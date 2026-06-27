# Go Coding Standard — Communication Service

> This document defines the conventions for the **communication-service** Go repository.
> The repo hosts **one microservice per platform** under `app/<platform>-service/`.
> **Spring Cloud Gateway** routes public traffic directly to each service. Platform services
> wrap a single vendor API. When in doubt, open an existing service and copy the shape.

**Stack:** Go 1.22 · Fiber v2 · GORM v2 + PostgreSQL · logrus · google/uuid ·
go-playground/validator · golang-jwt/v5 · go-redis · godotenv.

**Domain scope (per service):** channel connections, outbound/inbound messages, webhook
ingestion, OAuth tokens, and delivery status for **one** platform.

---

## 1. Project layout

The repository is a **modular multi-service** Go module. Each platform is a self-contained
service with an identical internal layout:

```
communication-service/
├─ go.mod
├─ app/
│  ├─ gateway/                     # Spring Cloud Gateway (Java)
│  ├─ facebook-service/
│  │  ├─ cmd/
│  │  │  └─ main.go                 # composition root
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
│  │  │  └─ client/                 # vendor HTTP/SDK — facebook-service only
│  │  ├─ pkg/                       # utils scoped to this service
│  │  ├─ Dockerfile
│  │  └─ .env.example
│  ├─ instagram-service/
│  │  └─ …                          # same layout; own internal/client/
│  └─ line-service/
│     └─ …
└─ pkg/                             # shared envelope + HTTP helpers (response, utils)
```

**Rule:** dependencies point **inward and downward** only:

```
handler → service → repository → models
                 → client (vendor API)
```

`mapper`, `request`, `response`, `config` are leaf packages. A `repository` never imports a
`service` or `client`. A `client` never imports `handler` or `service`.

**Cross-service rule:** never import another service's `internal/` package. Share code through
root [`pkg/response`](../pkg/response/) and [`pkg/utils`](../pkg/utils/).

### Module path

The module is rooted at the repository `go.mod`:

```
module github.com/laconic-smart-solution/communication-service
```

Imports are absolute from that root, e.g.

```
github.com/laconic-smart-solution/communication-service/app/line-service/internal/service
github.com/laconic-smart-solution/communication-service/app/line-service/pkg/utils
```

---

## 2. Import grouping

Exactly three groups, separated by a blank line, in this order:

```go
import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/models"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/repository"
)
```

1. Standard library. 2. Third-party. 3. This repository. `gofmt`/`goimports` enforces it.

---

## 3. The interface + impl + constructor pattern

Every **service**, **repository**, and **client** is an exported interface with an unexported
struct implementation and a `New…` constructor that returns the interface. Dependencies are
injected through the constructor — never package-level globals.

```go
type MessageService interface {
	Send(ctx context.Context, req request.SendMessageRequest, actor *uuid.UUID) (*response.MessageResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*response.MessageResponse, error)
}

type messageServiceImpl struct {
	logger  *logrus.Logger
	cfg     *config.Config
	repo    repository.MessageRepository
	channels repository.ChannelRepository
	client  client.LINEClient
}

var _ MessageService = (*messageServiceImpl)(nil)

func NewMessageService(
	logger *logrus.Logger,
	cfg *config.Config,
	repo repository.MessageRepository,
	channels repository.ChannelRepository,
	client client.LINEClient,
) MessageService {
	return &messageServiceImpl{logger: logger, cfg: cfg, repo: repo, channels: channels, client: client}
}
```

- Impl struct named `<thing>Impl` (e.g. `messageServiceImpl`, `lineClientImpl`).
- Constructor field order matches struct field order.
- Add `var _ Iface = (*impl)(nil)` for every service (catches drift).
- Receivers: `s` service, `r` repository, `h` handler, `c` client.

---

## 4. Client layer (vendor API wrapper)

Each service owns exactly **one** vendor integration under `internal/client/`. There is no
shared provider registry — the service already knows its platform.

```go
// app/line-service/internal/client/line_client.go
type SendMessageInput struct {
	ChannelAccessToken string
	RecipientID        string
	Text               string
}

type LINEClient interface {
	SendMessage(ctx context.Context, in SendMessageInput) (externalID string, err error)
	VerifyWebhook(ctx context.Context, signature string, body []byte) error
	ParseWebhook(ctx context.Context, body []byte) ([]client.InboundEvent, error)
	RefreshToken(ctx context.Context, refreshToken string) (client.TokenPair, error)
}
```

**Client rules:**

- All HTTP to Facebook / Instagram / LINE stays inside that service's `internal/client/`.
- Map vendor errors to sentinels (`ErrRateLimited`, `ErrInvalidToken`); the service translates
  those into user-safe messages.
- Never log access tokens, app secrets, or full webhook bodies.
- HTTP clients use timeouts from `config`; respect `ctx` cancellation.
- Webhook signature verification happens in the client; the handler only passes headers + raw body.

