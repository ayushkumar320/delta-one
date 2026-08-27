# Delta One — Event Ticketing Platform

A microservices learning project: Go backend, React frontend, Postgres and Redis.

## Layout

```
.
├── go.mod              # single Go module for the whole backend
├── Makefile            # task runner
├── services/           # one directory per Go service
│   ├── auth/           # signup, login, JWT issuing
│   ├── catalog/        # events, venues, seat maps (read-heavy)
│   ├── booking/        # seat hold and confirm (the hard one)
│   ├── payment/        # simulated payment gateway
│   ├── notification/   # async email/SMS worker
│   └── gateway/        # API gateway: routing, auth check, rate limit
├── shared/             # code imported by more than one service
│   ├── middleware/     # logging, recovery, request ID
│   ├── events/         # event payloads + the Redis Streams bus
│   ├── httpx/          # JSON helpers, error-to-status mapping, service client
│   ├── config/         # environment, Postgres and Redis clients
│   ├── migrate/        # applies a service's embedded SQL at startup
│   └── token/          # issues and verifies access tokens
├── migrations/         # SQL migrations, one directory per service database
└── frontend/           # React + TypeScript + Vite
```

Each service follows the same internal structure:

```
services/<name>/
├── cmd/server/         # main.go — wiring only
└── internal/
    ├── domain/         # entities and business rules; imports nothing
    ├── repository/     # database access
    ├── service/        # business logic
    └── transport/      # HTTP handlers and DTOs
```

Dependencies point inward. `domain` imports no other layer.

## Prerequisites

- Go 1.23 or newer
- Node.js 22 or newer
- Docker, for Postgres and Redis

## Getting started

Start the two pieces of infrastructure:

```bash
docker run -d --name delta-postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:17
docker run -d --name delta-redis -p 6379:6379 redis:8
```

Configure and create the databases:

```bash
cp .env.example .env         # then set DB_PASSWORD and JWT_SECRET
make databases               # creates one database per service
```

Run everything:

```bash
make run-all                 # all six services
make web                     # the React dev server on :5173
```

Each service applies its own migrations at startup, so there is no separate
migration step. The catalog seeds three demo events the first time it runs.

To run one service on its own:

```bash
make run s=booking
```

## How it fits together

The gateway on `:8080` is the only public entry point. It verifies the access
token, strips any client-supplied identity headers, rate limits by IP, and
proxies to the service that owns the route. Services behind it trust the
`X-User-ID`, `X-User-Email` and `X-User-Role` headers the gateway sets.

Synchronous calls go over HTTP: booking asks catalog what seats cost and asks
payment to charge them. Anything that can happen later is an event on a Redis
stream, which notification consumes as a consumer group.

Redis Streams stands in for RabbitMQ here. Consumer groups already give each
service every event exactly once with redelivery of anything unacknowledged,
and Redis is running anyway for rate limiting, so it is one fewer container.

### Where correctness actually lives

Two seats being sold twice, or one card being charged twice, are prevented by
database constraints rather than by application logic:

- `booking_seats` has a partial unique index over unreleased rows, so two
  concurrent holds on the same seat both reach the insert and exactly one
  commits.
- `payments` has a partial unique index over succeeded rows, so a retried
  charge returns the original payment instead of taking the money again.

A hold lasts ten minutes. Expired holds are released inside the next hold
transaction for that event, so an abandoned checkout stops blocking a seat
immediately; the background sweeper only controls how quickly the seat map
catches up and the customer is told.

## API

Everything below is served under `/api` by the gateway.

| Method | Path                         | Who        |
| ------ | ---------------------------- | ---------- |
| POST   | `/auth/register`             | anyone     |
| POST   | `/auth/login`                | anyone     |
| GET    | `/auth/me`                   | signed in  |
| GET    | `/events`                    | anyone     |
| GET    | `/events/{id}`               | anyone     |
| GET    | `/events/{id}/seats`         | anyone     |
| GET    | `/events/{id}/taken-seats`   | anyone     |
| POST   | `/events`, `/venues`         | organizers |
| POST   | `/bookings`                  | signed in  |
| GET    | `/bookings`, `/bookings/{id}`| signed in  |
| POST   | `/bookings/{id}/confirm`     | signed in  |
| DELETE | `/bookings/{id}`             | signed in  |

### Test cards

The payment service is simulated. `tok_visa` approves; `tok_decline` and
`tok_insufficient_funds` fail, which is how the checkout page offers them.

## Why no monorepo framework

The backend is a single Go module, so the Go toolchain handles builds and
caching. The frontend is one Vite app. Turborepo, Nx and Lerna orchestrate
multiple JavaScript packages — there is only one here, and they cannot build
the Go side at all. A Makefile covers the whole repository in twenty lines.

Revisit this if the frontend ever splits into several packages.
