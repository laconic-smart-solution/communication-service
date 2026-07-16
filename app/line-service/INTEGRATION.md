# LINE Service — Integration Guide

`line-service` is a **stateless** LINE Messaging API proxy on port **8083**.
It does **not** store channels, messages, or LINE credentials.
Calling applications own `LINE_CHANNEL_SECRET` and `LINE_CHANNEL_ACCESS_TOKEN` and pass them on each request.

Gateway path: `/comms/line/**` (rewrite strips `/comms/line`).

---

## Architecture

```
Calling app                    line-service                 LINE
     │                              │                         │
     │  POST /messages              │                         │
     │  + X-Line-Channel-Access-Token                         │
     │─────────────────────────────►│  push message           │
     │                              │────────────────────────►│
     │◄─────────────────────────────│                         │
     │                              │                         │
     │  (optional) callback         │  webhook                │
     │◄─────────────────────────────│◄────────────────────────│
     │                              │  ?channel_secret=...    │
```

| Owned by calling app | Owned by line-service |
|----------------------|------------------------|
| Channel secret | Gateway shared secret |
| Channel access token | HTTP proxy to LINE |
| Message history / tenant data | Signature verify + optional event forward |

---

## Prerequisites

| Requirement | Version |
|-------------|---------|
| Go | 1.22+ |
| LINE Messaging API channel | Channel secret + long-lived access token |
| Gateway | Same `GATEWAY_SECRET` as this service |

**No PostgreSQL** is required for line-service.

---

## Environment variables

```bash
cp app/line-service/.env.example app/line-service/.env
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LINE_SERVICE_PORT` | no | `8083` | HTTP listen port |
| `PORT` | no | — | Fallback if `LINE_SERVICE_PORT` is unset |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, `error` |
| `GATEWAY_SECRET` | yes | — | Must match gateway (`X-Gateway-Secret`) |

Do **not** set `LINE_CHANNEL_SECRET` or `LINE_CHANNEL_ACCESS_TOKEN` on this service.
Those values live in your application and are sent per request.

### Example `.env`

```bash
LINE_SERVICE_PORT=8083
LOG_LEVEL=info
GATEWAY_SECRET=hello
```

---

## Run locally

```bash
# from communication-service root
go run ./app/line-service/cmd/main.go
```

Health:

```bash
curl http://localhost:8083/api/v1/healthz
# {"success":true,"data":{"status":"ok","service":"line-service"}}
```

Via gateway (default local gateway port may be `8099` or `8080` depending on env):

```bash
curl http://localhost:8099/comms/line/api/v1/healthz
```

---

## API

Envelope for business endpoints:

```json
{ "success": true, "data": { ... }, "error": "only when success is false" }
```

### 1. Send text message (outbound)

`POST /api/v1/messages`  
Via gateway: `POST /comms/line/api/v1/messages`

**Headers (required):**

| Header | Description |
|--------|-------------|
| `X-Gateway-Secret` | Shared secret (injected by gateway; required if calling service directly) |
| `X-Tenant-ID` | Tenant UUID (gateway injects from JWT; optional locally with fallback) |
| `X-Actor-ID` | Actor UUID (gateway injects from JWT) |
| `X-Line-Channel-Access-Token` | LINE channel access token from **your** app |

**Body:**

```json
{
  "recipient_id": "Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "text": "Hello from my app"
}
```

**Success (201):**

```json
{
  "success": true,
  "data": {
    "recipient_id": "Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "status": "sent"
  }
}
```

**cURL (direct to service):**

```bash
curl -X POST http://localhost:8083/api/v1/messages \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002" \
  -H "X-Line-Channel-Access-Token: YOUR_CHANNEL_ACCESS_TOKEN" \
  -d '{"recipient_id":"Uxxxxxxxx","text":"Hello"}'
```

**cURL (via gateway):**

```bash
curl -X POST http://localhost:8099/comms/line/api/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT" \
  -H "X-Line-Channel-Access-Token: YOUR_CHANNEL_ACCESS_TOKEN" \
  -d '{"recipient_id":"Uxxxxxxxx","text":"Hello"}'
```

### 2. Inbound webhook

`POST /api/v1/webhooks`  
Via gateway: `POST /comms/line/api/v1/webhooks`  
Public (no JWT / no gateway secret required).

LINE cannot send your app headers, so the **channel secret** is supplied on the URL you register in the LINE Developers Console:

```
https://<your-gateway-host>/comms/line/api/v1/webhooks?channel_secret=<CHANNEL_SECRET>&callback_url=<URL_ENCODED_HTTPS_CALLBACK>
```

| Query | Required | Description |
|-------|----------|-------------|
| `channel_secret` | yes | LINE channel secret (used to verify `X-Line-Signature`) |
| `callback_url` | no | HTTPS URL on **your** app; receives parsed events as JSON |

**LINE Console setup:**

1. Open [LINE Developers Console](https://developers.line.biz/console/).
2. Messaging API → Webhook URL → paste the URL above (secret + optional callback).
3. Enable webhook; issue a long-lived **Channel access token** and store it in **your** application.

**Behavior:**

1. Verify HMAC-SHA256 signature with `channel_secret`.
2. Parse text message events.
3. If `callback_url` is set, `POST` to your app:

```json
{
  "events": [
    {
      "type": "message",
      "user_id": "Uxxxxxxxx",
      "message_id": "…",
      "text": "hi"
    }
  ]
}
```

4. Return HTTP `200` to LINE when the signature is valid.

Treat the webhook URL as **sensitive** (it embeds the channel secret). Use HTTPS only.

---

## Integration checklist (calling app)

1. Store `channel_secret` and `channel_access_token` in your own database/secrets store.
2. On send: pass token as `X-Line-Channel-Access-Token`; never persist tokens in line-service.
3. Register webhook URL with your `channel_secret` (and optional `callback_url`) in LINE Console.
4. Implement `callback_url` handler if you need inbound messages.
5. Use the same `GATEWAY_SECRET` / JWT flow as other `/comms/**` services.

---

## Errors

| Condition | Typical response |
|-----------|------------------|
| Missing access token header | `400` — `X-Line-Channel-Access-Token header is required` |
| Invalid / expired LINE token | `400` — `invalid line channel token` |
| LINE rate limit | `400` — `line rate limit exceeded` |
| Missing `channel_secret` on webhook | `400` — `channel_secret query parameter is required` |
| Bad `X-Line-Signature` | `400` — `invalid webhook signature` |
| `callback_url` not HTTPS | `400` — `callback_url must be a valid https URL` |
| Missing gateway secret (non-webhook) | `403` — `forbidden` |

---

## Security notes

- Prefer calling through the gateway with JWT.
- Never log `X-Line-Channel-Access-Token` or `channel_secret`.
- Prefer HTTPS for gateway and for `callback_url`.
- Rotate LINE tokens in the **calling app**; redeploying line-service is not required for credential rotation.
