# whatsapp-service

Stateless proxy for WhatsApp Cloud API (Meta Graph API). Mirrors line/telegram/viber-service.

## Endpoints

| Method | Path | Auth |
|--------|------|------|
| GET | `/api/v1/healthz` | Public |
| GET | `/api/v1/webhooks` | Public (hub.verify_token handshake) |
| POST | `/api/v1/webhooks` | Public (HMAC `X-Hub-Signature-256`) |
| POST | `/api/v1/messages` | Gateway + access token + phone number id |
| POST | `/api/v1/messages/batch` | Gateway + access token + phone number id |
| GET | `/api/v1/qr?phone=&text=&size=&format=json` | Gateway |

## Headers (outbound)

- `X-Gateway-Secret`
- `X-Tenant-ID` / `X-Actor-ID`
- `X-WhatsApp-Access-Token`
- `X-WhatsApp-Phone-Number-Id`

## Local run

```bat
run.bat
```

Port **8089**. Gateway: `/comms/whatsapp/**`.
