# Communication Service

A Go **multi-service repository** for social and messaging integrations. **Platform services**
under `app/<platform>-service/` wrap one vendor each (Facebook, Instagram, LINE). A
**normalization-service** exposes a unified **GraphQL** API and fans out to those REST
services. **Spring Cloud Gateway** (separate repo) handles CORS, rate limiting, and JWT at the
edge.

## What it does

| Capability | Description |
|------------|-------------|
| **Outbound messaging** | Send text, media, and template messages via the platform's native API. |
| **Channel connections** | Link, refresh, and revoke OAuth credentials per tenant. |
| **Inbound webhooks** | Verify signatures, normalize payloads, persist or forward events. |
| **Delivery tracking** | Store message status and correlate vendor message IDs. |

### Services

| Service | Path | Role |
|---------|------|------|
| `normalization-service` | `app/normalization-service/` | Unified GraphQL → REST fan-out |
| `facebook-service` | `app/facebook-service/` | Meta Graph API — Page post CRUD ([setup](app/facebook-service/SETUP.md)) |
| `instagram-service` | `app/instagram-service/` | Meta Graph API (Instagram DM) |
| `line-service` | `app/line-service/` | LINE Messaging API ([setup](app/line-service/SETUP.md)) |
| *future* | `app/<platform>-service/` | One folder per new provider |

Each service is **independently deployable** (own `cmd/main.go`, config, Dockerfile, port).
Laconic apps call **`POST /comms/graphql`** through the gateway — not platform REST directly.

## Architecture

```
Laconic apps ──┐
Meta / LINE  ──┼──▶ Spring Cloud Gateway ──▶ normalization-service ──▶ facebook / instagram / line
               │    (CORS, rate limit, JWT)   (GraphQL + webhook proxy)         │
               │                                                                 ▼
               └──────────────────────────────────────────────────────── Vendor APIs
```

The gateway talks **only** to `normalization-service`. Platform services are internal.

| Layer | Doc |
|-------|-----|
| Gateway | [docs/API_GATEWAY.md](docs/API_GATEWAY.md) |
| GraphQL normalization | [docs/NORMALIZATION_SERVICE.md](docs/NORMALIZATION_SERVICE.md) |
| Full architecture | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) |

Within each service the flow is the same:

```
handler → service → repository (Postgres)
                 → client   (vendor HTTP / SDK — platform-specific only)
```

Vendor API calls live only inside that service's `internal/client/` (or `pkg/`). No service
imports another service's `internal/` packages.

## Tech stack

| Layer | Stack |
|-------|-------|
| **API Gateway** (separate repo) | Spring Boot · Spring Cloud Gateway · OAuth2 JWT · CORS · rate limit |
| **Normalization** (this repo) | Go 1.22 · gqlgen · Fiber |
| **Platform services** (this repo) | Go 1.22 · Fiber v2 · GORM v2 · PostgreSQL |
| **Shared** | logrus · google/uuid · go-playground/validator · godotenv · Redis (optional) |

## Project layout

```
communication-service/
├─ go.mod
├─ docker-compose.yml
├─ .env.example
├─ app/
│  ├─ gateway/                    # Spring Cloud Gateway (Java)
│  ├─ normalization-service/
│  │  ├─ cmd/main.go
│  │  ├─ internal/graphql/     # schema + resolvers (gqlgen)
│  │  ├─ internal/client/      # REST clients → platform services
│  │  └─ …
│  ├─ facebook-service/
│  │  ├─ cmd/main.go              # composition root
│  │  ├─ Dockerfile
│  │  ├─ internal/
│  │  │  ├─ config/
│  │  │  ├─ database/
│  │  │  ├─ models/
│  │  │  ├─ repository/
│  │  │  ├─ request/
│  │  │  ├─ response/
│  │  │  ├─ mapper/
│  │  │  ├─ service/
│  │  │  ├─ handler/
│  │  │  ├─ middleware/
│  │  │  ├─ routes/
│  │  │  └─ client/               # Meta Graph API client (facebook only)
│  │  └─ pkg/utils/
│  ├─ instagram-service/
│  │  └─ …                        # same shape; internal/client/ for Instagram
│  └─ line-service/
│     └─ …                        # same shape; internal/client/ for LINE
└─ pkg/                           # optional shared code across services (envelope, crypto)
```

