# Communication Service

A Go **multi-service repository** for social, messaging, and marketplace integrations.
**Platform services** under `app/<platform>-service/` wrap one vendor each (Facebook, LINE,
Lazada). **Spring Cloud Gateway** routes public traffic directly to those services with JWT,
CORS, and a shared gateway secret.

## What it does

| Capability | Description |
|------------|-------------|
| **Outbound messaging** | Send text via LINE using caller-supplied channel access tokens. |
| **Channel / store connections** | Facebook (DB) and Lazada OAuth; LINE credentials stay in calling apps. |
| **Inbound webhooks** | Verify vendor signatures; LINE optionally forwards parsed events to your callback URL. |
| **Marketplace setup** | Lazada seller OAuth and store connection (Laco Invoice). |

### Services

| Service | Port | Role |
|---------|------|------|
| `gateway` | `8080` | Spring Cloud Gateway — JWT, CORS, routing |
| `facebook-service` | `8081` | Meta Graph API ([setup](app/facebook-service/SETUP.md)) |
| `line-service` | `8083` | LINE Messaging API proxy ([integration](app/line-service/INTEGRATION.md)) |
| `lazada-service` | `8084` | Lazada Open Platform store connect ([setup](app/lazada-service/SETUP.md)) |

Each service is **independently deployable** (own `cmd/main.go`, config, Dockerfile).

## Architecture

```
Clients / vendors
        ↓
Spring Cloud Gateway (:8080)   JWT + X-Gateway-Secret
        ↓
facebook / line / lazada services (:8081 / :8083 / :8084)
        ↓
Vendor APIs (Meta, LINE, Lazada)
```

Public REST paths:

| Path | Service |
|------|---------|
| `/comms/facebook/**` | facebook-service |
| `/comms/line/**` | line-service |
| `/comms/lazada/**` | lazada-service |

Within each Go service:

```
handler → service → client (vendor HTTP)
                 → repository (Postgres)   # facebook-service only among messaging peers
```

Shared response envelope lives in root [`pkg/response`](pkg/response/) and [`pkg/utils`](pkg/utils/).

| Document | Description |
|----------|-------------|
| [docs/GO_CODING_STANDARD.md](docs/GO_CODING_STANDARD.md) | Go layer rules, naming, PR checklist |
| [docs/API_GATEWAY.md](docs/API_GATEWAY.md) | Gateway routing and auth |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System context and request flows |

## Getting started

### Prerequisites

- Go 1.22+
- Java 21 + Maven (gateway)
- PostgreSQL 14+

### Run locally

```bash
cp .env.example .env
go mod download
docker compose up postgres -d

# terminals — platform services
go run ./app/line-service/cmd/main.go
go run ./app/lazada-service/cmd/main.go

# gateway
cd app/gateway && mvn spring-boot:run
```

Health checks (via gateway, public):

```
GET http://localhost:8080/comms/line/api/v1/healthz
GET http://localhost:8080/comms/lazada/api/v1/healthz
```

### Docker Compose

```bash
docker compose up --build
```

| Service | URL |
|---------|-----|
| Gateway | http://localhost:8080 |
| Postgres | localhost:5433 |

## API overview

Business routes require **JWT** at the gateway. The gateway injects `X-Gateway-Secret`,
`X-Tenant-ID`, and `X-Actor-ID` before forwarding to platform services.

Webhooks are **public** (no JWT): e.g. `POST /comms/line/api/v1/webhooks`.

All platform REST responses use the shared envelope:

```json
{ "success": true, "data": { ... } }
{ "success": false, "error": "message" }
```

Example (Lazada store connect):

```
GET  /comms/lazada/api/v1/stores/authorize-url   (JWT)
POST /comms/lazada/api/v1/stores/connect         (JWT)
```

## Development

```bash
go fmt ./...
go test ./...
go vet ./...
```

### Adding a new platform service

1. Copy `app/line-service/` → `app/<platform>-service/`.
2. Implement `internal/client/` for the vendor API.
3. Use root `pkg/response` and `pkg/utils` for HTTP envelopes.
4. Add a gateway route in `app/gateway/src/main/resources/application.yml`.
5. Add the service to `docker-compose.yml`.

## License

Proprietary — Laconic Smart Solution.
