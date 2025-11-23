# k6 Load Testing with Grafana Monitoring

This setup provides real-time visualization of k6 load tests using Grafana and InfluxDB.

## 🚀 Quick Start

### 1. Start the Monitoring Stack
```bash
make monitoring-up
```

This starts:
- **Grafana** at http://localhost:3000 (auto-login enabled)
- **InfluxDB** at http://localhost:8086

### 2. Run Tests with Visualization

#### Smoke Test (30s with 5 VUs)
```bash
make load-smoke-monitor
```

#### Load Test (5min with up to 200 VUs)
```bash
make load-test-monitor
```

#### Stress Test (16min with up to 1000 VUs)
```bash
make load-stress-monitor
```

#### Spike Test (4min with traffic bursts)
```bash
make load-spike-monitor
```

### 3. View Results
Open http://localhost:3000 in your browser to see:
- **Real-time metrics** as tests run
- **Per-endpoint response times** (p95, p99, avg)
- **Error rates** by endpoint
- **Request rates** and throughput
- **Virtual Users** over time

## 📊 Available Metrics

### Per-Endpoint Latency
- `event_post_duration` - POST /events response times
- `bulk_post_duration` - POST /events/bulk response times  
- `metrics_get_duration` - GET /metrics response times

Each includes: **min**, **avg**, **median**, **max**, **p(90)**, **p(95)**, **p(99)**

### Request Counts
- `event_posts_total` - Single event POST count
- `bulk_posts_total` - Bulk event POST count
- `metrics_gets_total` - Metrics GET count

### Error Rates
- `event_post_errors` - POST /events error rate
- `bulk_post_errors` - POST /events/bulk error rate
- `metrics_get_errors` - GET /metrics error rate

### Standard k6 Metrics
- `http_req_duration` - Overall response time
- `http_req_failed` - Overall error rate
- `http_reqs` - Total request rate
- `vus` - Active virtual users
- `iterations` - Test iterations

## 🛠️ Manual Usage

Run any k6 test with InfluxDB output:

```bash
# Start monitoring
docker compose --profile monitoring up -d

# Run test with metrics export
docker compose --profile load-test run --rm \
  -e K6_OUT=influxdb=http://influxdb:8086/k6 \
  k6 run /scripts/smoke.js
```

## 📈 Dashboard Features

The pre-configured dashboard shows:

1. **Virtual Users** - Active VUs over time
2. **Request Rate** - Requests per second
3. **Response Time by Endpoint**:
   - POST /events (p95, p99, avg)
   - POST /events/bulk (p95, p99, avg)
   - GET /metrics (p95, p99, avg)
4. **Error Rates** - Per-endpoint error percentages
5. **Request Counts** - Throughput by endpoint

All panels auto-refresh every 5 seconds during tests.

## 🧹 Cleanup

```bash
# Stop monitoring stack
make monitoring-down

# Stop all services
make down
```

## 💡 Tips

- **Run tests with monitoring** to see real-time graphs
- **Compare test runs** by looking at historical data in Grafana
- **Identify bottlenecks** by comparing response times across endpoints
- **Spot errors quickly** with the error rate panel
- Dashboard persists in Docker volume, survives restarts

## 📦 What's Included

```
config/grafana/provisioning/
├── datasources/
│   └── influxdb.yml          # InfluxDB datasource config
└── dashboards/
    ├── dashboard.yml          # Dashboard provider config
    └── k6-dashboard.json      # Pre-built k6 dashboard
```

## 🔗 Access URLs

- **Grafana Dashboard**: http://localhost:3000
- **InfluxDB**: http://localhost:8086
- **Application**: http://localhost:50051
- **ClickHouse UI**: http://localhost:8080
- **Redis UI**: http://localhost:8081

## 🐛 Troubleshooting

**Dashboard not showing data?**
- Ensure monitoring stack is running: `docker compose ps | grep -E "grafana|influxdb"`
- Check k6 output includes: `output: InfluxDBv1 (http://influxdb:8086)`
- Wait a few seconds after starting the test

**Can't access Grafana?**
- Check Grafana logs: `docker logs clickhouse-demo-grafana-kucukaslan`
- Verify port 3000 is not in use: `lsof -i :3000`

**Metrics not appearing?**
- Verify InfluxDB is healthy: `curl http://localhost:8086/ping`
- Check database exists: `curl -G http://localhost:8086/query --data-urlencode "q=SHOW DATABASES"`
