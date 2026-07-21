# WeChat Service — Setup Guide

`wechat-service` is a **stateless** WeChat Official Account (公众号) proxy on port **8090**.
It does **not** store AppID, AppSecret, access tokens, or OpenIDs. Calling apps (e.g. Mosonic Nest) own credentials and pass them per request.

Gateway path: `/comms/wechat/**` (rewrite strips `/comms/wechat`).

Mosonic end-to-end (Nest env, CONNECT UI, Prisma): see [`neutral-tex/docs/setup-wechat.md`](../../../neutral-tex/docs/setup-wechat.md).  
API contract summary: [`INTEGRATION.md`](./INTEGRATION.md).

---

## Prerequisites

| Requirement | Version / notes |
|-------------|-----------------|
| Go | 1.22+ |
| WeChat Official Account | AppID + AppSecret; public HTTPS for webhooks |
| Gateway | Same `GATEWAY_SECRET` as this service |

**No PostgreSQL** is required for wechat-service.

**Constraints**

- Outbound text uses `cgi-bin/message/custom/send` (customer-service API) — typically only within WeChat’s **~48h** interaction window after the user messages you (or certain events).
- API host is `api.weixin.qq.com`; this machine must reach China WeChat endpoints.
- Not WeCom / Mini Program.

---

## Architecture position

```
Calling app (Nest)                 wechat-service (:8090)          WeChat OA
     │                                    │                            │
     │  POST /api/v1/messages             │                            │
     │  + X-WeChat-Access-Token           │  custom/send               │
     │───────────────────────────────────►│───────────────────────────►│
     │                                    │                            │
     │  GET /api/v1/qr?scene_str=…        │  qrcode/create + showqrcode│
     │───────────────────────────────────►│───────────────────────────►│
     │◄───────────────────────────────────│                            │
     │                                    │                            │
     │  (optional) callback_url           │  XML webhook               │
     │◄───────────────────────────────────│◄───────────────────────────│
```

| Owned by calling app | Owned by wechat-service |
|----------------------|-------------------------|
| AppID / AppSecret / access_token | Gateway shared secret |
| OpenID ↔ user linking | HTTP proxy to WeChat |
| Welcome / CONNECT logic | Signature verify + optional event forward |
| Webhook business handling | Scene / fallback QR rendering |

---

## Environment variables

```bash
cp app/wechat-service/.env.example app/wechat-service/.env
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `WECHAT_SERVICE_PORT` | no | `8090` | HTTP listen port |
| `PORT` | no | — | Fallback if `WECHAT_SERVICE_PORT` is unset |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, `error` |
| `GATEWAY_SECRET` | yes | — | Must match gateway (`X-Gateway-Secret`) |
| `WECHAT_BATCH_MIN_INTERVAL_MS` | no | `250` | Delay between batch chunks |
| `WECHAT_BATCH_MAX_RECIPIENTS` | no | `500` | Max recipients per batch request |

Do **not** set `WECHAT_APP_ID` / `WECHAT_APP_SECRET` on this service. Those live in the calling app; Nest caches `access_token` and sends `X-WeChat-Access-Token`.

### Example `.env` (local)

```bash
WECHAT_SERVICE_PORT=8090
LOG_LEVEL=info
GATEWAY_SECRET=hello
```

Root Compose / gateway also need:

```bash
WECHAT_SERVICE_URI=http://wechat-service:8090   # docker
# or http://localhost:8090 for local gateway
```

---

## WeChat Official Account (公众平台) setup

1. Open [微信公众平台](https://mp.weixin.qq.com/) → your OA → **开发** → **基本配置**.
2. Copy **AppID** and **AppSecret** into the **calling app** (not this service).
3. **服务器配置** (for inbound):
   - Prefer pointing WeChat at your product webhook (Mosonic: `https://<host>/wechat/webhook/direct`).
   - Or point at this proxy and set `callback_url` to your app (see Webhooks below).
   - **Token**: shared string used for SHA1 signature verify (`X-WeChat-Token` on webhook requests).
   - Encoding: **明文** is enough for Mosonic; optional safe mode needs EncodingAESKey on the caller / headers.
4. Note OA **微信号 / username** if you use fallback deep-link QR (`oa_username` query).

---

## Run locally

### Windows

```bat
cd communication-service\app\wechat-service
copy .env.example .env
run.bat
```

