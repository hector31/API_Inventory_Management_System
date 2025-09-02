# Technical Design — Distributed Inventory (Concise README)

This document addresses the exercise’s three asks:

1. **Distributed architecture** that fixes consistency & latency.
2. **API design** for key inventory operations.
3. **Justification** of why these choices fit a distributed system with stores.

---

## 1) Distributed Architecture

**Goal:** Prevent overselling while removing the 15‑minute batch lag.

**CAP stance:** **CP for writes** — the **Central Inventory API** is the *single authority* for stock mutations. If a store can’t reach it, writes are **not committed locally** (they may be queued). Reads can continue from a **local snapshot** at the store (eventual consistency for reads only).

**High‑level flow**

```mermaid
flowchart LR
  subgraph Store Node(s)
    UI[POS / Checkout]
    SAPI[Store Node API]
    SCache[(store_state.json)]
  end
  UI -->|REST| SAPI
  SAPI -->|POST updates / sync| API[Central Inventory API]
  SAPI <-->|GET reads| SCache
  API --> SVC[Service: OCC + Idempotency]
  SVC --> MEM[(In‑memory State)]
  SVC --> EVT[events.log.jsonl]
  SVC -->|periodic| SNAP[state.json]
  SNAP --> BOOT
  EVT --> BOOT
  BOOT --> MEM
  SVC --> OTel[OpenTelemetry] --> COL[Collector] --> Jaeger[Jaeger]
```

**Write path (strong consistency)**

1. Store sends `POST /inventory/updates` with **`idempotencyKey`** (client‑generated, 256‑bit) and **`version`** (client’s last seen).
2. Central validates, enforces **OCC** per product, applies the mutation.
3. Central updates in‑memory state, appends an **event** to JSONL, snapshots periodically (atomic write).
4. Response returns new quantity & version. **Store Node immediately updates its local cache** for that product using the response.

**Read path (available during outages)**

* Store Node serves reads from its **local snapshot** when Central is down (responses include `"stale": true`).
* Store Node **never decrements locally** while offline; it either **queues** write intents to replay later or returns 503.

**Recovery**

* On restart, Central loads `state.json` then **replays** `events.log.jsonl` to rebuild exact state. Invalid events → `deadletter.jsonl`.

---

## 2) Store Synchronization (How local DB is updated)

We use **two complementary mechanisms** to keep stores fresh:

### A) Immediate local update on successful write

After a successful `POST /inventory/updates`, Store Node updates its `store_state.json` for the affected product using `newQuantity`/`newVersion` from the response. This gives **instant feedback** for items the store just touched.

### B) Background catch‑up via replication API (pull deltas)

Each Store Node runs a background syncer that **long‑polls** Central for **ordered change events** and applies them in order. Two endpoints:

**GET** `/replication/snapshot`

```json
{
  "state": { "SKU-1": {"qty":20,"version":3,"updatedAt":"..."}, "SKU-2": {"qty":5,"version":1,"updatedAt":"..."} },
  "lastOffset": 1287
}
```

Use on first run or if the store is too far behind.

**GET** `/replication/changes?fromOffset=1287&limit=500&longPollSeconds=20`

```json
{
  "events": [
    { "seq":1288, "type":"StockDecreased", "productId":"SKU-2", "storeId":"store-7", "delta":-1, "newVersion":2, "ts":"..." }
  ],
  "nextOffset": 1288,
  "hasMore": false
}
```

* **`seq`** is a monotonically increasing sequence assigned on commit.
* If `fromOffset` is older than retention, Central returns **410 Gone** → the store should call `/replication/snapshot` again.

**Recommended cadence**

* **Long‑poll** changes with `longPollSeconds=20` (near real‑time, low overhead).
* Fallback: simple polling every **2s** if you prefer no server hold logic.

**Local persistence at the store**

* `store_state.json` — the local read cache (atomic write).
* `store_offset.json` — `{ "lastOffset": <seq> }` to resume long‑poll after restarts.
* Snapshot the store cache every **10s** or after **100** applied events.

---

## 3) API Design (key operations)

**Common**

* Auth: `X-API-Key` header.
* Error shape:

```json
{ "code":"bad_request|unauthorized|conflict|unprocessable|internal", "message":"...", "details":[{"field":"delta","issue":"must be non-zero"}] }
```

* Mutations require **`idempotencyKey`** and **`version`**.

### 3.1 Mutate stock

**POST** `/inventory/updates`

