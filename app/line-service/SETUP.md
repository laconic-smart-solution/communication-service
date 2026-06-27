# LINE Service — Setup Guide

`line-service` runs on port **8083** and is reached via the gateway at `/comms/line/**`.

---

## Prerequisites

| Requirement | Version |
|-------------|---------|
| Go | 1.22+ |
| PostgreSQL | 14+ |
| LINE Developers account | Messaging API channel |

Optional: run the full stack via Docker Compose from the repo root (see [Full stack](#full-stack-docker-compose)).

---

## Architecture position

```
Client / LINE servers
        ↓
Spring Cloud Gateway (:8080)
        ↓
line-service (:8083)           ← this service
        ↓
LINE Messaging API
```

Public webhook URL (register in LINE Console):

```
https://<your-host>/comms/line/api/v1/webhooks
```

---

## Environment variables

Copy the example file and fill in values:

```bash
cp app/line-service/.env.example app/line-service/.env
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LINE_SERVICE_PORT` | no | `8083` | HTTP listen port |
| `PORT` | no | — | Fallback if `LINE_SERVICE_PORT` is unset |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, `error` |
| `DATABASE_URL` | yes | — | Postgres connection string |
| `DB_AUTO_MIGRATE` | no | `true` | Run GORM migrations on startup |
| `GATEWAY_SECRET` | yes | — | Must match gateway (`X-Gateway-Secret`) |
| `LINE_CHANNEL_SECRET` | yes* | — | Channel secret for webhook signature verification |
| `LINE_CHANNEL_ACCESS_TOKEN` | no** | — | Default token when channel record has none |

\* Required for webhooks.  
\** Used as fallback; per-channel tokens stored in DB are preferred.

### Example `.env` (local)

```bash
LINE_SERVICE_PORT=8083
LOG_LEVEL=info
DATABASE_URL=postgres://comms:comms@localhost:5433/communication?sslmode=disable
DB_AUTO_MIGRATE=true
GATEWAY_SECRET=hello
LINE_CHANNEL_SECRET=your_channel_secret
LINE_CHANNEL_ACCESS_TOKEN=your_long_lived_channel_access_token
```

---

## LINE Developers setup

1. Open [LINE Developers Console](https://developers.line.biz/console/).
2. Create a **Messaging API** channel (or use an existing Official Account).
3. Copy **Channel secret** → `LINE_CHANNEL_SECRET`.
4. Issue a **Channel access token** (long-lived) → `LINE_CHANNEL_ACCESS_TOKEN` or store per channel via the API.
5. Under **Messaging API → Webhook settings**:
   - Enable **Use webhook**
   - Set webhook URL: `https://<gateway-host>/comms/line/api/v1/webhooks`
   - Disable webhook if using a tunnel for local dev (e.g. ngrok)

---

## Run locally

### 1. Start PostgreSQL

```bash
# from repo root — Postgres on host port 5433
docker compose up postgres -d
```

### 2. Run line-service

```bash
# from repo root
go run ./app/line-service/cmd/main.go
```

Load env from `app/line-service/.env` or export variables in your shell (`godotenv` loads `.env` from the working directory).

### 3. Verify health

```bash
curl http://localhost:8083/api/v1/healthz
# {"service":"line-service","status":"ok"}
```

### 4. Run with gateway (recommended)

```bash
# terminal 1
go run ./app/line-service/cmd/main.go

# terminal 2
cd app/gateway && mvn spring-boot:run
```

---

## Full stack (Docker Compose)

From the repository root:

```bash
cp .env.example .env
# edit LINE_CHANNEL_SECRET, LINE_CHANNEL_ACCESS_TOKEN
docker compose up --build
```

| Service | URL |
|---------|-----|
| Gateway | http://localhost:8080 |
| line-service (internal) | http://localhost:8083 |
| Health via gateway | http://localhost:8080/comms/line/api/v1/healthz |

---

## API reference (internal REST)

All business routes require header `X-Gateway-Secret` (injected by gateway).  
Webhooks and health checks are exempt.

### Headers (from gateway)

| Header | Description |
|--------|-------------|
| `X-Gateway-Secret` | Shared secret |
| `X-Tenant-ID` | Tenant UUID |
| `X-Actor-ID` or `X-User-ID` | Caller identity |

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/healthz` | no | Liveness |
| `POST` | `/api/v1/webhooks` | signature | LINE webhook (`X-Line-Signature`) |
| `POST` | `/api/v1/channels` | yes | Connect Official Account |
| `GET` | `/api/v1/channels` | yes | List channels (`?page=&limit=`) |
| `DELETE` | `/api/v1/channels/:id` | yes | Revoke channel |
| `POST` | `/api/v1/messages` | yes | Send push message |
| `GET` | `/api/v1/messages/:id` | yes | Get message status |

### Connect channel

```bash
curl -X POST http://localhost:8083/api/v1/channels \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002" \
  -d '{
    "external_id": "your-line-channel-id",
    "display_name": "My Official Account",
    "access_token": "your-channel-access-token"
  }'
```

### Send message

```bash
curl -X POST http://localhost:8083/api/v1/messages \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002" \
  -d '{
    "channel_id": "<uuid-from-connect>",
    "recipient_id": "Uxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "text": "Hello from LINE!"
  }'
```

### Response envelope

```json
{
  "success": true,
  "data": { ... }
}
```

Errors return `"success": false` and `"error": "message"`.

---

## Public API (via gateway)

Clients call REST through the gateway with JWT:

```
POST http://localhost:8080/comms/line/api/v1/messages
Authorization: Bearer <jwt>
```

Example send message:

```bash
curl -X POST http://localhost:8080/comms/line/api/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <jwt>" \
  -d '{
    "channel_id": "<uuid>",
    "recipient_id": "Uxxxxxxxx...",
    "text": "Hello!"
  }'
```

---

## Database tables

| Table | Purpose |
|-------|---------|
| `line_channels` | Connected LINE Official Accounts |
| `line_messages` | Outbound and inbound messages |

Migrations run automatically when `DB_AUTO_MIGRATE=true`.

---

## Troubleshooting

| Issue | Check |
|-------|-------|
| `403 forbidden` | `GATEWAY_SECRET` mismatch; ensure `X-Gateway-Secret` header is sent |
| `invalid webhook signature` | `LINE_CHANNEL_SECRET` wrong; body must not be modified by proxy |
| `failed to send message` | Token expired or invalid; verify Channel access token |
| `db connect` failed | Postgres running; `DATABASE_URL` correct |
| Webhook not received | URL registered in LINE Console; gateway routes to line-service |

---

## Related docs

- [Repository README](../../README.md)
- [Architecture](../../docs/ARCHITECTURE.md)
- [Gateway](../../docs/API_GATEWAY.md)
- [Go coding standard](../../docs/GO_CODING_STANDARD.md)
