# Load Testing Metrics and Results

This document tracks performance metrics from k6 load tests for the ClickHouse Event Tracking API.

## Test Environment

- **Application**: ClickHouse Event Tracking API
- **Go Version**: 1.25.4
- **ClickHouse Version**: 25.10.2.65-alpine
- **Redis Version**: 8-alpine
- **Deployment**: Docker Compose (local)

## Key Performance Indicators (KPIs)

### Service Level Objectives (SLOs)

| Metric | Target | Critical Threshold |
|--------|--------|--------------------|
| **Availability** | 99.9% | 99% |
| **P95 Latency** | < 500ms | < 1000ms |
| **P99 Latency** | < 1000ms | < 2000ms |
| **Error Rate** | < 1% | < 5% |
| **Throughput** | > 1000 RPS | > 500 RPS |

### Endpoints

1. **POST /events** - Event ingestion (write-heavy)
2. **POST /metrics** - Metrics queries (read-heavy)

---

## Test Scenarios

### 1. Smoke Test

**Purpose**: Quick validation after deployments or code changes

**Configuration**:
- VUs: 5
- Duration: 1 minute
- Workload: 90% writes, 10% reads
- Think time: 100-500ms

**Command**: `make load-smoke`

**Expected Baseline** (to be filled after first run):
```
Metrics                          Target    Baseline    Last Run    Status
─────────────────────────────────────────────────────────────────────────
http_req_duration (p95)          < 1000ms   TBD         TBD         ⏳
http_req_duration (p99)          < 2000ms   TBD         TBD         ⏳
http_req_failed                  < 1%       TBD         TBD         ⏳
http_reqs (total)                -          TBD         TBD         ⏳
http_reqs (rate)                 -          TBD         TBD         ⏳
```

**Results History**:
| Date | VUs | Duration | P95 | P99 | Error Rate | Total Requests | RPS | Notes |
|------|-----|----------|-----|-----|------------|----------------|-----|-------|
| -    | 5   | 1m       | -   | -   | -          | -              | -   | Baseline pending |

---

### 2. Load Test

**Purpose**: Validate normal production capacity and sustained performance

**Configuration**:
- Max VUs: 200
- Duration: 5 minutes (with 30s ramp-up/down)
- Workload: 85% writes, 15% reads
- Think time: 50-300ms
- Stages:
  - 30s → 50 VUs
  - 1m → 100 VUs
  - 1m → 200 VUs
  - 2m @ 200 VUs (sustained)
  - 30s → 100 VUs
  - 30s → 0 VUs

**Command**: `make load-test`

**Expected Baseline** (to be filled after first run):
```
Metrics                          Target    Baseline    Last Run    Status
─────────────────────────────────────────────────────────────────────────
http_req_duration (p95)          < 500ms    TBD         TBD         ⏳
http_req_duration (p99)          < 1000ms   TBD         TBD         ⏳
http_req_failed                  < 1%       TBD         TBD         ⏳
http_reqs (rate)                 > 100/s    TBD         TBD         ⏳
events_posted (total)            -          TBD         TBD         ⏳
metrics_queried (total)          -          TBD         TBD         ⏳
```

**Results History**:
| Date | Max VUs | P95 (ms) | P99 (ms) | Error Rate | Total Requests | Avg RPS | Events Posted | Metrics Queried | Notes |
|------|---------|----------|----------|------------|----------------|---------|---------------|-----------------|-------|
| -    | 200     | -        | -        | -          | -              | -       | -             | -               | Baseline pending |

---

### 3. Stress Test

**Purpose**: Find system breaking point and identify capacity limits

**Configuration**:
- Max VUs: 1000
- Duration: 16 minutes
- Workload: 80% writes, 20% reads
- Think time: 10-100ms (minimal)
- Stages:
  - 2m → 100 VUs
  - 2m → 300 VUs
  - 2m → 500 VUs
  - 2m → 800 VUs
  - 2m → 1000 VUs
  - 3m @ 1000 VUs (sustained peak)
  - 2m → 500 VUs (ramp down)
  - 1m → 0 VUs

**Command**: `make load-stress`

**Expected Thresholds** (relaxed for stress conditions):
```
Metrics                          Threshold  Baseline    Last Run    Status
─────────────────────────────────────────────────────────────────────────
http_req_duration (p95)          < 2000ms   TBD         TBD         ⏳
http_req_duration (p99)          < 5000ms   TBD         TBD         ⏳
http_req_failed                  < 5%       TBD         TBD         ⏳
Breaking point (VUs)             -          TBD         TBD         ⏳
```

