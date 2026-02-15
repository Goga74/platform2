# Platform2

Multi-project Go backend platform. Each project lives in `projects/` with shared infrastructure.

## Architecture

```
platform2/
├── cmd/server/main.go              # Single entry point
├── projects/
│   └── strike2/                    # First project (migrated from strike2 repo)
│       ├── handlers/
│       ├── models/
│       ├── repository/
│       ├── migrations/
│       └── routes.go
├── internal/
│   ├── common/
│   │   ├── database/postgres.go    # PostgreSQL connection + schema support
│   │   ├── config/config.go        # Environment configuration
│   │   └── swagger/swagger.go      # Swagger UI
│   └── transport/                  # Shared transport (uTLS, proxies)
├── api/
│   └── openapi.yaml                # Unified API spec
├── docker-compose.yml              # Dev (includes PostgreSQL)
├── docker-compose.prod.yml         # Prod (managed DB)
└── Dockerfile                      # Multi-stage build
```

## Database Strategy

Single PostgreSQL instance with **schema-per-project**:
- `strike2` schema for Strike2 tables
- Future projects get their own schemas
- One `DATABASE_URL`, isolated data per project

## Quick Start

```bash
# 1. Copy environment file
cp .env.example .env
# Edit .env with your values

# 2. Start with Docker (includes PostgreSQL)
docker-compose up -d

# 3. Run migrations
make migrate

# 4. Access
# API:     http://localhost:8080/health
# Swagger: http://localhost:8080/swagger
# Strike2: http://localhost:8080/api/strike2/health
```

### Without Docker

```bash
# Requires Go 1.23+ and PostgreSQL running locally
make build
./bin/platform
```

## API Endpoints

| Endpoint | Description |
|---|---|
| `GET /health` | Platform health check |
| `GET /swagger` | API documentation |
| `GET /api/strike2/health` | Strike2 project health |

## Adding a New Project

1. Create `projects/yourproject/` directory
2. Add `routes.go` with `RegisterRoutes(rg *gin.RouterGroup, db *database.DB)`
3. Create migration in `projects/yourproject/migrations/`
4. Register in `cmd/server/main.go`:
   ```go
   yourGroup := r.Group("/api/yourproject")
   yourproject.RegisterRoutes(yourGroup, db)
   ```
5. Add config fields to `internal/common/config/config.go`
6. Update `api/openapi.yaml`

## Deployment

### DigitalOcean (Production)

```bash
# Uses managed PostgreSQL (no local DB container)
docker-compose -f docker-compose.prod.yml up -d
```

### Environment Variables

See `.env.example` for all available configuration.

## Makefile Commands

| Command | Description |
|---|---|
| `make build` | Build binary to `bin/platform` |
| `make run` | Run with `go run` |
| `make test` | Run all tests |
| `make docker-build` | Build Docker image |
| `make docker-up` | Start dev environment |
| `make docker-prod` | Start production |
| `make migrate` | Run database migrations |
