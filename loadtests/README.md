# Load Testing with k6

This directory contains k6 load test scripts for the ClickHouse Event Tracking API.

## Quick Start

### Prerequisites

1. **Start the application**:
   ```bash
   make up
   ```

2. **Verify services are healthy**:
   ```bash
   docker compose ps
   ```

### Running Tests

#### Smoke Test (Quick Validation)
```bash
make load-smoke
```
- **Duration**: 1 minute
- **VUs**: 5
- **Purpose**: Quick validation after deployments

#### Load Test (Normal Capacity)
```bash
make load-test
```
- **Duration**: 5 minutes
- **Max VUs**: 200
- **Purpose**: Test sustained production load

#### Stress Test (Find Limits)
```bash
make load-stress
```
- **Duration**: 16 minutes
- **Max VUs**: 1000
- **Purpose**: Identify breaking points and capacity limits

#### Spike Test (Traffic Bursts)
```bash
make load-spike
```
- **Duration**: 4 minutes
- **Pattern**: 10 → 500 → 10 → 700 → 10 → 300 → 10 VUs
- **Purpose**: Test sudden traffic spikes and recovery

### Custom Tests

Run custom configurations:

```bash
# Recent data, 100 VUs, 5 minutes
make load-custom VUS=100 DURATION=5m MODE=recent SCRIPT=load

# Historical data, 500 VUs, 10 minutes
make load-custom VUS=500 DURATION=10m MODE=historical SCRIPT=stress

# Mixed timestamps, 50 VUs, 2 minutes
make load-custom VUS=50 DURATION=2m MODE=mixed SCRIPT=smoke

# Custom load distribution: 99% events, 0% bulk, 0.01% metrics
make load-custom SCRIPT=smoke EVENT_PCT=0.99 BULK_PCT=0.0 METRICS_PCT=0.01
```

### Load Distribution Configuration

All tests support custom load distribution via environment variables:

- `EVENT_PCT` - Percentage of single event requests (default varies by test)
- `BULK_PCT` - Percentage of bulk event requests (default varies by test)  
- `METRICS_PCT` - Percentage of metrics query requests (default varies by test)

**Examples:**
```bash
# 99% events, 0.01% metrics, no bulk
make load-smoke-monitoring EVENT_PCT=0.99 BULK_PCT=0.0 METRICS_PCT=0.01

# 80% events, 15% bulk, 5% metrics
make load-test-monitor EVENT_PCT=0.80 BULK_PCT=0.15 METRICS_PCT=0.05

# Events only (no bulk or metrics)
make load-stress-monitor EVENT_PCT=1.0 BULK_PCT=0.0 METRICS_PCT=0.0
```

## Directory Structure

```
loadtests/
├── README.md              # This file
├── METRICS.md             # Detailed metrics tracking and results
├── lib/
│   └── data-generator.js  # Shared data generation utilities
├── results/               # Test results (auto-generated)
│   └── .gitkeep
├── smoke.js               # Smoke test scenario
├── load.js                # Load test scenario
├── stress.js              # Stress test scenario
└── spike.js               # Spike test scenario
```

## Test Scenarios

### 1. Smoke Test (`smoke.js`)
- **Load**: 5 VUs, 1 minute
- **Default Workload**: 70% single event (`POST /events`), 20% bulk events (`POST /events/bulk`), 10% metrics (`GET /metrics`)
- **Thresholds**: Error rate < 1%, P95 < 1s, P99 < 2s

### 2. Load Test (`load.js`)
- **Load**: Ramp up to 200 VUs over 5 minutes
- **Default Workload**: 65% single event, 20% bulk events (100 events/batch), 15% metrics
- **Thresholds**: Error rate < 1%, P95 < 500ms, P99 < 1s, > 100 RPS

### 3. Stress Test (`stress.js`)
- **Load**: Ramp up to 1000 VUs over 16 minutes
- **Default Workload**: 90% single event, 9% bulk events (200 events/batch), 1% metrics
- **Thresholds**: Error rate < 5%, P95 < 2s, P99 < 5s (relaxed)

### 4. Spike Test (`spike.js`)
- **Load**: Multiple spike bursts (500, 700, 300 VUs)
- **Default Workload**: 90% single event, 9% bulk events (150 events/batch), 1% metrics
- **Thresholds**: Error rate < 3%, P95 < 1.5s, P99 < 3s