| Document | Description |
|----------|-------------|
| [docs/GO_CODING_STANDARD.md](docs/GO_CODING_STANDARD.md) | Go layer rules, naming, PR checklist |
| [docs/API_GATEWAY.md](docs/API_GATEWAY.md) | Spring Cloud Gateway — CORS, rate limit, routing |
| [docs/NORMALIZATION_SERVICE.md](docs/NORMALIZATION_SERVICE.md) | GraphQL normalization layer |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System context and request flows |

## Getting started

### Prerequisites

- Go 1.22+
- PostgreSQL 14+
- (Optional) Redis 6+

### Environment variables

Each service has its own `.env.example` under `app/<service>/`. Copy and fill in:

```bash
cp app/normalization-service/.env.example app/normalization-service/.env
cp app/facebook-service/.env.example app/facebook-service/.env
cp app/line-service/.env.example app/line-service/.env
```

Common keys per service:

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | no | HTTP port (`8090` normalization, `8081` facebook, `8082` instagram, `8083` line) |
| `LOG_LEVEL` | no | `debug`, `info`, `warn`, `error` |
| `DATABASE_URL` | yes | Postgres DSN (separate DB or schema per service) |
| `DB_AUTO_MIGRATE` | no | Run GORM migrations on startup |
| `GATEWAY_SECRET` | yes | Service-to-service auth |
| `WEBHOOK_BASE_URL` | no | Public callback URL for this service |

Platform-specific keys (e.g. `FACEBOOK_APP_SECRET`, `LINE_CHANNEL_SECRET`) belong only on the
matching service.

### Run locally

```bash
cp .env.example .env
go mod download

# terminal 1 — Postgres required on DATABASE_URL (default port 5433 via docker-compose)
go run ./app/line-service/cmd/main.go

# terminal 2
go run ./app/normalization-service/cmd/main.go

# terminal 3 — Java 21 + Maven
cd app/gateway && mvn spring-boot:run
```

Health checks: gateway `GET /comms/healthz` · normalization `GET :8090/healthz` · line `GET :8083/api/v1/healthz`

### Docker Compose (full stack)

```bash
cp .env.example .env
docker compose up --build
```

| Service | URL |
|---------|-----|
| Gateway | http://localhost:8080 |
| GraphQL playground | http://localhost:8090/graphql (when `PLAYGROUND_ENABLED=true`) |
| LINE service (internal) | http://localhost:8083 |
| Postgres | localhost:5433 |

## API overview

### Client API (GraphQL via gateway)

Laconic apps use a single endpoint:

```
POST https://api.example.com/comms/graphql
Authorization: Bearer <jwt>
```

Example operations: `channels`, `sendMessage`, `connectChannel`, `revokeChannel` — see
[docs/NORMALIZATION_SERVICE.md](docs/NORMALIZATION_SERVICE.md).

### Platform REST (internal only)

Platform services expose `/api/v1` on internal ports. Only **normalization-service** calls them
(GraphQL resolvers or webhook proxy). Public webhook URLs still use the gateway host, but traffic
flows gateway → normalization → platform:

| Service | Internal (normalization → platform) | Public webhook URL |
|---------|-------------------------------------|--------------------|
| facebook-service | `:8081/api/v1/...` | `POST /comms/facebook/api/v1/webhooks` |
| instagram-service | `:8082/api/v1/...` | `POST /comms/instagram/api/v1/webhooks` |
| line-service | `:8083/api/v1/...` | `POST /comms/line/api/v1/webhooks` |

Platform REST uses the `APIResponse` envelope (`success`, `data`, `error`). GraphQL responses
use standard GraphQL JSON (`data` / `errors`).

## Development

```bash
go fmt ./...
go test ./...
go vet ./...
```

Coding conventions: [docs/GO_CODING_STANDARD.md](docs/GO_CODING_STANDARD.md)

### Adding a new platform service

1. Copy an existing service folder: `app/facebook-service/` → `app/<platform>-service/`.
2. Rename `AppName`, models, and routes for the new vendor.
3. Implement `internal/client/` against the vendor API.
4. Add `app/<platform>-service/.env.example` and `Dockerfile`.
5. Add resolvers, REST client, and webhook proxy route in `normalization-service`.
6. Gateway config unchanged — it only routes to normalization ([docs/API_GATEWAY.md](docs/API_GATEWAY.md)).

## License

Proprietary — Laconic Smart Solution.
