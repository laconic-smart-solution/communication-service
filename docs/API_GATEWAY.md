# API Gateway — Spring Cloud Gateway

**Spring Cloud Gateway** is the edge layer: CORS, JWT validation, and routing directly to
platform Go services.

```
Gateway  →  facebook / line / lazada services
```

Location: [`app/gateway/`](../app/gateway/)

---

## Architecture

```
                    ┌──────────────────────────────────────────┐
  Clients / vendors │        Spring Cloud Gateway               │
                    │  CORS · JWT · X-Gateway-Secret injection  │
                    └───────────┬──────────────┬────────────────┘
                                │              │
                    ┌───────────▼──┐  ┌────────▼────────┐
                    │ facebook-svc │  │ line / lazada   │
                    └──────────────┘  └─────────────────┘
```

### Gateway responsibilities

| Concern | Gateway |
|---------|---------|
| CORS | yes |
| JWT validation (business API routes) | yes |
| Inject `X-Gateway-Secret` on all outbound requests | yes |
| Inject `X-Tenant-ID`, `X-Actor-ID` from JWT | yes |
| Route `/comms/{platform}/**` to platform services | yes |
| Webhook signature verification | no — platform `internal/client/` |
| Vendor API calls | no — platform services |

---

## Public URL layout

| Public path | Upstream | Auth |
|-------------|----------|------|
| `/comms/facebook/**` | `facebook-service:8081/**` | JWT on `/api/v1/**` except webhooks + healthz |
| `/comms/line/**` | `line-service:8083/**` | same |
| `/comms/lazada/**` | `lazada-service:8084/**` | same |
| `/comms/telegram/**` | `telegram-service:8087/**` | same |
| `GET /comms/**/api/v1/healthz` | platform service | public |
| `POST/GET /comms/**/api/v1/webhooks` | platform service | public |

Path rewrite example:

```
GET /comms/lazada/api/v1/stores  →  lazada-service/api/v1/stores
```

---

## Authentication

### JWT required

All `/comms/**/api/v1/**` routes except webhooks and health checks.

Gateway forwards identity headers after JWT validation:

| Header | Description |
|--------|-------------|
| `X-Tenant-ID` | Tenant UUID |
| `X-Actor-ID` | User UUID |
| `X-Gateway-Secret` | Shared secret (always injected) |

### Public routes

| Path | Reason |
|------|--------|
| `POST/GET /comms/**/api/v1/webhooks` | Vendor callbacks |
| `GET /comms/**/api/v1/healthz` | Liveness |

---

## Configuration

[`application.yml`](../app/gateway/src/main/resources/application.yml):

```yaml
spring.cloud.gateway.routes:
  - id: comms-facebook
    uri: ${FACEBOOK_SERVICE_URI:http://localhost:8081}
    predicates: [Path=/comms/facebook/**]
    filters:
      - RewritePath=/comms/facebook/(?<segment>.*), /${segment}
  # line, lazada — same pattern
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `GATEWAY_PORT` | Gateway port (default `8080`) |
| `GATEWAY_SECRET` | Injected as `X-Gateway-Secret` |
| `GATEWAY_JWT_HMAC_SECRET` | JWT HMAC validation |
| `FACEBOOK_SERVICE_URI` | Upstream facebook-service |
| `LINE_SERVICE_URI` | Upstream line-service |
| `TELEGRAM_SERVICE_URI` | Upstream telegram-service |
| `LAZADA_SERVICE_URI` | Upstream lazada-service |
| `GATEWAY_CORS_ALLOWED_ORIGINS` | Browser origins |

---

## Response format

Platform services return a shared JSON envelope via root `pkg/response`:

```json
{ "success": true, "data": { ... } }
```

Webhooks return plain `200 OK` (vendor requirement).

---

## Health checks

| Check | URL |
|-------|-----|
| Gateway | `GET /actuator/health` |
| Line (via gateway) | `GET /comms/line/api/v1/healthz` |
| Telegram (via gateway) | `GET /comms/telegram/api/v1/healthz` |
| Lazada (via gateway) | `GET /comms/lazada/api/v1/healthz` |

---

## Adding a new platform service

1. Add `app/<platform>-service/`.
2. Add a route block in `application.yml`.
3. Add `/<PLATFORM>_SERVICE_URI` env var to gateway and docker-compose.
4. Register vendor webhooks at `https://api.example.com/comms/<platform>/api/v1/webhooks`.

---

## Related docs

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [GO_CODING_STANDARD.md](GO_CODING_STANDARD.md)