**Key Questions**:
- At what VU count do errors start increasing significantly?
- What is the maximum sustainable throughput (RPS)?
- Does latency degrade gracefully or spike suddenly?
- Are there timeout or connection errors?

**Results History**:
| Date | Max VUs | Breaking Point | P95 (ms) | P99 (ms) | P99.9 (ms) | Error Rate | Max RPS | Notes |
|------|---------|----------------|----------|----------|------------|------------|---------|-------|
| -    | 1000    | -              | -        | -        | -          | -          | -       | Baseline pending |

---

### 4. Spike Test

**Purpose**: Test system behavior under sudden traffic bursts

**Configuration**:
- Spike Pattern: 10 → 500 → 10 → 700 → 10 → 300 → 10 VUs
- Duration: 4 minutes
- Workload: 80% writes, 20% reads
- Think time: 10-50ms (very short)
- Stages:
  - 10s @ 10 VUs (baseline)
  - 10s → 500 VUs (SPIKE 1)
  - 30s @ 500 VUs
  - 10s → 10 VUs (drop)
  - 30s @ 10 VUs (recovery)
  - 10s → 700 VUs (SPIKE 2 - largest)
  - 30s @ 700 VUs
  - 10s → 10 VUs (drop)
  - 30s @ 10 VUs (recovery)
  - 10s → 300 VUs (SPIKE 3)
  - 20s @ 300 VUs
  - 10s → 10 VUs (drop)
  - 20s @ 10 VUs (final recovery)

**Command**: `make load-spike`

**Expected Thresholds**:
```
Metrics                          Threshold  Baseline    Last Run    Status
─────────────────────────────────────────────────────────────────────────
http_req_duration (p95)          < 1500ms   TBD         TBD         ⏳
http_req_duration (p99)          < 3000ms   TBD         TBD         ⏳
http_req_failed                  < 3%       TBD         TBD         ⏳
```

**Key Questions**:
- Does the system handle sudden traffic spikes without cascading failures?
- How quickly does it recover after spikes end?
- Are there circuit breaker activations or rate limiting?
- Do error rates spike during rapid transitions?

**Results History**:
| Date | Max Spike | P95 (ms) | P99 (ms) | Error Rate @ Baseline | Error Rate @ Spike | Recovery Time | Notes |
|------|-----------|----------|----------|-----------------------|--------------------|---------------|-------|
| -    | 700 VUs   | -        | -        | -                     | -                  | -             | Baseline pending |

---

## Custom Test Runs

Use `make load-custom` for custom configurations:

```bash
# Examples:
make load-custom VUS=100 DURATION=5m MODE=recent SCRIPT=smoke
make load-custom VUS=500 DURATION=10m MODE=mixed SCRIPT=load
make load-custom VUS=1500 DURATION=3m MODE=historical SCRIPT=stress
```

**Parameters**:
- `VUS`: Number of virtual users (default: 10)
- `DURATION`: Test duration (e.g., 1m, 5m, 10m)
- `MODE`: Timestamp mode - `recent`, `historical`, or `mixed` (default: recent)
- `SCRIPT`: Test script - `smoke`, `load`, `stress`, or `spike`

---

## Timestamp Modes

The `TIMESTAMP_MODE` environment variable controls event timestamp distribution:

### Recent Mode (default)
- **Range**: Last hour (now - 3600s to now)
- **Use case**: Test current data ingestion and real-time queries
- **Best for**: Smoke tests, load tests

### Historical Mode
- **Range**: 1-30 days ago
- **Use case**: Test queries on older data, backfill scenarios
- **Best for**: Metrics query performance testing

### Mixed Mode
- **Distribution**: 70% last 24 hours, 30% last 30 days
- **Use case**: Realistic production workload simulation
- **Best for**: Comprehensive load and stress tests

---

## Data Generation

### Shared Pools (initialized at test startup)

| Pool | Size | Description |
|------|------|-------------|
| **User IDs** | 1,000 | Format: `user_000001` to `user_001000` |
| **Campaign IDs** | 50 | Format: `summer_sale_2025_001`, etc. |
| **Event Types** | 25 | `purchase`, `page_view`, `button_click`, etc. |
| **Channels** | 10 | `web`, `mobile`, `ios`, `android`, etc. |
| **Tags** | 18 | `premium`, `free`, `trial`, `mobile`, etc. |

### Event Metadata

