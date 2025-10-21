# QuantumSpring AI Proxy Fork

Enhanced fork of [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) with persistent usage statistics and real-time visualization dashboard.

## Overview

This fork adds two critical features for production AI infrastructure:

1. **Persistent Usage Statistics** - SQLite-backed storage ensures usage data survives server restarts
2. **On-Box Visualization** - Web dashboard for monitoring token consumption, costs, and model usage

Perfect for teams using AI APIs (OpenAI, Claude, Gemini) who need granular usage tracking without sending data to external services.

---

## Features

✅ **Stats survive restarts** - SQLite persistence with configurable retention
✅ **Real-time metrics API** - JSON endpoints for `/metrics` with time-range queries
✅ **Beautiful dashboard** - Modern web UI with Chart.js visualizations
✅ **Localhost-first security** - Binds to `127.0.0.1` by default, optional Basic Auth
✅ **Zero external dependencies** - Pure Go SQLite, single binary deployment
✅ **Docker ready** - Pre-configured Dockerfile and docker-compose

---

## Quick Start

### Option 1: Run from Binary

```bash
# Build
go build -o cli-proxy-api ./cmd/server

# Configure
cp config.quantumspring.yaml config.yaml
# Edit config.yaml with your settings

# Run
./cli-proxy-api --config config.yaml
```

### Option 2: Run with Docker

```bash
# Build image
docker build -t quantumspring/ai-proxy .

# Run container
docker run -d \
  -p 8317:8317 \
  -v $(pwd)/data:/data \
  -v $(pwd)/config.yaml:/app/config.yaml \
  --name ai-proxy \
  quantumspring/ai-proxy
```

### Option 3: Docker Compose

```bash
docker compose up -d
```

---

## Endpoints

| Endpoint | Description |
|----------|-------------|
| `http://localhost:8317` | Proxy API (OpenAI/Claude/Gemini compatible) |
| `http://localhost:8317/_qs/health` | Health check (persistence status) |
| `http://localhost:8317/_qs/metrics` | Usage metrics (JSON) |
| `http://localhost:8317/_qs/metrics/ui` | Web dashboard |

---

## Configuration

### Minimal Configuration

```yaml
# config.yaml
port: 8317
usage-statistics-enabled: true

persistence:
  enabled: true
  type: sqlite
  path: "./data/usage.db"
  buffer-size: 100
  flush-interval: 10s
  retention-days: 90

quantumspring:
  enabled: true
  bind-address: "127.0.0.1"  # localhost only
  basic-auth:
    username: ""  # empty = no auth
    password: ""
```

### Production Configuration

```yaml
# config.yaml
port: 8317
usage-statistics-enabled: true

persistence:
  enabled: true
  type: sqlite
  path: "/data/usage.db"
  buffer-size: 250
  flush-interval: 5s
  retention-days: 90

quantumspring:
  enabled: true
  bind-address: "0.0.0.0"  # allow remote access
  basic-auth:
    username: "admin"
    password: "your-secure-password-here"

# Your existing proxy config...
api-keys:
  - "your-api-key-1"

claude-api-key:
  - api-key: "sk-ant-..."

codex-api-key:
  - api-key: "sk-..."
```

---

## API Reference

### GET `/_qs/health`

Health check endpoint.

**Response:**
```json
{
  "ok": true,
  "version": "1.0.0",
  "persistence": "sqlite",
  "statistics_enabled": true,
  "persistence_enabled": true,
  "total_records_persisted": 15234
}
```

---

### GET `/_qs/metrics`

Query usage metrics.

**Query Parameters:**
- `from` (optional) - ISO 8601 timestamp, default: 24h ago
- `to` (optional) - ISO 8601 timestamp, default: now
- `model` (optional) - filter by model name
- `interval` (optional) - `hour`, `day`, `week`, `month` (default: hour)

