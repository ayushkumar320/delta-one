# Delta One — Event Ticketing Platform

A microservices learning project: Go backend, React frontend, Docker-orchestrated.

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
│   ├── events/         # event payload structs (publisher + consumer)
│   └── httpx/          # JSON helpers, error-to-status mapping
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

## Getting started

```bash
cp .env.example .env    # then fill in the blanks
make help               # list available tasks
```

Run a backend service:

```bash
make run s=auth
```

Run the frontend dev server:

```bash
make web
```

## Why no monorepo framework

The backend is a single Go module, so the Go toolchain handles builds and
caching. The frontend is one Vite app. Turborepo, Nx and Lerna orchestrate
multiple JavaScript packages — there is only one here, and they cannot build
the Go side at all. A Makefile covers the whole repository in twenty lines.

Revisit this if the frontend ever splits into several packages.
