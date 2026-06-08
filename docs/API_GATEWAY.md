# API Gateway — Spring Cloud Gateway

**Spring Cloud Gateway** (Spring Boot) is the edge layer: CORS, rate limiting, JWT, and
routing. It has **one upstream** — `normalization-service`. It never calls platform services
directly.

```
Gateway  →  normalization-service  →  facebook / instagram / line
```

> The gateway lives in a **separate repository**.

---

## Architecture

```
                    ┌──────────────────────────────────────────┐
  All traffic       │        Spring Cloud Gateway               │
  (GraphQL,         │  CORS · rate limit · JWT · routing        │
   webhooks)   ────▶└───────────────────┬──────────────────────┘
                                          │
                                          │  only connection
                                          ▼
                               normalization-service :8090
                                          │
                          ┌───────────────┼───────────────┐
                          ▼               ▼               ▼
                   facebook-svc   instagram-svc    line-svc
```

### Gateway responsibilities (do)

| Concern | Gateway |
|---------|---------|
| CORS | yes |
| Rate limiting | yes |
| JWT validation (GraphQL routes) | yes |
| Request logging / tracing | yes |
| Route **all** `/comms/**` → normalization-service | yes |
| Inject `X-Tenant-ID`, `X-Actor-ID`, `X-Request-ID` | yes (on authenticated routes) |

### Gateway responsibilities (do not)

| Concern | Where |
|---------|-------|
| Route to facebook / instagram / line | **never** — normalization only |
| GraphQL | normalization-service |
| Webhook proxy / platform routing | normalization-service |
| Webhook signature verification | platform service `internal/client/` |
| Vendor API calls | platform service `internal/client/` |

---

## Public URL layout

Every public path hits normalization-service first:

| Public path | Gateway forwards to | Auth |
|-------------|---------------------|------|
| `POST /comms/graphql` | `normalization-service/graphql` | JWT |
| `GET /comms/graphql` | `normalization-service/graphql` | JWT (playground, dev) |
| `POST /comms/facebook/api/v1/webhooks` | `normalization-service` (proxy) | no |
| `GET /comms/facebook/api/v1/webhooks` | `normalization-service` (proxy) | no |
| `POST /comms/instagram/api/v1/webhooks` | `normalization-service` (proxy) | no |
| `POST /comms/line/api/v1/webhooks` | `normalization-service` (proxy) | no |
| `GET /comms/healthz` | `normalization-service/healthz` | no |

```
# Client (Laconic app)
POST https://api.example.com/comms/graphql

# Vendor webhook (LINE) — still enters via gateway, proxied through normalization
POST https://api.example.com/comms/line/api/v1/webhooks
X-Line-Signature: ...
```

---

## Authentication

### Routes that require JWT

- `POST /comms/graphql`
- `GET /comms/graphql` (if playground enabled)

Gateway validates JWT, then forwards to normalization with:

| Header | Description |
|--------|-------------|
| `X-Tenant-ID` | Tenant UUID |
| `X-Actor-ID` | User / service account UUID |
| `X-Request-ID` | Correlation ID |

### Public routes (no JWT)

| Path | Reason |
|------|--------|
| `POST /comms/*/api/v1/webhooks` | Vendor callbacks |
| `GET /comms/facebook/api/v1/webhooks` | Meta `hub.challenge` |
| `GET /comms/healthz` | Liveness |

Normalization decides how to proxy webhooks; gateway only forwards.

---

## Webhook forwarding rules (gateway → normalization)

1. **Forward raw body** — do not parse or re-serialize JSON.
2. **Preserve vendor headers** — Meta: `X-Hub-Signature-256`; LINE: `X-Line-Signature`.
3. **No auth filter** on webhook paths.
4. **Do not route webhooks to platform services** — always send to normalization-service.
5. Short upstream timeout (e.g. 15s).

