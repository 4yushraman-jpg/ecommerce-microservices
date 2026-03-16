# Cart Service — Roadmap

## Dependencies

**Depends on:** User Service (auth), Product Catalog Service (product details, price, stock).

- Cart needs `user_id` from JWT for authenticated users
- Cart needs to fetch product info (name, price, stock) from product-catalog-service
- Optionally: support anonymous carts (session/cookie)

---

## Database: Redis vs PostgreSQL

| Approach | Use When | Pros | Cons |
|----------|----------|------|------|
| **Redis** | Ephemeral carts, high throughput | Fast, TTL for expiry, simple | No persistence by default, separate infra |
| **PostgreSQL** | Persist carts, analytics | Same stack as others, durable, queryable | Heavier than Redis for frequent updates |

**Recommendation:** Start with **PostgreSQL** for consistency with user/product services. Add Redis later for performance if needed.

**Setup:** Create `carts_db` and set `DATABASE_URL` for cart-service.

---

## Build Order

```
Phase 1: Foundation
    ↓
Phase 2: Cart CRUD
    ↓
Phase 3: Integration with Product Service
    ↓
Phase 4: Auth & Polish
```

---

## Phase 1: Foundation (Day 1)

### 1.1 Project Structure

```
cart-service/
├── cmd/main.go
├── internal/
│   ├── handlers/     # cart.go
│   ├── models/       # cart.go
│   ├── database/     # db.go, migrations/
│   └── middleware/   # auth (reuse JWT pattern)
├── go.mod, go.sum
├── Dockerfile
└── .env.example
```

### 1.2 Tech Stack

- **Language:** Go (consistent with user-service, product-catalog-service)
- **Router:** Chi
- **DB:** PostgreSQL (pgx)
- **Auth:** JWT validation (same secret as user-service)

### 1.3 Schema

```
carts
├── id (SERIAL PRIMARY KEY)
├── user_id (INTEGER NOT NULL, UNIQUE)  # one cart per user
├── created_at, updated_at
└── expires_at (optional, for TTL)

cart_items
├── id (SERIAL PRIMARY KEY)
├── cart_id (FK → carts)
├── product_id (INTEGER NOT NULL)       # no FK; product-catalog owns products
├── quantity (INTEGER NOT NULL)
├── created_at, updated_at
└── UNIQUE(cart_id, product_id)         # one row per product per cart
```

---

## Phase 2: Cart CRUD (Days 2–3)

### 2.1 Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /carts/me | Yes | Get current user's cart |
| POST | /carts/me/items | Yes | Add item (product_id, quantity) |
| PUT | /carts/me/items/:product_id | Yes | Update quantity |
| DELETE | /carts/me/items/:product_id | Yes | Remove item |
| DELETE | /carts/me | Yes | Clear cart |

### 2.2 Models

- `Cart` — id, user_id, items, created_at, updated_at
- `CartItem` — product_id, quantity (and product details from product-service)
- `AddItemRequest` — product_id, quantity
- `UpdateItemRequest` — quantity

### 2.3 Logic

- Create cart on first add if none exists
- Validate quantity > 0
- Merge or replace when adding same product_id

---

## Phase 3: Integration with Product Service (Day 4)

### 3.1 Product Validation

- On add/update: call product-catalog-service `GET /products/:id` to verify product exists and get price/name
- Check stock_quantity >= requested quantity
- Store product_id and quantity only; enrich response with product details from product-service when returning cart

### 3.2 Cart Response

```json
{
  "id": 1,
  "user_id": 42,
  "items": [
    {
      "product_id": 5,
      "quantity": 2,
      "name": "Widget",
      "price": 999,
      "stock_quantity": 10
    }
  ]
}
```

- Fetch product details from product-catalog-service when building GET /carts/me response

### 3.3 Resilience

- Handle product-service unavailable (return cached name/price or error)
- Consider circuit breaker for product-service calls

---

## Phase 4: Auth & Polish (Day 5)

### 4.1 Auth

- All cart endpoints require JWT (AuthMiddleware)
- Use `user_id` from JWT claims; no `userId` in URL (use `/carts/me`)
- Same JWT_SECRET as user-service

### 4.2 Optional Enhancements

- **Anonymous carts:** Store in Redis by session_id, merge into user cart on login
- **Cart expiry:** TTL on Redis or `expires_at` in DB
- **Health check:** GET /health
- **Logging:** Request logging, error logging

---

## Quick Reference: What to Build When

| Day | Focus | Deliverables |
|-----|-------|--------------|
| 1 | Foundation | DB, migrations (carts, cart_items), main.go, router |
| 2 | Cart CRUD | Add, update, remove items; get cart |
| 3 | Handlers | Full CRUD, validation |
| 4 | Product integration | HTTP client to product-service, enrich cart response |
| 5 | Auth, polish | JWT middleware, health check, Docker |

---

## Next Step

Start with **Phase 1**: implement `internal/database/db.go`, migrations for `carts` and `cart_items`, and `cmd/main.go` with routes.
