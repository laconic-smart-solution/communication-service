# Lazada Service — Setup Guide

`lazada-service` wraps the [Lazada Open Platform](https://open.lazada.com/) OAuth and store-connection APIs for Laco Invoice marketplace setup. It runs on port **8084** and persists tenant-scoped Lazada seller tokens in PostgreSQL.

---

## Prerequisites

| Requirement | Version |
|-------------|---------|
| Go | 1.22+ |
| PostgreSQL | 14+ |
| Lazada Open Platform app | Seller authorization enabled |

---

## Architecture position

```
Laco Invoice UI / Gateway
        ↓
Spring Cloud Gateway (:8080)  →  /comms/lazada/**
        ↓
lazada-service (:8084)     ← this service
        ↓
Lazada Auth API (auth.lazada.com)
```

Public API example: `GET http://localhost:8080/comms/lazada/api/v1/stores/authorize-url` (JWT).

---

## Environment variables

Copy the example file and fill in values:

```bash
cp app/lazada-service/.env.example app/lazada-service/.env
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LAZADA_SERVICE_PORT` | no | `8084` | HTTP listen port |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, `error` |
| `DATABASE_URL` | yes | — | Postgres connection string |
| `DB_AUTO_MIGRATE` | no | `true` | Run GORM migrations on startup |
| `GATEWAY_SECRET` | yes | — | Service-to-service auth (`X-Gateway-Secret`) |
| `LAZADA_APP_KEY` | yes | — | Lazada app key from Open Platform console |
| `LAZADA_APP_SECRET` | yes | — | Lazada app secret |
| `LAZADA_REDIRECT_URI` | yes | — | OAuth callback URL registered in Lazada app |
| `LAZADA_COUNTRY` | no | — | Seller country (`th`, `sg`, `my`, …). Uses regional OAuth URL when set. |
| `LAZADA_FORCE_AUTH` | no | `false` | Set `true` only when you need to force a fresh Lazada auth session. |
| `LAZADA_AUTH_BASE_URL` | no | — | Override authorize URL (defaults to regional or `auth.lazada.com`). |

### Example `.env` (local)

```bash
LAZADA_SERVICE_PORT=8084
LOG_LEVEL=info
DATABASE_URL=postgres://comms:comms@localhost:5433/communication?sslmode=disable
DB_AUTO_MIGRATE=true
GATEWAY_SECRET=hello
LAZADA_APP_KEY=your_app_key
LAZADA_APP_SECRET=your_app_secret
LAZADA_REDIRECT_URI=https://dev-laco-invoice.laconic.co.th/stores/lazada/callback
LAZADA_COUNTRY=th
LAZADA_FORCE_AUTH=false
```

---

## Lazada Open Platform setup

1. Open [Lazada Open Platform](https://open.lazada.com/) and create an app.
2. Set the **Callback URL** to match `LAZADA_REDIRECT_URI`.
3. Note the **App Key** and **App Secret** for env vars.
4. Ensure seller authorization is enabled for your app category.

OAuth docs: [Generate Access Token (`/auth/token/create`)](https://open.lazada.com/apps/doc/api?path=%2Fauth%2Ftoken%2Fcreate)

---

## Run locally

```bash
cd communication-service
go run ./app/lazada-service/cmd/main.go
```

Health check:

```bash
curl http://localhost:8084/api/v1/healthz
```

---

## Connect flow

### 1. Get authorization URL

```bash
curl -s "http://localhost:8084/api/v1/stores/authorize-url" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
```

Open the returned `url` in a browser and authorize your Lazada seller account. Lazada redirects to your callback with a `code` query parameter (valid ~30 minutes, single use).

### 2. Exchange code and connect store

```bash
curl -s -X POST "http://localhost:8084/api/v1/stores/connect" \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002" \
  -d '{"code":"<oauth_code_from_callback>","display_name":"My Lazada Store"}'
```

### 3. List connected stores

```bash
curl -s "http://localhost:8084/api/v1/stores" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
```

### 4. Refresh tokens

```bash
curl -s -X POST "http://localhost:8084/api/v1/stores/<store_id>/refresh" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002"
```

### 5. Revoke store connection

```bash
curl -s -X DELETE "http://localhost:8084/api/v1/stores/<store_id>" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002"
```

---

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/healthz` | Health check (no gateway secret) |
| GET | `/api/v1/stores/authorize-url` | Lazada OAuth redirect URL |
| POST | `/api/v1/stores/connect` | Exchange OAuth code, persist store |
| GET | `/api/v1/stores` | List tenant stores (paginated) |
| POST | `/api/v1/stores/:id/refresh` | Refresh access token |
| DELETE | `/api/v1/stores/:id` | Soft-revoke store connection |

All business routes require `X-Gateway-Secret` and `X-Tenant-ID`. Mutating routes also require `X-Actor-ID` or `X-User-ID`.

---

## Docker Compose

From the repo root:

```bash
docker compose up --build lazada-service
```

Service URL: http://localhost:8084

---

## Token signing

Requests to Lazada use HMAC-SHA256 signing per LazOP standards:

1. Sort parameters by key (ASCII).
2. Concatenate: `apiPath + key1 + value1 + key2 + value2 + ...`
3. HMAC-SHA256 with app secret, uppercase hex.

Implemented in `internal/client/signature.go`.