`facebook-service` and `instagram-service` may share Meta Graph patterns but each keeps its
own `internal/client/` — no cross-import between services.

---

## 5. Models

- Embed `models.Base` in **every** table. Never redefine audit columns.
- UUID primary keys, `default:gen_random_uuid()`.
- Declare `func (T) TableName() string` on every model.
- Secrets (`AccessToken`, `RefreshToken`) use `json:"-"` and never appear in response DTOs.
- Store vendor external IDs alongside internal UUIDs for correlation.
- Table names may be prefixed per service (e.g. `line_channels`, `fb_messages`) when services
  share one Postgres instance.

```go
type Channel struct {
	Base
	TenantID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	ExternalID     string     `gorm:"type:varchar(128);not null" json:"external_id"`
	DisplayName    string     `gorm:"type:varchar(255)" json:"display_name"`
	AccessToken    string     `gorm:"type:text;not null" json:"-"`
	RefreshToken   string     `gorm:"type:text" json:"-"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty"`
}

func (Channel) TableName() string { return "line_channels" }
```

---

## 6. Repository

- One interface + impl per aggregate; holds only `db *gorm.DB`.
- Methods take/return models and primitives — **no Fiber, no DTOs, no client types**.
- Return raw GORM errors; the **service** translates them.
- Paginated `List` returns `([]T, total int64, error)` and clamps `page`/`limit` defensively.

Soft delete: write `deleted_by` then `db.Delete(...)`; treat `RowsAffected == 0` as not-found.

---

## 7. Service (business logic)

- The only layer with business rules.
- Receives `request.*` DTOs and `actor *uuid.UUID`; returns `response.*` DTOs.
- Loads channel tokens from the repository, delegates vendor calls to `internal/client`.

```go
ch, err := s.channels.GetByIDAndTenant(req.ChannelID, tenantID)
if err != nil {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("channel not found")
	}
	return nil, err
}