```json
// Request
{ "storeId":"store-7", "productId":"SKU-123", "delta":-1, "version":7, "idempotencyKey":"b64url-256bit" }
// 200 OK (applied)
{ "productId":"SKU-123", "newQuantity":19, "newVersion":8, "applied":true, "lastUpdated":"2025-09-01T03:10:00Z" }
// Errors: 400, 401, 409 (stale version), 422 (qty<0)
```

### 3.2 Bulk sync (reconciliation/cold start)

**POST** `/inventory/sync`

```json
{ "storeId":"store-7", "mode":"merge", "products":[ {"id":"SKU-1","qty":20,"version":3} ] }
// 200 OK: { "updated":10, "created":5, "skipped":2 }
```

### 3.3 Read product

**GET** `/inventory/{productId}` → `{ "productId":"SKU-123", "available":19, "version":8, "lastUpdated":"..." }`

### 3.4 Global availability

**GET** `/inventory/global/{productId}` → `{ "productId":"SKU-123", "totalAvailable":420, "perStore": {"store-1":50, "store-7":19} }`

### 3.5 Listing (cursor)

**GET** `/inventory?cursor=&limit=50` → `{ "items":[...], "nextCursor":"opaque-or-empty" }`

### 3.6 Replication (store sync)

* **GET** `/replication/snapshot` — bootstrap full state with `lastOffset`.
* **GET** `/replication/changes?fromOffset=&limit=&longPollSeconds=` — ordered deltas for long‑polling.

### 3.7 Store Node API (optional, for simulation)

* **GET** `/store/inventory/{productId}` — serve from local cache; add `"stale": true` when Central is down.
* **POST** `/store/inventory/updates` — forward to Central with the **same** `idempotencyKey` & `version`; if Central is down, queue or return 503.

---

## 4) Justification (why this fits the distributed scenario)

* **Consistency first (no oversell):** Single writer (Central) + **OCC** per product ensures only one winner per version; **idempotency** de‑duplicates retries/timeouts.
* **Low latency:** Immediate local cache update on successful writes + **long‑poll replication** keeps stores fresh within seconds. No 15‑minute batch windows.
* **Fault tolerance & auditability:** **Snapshot + ordered event log** allow deterministic recovery and time travel; **DLQ** isolates poison events.
* **Simplicity & evolvability:** Pure Go + JSON files are easy to run/grade; architecture cleanly upgrades to Kafka/Redpanda (for events) and SQLite/Postgres (for persistence) without changing business semantics.
* **Observability:** OpenTelemetry traces/metrics/logs across HTTP and domain ops → fast debugging and performance validation.

---

## 5) Setup & Quickstart

**Prereqs:** Go 1.22+, (optional) Docker for OTel Collector + Jaeger.

```bash
# Central API
make run              # or: go run ./cmd/server
# Env (defaults)
# PORT=8080 SNAPSHOT_INTERVAL=10s RATE_TOKENS=100 RATE_PERIOD=60s \
# OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 API_KEYS=demo

# Optional Store Node on 8081 (reads cached, writes forwarded)
make run-store        # or: go run ./cmd/store

# Observability stack (optional)
docker compose up -d  # starts Collector + Jaeger (http://localhost:16686)
```

**Smoke test**

```bash
# Write via Central
curl -H 'X-API-Key: demo' -H 'Content-Type: application/json' \
  -X POST :8080/inventory/updates \
  -d '{"storeId":"store-7","productId":"SKU-123","delta":5,"version":0,"idempotencyKey":"k1"}'

# Read via Central
curl -H 'X-API-Key: demo' :8080/inventory/SKU-123

# Read via Store Node (uses local cache if Central is down)
curl :8081/store/inventory/SKU-123
```

---

## 6) Observability & Performance

* **OpenTelemetry**: `otelhttp` on routes; spans for `ApplyDelta`, `PersistSnapshot`, `ReplayEvent`; metrics `inventory_updates_total`, `request_latency_ms`, `snapshot_write_bytes`, `dlq_events_total`.
* **JMeter**: plans for read‑heavy, write‑heavy, and mixed; target p95: GET < 200ms, POST < 300ms locally.

---

## 7) Evolution (nice‑to‑haves)

* Replace JSONL with **Kafka/Redpanda** topic `inventory.events` (keyed by `productId`).
* Swap JSON snapshot for **SQLite/Postgres** with ACID and better concurrency.
* Add **gRPC** for low‑latency clients; keep REST for compatibility.
