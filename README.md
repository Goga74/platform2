## Environment Variables - CRITICAL

**DO NOT RENAME existing variables!**

Strike2 production variables (LOCKED):
- `STRIKE2_PROXY_TOKEN`
- `STRIKE2_CAPTCHA_KEY`

# Platform2

Multi-project Go backend platform. Each project lives in `projects/` with shared infrastructure.

## Architecture

```
platform2/
├── cmd/server/main.go              # Single entry point
├── projects/
│   └── strike2/                    # Strike2 proxy service
│       ├── auth/auth.go            # Simple token authentication
│       ├── captcha/solver.go       # 2Captcha integration
│       ├── proxy/handler.go        # HTTP/HTTPS proxy with JA3 spoofing
│       ├── scraper/scraper.go      # URL fetcher with worker pool
│       ├── strike2.go              # Project initialization
│       └── routes.go               # API route handlers
├── internal/
│   ├── common/
│   │   ├── config/config.go        # Environment configuration
│   │   └── swagger/swagger.go      # Swagger UI
│   └── transport/                  # Shared uTLS transport
│       ├── fingerprints.go         # Chrome/Firefox/Safari fingerprints
│       └── utls_client.go          # uTLS HTTP client with HTTP/2
├── api/openapi.yaml                # Unified API spec
├── docker-compose.yml              # Dev deployment
├── docker-compose.prod.yml         # Production deployment
└── Dockerfile                      # Multi-stage build
```

## Strike2

Strike2 is a transparent HTTP/HTTPS proxy with JA3 fingerprint spoofing. It works in two modes:

**Proxy Mode** — Use as HTTP/HTTPS proxy to route traffic through Strike2:
```bash
curl -x http://user:YOUR_TOKEN@localhost:8075 https://httpbin.org/ip
```

**API Mode** — Call REST endpoints to fetch URLs:
```bash
curl -X POST http://localhost:8075/api/strike2/fetch \
  -H "Content-Type: application/json" \
  -d '{"url": "https://httpbin.org/ip", "fingerprint": "chrome"}'
```

## Quick Start

```bash
# 1. Copy environment file
cp .env.example .env
# Edit .env — set PROXY_TOKEN at minimum

# 2. Start with Docker
docker-compose up -d

# 3. Test
curl http://localhost:8075/health
curl http://localhost:8075/api/strike2/health
curl -x http://user:YOUR_TOKEN@localhost:8075 https://httpbin.org/ip
```

### Without Docker

```bash
# Requires Go 1.23+
export PROXY_TOKEN=your_token_here
make run
```

## API Endpoints

### Platform

| Endpoint | Description |
|---|---|
| `GET /health` | Platform health check |
| `GET /swagger` | API documentation |

### Strike2

| Endpoint | Description |
|---|---|
| `GET /api/strike2/health` | Strike2 health + feature status |
| `POST /api/strike2/fetch` | Fetch single URL |
| `POST /api/strike2/v1/fetch` | Fetch single URL (v1) |
| `POST /api/strike2/v1/batch` | Batch fetch (max 100 URLs) |
| `GET /api/strike2/v1/fingerprints` | List available fingerprints |
| `POST /api/strike2/v1/captcha/solve/amazon-waf` | Solve Amazon WAF captcha |
| `GET /api/strike2/v1/captcha/balance` | 2Captcha account balance |

### Proxy

| Method | Description |
|---|---|
| `CONNECT host:443` | HTTPS tunnel (requires Proxy-Authorization) |
| `GET http://...` | HTTP proxy (requires Proxy-Authorization) |

## Authentication

Strike2 uses Simple Auth — a single `PROXY_TOKEN` environment variable.

**Proxy requests** require `Proxy-Authorization: Basic base64(user:token)` header. The username is ignored; only the token is validated.

**API requests** (`/api/strike2/*`) do not require authentication.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Internal server port |
| `PROXY_TOKEN` | — | Token for proxy authentication |
| `CAPTCHA_API_KEY` | — | 2Captcha API key (optional) |
| `UPSTREAM_PROXY` | — | Upstream proxy URL (optional) |
| `FINGERPRINT` | `chrome` | Default browser fingerprint |
| `WORKERS` | `500` | Worker pool size |

## Ports

- **8080** — Internal container port
- **8075** — External exposed port (docker-compose)

## Adding a New Project

1. Create `projects/yourproject/` directory
2. Add initialization and route registration
3. Register in `cmd/server/main.go`
4. Add config fields to `internal/common/config/config.go`
5. Update `api/openapi.yaml`

## Makefile Commands

| Command | Description |
|---|---|
| `make build` | Build binary to `bin/platform` |
| `make run` | Run with `go run` |
| `make test` | Run all tests |
| `make docker-build` | Build Docker image |
| `make docker-up` | Start dev environment |
| `make docker-prod` | Start production |
