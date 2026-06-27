# Facebook Service — Setup Guide

`facebook-service` runs on port **8081** and is reached via the gateway at `/comms/facebook/**`.

---

## Prerequisites

| Requirement | Version |
|-------------|---------|
| Go | 1.22+ |
| PostgreSQL | 14+ |
| Meta Developer account | Facebook App with Pages permissions |

Optional: Docker Compose from the repo root for the full stack.

---

## Architecture position

```
Client / Meta servers
        ↓
Spring Cloud Gateway (:8080)
        ↓
facebook-service (:8081)         ← this service
        ↓
Meta Graph API
```

Public webhook URL (register in Meta App Dashboard):

```
https://<your-host>/comms/facebook/api/v1/webhooks
```

---

## Environment variables

```bash
cp app/facebook-service/.env.example app/facebook-service/.env
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `FACEBOOK_SERVICE_PORT` | no | `8081` | HTTP listen port |
| `PORT` | no | — | Fallback if `FACEBOOK_SERVICE_PORT` is unset |
| `LOG_LEVEL` | no | `info` | Log level |
| `DATABASE_URL` | yes | — | Postgres connection string |
| `DB_AUTO_MIGRATE` | no | `true` | Run GORM migrations on startup |
| `GATEWAY_SECRET` | yes | — | Must match gateway (`X-Gateway-Secret`) |
| `FACEBOOK_APP_ID` | no | — | Meta app ID (reference / future OAuth) |
| `FACEBOOK_APP_SECRET` | yes* | — | App secret for webhook `X-Hub-Signature-256` |
| `FACEBOOK_VERIFY_TOKEN` | yes* | — | Custom string for webhook subscription verify |
| `FACEBOOK_GRAPH_API_VERSION` | no | `v21.0` | Graph API version |

\* Required for webhooks.

### Example `.env` (local)

```bash
FACEBOOK_SERVICE_PORT=8081
LOG_LEVEL=info
DATABASE_URL=postgres://comms:comms@localhost:5433/communication?sslmode=disable
DB_AUTO_MIGRATE=true
GATEWAY_SECRET=hello
FACEBOOK_APP_ID=1234567890
FACEBOOK_APP_SECRET=your_app_secret
FACEBOOK_VERIFY_TOKEN=my-custom-verify-token
FACEBOOK_GRAPH_API_VERSION=v21.0
```

---

## Meta / Facebook Developers setup

1. Go to [Meta for Developers](https://developers.facebook.com/) and create an app.
2. Add the **Facebook Login** or **Webhooks** product as needed.
3. Under **Settings → Basic**, copy **App ID** and **App Secret**.
4. Set `FACEBOOK_APP_SECRET` and choose a random `FACEBOOK_VERIFY_TOKEN`.
5. Request **Pages** permissions and obtain a **Page access token** with `pages_manage_posts`, `pages_read_engagement` (and related scopes your use case needs).
6. Under **Webhooks → Page**, subscribe to `feed` (and other fields as needed):
   - Callback URL: `https://<gateway-host>/comms/facebook/api/v1/webhooks`
   - Verify token: same as `FACEBOOK_VERIFY_TOKEN`
7. Connect the Page to your app and generate a long-lived **Page access token**.

Graph API post operations used by this service:

| Action | Graph API |
|--------|-----------|
| Create post | `POST /{page-id}/feed` |
| Read post | `GET /{post-id}?fields=id,message` |
| Update post | `POST /{post-id}` with `message` |
| Delete post | `DELETE /{post-id}` |