externalID, err := s.client.SendMessage(ctx, client.SendMessageInput{
	ChannelAccessToken: ch.AccessToken,
	RecipientID:        req.RecipientID,
	Text:               req.Text,
})
```

- **Error style:**
  - User-facing → `errors.New("channel already connected")` (lowercase, no punctuation).
  - Internal → log then generic: `s.logger.WithError(err).Error("send message")` →
    `errors.New("failed to send message")`.
  - Wrapping → `fmt.Errorf("line api: %w", err)`.
  - Sentinels → `var ErrTokenExpired = errors.New(...)`, use `errors.Is`.
- Never leak raw vendor/DB errors to API clients.
- Outbound flow: persist as `pending` → call client → update `status` + `external_id`.
- Convert to DTOs via `mapper`; never hand-build response structs in the service.

---

## 8. Handler (HTTP boundary)

Thin and uniform: **parse → validate → call service → map error to status → respond**.

Webhook handlers read the **raw body** before parsing (required for signature verification):

```go
func (h *WebhookHandler) Handle(c *fiber.Ctx) error {
	body := c.Body()
	if err := h.svc.HandleWebhook(c.Context(), c.Get("X-Line-Signature"), body); err != nil {
		return utils.BadRequest(c, err)
	}
	return c.SendStatus(fiber.StatusOK)
}
```

- `New…Handler` returns a concrete pointer (`*MessageHandler`).

---

## 9. Response envelope & helpers

Every response uses `pkg/response.APIResponse`, produced via `pkg/utils` helpers.

```go
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
```

Helpers: `utils.OK`, `utils.Created`, `utils.BadRequest`, `utils.NotFound`,
`utils.Unauthorized`, `utils.Internal`, `utils.TooManyRequests`.

Pagination: `response.PagedResponse[T]` + `response.PageMeta`.

Prefer identical envelope code in root `pkg/utils/`; service `internal/response/` holds only
platform-specific DTOs (e.g. `ChannelResponse`, `StoreResponse`).

---

## 10. Request DTOs & validation

- Inbound payloads use `json` + `validate` tags.
- Create requests use value fields with `required`; updates use pointers for partial patches.
- No `platform` field on requests — the service **is** the platform.
- Query params bind with `query:"…"` tags.

```go
type SendMessageRequest struct {
	ChannelID   uuid.UUID `json:"channel_id"   validate:"required"`
	RecipientID string    `json:"recipient_id" validate:"required"`
	Text        string    `json:"text"         validate:"required_without=MediaURL"`
	MediaURL    string    `json:"media_url"    validate:"omitempty,url"`
}
```

---

## 11. Mapper

Pure functions: models → response DTOs. Never include tokens or secrets.

---

## 12. Routes

`routes.Handlers` aggregates handler pointers; `routes.Register(app, h, auth)` wires
`/api/v1` on **this service's port**.

- `GET /healthz` — unguarded.
- `POST /webhooks` (and `GET /webhooks` for Meta verify) — unguarded, signature-verified in client.
- Business routes (`/channels`, `/messages`) — gateway auth required.
- Register literal paths before `/:id` params.

```go
// app/line-service/internal/routes/routes.go
func Register(app *fiber.App, h Handlers, auth fiber.Handler) {
	api := app.Group("/api/v1")
	api.Get("/healthz", healthz)
	api.Post("/webhooks", h.Webhook.Handle)

	channels := api.Group("/channels", auth)
	channels.Post("/", h.Channel.Connect)
	channels.Get("/", h.Channel.List)
	channels.Delete("/:id", h.Channel.Revoke)

	messages := api.Group("/messages", auth)
	messages.Post("/", h.Message.Send)
	messages.Get("/:id", h.Message.GetByID)
}
```

---

## 13. Config

- Typed `config.Config` per service via `config.Load()`.
- No scattered `os.Getenv` — everything funnels through `Config`.
- Platform credentials live only on the matching service.

| Service | Example env vars |
|---------|------------------|
| `facebook-service` | `FACEBOOK_APP_ID`, `FACEBOOK_APP_SECRET`, `FACEBOOK_VERIFY_TOKEN` |
| `instagram-service` | `INSTAGRAM_APP_ID`, `INSTAGRAM_APP_SECRET` |
| `line-service` | `LINE_CHANNEL_SECRET`, `LINE_CHANNEL_ACCESS_TOKEN` |
| all | `PORT`, `DATABASE_URL`, `GATEWAY_SECRET`, `LOG_LEVEL` |

Per-tenant tokens are stored in the database, not in env vars.

---

## 14. Database

- Each service owns its `database.Connect`, `autoMigrate`, and optional `seed`.
- Prefer a separate database or schema per service in production.
- `autoMigrate` lists every model explicitly; gated by `cfg.DBAutoMigrate`.

---

## 15. main.go (composition root)

Each service's `app/<platform>-service/cmd/main.go` follows the same sequence:

1. `cfg := config.Load()` → `log := newLogger(cfg.LogLevel)`.
2. Assert required secrets (`log.Fatal` if missing).
3. `database.Connect` → optional seed.
4. Construct `client.New…(cfg, log)`.
5. Construct **repositories → services → handlers**.
6. `fiber.New` with `AppName: "line-service"` (match folder name); middleware; `routes.Register`.
7. Listen; graceful shutdown on `SIGINT`/`SIGTERM` (10s timeout).

---

## 16. Logging

- One `*logrus.Logger`, JSON formatter, injected into services and clients.
- Internal errors: short lowercase operation name (`"send message"`).
- Never log OAuth tokens, webhook secrets, or message PII.

---

## 17. Context

- Service and client I/O methods take `ctx context.Context` as the **first** parameter.
- Handlers pass `c.Context()`.

---

## 18. Security defaults

- Gateway (Spring Cloud Gateway) handles CORS, JWT, and routes `/comms/{platform}/**` to
  platform services — see [API_GATEWAY.md](API_GATEWAY.md).
- Gateway injects `X-Gateway-Secret`; platform middleware validates it on business routes.
- Platform services are internal; webhook signatures verified in platform `internal/client/`.
- Tokens: `json:"-"` on models, never in response DTOs.
- Scope all queries by `tenant_id`.
- Idempotent webhook processing: deduplicate by vendor event ID.

---

## 19. Naming & style

| Thing | Convention | Example |
|-------|-----------|---------|
| Service folder | `<platform>-service` | `app/line-service/` |
| Files | `snake_case.go` | `message_repository.go` |
| Interfaces | noun, exported | `MessageService`, `LINEClient` |
| Impl structs | `<iface>Impl`, unexported | `messageServiceImpl` |
| Constructors | `New<Thing>` | `NewMessageService` |
| Receivers | 1 letter by layer | `s`, `r`, `h`, `c` |
| Errors | lowercase, no punctuation | `errors.New("channel not found")` |
| Sentinels | `Err…` | `ErrTokenExpired` |

Run `gofmt`/`goimports`. Exported identifiers get a doc comment starting with the name.

---

## 20. Checklist before opening a PR

- [ ] Changes are scoped to one `app/<platform>-service/` (or root `pkg/` if truly shared).
- [ ] No imports across services' `internal/` packages.
- [ ] Vendor HTTP/SDK code only in that service's `internal/client/`.
- [ ] Service/repo expose interface + `New…`; `var _ Iface = (*impl)(nil)` on services.
- [ ] Errors: user-safe out, internal logged + wrapped.
- [ ] Handler only parses/validates/maps; webhook handlers pass raw body.
- [ ] All responses use `utils.*` envelope helpers.
- [ ] New models embed `Base`, declare `TableName`, listed in `autoMigrate`.
- [ ] `gofmt`/`goimports` clean; imports in three-group order.