### Go directly (from `communication-service` root)

```bash
go run ./app/wechat-service/cmd
```

### Health

```bash
curl http://localhost:8090/api/v1/healthz
```

Via gateway (optional):

```bash
curl http://localhost:8099/comms/wechat/api/v1/healthz
```

### Full stack (Docker Compose)

From `communication-service`:

```bash
cp .env.example .env
docker compose up --build wechat-service gateway
```

| Service | URL |
|---------|-----|
| wechat-service | http://localhost:8090 |
| Gateway | http://localhost:8099 |
| Health via gateway | http://localhost:8099/comms/wechat/api/v1/healthz |

---

## API reference

Business routes require `X-Gateway-Secret` plus WeChat credentials as noted. Health and webhook verify/callback are public (signature / token as documented).

### Headers (outbound send / QR)

| Header | Description |
|--------|-------------|
| `X-Gateway-Secret` | Shared secret |
| `X-Tenant-ID` | Tenant UUID |
| `X-Actor-ID` | Caller identity UUID |
| `X-WeChat-Access-Token` | Cached `access_token` from calling app |

Webhook verify / forward (when using this service as OA URL):

| Header / query | Description |
|----------------|-------------|
| `X-WeChat-Token` | OA server config Token (signature) |
| `X-WeChat-Encoding-Aes-Key` | Optional safe mode |
| `callback_url` | Where to POST normalized events after parse |

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/healthz` | no | Liveness |
| `GET` | `/api/v1/webhooks` | Token signature | OA URL verify (`echostr`) |
| `POST` | `/api/v1/webhooks` | Token signature | XML inbound → optional `callback_url` |
| `POST` | `/api/v1/messages` | Gateway + access token | Send text to OpenID |
| `POST` | `/api/v1/messages/batch` | Gateway + access token | Batch send |
| `GET` | `/api/v1/qr` | Gateway (+ token for scene) | Scene QR PNG/JSON or OA fallback |

### Send message

```bash
curl -X POST http://localhost:8090/api/v1/messages \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002" \
  -H "X-WeChat-Access-Token: <access_token>" \
  -d '{
    "recipient_id": "<openid>",
    "text": "Hello from wechat-service"
  }'
```

Proxies to `POST https://api.weixin.qq.com/cgi-bin/message/custom/send`.

### Scene QR (CONNECT / scan-to-link)

```bash
curl "http://localhost:8090/api/v1/qr?scene_str=CONNECT%20ABC123&format=json&size=280" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002" \
  -H "X-WeChat-Access-Token: <access_token>"
```

Creates a temporary `QR_STR_SCENE` ticket and returns PNG (default) or JSON (`qr_image` data URI + `chat_url`).

### Fallback OA QR (no scene)

```bash
curl "http://localhost:8090/api/v1/qr?oa_username=gh_xxxxx&format=json" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002"
```

Renders a deep-link QR; user must still send `CONNECT <code>` in chat unless you use scene QR.

---

## Webhooks (optional forwarder)

If OA server URL points at **this** service:

```
https://<gateway>/comms/wechat/api/v1/webhooks?callback_url=https://<app>/wechat/webhook?token=<SHARED>
```

Pass `X-WeChat-Token` (and optional EncodingAESKey) so the service can verify GET `echostr` and POST XML, then forward normalized events to `callback_url`.

For Mosonic, prefer **direct** Nest webhook (no forwarder):

```
https://<public>/wechat/webhook/direct
```

---

## Troubleshooting

| Symptom | Likely cause |
|---------|----------------|
| Gateway 401 | `GATEWAY_SECRET` mismatch |
| Send / QR 400 missing token | Missing `X-WeChat-Access-Token` |
| WeChat API `45015` / similar | Outside 48h customer-service window |
| Invalid credential | Stale `access_token` or wrong AppID/Secret on caller |
| Webhook verify fails | Wrong Token or Nest/proxy not reachable over HTTPS |
| Unreachable from Nest | Wrong `WECHAT_SERVICE_URL` / service not running |

---

## Related

- [`INTEGRATION.md`](./INTEGRATION.md) — short contract
- Mosonic: [`docs/setup-wechat.md`](../../../neutral-tex/docs/setup-wechat.md)
- Sibling proxies: `line-service`, `telegram-service`, `viber-service`, `whatsapp-service`
