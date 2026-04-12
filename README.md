# E-Commerce Microservices

A Go-based e-commerce backend built with a microservices architecture.

This repository includes:
- API gateway routing with Traefik
- Independent services for users, catalog, cart, orders, payments, and notifications
- PostgreSQL database per service
- Kafka for async event flow (order -> payment -> notification)
- Docker Compose orchestration for local development

## Project Structure

```text
ecommerce-microservices/
├── gateway/
├── infrastructure/
├── services/
│   ├── user-service/
│   ├── product-catalog-service/
│   ├── cart-service/
│   ├── order-service/
│   ├── payment-service/
│   └── notification-service/
├── shared/
├── docs/
├── docker-compose.yml
├── docker-compose.override.yml
└── .env.example
```

## Services

| Service | Purpose | Database | Access Pattern |
|---|---|---|---|
| user-service | Signup, login, profile | users | Sync via gateway |
| product-catalog-service | Products, categories, stock | products | Sync via gateway |
| cart-service | User carts and line items | carts | Sync via gateway |
| order-service | Order creation and order lifecycle | orders | Sync via gateway |
| payment-service | Stripe payment processing and webhooks | payments | Sync via gateway + Kafka |
| notification-service | Email notification pipeline (SendGrid) | notifications | Async (Kafka consumer) |

## Gateway Routes

Traefik is exposed on port 80 with the following route prefixes:

- /user -> user-service
- /catalog -> product-catalog-service
- /cart -> cart-service
- /order -> order-service
- /payments -> payment-service

Traefik dashboard: http://localhost:8088

## Prerequisites

- Docker Desktop (or Docker Engine + Compose)
- Go 1.22+ (only needed for running services outside Docker)
- Stripe CLI (optional, for local webhook testing)

## Quick Start (Docker)

1. Create root environment file.

```powershell
Copy-Item .env.example .env
```

2. Add missing variables in .env if needed.

The current .env.example includes user/product/cart database URLs and JWT secret.
Add this for order-service as well:

```env
ORDER_DATABASE_URL=postgres://postgres:postgres@orders-db:5432/orders?sslmode=disable
```

3. Create service-specific env files used by Compose.

```powershell
Copy-Item services/payment-service/.env.example services/payment-service/.env
Copy-Item services/notification-service/.env.example services/notification-service/.env
```

4. Start the full stack.

```powershell
docker compose up --build
```

5. Stop the stack.

```powershell
docker compose down
```

To also remove volumes:

```powershell
docker compose down -v
```

## Local Development (Without Docker)

Each service can be run independently:

1. Create a PostgreSQL database for the service.
2. Copy its .env.example to .env and configure DATABASE_URL, JWT_SECRET, and PORT.
3. Run migrations from internal/database/migrations.
4. Start service from its folder:

```powershell
go run ./cmd
```

## Testing

Manual integration flow for user, catalog, and cart services is documented in:

- docs/TESTING.md

## Roadmaps

- Root implementation roadmap: ROADMAP.md
- Service-level roadmaps:
  - services/cart-service/ROADMAP.md
  - services/product-catalog-service/ROADMAP.md

## Notes

- All services that validate JWT should share the same JWT secret.
- payment-service expects Stripe keys in services/payment-service/.env.
- notification-service expects SendGrid credentials in services/notification-service/.env.
- Kafka broker is available at kafka:9092 within the Compose network.

## Project Page URL

- https://roadmap.sh/projects/scalable-ecommerce-platform
