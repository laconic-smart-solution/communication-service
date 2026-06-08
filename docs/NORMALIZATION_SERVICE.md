# Normalization Service — GraphQL + Routing Hub

`normalization-service` is the **only** backend the gateway talks to. It sits between Spring
Cloud Gateway and the platform microservices:

```
Gateway  →  normalization-service  →  facebook / instagram / line
```

It provides:

1. **GraphQL** — unified API for Laconic apps (queries, mutations).
2. **HTTP proxy** — transparent forward of webhooks and other paths to the correct platform
   service (raw body + headers preserved).

Platform services are **not** reachable from the gateway. They run on the internal network
and accept traffic only from normalization-service (and K8s internal probes).

> Location: `app/normalization-service/` in this repository.

---

## Position in the stack

```
Laconic apps ──┐
Meta / LINE  ──┼──▶ Spring Cloud Gateway  ──▶  normalization-service
               │         (CORS, JWT)              │
               │                                  ├── /graphql  → resolvers → REST
               │                                  └── /proxy/*  → webhook proxy → REST
               │                                              │
               └──────────────────────────────────────────────┼──▶ platform services
                                                              ▼
                                                    Vendor APIs
```

---

## Responsibilities

### normalization-service (do)

| Concern | Description |
|---------|-------------|
| GraphQL schema | Unified `Channel`, `Message`, `Platform` types |
| Resolvers | GraphQL → REST calls to platform services |
| Webhook proxy | Forward `/comms/<platform>/api/v1/webhooks` with raw body + headers |
| Platform routing | Select facebook / instagram / line by path or GraphQL `platform` arg |
| Header propagation | Forward `X-Tenant-ID`, `X-Actor-ID`, `X-Request-ID` on REST calls |
| Response shaping | Map platform `APIResponse` → GraphQL types |

### normalization-service (do not)

| Concern | Where |
|---------|-------|
| CORS / rate limiting | Spring Cloud Gateway |
| JWT validation | Spring Cloud Gateway |
| Webhook **signature verification** | Platform service `internal/client/` |
| OAuth token storage | Platform service + Postgres |
| Vendor API calls | Platform service `internal/client/` |

---

## Tech stack

