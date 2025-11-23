.PHONY: help swagger build up down rebuild logs clean test

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

swagger: ## Generate Swagger documentation locally
	@echo "Generating Swagger documentation..."
	@cd src && swag init -g main.go --output ./docs
	@echo "Swagger docs generated in src/docs/"

build: ## Build the Go application
	@echo "Building Go application..."
	@cd src && go build -o ../tmp/main .
	@echo "Build complete: tmp/main"

up: ## Start all Docker containers
	@echo "Starting Docker containers..."
	@docker compose up -d
	@echo "Containers started. Swagger UI: http://localhost:50051/swagger/index.html"

down: ## Stop all Docker containers
	@echo "Stopping Docker containers..."
	@docker compose down
	@echo "Containers stopped."

rebuild: down ## Rebuild and restart containers (no cache)
	@echo "Rebuilding Docker containers..."
	@docker compose build --no-cache
	@docker compose up -d
	@echo "Containers rebuilt and started. Swagger UI: http://localhost:50051/swagger/index.html"

logs: ## View logs from all containers
	@docker compose logs -f

logs-app: ## View logs from the application container only
	@docker compose logs -f clickhouse-demo

clean: ## Clean up generated files and Docker resources
	@echo "Cleaning up..."
	@rm -rf src/docs tmp/main
	@docker compose down -v
	@echo "Cleanup complete."

test: ## Run tests
	@echo "Running tests..."
	@cd src && go test ./... -v

install-swagger: ## Install swag CLI tool
	@echo "Installing swag CLI tool..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "swag installed."

# Load Testing Targets
load-smoke: ## Run smoke test (5 VUs, 1 minute)
	@echo "Running smoke test..."
	@docker compose --profile load-test run --rm k6 run /scripts/smoke.js
	@echo "Smoke test complete!"

load-test: ## Run load test (200 VUs, 5 minutes with ramp-up)
	@echo "Running load test..."
	@docker compose --profile load-test run --rm k6 run /scripts/load.js
	@echo "Load test complete!"

load-stress: ## Run stress test (up to 1000 VUs, 16 minutes)
	@echo "Running stress test (this will take ~16 minutes)..."
	@docker compose --profile load-test run --rm k6 run /scripts/stress.js
	@echo "Stress test complete!"

load-spike: ## Run spike test (traffic bursts, 4 minutes)
	@echo "Running spike test..."
	@docker compose --profile load-test run --rm k6 run /scripts/spike.js
	@echo "Spike test complete!"

load-custom: ## Run custom load test (use: make load-custom VUS=100 DURATION=5m MODE=recent SCRIPT=smoke EVENT_PCT=0.99 BULK_PCT=0.0 METRICS_PCT=0.01)
	@echo "Running custom load test..."
	@VUS=$(VUS) DURATION=$(DURATION) TIMESTAMP_MODE=$(MODE) EVENT_PCT=$(EVENT_PCT) BULK_PCT=$(BULK_PCT) METRICS_PCT=$(METRICS_PCT) docker compose --profile load-test run --rm k6 run /scripts/$(SCRIPT).js
	@echo "Custom test complete!"

# Monitoring Targets
monitoring-up: ## Start Grafana and InfluxDB for k6 metrics visualization
	@echo "Starting monitoring stack (Grafana + InfluxDB)..."
	@docker compose --profile monitoring up -d
	@echo "✓ Grafana: http://localhost:3000"
	@echo "✓ InfluxDB: http://localhost:8086"
	@echo "Dashboard will auto-load when you run tests with monitoring"

monitoring-down: ## Stop Grafana and InfluxDB
	@echo "Stopping monitoring stack..."
	@docker compose --profile monitoring down
	@echo "Monitoring stopped."

