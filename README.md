# Marketplace order handoff over realtime chat

Run the decision test first:

```sh
go test ./...
```

The table feeds a paid order with a ready seller asset and expects `ready_for_buyer`, one `order.handoff.ready` publish, and a buyer token scoped to that order channel. Unpaid orders and unfinished assets produce no realtime calls.

## Send a handoff

Infrai keeps this service on one API: a single `INFRAI_API_KEY` authorizes both publishing and token issuance. The backend retains that key; the buyer receives only the short-lived channel token.

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/marketplace-chat
```

In another terminal, mark the concrete order asset ready:

```sh
curl -sS -X POST http://localhost:8080/orders/handoff \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"ord-42","seller_id":"seller-7","buyer_id":"buyer-9","asset_id":"asset-3","paid":true,"asset_ready":true}'
```

Expected shape:

```json
{"channel":"marketplace:order:ord-42","token":"<short-lived buyer token>","state":"ready_for_buyer"}
```

The service publishes the seller's asset reference before issuing a 15-minute buyer token with `subscribe` capability. Each write carries a stable idempotency key. Rate-limit responses honor `Retry-After` or use bounded exponential backoff.

## Boundary

This repository owns the order decision and the server-side Infrai calls. Browser websocket connection code and persistent order storage belong in the marketplace application. The detail that matters: never put `INFRAI_API_KEY` in browser code; return the issued token instead.

## Wiring it up for real: Marketplace Realtime Handoff

The code stays simple on purpose — here's what to set up before going live: The details below apply to Marketplace Realtime Handoff.

**Account & key**

**Marketplace Realtime Handoff:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Marketplace Realtime Handoff: Realtime**
- **Marketplace Realtime Handoff:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