See [Graph API reference](https://developers.facebook.com/docs/graph-api/) for current version and permissions.

---

## Run locally

### 1. Start PostgreSQL

```bash
docker compose up postgres -d
```

### 2. Run facebook-service

```bash
go run ./app/facebook-service/cmd/main.go
```

### 3. Verify health

```bash
curl http://localhost:8081/api/v1/healthz
# {"service":"facebook-service","status":"ok"}
```

### 4. Full stack

```bash
go run ./app/facebook-service/cmd/main.go   # terminal 1
cd app/gateway && mvn spring-boot:run       # terminal 2
```

---

## Full stack (Docker Compose)

```bash
cp .env.example .env
# set FACEBOOK_APP_SECRET, FACEBOOK_VERIFY_TOKEN
docker compose up --build
```

| Service | URL |
|---------|-----|
| Gateway | http://localhost:8080 |
| facebook-service (internal) | http://localhost:8081 |
| Health via gateway | http://localhost:8080/comms/facebook/api/v1/healthz |

---

## API reference (internal REST)

Business routes require `X-Gateway-Secret`. Webhooks and health are exempt.

### Headers

| Header | Description |
|--------|-------------|
| `X-Gateway-Secret` | Shared secret |
| `X-Tenant-ID` | Tenant UUID |
| `X-Actor-ID` or `X-User-ID` | Caller identity |

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/healthz` | no | Liveness |
| `GET` | `/api/v1/webhooks` | verify token | Meta subscription challenge |
| `POST` | `/api/v1/webhooks` | signature | Meta webhook events |
| `POST` | `/api/v1/channels` | yes | Connect Facebook Page |
| `GET` | `/api/v1/channels` | yes | List Pages |
| `DELETE` | `/api/v1/channels/:id` | yes | Revoke Page connection |
| `POST` | `/api/v1/posts` | yes | Create & publish post |
| `GET` | `/api/v1/posts` | yes | List posts (`?channel_id=&page=&limit=`) |
| `GET` | `/api/v1/posts/:id` | yes | Get post |
| `PUT` | `/api/v1/posts/:id` | yes | Update post message |
| `DELETE` | `/api/v1/posts/:id` | yes | Delete post on Facebook |

### Connect Page

```bash
curl -X POST http://localhost:8081/api/v1/channels \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002" \
  -d '{
    "external_id": "123456789012345",
    "display_name": "My Facebook Page",
    "access_token": "your-page-access-token"
  }'
```

`external_id` is the **Facebook Page ID**.

### Create post

```bash
curl -X POST http://localhost:8081/api/v1/posts \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002" \
  -d '{
    "channel_id": "<uuid-from-connect>",
    "message": "Hello from communication-service!"
  }'
```

### Update post

```bash
curl -X PUT http://localhost:8081/api/v1/posts/<post-uuid> \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002" \
  -d '{"message": "Updated post text"}'
```

### Delete post

```bash
curl -X DELETE http://localhost:8081/api/v1/posts/<post-uuid> \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002"
```

### Webhook verification (Meta calls this)

```
GET /api/v1/webhooks?hub.mode=subscribe&hub.verify_token=<FACEBOOK_VERIFY_TOKEN>&hub.challenge=<challenge>
```

Returns the `hub.challenge` string on success.

---

## Public API (via gateway)

```
POST http://localhost:8080/comms/facebook/api/v1/posts
Authorization: Bearer <jwt>
```

Example create post:

```bash
curl -X POST http://localhost:8080/comms/facebook/api/v1/posts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <jwt>" \
  -d '{
    "channel_id": "<page-channel-uuid>",
    "message": "Published via gateway"
  }'
```

---

## Database tables

| Table | Purpose |
|-------|---------|
| `fb_channels` | Connected Facebook Pages (ID + access token) |
| `fb_posts` | Post records and Graph API `external_id` |

Post statuses: `draft` → `published` | `failed` | `deleted`.

---

## Troubleshooting

| Issue | Check |
|-------|-------|
| `403 forbidden` | `GATEWAY_SECRET` / `X-Gateway-Secret` mismatch |
| `invalid webhook signature` | `FACEBOOK_APP_SECRET` wrong; proxy must not alter POST body |
| `invalid verify token` | `FACEBOOK_VERIFY_TOKEN` must match Meta Dashboard |
| `invalid facebook page token` | Page token expired; regenerate with correct permissions |
| `failed to publish post` | Page lacks `pages_manage_posts`; token is user token not Page token |
| `post is not published on facebook` | Cannot update a local-only / failed post |

---

## Related docs

- [Repository README](../../README.md)
- [Architecture](../../docs/ARCHITECTURE.md)
- [Gateway](../../docs/API_GATEWAY.md)
- [Meta Graph API](https://developers.facebook.com/docs/graph-api/)
- [Go coding standard](../../docs/GO_CODING_STANDARD.md)