load-smoke-monitor: ## Run smoke test with Grafana monitoring (optional: EVENT_PCT=0.99 BULK_PCT=0.0 METRICS_PCT=0.01)
	@echo "Starting monitoring stack..."
	@docker compose --profile monitoring up -d
	@sleep 3
	@echo "Running smoke test with metrics to InfluxDB..."
	@docker compose --profile load-test run --rm \
		-e K6_OUT=influxdb=http://influxdb:8086/k6 \
		-e K6_INFLUXDB_PUSH_INTERVAL=5s \
		-e K6_INFLUXDB_CONCURRENT_WRITES=4 \
		-e EVENT_PCT=$(EVENT_PCT) -e BULK_PCT=$(BULK_PCT) -e METRICS_PCT=$(METRICS_PCT) \
		k6 run /scripts/smoke.js
	@echo "✓ Test complete! View results at http://localhost:3000"

load-test-monitor: ## Run load test with Grafana monitoring (optional: EVENT_PCT=0.99 BULK_PCT=0.0 METRICS_PCT=0.01)
	@echo "Starting monitoring stack..."
	@docker compose --profile monitoring up -d
	@sleep 3
	@echo "Running load test with metrics to InfluxDB..."
	@docker compose --profile load-test run --rm \
		-e K6_OUT=influxdb=http://influxdb:8086/k6 \
		-e K6_INFLUXDB_PUSH_INTERVAL=5s \
		-e K6_INFLUXDB_CONCURRENT_WRITES=4 \
		-e EVENT_PCT=$(EVENT_PCT) -e BULK_PCT=$(BULK_PCT) -e METRICS_PCT=$(METRICS_PCT) \
		k6 run /scripts/load.js
	@echo "✓ Test complete! View results at http://localhost:3000"

load-stress-monitor: ## Run stress test with Grafana monitoring (optional: EVENT_PCT=0.99 BULK_PCT=0.0 METRICS_PCT=0.01)
	@echo "Starting monitoring stack..."
	@docker compose --profile monitoring up -d
	@sleep 3
	@echo "Running stress test with metrics to InfluxDB..."
	@docker compose --profile load-test run --rm \
		-e K6_OUT=influxdb=http://influxdb:8086/k6 \
		-e K6_INFLUXDB_PUSH_INTERVAL=15s \
		-e K6_INFLUXDB_CONCURRENT_WRITES=10 \
		-e K6_INFLUXDB_TAGS_AS_FIELDS=vu:int,iter:int \
		-e EVENT_PCT=$(EVENT_PCT) -e BULK_PCT=$(BULK_PCT) -e METRICS_PCT=$(METRICS_PCT) \
		k6 run /scripts/stress.js
	@echo "✓ Test complete! View results at http://localhost:3000"

load-spike-monitor: ## Run spike test with Grafana monitoring (optional: EVENT_PCT=0.99 BULK_PCT=0.0 METRICS_PCT=0.01)
	@echo "Starting monitoring stack..."
	@docker compose --profile monitoring up -d
	@sleep 3
	@echo "Running spike test with metrics to InfluxDB..."
	@docker compose --profile load-test run --rm \
		-e K6_OUT=influxdb=http://influxdb:8086/k6 \
		-e K6_INFLUXDB_PUSH_INTERVAL=10s \
		-e K6_INFLUXDB_CONCURRENT_WRITES=10 \
		-e K6_INFLUXDB_TAGS_AS_FIELDS=vu:int,iter:int \
		-e EVENT_PCT=$(EVENT_PCT) -e BULK_PCT=$(BULK_PCT) -e METRICS_PCT=$(METRICS_PCT) \
		k6 run /scripts/spike.js
	@echo "✓ Test complete! View results at http://localhost:3000"

load-results: ## Display latest load test results
	@echo "Latest load test results:"
	@if [ -d "loadtests/results" ] && [ "$$(ls -A loadtests/results)" ]; then \
		ls -lt loadtests/results/ | head -10; \
	else \
		echo "No results found. Run a load test first."; \
	fi

load-clean: ## Clean load test results
	@echo "Cleaning load test results..."
	@rm -rf loadtests/results/*
	@echo "Results cleaned."