Metadata fields are generated based on event type:
- **Purchase/Checkout**: `amount`, `currency`, `items_count`, `payment_method`
- **Cart Events**: `product_id`, `quantity`, `price`
- **Page Views**: `page_url`, `referrer`, `duration_seconds`
- **Search**: `query`, `results_count`
- **Video**: `video_id`, `position_seconds`, `duration_seconds`
- **Common**: `session_id`, `user_agent`, optional `ab_test_variant`, `feature_flag`

---

## Performance Optimization Checklist

### Before Running Tests
- [ ] Ensure Docker containers are running: `make up`
- [ ] Check system resources (CPU, memory, disk space)
- [ ] Close unnecessary applications
- [ ] Verify ClickHouse and Redis are healthy: `docker compose ps`

### During Tests
- [ ] Monitor container logs: `make logs`
- [ ] Watch Docker stats: `docker stats`
- [ ] Check for errors in application logs

### After Tests
- [ ] Review k6 summary output
- [ ] Check `loadtests/results/` for detailed reports
- [ ] Update this document with results
- [ ] Document any anomalies or issues
- [ ] Compare with baseline metrics

---

## Analyzing Results

### Key Metrics to Watch

1. **HTTP Request Duration**
   - `http_req_duration`: Full request-response time
   - Focus on p(95) and p(99) percentiles
   - Look for latency spikes during ramp-ups

2. **Error Rates**
   - `http_req_failed`: Percentage of failed requests
   - Target: < 1% for normal load, < 5% for stress
   - Investigate 4xx vs 5xx error patterns

3. **Throughput**
   - `http_reqs`: Total requests and rate (RPS)
   - Compare to capacity requirements
   - Identify throughput plateaus

4. **Custom Metrics**
   - `events_posted`: Total events ingested
   - `metrics_queried`: Total metric queries
   - `event_post_duration`: Specific latency for event writes
   - `metrics_query_duration`: Specific latency for metric reads

### Red Flags 🚩

- Error rate > 5%
- P99 latency > 5 seconds
- Request timeouts or connection errors
- Memory leaks (increasing memory over time)
- Database connection pool exhaustion
- Sudden latency spikes without load increase

### Green Flags ✅

- Error rate < 1%
- P95 latency < 500ms
- Stable memory and CPU usage
- Linear throughput scaling with VUs
- Fast recovery after spike tests
- Consistent performance over sustained load

---

## Continuous Monitoring

### Recommended Test Cadence

| Test Type | Frequency | Trigger |
|-----------|-----------|---------|
| **Smoke** | After every deployment | CI/CD pipeline |
| **Load** | Weekly | Scheduled job |
| **Stress** | Monthly | Before major releases |
| **Spike** | Bi-weekly | Capacity planning |

### Baseline Update Strategy

Update baseline metrics when:
1. New hardware/infrastructure is deployed
2. Major code optimizations are implemented
3. Database schema changes affect performance
4. Significant dependency upgrades occur

---

## Troubleshooting

### Common Issues

**High Error Rates**
- Check application logs: `make logs-app`
- Verify ClickHouse connection pool size
- Check Redis connection limits
- Increase timeout values in k6 scripts

**High Latency**
- Profile database queries
- Check for table locks in ClickHouse
- Monitor Redis memory usage
- Review network latency between containers

**Connection Timeouts**
- Increase Docker container resources
- Adjust `IdleTimeout` in Fiber config
- Scale ClickHouse replicas
- Add connection pooling

**Inconsistent Results**
- Run tests multiple times and average results
- Ensure no other processes are consuming resources
- Check for background Docker operations
- Verify consistent test data generation

---

## Results Export

k6 automatically exports results to `loadtests/results/` directory:

```bash
# View latest results
make load-results

# Clean old results
make load-clean
```

Result files include:
- Console summary output
- JSON detailed metrics (if configured)
- CSV time-series data (if configured)

---

## Next Steps

1. **Run baseline tests**: Execute all test scenarios and fill in the baseline metrics
2. **Set up monitoring**: Consider adding Prometheus + Grafana for real-time metrics
3. **Automate testing**: Integrate k6 tests into CI/CD pipeline
4. **Scale testing**: Test against production-like infrastructure
5. **Profile bottlenecks**: Use pprof and ClickHouse query logs for optimization

---

## Resources

- [k6 Documentation](https://k6.io/docs/)
- [ClickHouse Performance Tips](https://clickhouse.com/docs/en/operations/tips)
- [Fiber Performance Tuning](https://docs.gofiber.io/)
- [Load Testing Best Practices](https://k6.io/docs/test-types/introduction/)

---

**Last Updated**: Never (awaiting first test run)  
**Baseline Status**: 🔴 Not established - Run tests to create baseline metrics