**Example:**
```bash
curl "http://localhost:8317/_qs/metrics?from=2025-01-20T00:00:00Z&to=2025-01-21T23:59:59Z&interval=hour"
```

**Response:**
```json
{
  "totals": {
    "requests": 1523,
    "tokens": 2450000,
    "prompt_tokens": 1200000,
    "completion_tokens": 1250000,
    "reasoning_tokens": 0,
    "cached_tokens": 45000,
    "failed_requests": 12,
    "success_rate": 99.2,
    "avg_latency_ms": 2345.67
  },
  "by_model": [
    {
      "model": "gpt-4",
      "requests": 850,
      "tokens": 1500000,
      "prompt_tokens": 700000,
      "completion_tokens": 800000,
      "avg_latency_ms": 2345,
      "failed_requests": 5
    },
    {
      "model": "claude-3-5-sonnet-20241022",
      "requests": 673,
      "tokens": 950000,
      "prompt_tokens": 500000,
      "completion_tokens": 450000,
      "avg_latency_ms": 1890,
      "failed_requests": 7
    }
  ],
  "by_api_key": [
    {
      "api_key": "***abc",
      "requests": 234,
      "tokens": 345000
    }
  ],
  "by_provider": [
    {
      "provider": "openai",
      "requests": 850,
      "tokens": 1500000,
      "avg_latency_ms": 2345
    }
  ],
  "timeseries": [
    {
      "bucket_start": "2025-01-21T00:00:00Z",
      "requests": 45,
      "tokens": 125000,
      "prompt_tokens": 60000,
      "completion_tokens": 65000,
      "avg_latency_ms": 2100,
      "failed_requests": 1
    }
  ],
  "query_period": {
    "from": "2025-01-20T00:00:00Z",
    "to": "2025-01-21T23:59:59Z"
  }
}
```

---

### GET `/_qs/metrics/ui`

Web dashboard with real-time visualizations.

Open in browser: `http://localhost:8317/_qs/metrics/ui`

Features:
- **KPI Cards** - Total requests, tokens, success rate, avg latency
- **Tokens Over Time** - Line chart showing hourly/daily trends
- **Usage by Model** - Doughnut chart of token distribution
- **Requests by Provider** - Bar chart of provider usage
- **Auto-refresh** - Updates every 30 seconds

---

## Security

### Localhost-Only Mode (Default)

```yaml
quantumspring:
  bind-address: "127.0.0.1"  # Only accessible from same machine
```

### Remote Access with Basic Auth

```yaml
quantumspring:
  bind-address: "0.0.0.0"
  basic-auth:
    username: "admin"
    password: "your-secure-password"
```

**Access with authentication:**
```bash
curl -u admin:your-secure-password http://your-server:8317/_qs/metrics
```

### Best Practices

- **Never log secrets** - Authorization headers are automatically sanitized
- **Rotate credentials** - Change Basic Auth password regularly
- **Use TLS** - Put reverse proxy (nginx/Caddy) in front with HTTPS
- **Firewall** - Restrict access to metrics endpoints at network level
- **API key masking** - Keys are automatically masked in responses (`***abc`)

---

## Persistence

### SQLite Backend

Default storage with zero configuration.

**Pros:**
- ✅ Single file database
- ✅ Zero external dependencies
- ✅ Fast queries with indexes
- ✅ ACID compliance
- ✅ Works in Docker/containers

**Data location:**
- Default: `./data/usage.db`
- Customizable via `persistence.path`

### Retention Policy

```yaml
persistence:
  retention-days: 90  # Delete records older than 90 days
```

Set to `0` for infinite retention (not recommended).

**Cleanup runs automatically:**
- Daily at midnight
- Deletes records older than `retention-days`
- Logged with count of deleted records

### Database Schema

