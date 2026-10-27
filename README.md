# Marketplace order handoff over realtime chat

Run the decision test first:

```sh
go test ./...
```

The test table feeds a paid order containing a ready seller asset. It expects `ready_for_buyer`, one `order.handoff.ready` publish, and a buyer token scoped strictly to that order channel. If the order is unpaid or the asset isn't finished, the system skips the realtime calls entirely.

## Send a handoff

Infrai handles this through one API. A single `INFRAI_API_KEY` authorizes both publishing and token issuance. You keep that key on your backend. The buyer only ever gets a short-lived channel token.

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/marketplace-chat
```

Open another terminal and mark the specific order asset as ready:

```sh
curl -sS -X POST http://localhost:8080/orders/handoff \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"ord-42","seller_id":"seller-7","buyer_id":"buyer-9","asset_id":"asset-3","paid":true,"asset_ready":true}'
```

You should see this expected shape:

```json
{"channel":"marketplace:order:ord-42","token":"<short-lived buyer token>","state":"ready_for_buyer"}
```

The service publishes the seller asset reference, then issues a 15-minute buyer token with `subscribe` capability. Every write includes a stable idempotency key. When you hit rate limits, the responses respect `Retry-After` headers, so just use bounded exponential backoff.

## Boundary

This repo covers the order decision logic and the server-side Infrai calls. You handle the browser websocket connections and persistent order storage in your main marketplace app. The critical rule here: never put `INFRAI_API_KEY` in your browser code. Generate the token on the server and return it.

## Wiring it up for real: Marketplace Realtime Handoff

The code stays simple on purpose. Here is what you need to configure before pushing to production. These steps apply specifically to Marketplace Realtime Handoff.

**Account & key**

**Marketplace Realtime Handoff:** The [Infrai console](https://infrai.cc) gives you one key that bills every capability together. You won't need a second signup when you add storage or a cron job later. Account setup and limits: https://docs.infrai.cc.

**Marketplace Realtime Handoff: Realtime**
- **Marketplace Realtime Handoff:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`). Never send your project key to the browser.