Normalization then proxies to the correct platform service unchanged. See
[NORMALIZATION_SERVICE.md](NORMALIZATION_SERVICE.md#webhook-proxy).

---

## Spring Cloud Gateway configuration

### Dependencies (`pom.xml`)

```xml
<dependency>
    <groupId>org.springframework.cloud</groupId>
    <artifactId>spring-cloud-starter-gateway</artifactId>
</dependency>
<dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-oauth2-resource-server</artifactId>
</dependency>
```

### Routes (`application.yml`)

The gateway needs **two route groups** — both point to normalization-service:

```yaml
spring:
  application:
    name: communication-api-gateway
  cloud:
    gateway:
      routes:
        # ── GraphQL (authenticated) ───────────────────────────
        - id: comms-graphql
          uri: ${NORMALIZATION_SERVICE_URI:http://normalization-service:8090}
          predicates:
            - Path=/comms/graphql
          filters:
            - RewritePath=/comms/graphql, /graphql

        # ── Webhooks + health (public) ────────────────────────
        - id: comms-proxy
          uri: ${NORMALIZATION_SERVICE_URI:http://normalization-service:8090}
          predicates:
            - Path=/comms/**
          filters:
            - RewritePath=/comms/(?<segment>.*), /proxy/${segment}
          metadata:
            public: true

      default-filters:
        - AddRequestHeader=X-Request-ID, #{T(java.util.UUID).randomUUID()}
        - name: RequestRateLimiter
          args:
            redis-rate-limiter.replenishRate: 100
            redis-rate-limiter.burstCapacity: 200

server:
  port: 8080
```

> `/comms/graphql` is listed first so it matches before the catch-all `/comms/**` proxy route.
> Normalization exposes `/proxy/{platform}/api/v1/webhooks` (or similar) to forward to platform
> services — see [NORMALIZATION_SERVICE.md](NORMALIZATION_SERVICE.md).

There are **no** `uri: http://facebook-service:8081` (or similar) entries in the gateway.

### JWT filter (conceptual)

```java
if (route.getMetadata().getOrDefault("public", false).equals(true)) {
    return chain.filter(exchange);
}
// validate JWT → set X-Tenant-ID, X-Actor-ID
```

### CORS (gateway only)

```yaml
spring:
  cloud:
    gateway:
      globalcors:
        cors-configurations:
          '[/**]':
            allowedOrigins: "${CORS_ALLOWED_ORIGINS}"
            allowedMethods: GET,POST,OPTIONS
            allowedHeaders: "*"
            allowCredentials: true
```

---

## Environment variables

| Variable | Description |
|----------|-------------|
| `SERVER_PORT` | Gateway port (default `8080`) |
| `SPRING_SECURITY_OAUTH2_RESOURCESERVER_JWT_ISSUER_URI` | JWT issuer |
| `NORMALIZATION_SERVICE_URI` | **Only** upstream — `http://normalization-service:8090` |
| `CORS_ALLOWED_ORIGINS` | Browser client origins |
| `REDIS_URL` | Rate limiter backend (optional) |

No `FACEBOOK_SERVICE_URI`, `LINE_SERVICE_URI`, etc. on the gateway — normalization holds those.

---

## Health checks

| Check | URL |
|-------|-----|
| Gateway | `GET /actuator/health` |
| Stack (via gateway) | `GET /comms/healthz` |

For Kubernetes, you may additionally probe platform services on the internal network
(`facebook-service:8081/api/v1/healthz`) — that is not a gateway route.

---

## Adding a new platform service

1. Add `app/<platform>-service/`.
2. Add REST client + resolvers in `normalization-service`.
3. Add webhook proxy route in `normalization-service` (not in the gateway).
4. Register vendor webhook URL: `https://api.example.com/comms/<platform>/api/v1/webhooks`.
5. Gateway config unchanged — still only points to normalization.

---

## Related docs

- [NORMALIZATION_SERVICE.md](NORMALIZATION_SERVICE.md) — GraphQL + webhook proxy
- [ARCHITECTURE.md](ARCHITECTURE.md) — system context
- [GO_CODING_STANDARD.md](GO_CODING_STANDARD.md) — Go conventions