```sql
CREATE TABLE usage_records (
    id INTEGER PRIMARY KEY,
    timestamp DATETIME NOT NULL,
    request_id TEXT,
    api_key TEXT,
    source TEXT,
    auth_id TEXT,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,

    -- Token metrics
    prompt_tokens INTEGER,
    completion_tokens INTEGER,
    reasoning_tokens INTEGER,
    cached_tokens INTEGER,
    total_tokens INTEGER,

    -- Request status
    status INTEGER,
    failed BOOLEAN,
    latency_ms INTEGER,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for fast queries
CREATE INDEX idx_usage_timestamp ON usage_records(timestamp);
CREATE INDEX idx_usage_model ON usage_records(model);
CREATE INDEX idx_usage_api_key ON usage_records(api_key);
```

---

## Deployment

### Single Binary

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -o cli-proxy-api ./cmd/server

# Deploy to server
scp cli-proxy-api user@server:/opt/ai-proxy/
scp config.yaml user@server:/opt/ai-proxy/

# Run with systemd
sudo systemctl start ai-proxy
```

### Docker

```bash
# Build
docker build -t quantumspring/ai-proxy .

# Run with volume persistence
docker run -d \
  -p 8317:8317 \
  -v /opt/ai-proxy/data:/data \
  -v /opt/ai-proxy/config.yaml:/app/config.yaml \
  --restart unless-stopped \
  quantumspring/ai-proxy
```

### Docker Compose

```yaml
version: '3.8'

services:
  ai-proxy:
    image: quantumspring/ai-proxy:latest
    ports:
      - "8317:8317"
    volumes:
      - ./data:/data
      - ./config.yaml:/app/config.yaml
    environment:
      - TZ=America/New_York
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8317/_qs/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

---

## Monitoring & Observability

### Prometheus Metrics (Future)

Planned for v2.0:
- Prometheus endpoint at `/_qs/metrics/prometheus`
- Standard metrics (requests, latency, errors)
- Custom metrics (tokens, costs by model)

### Grafana Dashboard (Future)

Pre-built dashboard for Grafana:
- Real-time token consumption
- Cost tracking by model/team
- Anomaly detection
- Quota alerts

---

## Troubleshooting

### Persistence not working

**Symptom:** Stats reset on restart

**Check:**
```yaml
persistence:
  enabled: true  # Must be true
usage-statistics-enabled: true  # Must be true
```

**Verify:**
```bash
curl http://localhost:8317/_qs/health
# Check: "persistence_enabled": true
```

### UI not loading

**Symptom:** 404 on `/_qs/metrics/ui`

**Check:**
```yaml
quantumspring:
  enabled: true  # Must be true
```

**Verify:**
```bash
curl http://localhost:8317/_qs/health
# Should return 200 OK
```

### Permission denied (Docker)

**Symptom:** Cannot write to `/data/usage.db`

**Solution:**
```bash
# Create data directory with correct permissions
mkdir -p data
chmod 777 data  # Or use proper user/group

# Run Docker with user
docker run -u $(id -u):$(id -g) ...
```

---

## Development

### Running Tests

```bash
# Unit tests
go test ./internal/persistence/...

# Integration tests
go test ./internal/api/handlers/quantumspring/...

# All tests
go test ./...
```

### Building from Source

```bash
git clone https://github.com/your-fork/CLIProxyAPI.git
cd CLIProxyAPI
go mod download
go build -o cli-proxy-api ./cmd/server
```

---

## License

MIT License - see [LICENSE](LICENSE) file for details.

---

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

---

## Support

- **Issues:** [GitHub Issues](https://github.com/your-fork/CLIProxyAPI/issues)
- **Discussions:** [GitHub Discussions](https://github.com/your-fork/CLIProxyAPI/discussions)
- **Email:** support@quantumspring.ai

---

## Acknowledgments

Built on top of the excellent [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) project.

Special thanks to:
- Original CLIProxyAPI maintainers
- Chart.js for visualization library
- modernc.org/sqlite for pure Go SQLite

---

**Made with ❤️ for QuantumSpring AI Infrastructure**
