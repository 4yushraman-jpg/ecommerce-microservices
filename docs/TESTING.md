# E-Commerce Platform — Manual Testing Guide

Run through this flow to validate user-service, product-catalog-service, and cart-service.

---

## Prerequisites

1. **PostgreSQL** — Create 3 databases: `users`, `products`, `carts`
2. **Run migrations** for each service (see each service README)
3. **Start all 3 services** (each on different port, or use defaults: 8080, 8081, 8082)

### Suggested Ports

| Service | Port | Base URL |
|---------|------|----------|
| user-service | 8080 | http://localhost:8080 |
| product-catalog-service | 8081 | http://localhost:8081 |
| cart-service | 8082 | http://localhost:8082 |

Set `PORT` in each service's `.env` before starting.

---

## 1. User Service

### 1.1 Register
```bash
curl -X POST http://localhost:8080/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```
Expected: `201` with `{"message":"Signup successful","user_id":1}`

### 1.2 Login
```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```
Expected: `200` with `{"token":"eyJ..."}` — save the token for later.

### 1.3 Get Profile (use token from login)
```bash
curl -X GET http://localhost:8080/users/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```
Expected: `200` with user object

---

## 2. Product Catalog Service

### 2.1 Create Category (admin required)

First, create an admin user in the users DB (set `role = 'admin'`) or use a user that has admin role. Then:

```bash
curl -X POST http://localhost:8081/categories \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"name":"Electronics","slug":"electronics","parent_id":null}'
```
Expected: `201` with `{"message":"Category created"}`

### 2.2 Create Product (admin)
```bash
curl -X POST http://localhost:8081/products \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"name":"Laptop","description":"A great laptop","price":99900,"sku":"LAP-001","category_id":1,"stock_quantity":10}'
```
Expected: `201` with `{"message":"Product created","id":1}`

### 2.3 List Products (no auth)
```bash
curl -X GET "http://localhost:8081/products?limit=10&page=1"
```
Expected: `200` with array of products

### 2.4 Get Product by ID
```bash
curl -X GET http://localhost:8081/products/1
```
Expected: `200` with product object

### 2.5 List Categories
```bash
curl -X GET http://localhost:8081/categories
```
Expected: `200` with array of categories

---

## 3. Cart Service

### 3.1 Add Item to Cart
```bash
curl -X POST http://localhost:8082/carts/me/items \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"product_id":1,"quantity":2}'
```
Expected: `201` with `{"message":"Product added to cart successfully"}`

### 3.2 Get Cart
```bash
curl -X GET http://localhost:8082/carts/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```
Expected: `200` with cart (id, user_id, items)

### 3.3 Update Item Quantity
```bash
curl -X PUT http://localhost:8082/carts/me/items/1 \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"quantity":3}'
```
Expected: `200` with `{"message":"Product quantity updated"}`

### 3.4 Remove Item (quantity = 0)
```bash
curl -X PUT http://localhost:8082/carts/me/items/1 \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"quantity":0}'
```
Expected: `204` No Content

### 3.5 Clear Cart
```bash
curl -X DELETE http://localhost:8082/carts/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```
Expected: `204` No Content

---

## End-to-End Flow Summary

1. Register user → Login → Save token
2. (Optional) Create admin user and add category + product via product-catalog-service
3. Add product to cart → Get cart → Update quantity → (optional) remove item or clear cart

---

## Health Checks

| Service | Endpoint | Auth |
|---------|----------|------|
| product-catalog-service | GET /health | No |
| cart-service | GET /health | No |
| user-service | GET /health | No |

```bash
curl http://localhost:8080/health   # user
curl http://localhost:8081/health   # product (if implemented)
curl http://localhost:8082/health   # cart
```

---

## Common Issues

- **401 Unauthorized** — Token missing, expired, or invalid. Login again.
- **404 Cart not found** — Cart doesn't exist yet; add an item first.
- **Database connection failed** — Ensure PostgreSQL is running and `DATABASE_URL` in `.env` is correct.
- **JWT_SECRET mismatch** — All services must use the same `JWT_SECRET`.
