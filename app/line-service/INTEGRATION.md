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
     │  POST /messages/batch        │  multicast (chunks)     │
     │─────────────────────────────►│────────────────────────►│
     │◄─────────────────────────────│  rate-limit + retry     │
     │                              │                         │
     │  GET /qr?basic_id=@…         │  (no LINE call)         │
     │─────────────────────────────►│                         │
     │◄─────────────────────────────│                         │
     │        image/png             │                         │
     │                              │                         │
     │  (optional) callback         │  webhook                │
     │◄─────────────────────────────│◄────────────────────────│
     │                              │  ?channel_secret=...    │
     │                              │                         │
     │  follow ⇒ POST /messages (auto-greet, driven by you)   │
     │─────────────────────────────►│────────────────────────►│
```

| Owned by calling app | Owned by line-service |
|----------------------|------------------------|
| Channel secret | Gateway shared secret |
| Channel access token | HTTP proxy to LINE |
| Message history / tenant data | Signature verify + optional event forward |
| Welcome / auto-reply copy | Add-friend QR rendering |

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
| `LINE_BATCH_MIN_INTERVAL_MS` | no | `250` | Delay between multicast chunks |
| `LINE_BATCH_MAX_RECIPIENTS` | no | `10000` | Max unique recipient IDs per batch request |

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
# from app/line-service (Windows)
run.bat

# or from communication-service root
go run ./app/line-service/cmd
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

### 2. Send batch / multicast (outbound)

`POST /api/v1/messages/batch`  
Via gateway: `POST /comms/line/api/v1/messages/batch`

Sends the same text to many LINE user IDs using LINE **Multicast**
(`POST https://api.line.me/v2/bot/message/multicast`).

**Behavior:**

1. Trim/dedupe `recipient_ids` (empty strings ignored).
2. Reject if count exceeds `LINE_BATCH_MAX_RECIPIENTS` (default **10000**).
3. Chunk into groups of **500** (LINE multicast limit).
4. Send chunks sequentially with a minimum interval (`LINE_BATCH_MIN_INTERVAL_MS`, default **250**).
5. On HTTP `429` from LINE: retry up to 3 times with exponential backoff (1s / 2s / 4s), then mark that chunk failed and continue.
6. Return a summary; partial success is possible (`sent` + `failed`).

line-service still does **not** own a recipient database — your app loads LINE user IDs and passes them here.

**Headers (required):** same as single send (`X-Gateway-Secret`, `X-Tenant-ID`, `X-Actor-ID`, `X-Line-Channel-Access-Token`).

**Body:**

```json
{
  "recipient_ids": [
    "Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "Uyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"
  ],
  "text": "Global announcement"
}
```

**Success (201):**

```json
{
  "success": true,
  "data": {
    "total": 2,
    "sent": 2,
    "failed": 0,
    "chunk_count": 1,
    "failed_recipient_ids": []
  }
}
```

`failed_recipient_ids` is omitted when empty; when present it lists IDs from failed chunks only.

**cURL (direct to service):**

```bash
curl -X POST http://localhost:8083/api/v1/messages/batch \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002" \
  -H "X-Line-Channel-Access-Token: YOUR_CHANNEL_ACCESS_TOKEN" \
  -d '{"recipient_ids":["Uxxxxxxxx","Uyyyyyyyy"],"text":"Hello everyone"}'
```

**Optional env (line-service):**

| Variable | Default | Description |
|----------|---------|-------------|
| `LINE_BATCH_MIN_INTERVAL_MS` | `250` | Delay between multicast chunks |
| `LINE_BATCH_MAX_RECIPIENTS` | `10000` | Max unique IDs per request |

### 3. Add-friend QR code

`GET /api/v1/qr`  
Via gateway: `GET /comms/line/api/v1/qr`

Renders a QR code for an official account's add-friend page
(`https://line.me/R/ti/p/%40<basic_id>`). Scanning it opens the OA profile in LINE.
No LINE credentials are involved — this endpoint calls no LINE API.

