# Telegram Service — Integration Guide

`telegram-service` is a **stateless** Telegram Bot API proxy on port **8087**.
It does **not** store bots, chats, or bot tokens.
Calling applications own the bot token and webhook secret and pass them on each request.

Gateway path: `/comms/telegram/**` (rewrite strips `/comms/telegram`).

---

## Architecture

```
Calling app                    telegram-service              Telegram
     │                              │                         │
     │  POST /messages              │                         │
     │  + X-Telegram-Bot-Token      │  sendMessage            │
     │─────────────────────────────►│────────────────────────►│
     │                              │                         │
     │  POST /messages/batch        │  sendMessage (loop)     │
     │─────────────────────────────►│────────────────────────►│
     │                              │  rate-limit + retry     │
     │                              │                         │
     │  GET /qr?bot_username=…      │  (no Telegram call)     │
     │─────────────────────────────►│                         │
     │◄─────────────────────────────│  image/png or JSON      │
     │                              │                         │
     │  (optional) callback         │  webhook update         │
     │◄─────────────────────────────│◄────────────────────────│
     │                              │  ?secret_token=...      │
```

| Owned by calling app | Owned by telegram-service |
|----------------------|---------------------------|
| Bot token | Gateway shared secret |
| Webhook secret token | HTTP proxy to Telegram |
| Chat IDs / tenant data | Secret verify + optional event forward |
| Welcome / auto-reply copy | Bot deep-link QR rendering |

---

## Environment variables

```bash
cp app/telegram-service/.env.example app/telegram-service/.env
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TELEGRAM_SERVICE_PORT` | no | `8087` | HTTP listen port |
| `LOG_LEVEL` | no | `info` | Log level |
| `GATEWAY_SECRET` | yes | — | Must match gateway |
| `TELEGRAM_BATCH_MIN_INTERVAL_MS` | no | `50` | Delay between sends |
| `TELEGRAM_BATCH_MAX_RECIPIENTS` | no | `10000` | Max unique chat IDs per batch |

Do **not** set the bot token on this service — pass `X-Telegram-Bot-Token` per request.

---

## Run locally

```bat
rem from app/telegram-service
run.bat
```

```bash
go run ./app/telegram-service/cmd
curl http://localhost:8087/api/v1/healthz
curl http://localhost:8099/comms/telegram/api/v1/healthz
```

---

## API

### 1. Send text

`POST /api/v1/messages`  
Via gateway: `POST /comms/telegram/api/v1/messages`

| Header | Description |
|--------|-------------|
| `X-Gateway-Secret` | Shared secret |
| `X-Tenant-ID` / `X-Actor-ID` | Identity |
| `X-Telegram-Bot-Token` | Bot token from **your** app |

```json
{ "recipient_id": "123456789", "text": "Hello from my app" }
```

`recipient_id` is the Telegram **chat_id** (user or group).

### 2. Batch send

`POST /api/v1/messages/batch`

Telegram has no multicast API — the service sends `sendMessage` per chat with pacing and 429 retries.

```json
{
  "recipient_ids": ["111", "222"],
  "text": "Announcement"
}
```

### 3. Bot QR / deep link

`GET /api/v1/qr?bot_username=MyCoolBot&size=256`  
Optional: `start=<payload>`, `format=json`

Renders a QR for `https://t.me/<bot>` (or with `?start=`). No Telegram API call.

### 4. Inbound webhook

`POST /api/v1/webhooks` (public)

Register in BotFather / `setWebhook`:

```
https://<gateway>/comms/telegram/api/v1/webhooks?secret_token=<SECRET>&callback_url=<HTTPS_CALLBACK>
```

Use the same `secret_token` as Telegram’s `secret_token` so Telegram sends
`X-Telegram-Bot-Api-Secret-Token`.

Forwarded event types (same shape as LINE):

| `type` | Meaning |
|--------|---------|
| `message` | Inbound text |
| `follow` | User started bot (`/start`) or `my_chat_member` → member |
| `unfollow` | User blocked/stopped bot (`kicked` / `left`) |

Callback body:

```json
{
  "events": [
    { "type": "follow", "user_id": "123" },
    { "type": "message", "user_id": "123", "message_id": "9", "text": "hi" }
  ]
}
```

---

## Integration checklist

1. Create a bot with [@BotFather](https://t.me/BotFather); store the token in your app.
2. On send: pass `X-Telegram-Bot-Token`; never store tokens in telegram-service.
3. `setWebhook` with `secret_token` matching the URL query param.
4. Persist chat IDs from `follow` / `message` events for outbound sends.
5. Show a join QR via `GET /api/v1/qr?bot_username=YourBot`.
6. Broadcasts: `POST /api/v1/messages/batch` with stored chat IDs.
