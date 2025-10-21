# QuantumSpring Testing Guide

## 🚀 Quick Start

### Option 1: Automated Test Suite (Recommended)

```bash
cd /Users/miroslavlalik/repos/CLIProxyAPI
./test_quantumspring.sh
```

This will:
- ✅ Check all required files
- ✅ Download dependencies
- ✅ Verify code compiles
- ✅ Run unit tests
- ✅ Run integration tests
- ✅ Generate coverage report

---

## 🧪 Manual Testing

### Step 1: Build the Binary

```bash
cd /Users/miroslavlalik/repos/CLIProxyAPI

# Build
go build -o cli-proxy-api ./cmd/server

# Verify
./cli-proxy-api --version
```

**Expected:** Binary compiles without errors

---

### Step 2: Test Configuration

```bash
# Copy example config
cp config.quantumspring.yaml config.yaml

# Verify config syntax
cat config.yaml
```

**Expected:** Valid YAML configuration

---

### Step 3: Run Unit Tests

```bash
# Persistence layer tests
go test -v ./internal/persistence/

# Expected output:
# === RUN   TestNewSQLiteStorage
# --- PASS: TestNewSQLiteStorage
# === RUN   TestInsert
# --- PASS: TestInsert
# === RUN   TestInsertBatch
# --- PASS: TestInsertBatch
# === RUN   TestGetTotals
# --- PASS: TestGetTotals
# === RUN   TestGetByModel
# --- PASS: TestGetByModel
# === RUN   TestGetTimeseries
# --- PASS: TestGetTimeseries
# === RUN   TestCleanup
# --- PASS: TestCleanup
# === RUN   TestQuery
# --- PASS: TestQuery
# PASS
```

---

### Step 4: Run Integration Tests

```bash
# API handler tests
go test -v ./internal/api/handlers/quantumspring/

# Expected output:
# === RUN   TestGetHealth
# --- PASS: TestGetHealth
# === RUN   TestGetMetricsWithoutPersistence
# --- PASS: TestGetMetricsWithoutPersistence
# === RUN   TestServeUI
# --- PASS: TestServeUI
# === RUN   TestServeUIJavaScript
# --- PASS: TestServeUIJavaScript
# PASS
```

---

### Step 5: Start the Server

```bash
# Run server
./cli-proxy-api --config config.yaml
```

**Expected log output:**
```
INFO[0000] CLIProxyAPI Version: 1.0.0, Commit: xxx, BuiltAt: xxx
INFO[0000] SQLite storage initialized                    path="./data/usage.db"
INFO[0000] Persistence initialized successfully          buffer_size=100 flush_interval=10s retention_days=90 type=sqlite
INFO[0000] QuantumSpring metrics API registered          auth_enabled=false bind_address="127.0.0.1" prefix="/_qs"
INFO[0000] Server started on :8317
```

---

### Step 6: Test Health Endpoint

```bash
# In another terminal
curl http://localhost:8317/_qs/health | jq
```

**Expected response:**
```json
{
  "ok": true,
  "version": "1.0.0",
  "persistence": "sqlite",
  "statistics_enabled": true,
  "persistence_enabled": true,
  "total_records_persisted": 0
}
```

---

### Step 7: Test Metrics Endpoint

```bash
curl http://localhost:8317/_qs/metrics | jq
```

**Expected response:**
```json
{
  "totals": {
    "requests": 0,
    "tokens": 0,
    "prompt_tokens": 0,
    "completion_tokens": 0,
    "failed_requests": 0,
    "success_rate": 0,
    "avg_latency_ms": 0
  },
  "by_model": [],
  "timeseries": [],
  "query_period": {
    "from": "2025-01-20T...",
    "to": "2025-01-21T..."
  }
}
```

---

### Step 8: Test Web UI

```bash
# Open in browser
open http://localhost:8317/_qs/metrics/ui
```

**Expected:**
- ✅ Dashboard loads
- ✅ Shows "Online" status badge
- ✅ Displays 4 KPI cards (all showing 0)
- ✅ Shows empty charts
- ✅ Auto-refresh message at bottom

---

### Step 9: Make a Test Request

```bash
# Make a test API request to generate usage data
curl -X POST http://localhost:8317/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-key" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

**Note:** This will fail (no real API key), but should still log usage

---

### Step 10: Verify Persistence

```bash
# Check database file was created
ls -lh ./data/usage.db