**Headers (required):**

| Header | Description |
|--------|-------------|
| `X-Gateway-Secret` | Shared secret (injected by gateway; required if calling service directly) |
| `X-Tenant-ID` | Tenant UUID (optional locally with fallback) |

**Query:**

| Query | Required | Default | Description |
|-------|----------|---------|-------------|
| `basic_id` | yes | — | OA basic ID, e.g. `@136laxkx` (a missing `@` is added) |
| `size` | no | `256` | PNG width/height in px, `64`–`1024` |
| `format` | no | — | `json` returns the envelope below; otherwise raw `image/png` |

**PNG (default):** responds `image/png` with `Cache-Control: public, max-age=86400`.

**JSON (`format=json`, 200):**

```json
{
  "success": true,
  "data": {
    "basic_id": "@136laxkx",
    "add_friend_url": "https://line.me/R/ti/p/%40136laxkx",
    "qr_image": "data:image/png;base64,iVBORw0KGgo..."
  }
}
```

**cURL:**

```bash
curl -H "X-Gateway-Secret: hello" \
  "http://localhost:8083/api/v1/qr?basic_id=@136laxkx&size=300" --output qr.png
```

### 4. Inbound webhook

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
2. Parse events. Forwarded types:

   | `type` | Meaning | Fields |
   |--------|---------|--------|
   | `message` | Inbound **text** message (other message types are dropped) | `user_id`, `reply_token`, `message_id`, `text` |
   | `follow` | User added the OA as a friend | `user_id`, `reply_token` |
   | `unfollow` | User blocked / removed the OA | `user_id` |

3. If `callback_url` is set **and** at least one event survived parsing, `POST` to your app:

```json
{
  "events": [
    { "type": "follow", "user_id": "Uxxxxxxxx", "reply_token": "…" },
    { "type": "message", "user_id": "Uxxxxxxxx", "reply_token": "…", "message_id": "…", "text": "hi" }
  ]
}
```

4. Return HTTP `200` to LINE when the signature is valid.

> **Adding a friend needs no approval.** LINE has no "accept friend request" API —
> a `follow` is already final when it reaches you. To auto-greet a new friend,
> react to the `follow` event by calling `POST /api/v1/messages` with `user_id`
> as `recipient_id`. line-service holds no token and so never replies on your behalf.

**Note:** line-service sends **no auth header** on the callback. Put a secret token
on your `callback_url` query string and verify it in your handler.

Treat the webhook URL as **sensitive** (it embeds the channel secret). Use HTTPS only.

---

## Integration checklist (calling app)

1. Store `channel_secret` and `channel_access_token` in your own database/secrets store.
2. On send: pass token as `X-Line-Channel-Access-Token`; never persist tokens in line-service.
3. Register webhook URL with your `channel_secret` (and optional `callback_url`) in LINE Console.
4. Implement `callback_url` handler if you need inbound messages or `follow` auto-greeting;
   protect it with a token on the query string.
5. Show the add-friend QR with `GET /api/v1/qr?basic_id=@yourid`.
6. For broadcasts: load your stored LINE user IDs and call `POST /api/v1/messages/batch`.
7. Use the same `GATEWAY_SECRET` / JWT flow as other `/comms/**` services.

---

## Errors

| Condition | Typical response |
|-----------|------------------|
| Missing access token header | `400` — `X-Line-Channel-Access-Token header is required` |
| Invalid / expired LINE token | `400` — `invalid line channel token` |
| LINE rate limit (single send) | `400` — `line rate limit exceeded` |
| Batch exceeds max recipients | `400` — `recipient_ids exceeds maximum of N` |
| Batch chunk still rate-limited after retries | counted in `failed` / `failed_recipient_ids` (HTTP 201 with partial success) |
| Missing `basic_id` on QR | `400` — `basic_id query parameter is required` |
| Malformed `basic_id` | `400` — `basic_id must look like @136laxkx` |
| QR `size` out of range | `400` — `size must be between 64 and 1024` |
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
