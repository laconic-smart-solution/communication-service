# Architecture

Multi-service communication and marketplace integration: **Spring Cloud Gateway** at the edge,
**Go platform microservices** behind it, one vendor per service.

**Golden rule:** the gateway routes `/comms/{platform}/**` directly to the matching platform
service and injects `X-Gateway-Secret` plus JWT identity headers.

```
Gateway  →  platform service  →  vendor API
```

---

## System context

```
┌─────────────┐     ┌─────────────┐     ┌─────────────────────┐
│ Laconic apps│     │ Meta / LINE │     │ Lazada sellers      │
└──────┬──────┘     └──────┬──────┘     └──────────┬──────────┘
       │ JWT REST          │ webhooks              │ OAuth
       ▼                   ▼                       ▼
┌──────────────────────────────────────────────────────────────┐
│              Spring Cloud Gateway (:8080)                     │
│  CORS · JWT · X-Gateway-Secret · path rewrite                │
└───────┬────────────────┬─────────────────┬───────────────────┘
        │                │                 │
        ▼                ▼                 ▼
 facebook-svc      line-svc          lazada-svc
   :8081             :8083              :8084
        │                │                 │
        └────────────────┴─────────────────┘
                         ▼
        Vendor APIs + Postgres
        (line-service is stateless; no DB)
```

---

## Request flows

### Authenticated REST (via gateway)

```
1. Client      POST /comms/lazada/api/v1/stores/connect  + JWT
2. Gateway     validate JWT → set X-Tenant-ID, X-Actor-ID, X-Gateway-Secret
3. lazada-svc  verify secret → business logic → Postgres
4. Response    { "success": true, "data": { ... } }
```

### Inbound webhook (LINE)

```
1. LINE         POST /comms/line/api/v1/webhooks?channel_secret=...&callback_url=...
2. Gateway      forward raw body + X-Line-Signature (no JWT)
3. line-svc     verify signature with query channel_secret → optional POST to callback_url → 200
```

Calling apps own channel secrets/tokens; line-service does not persist them.

### Local development (bypass gateway)

Platform services accept direct calls on internal ports when `X-Gateway-Secret` is set.
JWT headers optional (dev fallbacks in `pkg/utils`).

---

## Components

| Component | Path | Role |
|-----------|------|------|
| Gateway | `app/gateway/` | Edge routing, JWT, CORS |
| Platform services | `app/<platform>-service/` | Vendor integration + REST API |
| Shared envelope | `pkg/response`, `pkg/utils` | Uniform JSON responses |

---

## Security

| Layer | Mechanism |
|-------|-----------|
| Gateway | JWT on business routes; public webhooks + healthz |
| Gateway → platform | `X-Gateway-Secret` (constant-time compare in each service) |
| Platform → vendor | Caller-supplied tokens (LINE headers / Lazada query); OAuth tokens in DB where the service is stateful (Facebook); webhook HMAC in `internal/client/` |
| Tenancy | Stateful services scope by `tenant_id` from `X-Tenant-ID`; LINE proxy does not store tenant data |

---

## Response contract

All platform business endpoints return:

```json
{
  "success": true,
  "message": "optional",
  "data": {},
  "error": "only when success is false"
}
```

Health checks:

```json
{
  "success": true,
  "data": { "status": "ok", "service": "line-service" }
}
```

Webhooks: plain HTTP 200, no envelope.

---

## Related docs

- [API_GATEWAY.md](API_GATEWAY.md)
- [GO_CODING_STANDARD.md](GO_CODING_STANDARD.md)
