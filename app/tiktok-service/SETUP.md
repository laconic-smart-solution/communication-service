# TikTok Shop Service — Setup Guide

`tiktok-service` wraps the [TikTok Shop Partner Center](https://partner.tiktokshop.com/) OAuth and marketplace APIs for Laco Invoice. It is **stateless** (no database): it proxies TikTok OAuth, products, and orders. Connection tokens are persisted in **laco-invoice** (`store_marketplace_connections`).

Default port: **8086**.

---

## Prerequisites

| Requirement | Notes |
|-------------|--------|
| Go | 1.22+ |
| TikTok Shop Partner app | Seller authorization enabled |
| Gateway secret | Shared with Spring Cloud Gateway / laco-invoice |
| laco-invoice | Holds `TIKTOK_APP_KEY` / `TIKTOK_APP_SECRET` and store OAuth tokens |

---

## Architecture

```
Laco Invoice UI
        ↓
laco-invoice (BFF)  — adds X-TikTok-App-Key / X-TikTok-App-Secret
        ↓
Spring Cloud Gateway  →  /comms/tiktok/**
        ↓
tiktok-service (:8086)     ← this service
        ↓
TikTok Auth  (auth.tiktok-shops.com)
TikTok Open API  (open-api.tiktokglobalshop.com)
```

Public path example (via gateway):

`GET /comms/tiktok/api/v1/stores/authorize-url`

---

## Environment variables (tiktok-service)

```bash
cp app/tiktok-service/.env.example app/tiktok-service/.env
```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TIKTOK_SERVICE_PORT` | no | `8086` | HTTP listen port |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, `error` |
| `GATEWAY_SECRET` | yes | — | Service-to-service auth (`X-Gateway-Secret`) |
| `TIKTOK_REDIRECT_URI` | yes | — | OAuth callback URL registered in Partner Center |
| `TIKTOK_AUTH_HOST` | no | `https://auth.tiktok-shops.com` | Token host |
| `TIKTOK_API_HOST` | no | `https://open-api.tiktokglobalshop.com` | Signed API host |

**App key and app secret are not configured in tiktok-service.**  
Callers must send them on every request:

| Header | Source |
|--------|--------|
| `X-TikTok-App-Key` | laco-invoice `TIKTOK_APP_KEY` / `app.communication.tiktok-app-key` |
| `X-TikTok-App-Secret` | laco-invoice `TIKTOK_APP_SECRET` / `app.communication.tiktok-app-secret` |

### Example `.env` (tiktok-service)

```bash
TIKTOK_SERVICE_PORT=8086
LOG_LEVEL=info
GATEWAY_SECRET=hello
TIKTOK_REDIRECT_URI=https://dev-laco-invoice.laconic.co.th/stores/tiktok/callback
```

### Example config (laco-invoice)

```bash
# env or application-*.yml under app.communication
TIKTOK_APP_KEY=your_partner_app_key
TIKTOK_APP_SECRET=your_partner_app_secret
COMMUNICATION_TIKTOK_BASE_PATH=/comms/tiktok
GATEWAY_SECRET=hello
```

---

## TikTok Partner Center setup

1. Open [TikTok Shop Partner Center](https://partner.tiktokshop.com/) and create (or open) your app.
2. Set the **Redirect / Callback URL** to exactly match `TIKTOK_REDIRECT_URI`.
   - Local UI default: `http://localhost:5173/stores/tiktok/callback`
   - Deployed example: `https://dev-laco-invoice.laconic.co.th/stores/tiktok/callback`
3. Copy **App Key** and **App Secret** into **laco-invoice** (`TIKTOK_APP_KEY` / `TIKTOK_APP_SECRET`), not into tiktok-service.
4. Enable the API scopes your app needs (at minimum: authorization, product read, order read).
5. Authorize a seller shop against this app (sandbox or production as appropriate).

Useful docs:

- [Sign your API request](https://partner.tiktokshop.com/docv2/page/sign-your-api-request)
- OAuth authorize: `https://services.tiktokshop.com/open/authorize`
- Token: `POST {TIKTOK_AUTH_HOST}/api/v2/token/get`

---

## Run locally

```bash
cd communication-service
go run ./app/tiktok-service/cmd/main.go
```

Health check (no gateway secret):

```bash
curl http://localhost:8086/api/v1/healthz
```

Docker Compose:

```bash
docker compose up --build tiktok-service
```

Service URL: http://localhost:8086

---

## Connect flow (UI)

1. In Laco Invoice, open a store → connect **TikTok Shop**.
2. Browser opens TikTok authorize URL (built by this service).
3. Seller approves → redirects to `/stores/tiktok/callback?code=...&state=...`.
4. laco-invoice calls `POST /api/v1/stores/connect` with the code and stores tokens locally.
5. Sync **Products** / **Orders** from the UI. Customers are upserted during **order sync**.

---

## Connect flow (curl / direct service)

Replace secrets and IDs as needed. All business routes need `X-Gateway-Secret`. Connect/refresh also need `X-Tenant-ID` and `X-Actor-ID`. Every TikTok call needs app credential headers.

### 1. Authorization URL

```bash
curl -s "http://localhost:8086/api/v1/stores/authorize-url?state=demo" \
  -H "X-Gateway-Secret: hello" \
  -H "X-TikTok-App-Key: YOUR_APP_KEY" \
  -H "X-TikTok-App-Secret: YOUR_APP_SECRET"
```

Open `data.url` in a browser, authorize, then copy `code` from the callback query string.

### 2. Exchange code (connect)

```bash
curl -s -X POST "http://localhost:8086/api/v1/stores/connect" \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002" \
  -H "X-TikTok-App-Key: YOUR_APP_KEY" \
  -H "X-TikTok-App-Secret: YOUR_APP_SECRET" \
  -d '{"code":"<oauth_code>","display_name":"My TikTok Shop"}'
```

Response `data` includes `seller_id`, `access_token`, `refresh_token`, expiry timestamps. Persist these in laco-invoice (the service does not store them).

### 3. Refresh tokens

```bash
curl -s -X POST "http://localhost:8086/api/v1/stores/refresh" \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-Actor-ID: 00000000-0000-0000-0000-000000000002" \
  -H "X-TikTok-App-Key: YOUR_APP_KEY" \
  -H "X-TikTok-App-Secret: YOUR_APP_SECRET" \
  -d '{"refresh_token":"<refresh_token>"}'
```

### 4. List products

```bash
curl -s "http://localhost:8086/api/v1/products?access_token=<token>&seller_id=<shop_id>&page=1&limit=20" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-TikTok-App-Key: YOUR_APP_KEY" \
  -H "X-TikTok-App-Secret: YOUR_APP_SECRET"
```

### 5. Product detail

```bash
curl -s "http://localhost:8086/api/v1/products/<external_product_id>?access_token=<token>&seller_id=<shop_id>" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-TikTok-App-Key: YOUR_APP_KEY" \
  -H "X-TikTok-App-Secret: YOUR_APP_SECRET"
```

### 6. List orders (includes line items + customer fields)

```bash
curl -s "http://localhost:8086/api/v1/orders?access_token=<token>&seller_id=<shop_id>&page=1&limit=20" \
  -H "X-Gateway-Secret: hello" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-TikTok-App-Key: YOUR_APP_KEY" \
  -H "X-TikTok-App-Secret: YOUR_APP_SECRET"
```

Order `data` shape matches the shared marketplace contract (`items` + `total`), including:

- `customer_name`, `customer_email`, `customer_phone`, `customer_address`
- `external_customer_id` (TikTok `user_id`)
- nested `items[]` (SKU, price, quantity, optional `buyer_id` / `currency`)

---

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/healthz` | Health check |
| GET | `/api/v1/stores/authorize-url` | TikTok OAuth URL |
| POST | `/api/v1/stores/connect` | Exchange OAuth `code` → token payload |
| POST | `/api/v1/stores/refresh` | Refresh tokens |
| GET | `/api/v1/products` | Product search/list |
| GET | `/api/v1/products/:externalProductId` | Product detail |
| GET | `/api/v1/orders` | Orders + line items + customer info |

There is **no** list/revoke stores API in this service — laco-invoice owns connection lifecycle.

---

## Vendor APIs used

| Capability | TikTok API |
|------------|------------|
| Authorized shops / cipher | `GET /authorization/202309/shops` |
| Products list | `POST /product/202309/products/search` |
| Product detail | `GET /product/202309/products/{id}` |
| Orders search | `POST /order/202309/orders/search` |
| Order details | `GET /order/202309/orders?ids=` |

Access token is sent as header `x-tts-access-token`. Requests are signed with HMAC-SHA256 (`internal/client/signature.go`).

---

## Troubleshooting

| Symptom | Likely cause |
|---------|----------------|
| `X-TikTok-App-Key header is required` | laco-invoice missing `TIKTOK_APP_KEY` / `TIKTOK_APP_SECRET`, or headers not forwarded |
| `TikTok app credentials are not configured` | Same — configure on **laco-invoice**, then restart |
| Empty products/orders | Shop not authorized, wrong token, missing API scopes, or no `shop_cipher` shops for token |
| OAuth callback mismatch | `TIKTOK_REDIRECT_URI` must exactly equal Partner Center redirect URL |
| Gateway 401/403 | `GATEWAY_SECRET` mismatch between gateway, invoice, and tiktok-service |

Shared contract: [`docs/marketplace-service-contract.md`](../../docs/marketplace-service-contract.md)
