# Marketplace Service Contract

Every marketplace microservice in `communication-service` (Lazada, Shopee, TikTok Shop, etc.) exposes the same REST contract so `laco-invoice` can integrate via a uniform BFF layer.

Marketplace services are **stateless** — they proxy vendor OAuth and APIs only. Connection metadata and OAuth tokens are persisted in **laco-invoice** (`store_marketplace_connections`).

## Gateway routing

| Public path | Internal service |
|-------------|------------------|
| `/comms/lazada/**` | `lazada-service` (:8084) |
| `/comms/shopee/**` | `shopee-service` (:8085) |
| `/comms/tiktok/**` | `tiktok-service` (:8086) |

Gateway rewrites `/comms/{platform}/{segment}` → `/{segment}`.

## Authentication headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization: Bearer <jwt>` | At gateway | End-user JWT |
| `X-Gateway-Secret` | Yes (service) | Service-to-service secret |
| `X-Tenant-ID` | Connect / orders | Per-store tenant UUID from laco-invoice |
| `X-Actor-ID` | Connect / refresh | Per-user actor UUID |

## Standard store connection API

Base path: `/api/v1/stores`

| Method | Path | Tenant | Actor | Description |
|--------|------|--------|-------|-------------|
| GET | `/healthz` | No | No | Health check |
| GET | `/authorize-url?state=` | No | No | Build vendor OAuth URL |
| POST | `/connect` | Yes | Yes | Exchange OAuth `code`; returns tokens |
| POST | `/refresh` | Yes | Yes | Refresh tokens from `refresh_token` body |

There is **no** list or revoke endpoint in marketplace services — laco-invoice owns connection lifecycle in its database.

## Response envelope

```json
{
  "success": true,
  "message": "",
  "data": { }
}
```

## Connect / refresh response (token payload)

Returned by `POST /connect` and `POST /refresh`. Caller (laco-invoice) persists tokens locally.

| Field | Type | Description |
|-------|------|-------------|
| `seller_id` | string | Marketplace seller/shop ID |
| `display_name` | string | Human-readable name |
| `country` | string | Region/country code |
| `account` | string | Optional account label |
| `short_code` | string | Optional short code |
| `access_token` | string | OAuth access token |
| `refresh_token` | string | OAuth refresh token |
| `token_expires_at` | ISO8601 | Access token expiry |
| `refresh_expires_at` | ISO8601 | Refresh token expiry |

## Connect request

```json
{
  "code": "oauth-authorization-code",
  "display_name": "optional label",
  "shop_id": "required for Shopee when not in token response"
}
```

## Refresh request

```json
{
  "refresh_token": "oauth-refresh-token",
  "shop_id": "required for Shopee"
}
```

Lazada and TikTok only require `refresh_token`.

## OAuth state (laco-invoice)

Base64url JSON passed to `authorize-url`:

```json
{ "storeId": 123, "marketplace": "SHOPEE" }
```

Legacy Lazada states with `{ "storeId": 123 }` only remain supported during migration.

## Orders API

Base path: `/api/v1/orders`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/?access_token=&seller_id=&updated_from=` | Fetch orders using caller-supplied access token |

laco-invoice reads `access_token` from `store_marketplace_connections` and passes it to the marketplace service.

## Products API

Base path: `/api/v1/products`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/?access_token=&seller_id=&shop_id=&page=&limit=&search=` | Fetch products using caller-supplied access token |
| GET | `/{externalProductId}?access_token=&seller_id=&shop_id=&sku=` | Fetch single product detail |

### Query parameters (list)

| Parameter | Required | Description |
|-----------|----------|-------------|
| `access_token` | Yes | OAuth access token from laco-invoice |
| `seller_id` | No | Seller ID (Lazada) |
| `shop_id` | No | Shop ID (Shopee/TikTok; may duplicate `seller_id`) |
| `page` | No | Page number (default `1`) |
| `limit` | No | Page size (default `20`, max `100`) |
| `search` | No | Vendor-specific search filter |

### Product list response (`data`)

```json
{
  "items": [
    {
      "external_product_id": "123456",
      "sku": "SKU-001",
      "name": "Product name",
      "price": "99.00",
      "currency": "THB",
      "status": "active"
    }
  ],
  "total": 128
}
```

Lazada implements `/products/get` against the regional REST API. Shopee and TikTok return the same envelope; vendor calls may return empty lists until implemented.

### Product detail response (`data`)

```json
{
  "external_product_id": "123456",
  "sku": "SKU-001",
  "name": "Product name",
  "description": "Description",
  "price": "99.00",
  "currency": "THB",
  "status": "active",
  "images": ["https://..."],
  "variants": [
    {
      "sku": "SKU-001",
      "price": "99.00",
      "currency": "THB",
      "status": "active",
      "quantity": "10"
    }
  ]
}
```

Lazada implements `/product/item/get`. Shopee and TikTok detail endpoints return `404` until implemented.

## Adding a new marketplace

1. Copy `app/lazada-service/` layout to `app/{platform}-service/` (stateless, no database).
2. Implement vendor OAuth client in `internal/client/`.
3. Add gateway route and docker-compose service (no `DATABASE_URL`).
4. Add laco-invoice `integration/{platform}/` client + `features/store/{platform}/` BFF.
5. Register marketplace in client `SUPPORTED_MARKETPLACES` config.
6. Implement `MarketplaceOrderAdapter` for order sync.
7. Implement `MarketplaceProductAdapter` for read-through product listing.
