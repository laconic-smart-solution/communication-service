# Marketplace Service Contract

Every marketplace microservice in `communication-service` (Lazada, Shopee, TikTok Shop, etc.) exposes the same REST contract so `laco-invoice` can integrate via a uniform BFF layer.

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
| `X-Tenant-ID` | Mutations + list | Per-store tenant UUID from laco-invoice |
| `X-Actor-ID` | Mutations | Per-user actor UUID |

## Standard store connection API

Base path: `/api/v1/stores`

| Method | Path | Tenant | Actor | Description |
|--------|------|--------|-------|-------------|
| GET | `/healthz` | No | No | Health check |
| GET | `/authorize-url?state=` | No | No | Build vendor OAuth URL |
| POST | `/connect` | Yes | Yes | Exchange OAuth `code` |
| GET | `/?page=&limit=` | Yes | No | List tenant connections |
| POST | `/{id}/refresh` | Yes | Yes | Refresh tokens |
| DELETE | `/{id}` | Yes | Yes | Revoke connection |

## Response envelope

```json
{
  "success": true,
  "message": "",
  "data": { }
}
```

Paged list `data`:

```json
{
  "items": [ ],
  "meta": { "page": 1, "limit": 20, "total": 0, "total_pages": 0 }
}
```

## Store object (normalized)

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | External connection ID |
| `tenant_id` | UUID | laco-invoice store scope |
| `seller_id` | string | Marketplace seller/shop ID |
| `display_name` | string | Human-readable name |
| `country` | string | Region/country code |
| `account` | string | Optional account label |
| `short_code` | string | Optional short code |
| `token_expires_at` | ISO8601 | Access token expiry |
| `refresh_expires_at` | ISO8601 | Refresh token expiry |
| `created_at` | ISO8601 | Connection created |

Tokens are stored in the marketplace service database only — never returned in API responses.

## Connect request

```json
{
  "code": "oauth-authorization-code",
  "display_name": "optional label"
}
```

## OAuth state (laco-invoice)

Base64url JSON passed to `authorize-url`:

```json
{ "storeId": 123, "marketplace": "SHOPEE" }
```

Legacy Lazada states with `{ "storeId": 123 }` only remain supported during migration.

## Orders API (Phase 3)

Base path: `/api/v1/orders`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/?store_id=&updated_from=` | Fetch orders for a connected store |

## Adding a new marketplace

1. Copy `app/lazada-service/` layout to `app/{platform}-service/`.
2. Implement vendor OAuth client in `internal/client/`.
3. Add gateway route and docker-compose service.
4. Add laco-invoice `integration/{platform}/` client + `features/store/{platform}/` BFF.
5. Register marketplace in client `SUPPORTED_MARKETPLACES` config.
6. Implement `MarketplaceOrderAdapter` for order sync.