| Piece | Choice |
|-------|--------|
| Language | Go 1.22 |
| GraphQL | [gqlgen](https://github.com/99designs/gqlgen) |
| Webhook proxy | Fiber handler — `c.Body()` passthrough via `httputil.ReverseProxy` or raw `http.Client` |
| REST clients | `net/http` / resty — one client per platform service |

Stateless in v1 — no Postgres. Optional Redis later for caching.

---

## Project layout

```
app/normalization-service/
├─ cmd/main.go
├─ internal/
│  ├─ config/
│  ├─ graphql/
│  │  ├─ schema.graphqls
│  │  ├─ resolver/
│  │  └─ generated/
│  ├─ client/                 # REST clients → platform services
│  │  ├─ facebook_client.go
│  │  ├─ instagram_client.go
│  │  └─ line_client.go
│  ├─ proxy/                  # webhook + path proxy handlers
│  │  └── webhook_proxy.go
│  ├─ middleware/
│  └─ routes/
├─ pkg/utils/
├─ gqlgen.yml
├─ Dockerfile
└─ .env.example
```

Two entry paths inside normalization:

```
POST /graphql          →  resolver  →  client (REST)  →  platform service
POST /proxy/line/...   →  proxy     →  platform service (raw forward)
```

---

## Routes (normalization-service)

Gateway rewrites public paths before they arrive:

| Gateway public path | Arrives at normalization | Action |
|--------------------|--------------------------|--------|
| `POST /comms/graphql` | `POST /graphql` | GraphQL handler |
| `POST /comms/line/api/v1/webhooks` | `POST /proxy/line/api/v1/webhooks` | Proxy → line-service |
| `GET /comms/facebook/api/v1/webhooks` | `GET /proxy/facebook/api/v1/webhooks` | Proxy → facebook-service |
| `GET /comms/healthz` | `GET /healthz` | Aggregate platform health |

### Webhook proxy

The proxy handler must:

1. Read **raw body** from the incoming request.
2. Copy vendor headers (`X-Line-Signature`, `X-Hub-Signature-256`, etc.).
3. Forward to `{PLATFORM_SERVICE_URL}/api/v1/webhooks` without modifying the body.
4. Return the platform service status code and body as-is.

```go
// Conceptual — app/normalization-service/internal/proxy/webhook_proxy.go
func (p *WebhookProxy) Handle(c *fiber.Ctx, platform string) error {
    target := p.cfg.PlatformURL(platform) + "/api/v1/webhooks"
    // forward method, query string, raw body, signature headers
    return p.forward(c, target)
}
```

**Do not** parse webhook JSON in normalization. Signature verification happens in the
platform service.

---

## GraphQL schema (sketch)

```graphql
enum Platform {
  FACEBOOK
  INSTAGRAM
  LINE
}

type Channel {
  id: ID!
  platform: Platform!
  externalId: String!
  displayName: String
  createdAt: Time!
}

type Message {
  id: ID!
  platform: Platform!
  channelId: ID!
  recipientId: String!
  body: String
  status: String!
  externalId: String
  createdAt: Time!
}

type Query {
  channel(id: ID!, platform: Platform!): Channel
  channels(platform: Platform, page: Int, limit: Int): ChannelConnection!
  message(id: ID!, platform: Platform!): Message
}

type Mutation {
  sendMessage(input: SendMessageInput!): Message!
  connectChannel(input: ConnectChannelInput!): Channel!
  revokeChannel(id: ID!, platform: Platform!): Boolean!
}

input SendMessageInput {
  platform: Platform!
  channelId: ID!
  recipientId: String!
  text: String
  mediaUrl: String
}
```

---

## Downstream REST mapping (GraphQL resolvers)

| GraphQL operation | HTTP call |
|-------------------|-----------|
| `channels(platform: LINE)` | `GET http://line-service:8083/api/v1/channels` |
| `sendMessage(platform: FACEBOOK, …)` | `POST http://facebook-service:8081/api/v1/messages` |
| `revokeChannel(platform: INSTAGRAM, …)` | `DELETE http://instagram-service:8082/api/v1/channels/:id` |

| Env var | Default |
|---------|---------|
| `FACEBOOK_SERVICE_URL` | `http://facebook-service:8081` |
| `INSTAGRAM_SERVICE_URL` | `http://instagram-service:8082` |
| `LINE_SERVICE_URL` | `http://line-service:8083` |

---

## Request flows

### Send message via GraphQL

```
1. Client       →  POST /comms/graphql
2. Gateway      →  JWT, headers  →  normalization:8090/graphql
3. Resolver     →  POST line-service:8083/api/v1/messages
4. line-service →  vendor API, persist, return APIResponse
5. Resolver     →  GraphQL Message
6. Client       ←  GraphQL JSON
```

### Inbound webhook (via normalization)

```
1. LINE         →  POST /comms/line/api/v1/webhooks
2. Gateway      →  normalization:8090/proxy/line/api/v1/webhooks
3. Proxy        →  POST line-service:8083/api/v1/webhooks  (raw body + X-Line-Signature)
4. line-service →  VerifyWebhook → ParseWebhook → persist
5. 200 OK       ←  back through normalization ← gateway ← LINE
```

---

## Environment variables

```bash
PORT=8090
LOG_LEVEL=info

FACEBOOK_SERVICE_URL=http://facebook-service:8081
INSTAGRAM_SERVICE_URL=http://instagram-service:8082
LINE_SERVICE_URL=http://line-service:8083

GATEWAY_SECRET=          # optional HMAC for internal trust
PLAYGROUND_ENABLED=false
```

Register webhooks with vendors using the **gateway** public URL:

```
https://api.example.com/comms/line/api/v1/webhooks
```

---

## Security

- Trust `X-Tenant-ID` / `X-Actor-ID` only from the gateway (private network / mTLS).
- Platform services should **reject** requests not from normalization (internal network ACL
  or shared `GATEWAY_SECRET` header on normalization → platform calls).
- GraphQL depth/complexity limits in gqlgen.
- Webhook proxy: zero body mutation.

---

## Adding a new platform

1. Add `app/<platform>-service/`.
2. Add `internal/client/<platform>_client.go` for GraphQL resolvers.
3. Register proxy path `/proxy/<platform>/**` → new service URL.
4. Extend GraphQL `Platform` enum and resolvers.
5. Gateway config **unchanged** — still only routes to normalization.

---

## Related docs

- [ARCHITECTURE.md](ARCHITECTURE.md) — system context
- [API_GATEWAY.md](API_GATEWAY.md) — gateway routes (normalization only)
- [GO_CODING_STANDARD.md](GO_CODING_STANDARD.md) — Go conventions
