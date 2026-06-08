# Architecture Overview

High-level view of the communication platform: Spring Cloud Gateway at the edge,
**normalization-service** as the single backend entry point, and Go microservices per social
provider behind it.

**Golden rule:** the gateway talks **only** to `normalization-service`. It never calls
`facebook-service`, `instagram-service`, or `line-service` directly. All traffic — GraphQL,
webhooks, health — flows:

```
Gateway  →  normalization-service  →  platform service(s)
```

---

## System context

```
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ Laconic apps │   │ Meta / LINE  │   │   Operators  │
│ (HRMS, etc.) │   │  webhooks    │   │   / admins   │
└──────┬───────┘   └──────┬───────┘   └──────┬───────┘
       │                  │                  │
       └──────────────────┼──────────────────┘
                          ▼
              ┌───────────────────────┐
              │ Spring Cloud Gateway  │  Java / Spring Boot
              │  (separate repo)      │  CORS · rate limit · JWT
              └───────────┬───────────┘
                          │  only upstream
                          ▼
              ┌───────────────────────┐
              │ normalization-service │  Go / gqlgen + HTTP proxy
              │  GraphQL + webhooks   │
              └───────────┬───────────┘
                          │  REST (internal)
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
 ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
 │  facebook   │  │  instagram  │  │    line     │   Go / Fiber
 │  -service   │  │  -service   │  │  -service   │
 └──────┬──────┘  └──────┬──────┘  └──────┬──────┘
        │                │                │
        ▼                ▼                ▼
   PostgreSQL       PostgreSQL       PostgreSQL
        │                │                │
        └────────────────┼────────────────┘
                         ▼
              Meta Graph API / LINE Messaging API
```

---

## Traffic routing

| Traffic type | Public path | Flow |
|--------------|-------------|------|
| Laconic apps (GraphQL) | `POST /comms/graphql` | gateway → normalization → platform REST |
| Vendor webhooks | `POST /comms/<platform>/api/v1/webhooks` | gateway → normalization (proxy) → platform |
| Meta verify | `GET /comms/<platform>/api/v1/webhooks` | gateway → normalization (proxy) → platform |
| Health (via gateway) | `GET /comms/healthz` | gateway → normalization → fan-out to platforms |

> **Kubernetes note:** cluster internal probes may hit platform services directly on
> `:8081/:8082/:8083` for faster failure detection. That is ops-only and does not change
> the public rule: **no gateway → platform direct path**.

---

## Repository split

| Component | Location | Language | Role |
|-----------|----------|----------|------|
| API Gateway | Separate repo | Java, Spring Cloud Gateway | Edge: CORS, rate limit, JWT |
| Normalization | `app/normalization-service/` | Go, gqlgen | **Only** gateway upstream; GraphQL + webhook proxy |
| Platform services | `app/<platform>-service/` | Go, Fiber | REST, webhooks, vendor client, Postgres |
| Shared Go helpers | `pkg/` (minimal) | Go | Optional shared utilities |

---

## Layer responsibilities

| Layer | Does | Does not |
|-------|------|----------|
| **Gateway** | CORS, rate limiting, JWT, forward **all** `/comms/**` to normalization | Talk to platform services; GraphQL; business logic |
| **Normalization** | GraphQL, webhook HTTP proxy, route to correct platform service | Verify webhook signatures; call vendor APIs; persist data |
| **Platform service** | REST, webhook handling, persistence, vendor `internal/client/` | GraphQL; receive traffic directly from gateway |

---

## Request flows

### Outbound message via GraphQL

```
1. Client     →  POST /comms/graphql  { mutation sendMessage(platform: LINE, …) }
2. Gateway    →  JWT, CORS, rate limit; set X-Tenant-ID, X-Actor-ID
3. Gateway    →  normalization-service:8090/graphql
4. Resolver   →  POST line-service:8083/api/v1/messages  (+ trusted headers)
5. line-svc   →  repository → internal/client → LINE API
6. Resolver   →  GraphQL Message
7. Client     ←  GraphQL JSON
```

### Inbound webhook (through normalization)

```
1. LINE       →  POST /comms/line/api/v1/webhooks  (+ X-Line-Signature)
2. Gateway    →  forward to normalization-service (no JWT on webhook paths)
3. Normalization  →  proxy handler: forward raw body + headers unchanged
4. Normalization  →  POST line-service:8083/api/v1/webhooks
5. line-svc   →  client.VerifyWebhook → client.ParseWebhook → persist
6. line-svc   →  200 OK  →  normalization  →  gateway  →  LINE
```

### Meta webhook verification

```
1. Meta  →  GET /comms/facebook/api/v1/webhooks?hub.mode=subscribe&hub.challenge=…
2. Gateway  →  normalization-service
3. Normalization  →  proxy GET + query string to facebook-service:8081/api/v1/webhooks
4. facebook-service  →  echo hub.challenge
```

---

## Why this three-tier shape

| Choice | Rationale |
|--------|-----------|
| **Single gateway upstream** | One place to route; platform services stay private on the internal network |
| **Spring gateway** | Org-standard edge for CORS and rate limiting |
| **Normalization** | Unified GraphQL for clients; central routing to the right platform service |
| **Webhook proxy in normalization** | Keeps gateway dumb; preserves raw signed bodies via transparent HTTP forward |
| **Go platform services** | Isolated vendor logic, signature verification, persistence |

See [NORMALIZATION_SERVICE.md](NORMALIZATION_SERVICE.md) and [API_GATEWAY.md](API_GATEWAY.md).

---

## Data ownership

Each **platform service** owns its tables. **Normalization-service** is stateless in v1 —
no Postgres. It does not duplicate data; it routes and translates.

---

## Security summary

| Layer | Mechanism |
|-------|-----------|
| Gateway | JWT on GraphQL; no JWT on webhooks; CORS; rate limiting |
| Normalization | Trust gateway headers; GraphQL limits; webhook proxy (no body mutation) |
| Platform service | Webhook signature verification; trusted headers on REST |
| Network | Platform services not reachable from the public internet |

---

## Documentation index

| Document | Contents |
|----------|----------|
| [README.md](../README.md) | Quick start, service list |
| [ARCHITECTURE.md](ARCHITECTURE.md) | This file |
| [API_GATEWAY.md](API_GATEWAY.md) | Gateway → normalization only |
| [NORMALIZATION_SERVICE.md](NORMALIZATION_SERVICE.md) | GraphQL + webhook proxy |
| [GO_CODING_STANDARD.md](GO_CODING_STANDARD.md) | Go conventions |