# Expected: File exists with non-zero size

# Query database directly (optional)
sqlite3 ./data/usage.db "SELECT COUNT(*) FROM usage_records;"
```

---

### Step 11: Test Time-Range Queries

```bash
# Query last hour
from=$(date -u -v-1H +"%Y-%m-%dT%H:%M:%SZ")
to=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

curl "http://localhost:8317/_qs/metrics?from=$from&to=$to" | jq
```

---

### Step 12: Test Server Restart (Persistence)

```bash
# Stop server (Ctrl+C)

# Restart
./cli-proxy-api --config config.yaml

# Check metrics again
curl http://localhost:8317/_qs/metrics | jq

# Expected: Previous data still there!
```

---

## 🐳 Docker Testing

### Build Docker Image

```bash
docker build -f Dockerfile.quantumspring -t quantumspring/ai-proxy .
```

**Expected:** Build completes without errors

---

### Run Container

```bash
# Create config
mkdir -p $(pwd)/config
cp config.quantumspring.yaml $(pwd)/config/config.yaml

# Run container
docker run -d \
  -p 8317:8317 \
  -v $(pwd)/data:/data \
  -v $(pwd)/config:/app/config \
  --name ai-proxy-test \
  quantumspring/ai-proxy
```

---

### Test Docker Health

```bash
# Check container is running
docker ps | grep ai-proxy-test

# Check health
docker exec ai-proxy-test wget -O- http://localhost:8317/_qs/health

# Check logs
docker logs ai-proxy-test
```

---

### Cleanup

```bash
docker stop ai-proxy-test
docker rm ai-proxy-test
```

---

## 📊 Coverage Testing

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./internal/persistence/ ./internal/api/handlers/quantumspring/

# View coverage summary
go tool cover -func=coverage.out

# Open HTML coverage report
go tool cover -html=coverage.out
```

**Expected coverage:**
- `persistence/sqlite.go`: >80%
- `handlers/quantumspring/metrics.go`: >70%

---

## 🔍 Troubleshooting

### Issue: Tests fail with "cannot find package"

**Solution:**
```bash
go mod tidy
go mod download
```

---

### Issue: Database locked error

**Solution:**
```bash
# Stop all running instances
pkill cli-proxy-api

# Remove lock files
rm -f ./data/usage.db-wal ./data/usage.db-shm
```

---

### Issue: Port 8317 already in use

**Solution:**
```bash
# Find process using port
lsof -i :8317

# Kill process
kill -9 <PID>

# Or change port in config.yaml
```

---

### Issue: Web UI doesn't load

**Check:**
1. Server is running: `curl http://localhost:8317/_qs/health`
2. QuantumSpring is enabled in config: `quantumspring.enabled: true`
3. Check browser console for errors
4. Verify embedded assets: Files exist in `internal/api/handlers/quantumspring/web/quantumspring/`

---

## ✅ Success Criteria

All tests pass if:

- ✅ `go test ./internal/persistence/` passes all tests
- ✅ `go test ./internal/api/handlers/quantumspring/` passes all tests
- ✅ Binary compiles without errors
- ✅ Server starts and binds to port 8317
- ✅ Health endpoint returns 200 OK
- ✅ Metrics endpoint returns valid JSON
- ✅ Web UI loads in browser
- ✅ Database file is created in `./data/`
- ✅ Data persists after restart
- ✅ Docker image builds and runs

---

## 📝 Test Checklist

Copy this checklist and mark as you test:

```
[ ] Code compiles
[ ] Unit tests pass (persistence)
[ ] Integration tests pass (API)
[ ] Server starts successfully
[ ] Health endpoint works
[ ] Metrics endpoint works
[ ] Web UI loads
[ ] Charts render (with data)
[ ] Database file created
[ ] Data persists after restart
[ ] Docker build succeeds
[ ] Docker container runs
[ ] All logs show no errors
```

---

## 🎯 Next Steps

Once all tests pass:

1. Configure your actual API keys in `config.yaml`
2. Set up Basic Auth if needed
3. Deploy to production
4. Monitor dashboard at `/_qs/metrics/ui`
5. Set up retention policy
6. Configure backups for `./data/usage.db`

---

**Happy Testing! 🚀**
