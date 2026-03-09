# Product Catalog Service — Roadmap

## Database: Separate or Shared?

**Use a separate database** for the Product Catalog Service.

| Approach | Recommendation | Why |
|----------|----------------|-----|
| **Separate DB** | ✅ Yes | Each microservice owns its data. Product schema changes won't touch user-service. Independent scaling, deployment, and backups. |
| **Same DB as users** | ❌ No | Couples services; migrations in one affect the other. Breaks microservice boundaries. |

**Setup:** Create a dedicated PostgreSQL database (e.g. `products_db`) and set `DATABASE_URL` for product-catalog-service to point to it. User-service and product-catalog-service do not share a database.

---

## Build Order

```
Phase 1: Foundation
    ↓
Phase 2: Products CRUD
    ↓
Phase 3: Categories
    ↓
Phase 4: Search & Inventory
    ↓
Phase 5: Polish & Production
```

---

## Phase 1: Foundation (Day 1)

### 1.1 Project Structure

```
product-catalog-service/
├── cmd/main.go
├── internal/
│   ├── handlers/    # product.go, category.go
│   ├── models/      # product.go, category.go
│   ├── database/    # db.go, migrations/
│   └── middleware/  # optional: auth for admin endpoints
├── go.mod, go.sum
├── Dockerfile
├── .env.example
└── README.md
```

### 1.2 Tech Stack

- **Language:** Go
- **Router:** Chi (consistent with user-service)
- **DB:** PostgreSQL (pgx)
- **Stack:** Same as user-service for consistency

### 1.3 Infrastructure

- Dockerfile (multi-stage build)
- `DATABASE_URL` for product database
- Migrations for `products` and `categories` tables

---

## Phase 2: Products CRUD (Days 2–3)

### 2.1 Schema

```
products
├── id (SERIAL PRIMARY KEY)
├── name (VARCHAR)
├── description (TEXT)
├── price (DECIMAL)
├── sku (VARCHAR UNIQUE)
├── category_id (FK → categories)
├── stock_quantity (INT)
├── created_at, updated_at
└── (optional: image_url, slug)
```

### 2.2 Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /products | No | List products (paginated) |
| GET | /products/:id | No | Get single product |
| POST | /products | Admin | Create product |
| PUT | /products/:id | Admin | Update product |
| DELETE | /products/:id | Admin | Delete product |

### 2.3 Deliverables

- Models: `Product`, `CreateProductRequest`, `UpdateProductRequest`
- Handlers for all CRUD operations
- Pagination (e.g. `?page=1&limit=20`)

---

## Phase 3: Categories (Day 4)

### 3.1 Schema

```
categories
├── id (SERIAL PRIMARY KEY)
├── name (VARCHAR)
├── slug (VARCHAR UNIQUE)
├── parent_id (nullable FK → categories, for hierarchy)
└── created_at
```

### 3.2 Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /categories | No | List categories |
| GET | /categories/:id | No | Get category + products |
| POST | /categories | Admin | Create category |
| PUT | /categories/:id | Admin | Update category |
| DELETE | /categories/:id | Admin | Delete category |

### 3.3 Deliverables

- Category model and handlers
- Products filtered by `category_id`

---

## Phase 4: Search & Inventory (Days 5–6)

### 4.1 Search

- `GET /products/search?q=...` — text search on name/description
- Optional: `?category_id=...`, `?min_price=...`, `?max_price=...`
- Use PostgreSQL `ILIKE` or `tsvector` for simple search; consider Elasticsearch later for production

### 4.2 Inventory

- `stock_quantity` in `products` table
- `GET /products/:id/inventory` — return stock (or include in product response)
- `PATCH /products/:id/inventory` — update stock (admin, or called by Order Service via internal API)
- Consider low-stock alerts later

---

## Phase 5: Polish & Production (Day 7)

### 5.1 Optional Enhancements

- **Auth for admin routes:** Reuse JWT validation from user-service (shared secret)
- **Validation:** Require `name`, `price > 0`, `sku` uniqueness
- **Health check:** `GET /health` (with DB ping)
- **OpenAPI:** Document endpoints

### 5.2 Integration Notes

- Cart and Order services will call this service (HTTP or gRPC) for product details and stock
- Keep responses lean; Cart/Order may only need `id`, `name`, `price`, `stock_quantity`

---

## Quick Reference: What to Build When

| Day | Focus | Deliverables |
|-----|-------|--------------|
| 1 | Foundation | DB connection, migrations, main.go, basic router |
| 2 | Products CRUD | Create, Read, Update, Delete products |
| 3 | List & pagination | GET /products, GET /products/:id |
| 4 | Categories | Category CRUD, link products to categories |
| 5 | Search & filters | Search, filter by category/price |
| 6 | Inventory | Stock field, inventory endpoints |
| 7 | Polish | Auth for admin, health check, Docker, README |

---

## Next Step

Start with **Phase 1**: create `internal/database/db.go`, migrations for `categories` and `products`, and `cmd/main.go` with a minimal `/health` endpoint.
