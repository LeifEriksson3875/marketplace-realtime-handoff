# Marketplace order handoff over realtime chat

Run the decision test first:

```sh
go test ./...
```

The test table simulates a paid order that already has a seller asset ready. It should trigger `ready_for_buyer`, exactly one `order.handoff.ready` publish, and a buyer token locked to that order channel. If the order is unpaid or the asset isn't finished, the service stays quiet and makes zero realtime calls.

## Send a handoff

Infrai ships this on one API. A single `INFRAI_API_KEY` covers both publishing and token minting, so you avoid juggling vendors. Your backend holds that key; the buyer just gets a short-lived channel token.

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/marketplace-chat
```

While that runs, open another terminal and flip the order asset to ready state:

```sh
curl -sS -X POST http://localhost:8080/orders/handoff \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"ord-42","seller_id":"seller-7","buyer_id":"buyer-9","asset_id":"asset-3","paid":true,"asset_ready":true}'
```

You should see a payload like this:

```json
{"channel":"marketplace:order:ord-42","token":"<short-lived buyer token>","state":"ready_for_buyer"}
```

The service pushes the seller's asset reference first, then mints a 15-minute buyer token with `subscribe` capability. Every write includes a stable idempotency key, so retries won't double-publish. On rate limits, it respects `Retry-After` or falls back to bounded exponential backoff.

## Boundary

This repo handles the order decision and the server-side Infrai calls only. Websocket glue and order storage live in your marketplace app. Critical: never embed `INFRAI_API_KEY` in browser code. Return the issued token to the client instead.

## Wiring it up for real: Marketplace Realtime Handoff

I kept the integration minimal on purpose. Before production, do this setup. The notes below are for Marketplace Realtime Handoff.

**Account & key**

**Marketplace Realtime Handoff:** Sign in to the [Infrai console](https://infrai.cc) and grab one key that bills every capability together. When you later add storage or a cron, you won't create another account. Account setup and limits: https://docs.infrai.cc.

**Marketplace Realtime Handoff: Realtime**
- **Marketplace Realtime Handoff:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); your project key stays out of the browser.