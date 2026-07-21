# WeChat Service — Integration Guide

`wechat-service` is a **stateless** WeChat Official Account (公众号) proxy on port **8090**.
It does **not** store OA credentials or OpenIDs. Calling apps own AppID/AppSecret / access_token.

Gateway path: `/comms/wechat/**`.

**Full setup (env, run, curl, OA admin):** see [`SETUP.md`](./SETUP.md).  
**Mosonic product setup:** [`neutral-tex/docs/setup-wechat.md`](../../../neutral-tex/docs/setup-wechat.md).

## Routes

| Method | Path | Auth |
|--------|------|------|
| GET | `/api/v1/healthz` | public |
| GET | `/api/v1/webhooks` | public (URL verify / echostr) |
| POST | `/api/v1/webhooks` | public (XML → optional callback) |
| POST | `/api/v1/messages` | `X-Gateway-Secret` + `X-WeChat-Access-Token` |
| POST | `/api/v1/messages/batch` | same |
| GET | `/api/v1/qr?scene_str=…` | same (+ access token for scene QR) |

## Send

```json
{ "recipient_id": "<openid>", "text": "hello" }
```

Proxies to `cgi-bin/message/custom/send` (48h customer-service window).

## QR

- `scene_str` + access token → temporary WeChat scene QR (`QR_STR_SCENE`)
- or `oa_username` → fallback deep-link QR (user must send CONNECT in chat)

## Headers

- `X-Gateway-Secret`
- `X-WeChat-Access-Token`
- `X-Tenant-ID` / `X-Actor-ID` (UUIDs)