**Note:** All workload distributions can be customized using `EVENT_PCT`, `BULK_PCT`, and `METRICS_PCT` environment variables.
- **Workload**: 60% single event, 20% bulk events (150 events/batch), 20% metrics
- **Thresholds**: Error rate < 3%, P95 < 1.5s, P99 < 3s

## Timestamp Modes

Control event timestamp distribution with `TIMESTAMP_MODE`:

### `recent` (default)
- Events timestamped within the last hour
- Best for: Testing real-time ingestion and current data queries

### `historical`
- Events timestamped 1-30 days ago
- Best for: Testing queries on older data, backfill scenarios

### `mixed`
- 70% last 24 hours, 30% last 30 days
- Best for: Realistic production workload simulation

## Data Generation

### Pre-initialized Pools
- **1,000 users**: `user_000001` to `user_001000`
- **50 campaigns**: `summer_sale_2025_001`, `winter_promo_2025_002`, etc.
- **25 event types**: `purchase`, `page_view`, `button_click`, etc.
- **10 channels**: `web`, `mobile`, `ios`, `android`, etc.
- **18 tags**: `premium`, `free`, `trial`, `mobile`, etc.

### Event Metadata
Realistic metadata is generated based on event type:
- **Purchase events**: `amount`, `currency`, `items_count`, `payment_method`
- **Page views**: `page_url`, `referrer`, `duration_seconds`
- **Search events**: `query`, `results_count`
- **Video events**: `video_id`, `position_seconds`

## Understanding Results

### Key Metrics

```
http_req_duration............: avg=123.45ms p(95)=234.56ms p(99)=345.67ms
http_req_failed..............: 0.50%
http_reqs....................: 12345 (205.75 RPS)
events_posted................: 11000
bulk_events_posted...........: 50000
metrics_queried..............: 1345
```

- **http_req_duration**: Response time (focus on p95/p99)
- **http_req_failed**: Error rate percentage
- **http_reqs**: Total requests and rate (RPS)
- **events_posted**: Count of single event POSTs
- **bulk_events_posted**: Count of events inserted via bulk endpoint
- **metrics_queried**: Count of metrics queries

### Good Performance Indicators ✅
- Error rate < 1%
- P95 latency < 500ms
- Consistent throughput
- No timeouts or connection errors

### Warning Signs 🚩
- Error rate > 5%
- P99 latency > 5s
- Connection timeouts
- Memory or CPU spikes

## Viewing Results

```bash
# List recent test results
make load-results

# View detailed logs during test
make logs-app

# Monitor Docker container stats
docker stats
```

## Cleaning Up

```bash
# Remove test result files
make load-clean

# Stop all containers
make down
```

## Tips for Effective Load Testing

1. **Start Small**: Run smoke test first to validate basic functionality
2. **Establish Baseline**: Document initial performance metrics in `METRICS.md`
3. **Incremental Testing**: Progress from smoke → load → stress → spike
4. **Monitor Resources**: Watch CPU, memory, and disk I/O during tests
5. **Multiple Runs**: Run tests 2-3 times to average out variance
6. **Document Results**: Update `METRICS.md` with findings after each test
7. **Optimize & Repeat**: Make improvements and re-run tests to measure impact

## Troubleshooting

### "connection refused" errors
- Ensure services are running: `docker compose ps`
- Check application is healthy: `curl http://localhost:50051/health`

### High error rates during tests
- Check application logs: `make logs-app`
- Verify database connections: `docker compose logs clickhouse`
- Review Redis status: `docker compose logs redis`

### Inconsistent results
- Close other applications consuming resources
- Run multiple iterations and average results
- Ensure no background Docker operations

### k6 container not found
- The k6 service uses a Docker Compose profile
- Tests automatically use `--profile load-test` flag
- No manual profile activation needed

## Next Steps

1. **Run your first test**: `make load-smoke`
2. **Review results**: Check console output for metrics
3. **Document baseline**: Update `METRICS.md` with your results
4. **Run comprehensive tests**: Execute load, stress, and spike tests
5. **Optimize**: Identify bottlenecks and make improvements
6. **Automate**: Integrate into CI/CD for continuous performance monitoring

## Resources

- [METRICS.md](./METRICS.md) - Detailed metrics tracking and analysis
- [k6 Documentation](https://k6.io/docs/)
- [k6 Test Types](https://k6.io/docs/test-types/introduction/)
- [k6 Thresholds](https://k6.io/docs/using-k6/thresholds/)
- [k6 Metrics](https://k6.io/docs/using-k6/metrics/)
