# viber-service

Stateless proxy for Viber Bot API (mirrors telegram-service / line-service).

## Endpoints

| Method | Path | Auth |
|--------|------|------|
| GET | `/api/v1/healthz` | Public |
| POST | `/api/v1/webhooks` | Public (HMAC) |
| POST | `/api/v1/messages` | Gateway + `X-Viber-Auth-Token` |
| POST | `/api/v1/messages/batch` | Gateway + `X-Viber-Auth-Token` |
| GET | `/api/v1/qr?uri=&start=&size=&format=json` | Gateway |

## Headers

- `X-Gateway-Secret`
- `X-Tenant-ID` / `X-Actor-ID` (messages)
- `X-Viber-Auth-Token` (messages)
- Webhook: `X-Viber-Content-Signature` + query `auth_token` (HMAC key) + optional `callback_url`

## Local run

```bat
run.bat
```

Default port **8088**. Gateway path: `/comms/viber/**`.

## Nest wiring

neutral-tex server uses `VIBER_SERVICE_URL`, `VIBER_AUTH_TOKEN`, `VIBER_URI`.
Viber requires HTTPS for inbound webhooks (no long-polling).
