/**
 * k6 Smoke Test
 * Purpose: Quick validation that the API is working
 * Load: 5 VUs for 1 minute
 * Use case: Run after deployments or code changes to verify basic functionality
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';
import { initPools, generateEvent, generateMetricRequest, generateBulkEvents } from './lib/data-generator.js';

// Initialize data pools at module level (runs once when script loads)
initPools();

// Custom metrics per endpoint
const eventPostDuration = new Trend('event_post_duration', true);
const bulkPostDuration = new Trend('bulk_post_duration', true);
const metricsGetDuration = new Trend('metrics_get_duration', true);

const eventPostCount = new Counter('event_posts_total');
const bulkPostCount = new Counter('bulk_posts_total');
const metricsGetCount = new Counter('metrics_gets_total');

const eventPostErrors = new Rate('event_post_errors');
const bulkPostErrors = new Rate('bulk_post_errors');
const metricsGetErrors = new Rate('metrics_get_errors');

// Test configuration
export const options = {
  vus: __ENV.VUS || 5,
  duration: __ENV.DURATION || '1m',
  
  // Reduce metric volume
  discardResponseBodies: true,
  
  thresholds: {
    http_req_failed: ['rate<0.01'],        // Less than 1% errors
    http_req_duration: ['p(95)<1000'],     // 95% of requests under 1s
    http_req_duration: ['p(99)<2000'],     // 99% of requests under 2s
  },
  
  summaryTrendStats: ['min', 'avg', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

// Setup phase - runs once before the test starts
export function setup() {
  const baseUrl = __ENV.BASE_URL || 'http://localhost:50051';
  console.log(`Target URL: ${baseUrl}`);
  
  // Wait for app to be ready
  console.log('Checking if application is healthy...');
  let healthy = false;
  for (let i = 0; i < 30; i++) {
    try {
      const res = http.get(`${baseUrl}/health`, { timeout: '5s' });
      if (res.status === 200) {
        console.log('✓ Application is healthy and ready');
        healthy = true;
        break;
      }
    } catch (e) {
      // Ignore and retry
    }
    console.log(`Waiting for application to be ready... (attempt ${i + 1}/30)`);
    sleep(1);
  }
  
  if (!healthy) {
    throw new Error('Application did not become healthy in time');
  }
  
  console.log(`VUs: ${options.vus}, Duration: ${options.duration}`);
  console.log(`Timestamp Mode: ${__ENV.TIMESTAMP_MODE || 'recent'}`);
  
  // Display load distribution
  const eventPct = parseFloat(__ENV.EVENT_PCT || '0.70');
  const bulkPct = parseFloat(__ENV.BULK_PCT || '0.20');
  const metricsPct = parseFloat(__ENV.METRICS_PCT || '0.10');
  console.log(`Load Distribution: Events=${(eventPct*100).toFixed(1)}%, Bulk=${(bulkPct*100).toFixed(1)}%, Metrics=${(metricsPct*100).toFixed(1)}%`);
  
  return { baseUrl };
}

// Main test function - runs repeatedly for each VU during the test duration
export default function (data) {
  const baseUrl = data.baseUrl;
  const timestampMode = __ENV.TIMESTAMP_MODE || 'recent';
  
  // Load distribution - configurable via env vars
  const eventPct = parseFloat(__ENV.EVENT_PCT || '0.70');
  const bulkPct = parseFloat(__ENV.BULK_PCT || '0.20');
  const metricsPct = parseFloat(__ENV.METRICS_PCT || '0.10');
  
  const action = Math.random();
  
  if (action < eventPct) {
    // POST single event
    const event = generateEvent(timestampMode);
    const eventsUrl = `${baseUrl}/events`;
    
    const startTime = Date.now();
    const eventsRes = http.post(eventsUrl, JSON.stringify(event), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'PostEvent' },
    });
    const duration = Date.now() - startTime;
    
    eventPostDuration.add(duration);
    eventPostCount.add(1);
    
    const success = check(eventsRes, {
      'events: status is 200': (r) => r.status === 200,
      'events: response has success field': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.hasOwnProperty('success');
        } catch {
          return false;
        }
      },
    });
    
    eventPostErrors.add(!success || eventsRes.status !== 200);
    
    // Log errors for debugging
    if (!success || eventsRes.status !== 200) {
      console.error(`[ERROR] POST /events failed - Status: ${eventsRes.status}, Body: ${eventsRes.body}, Event: ${JSON.stringify(event)}`);
    }
  } else if (action < eventPct + bulkPct) {
    // POST bulk events
    const bulkRequest = generateBulkEvents(50, timestampMode); // 50 events per bulk request
    const bulkUrl = `${baseUrl}/events/bulk`;
    
    const startTime = Date.now();
    const bulkRes = http.post(bulkUrl, JSON.stringify(bulkRequest), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'PostEventsBulk' },
    });
    const duration = Date.now() - startTime;
    
    bulkPostDuration.add(duration);
    bulkPostCount.add(1);
    
    const success = check(bulkRes, {
      'bulk: status is 200': (r) => r.status === 200,
      'bulk: response has success field': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.hasOwnProperty('success');
        } catch {
          return false;
        }
      },
      'bulk: has success_count': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.hasOwnProperty('success_count');
        } catch {
          return false;
        }
      },
    });
    
    bulkPostErrors.add(!success || bulkRes.status !== 200);
    
    // Log errors for debugging
    if (!success || bulkRes.status !== 200) {
      console.error(`[ERROR] POST /events/bulk failed - Status: ${bulkRes.status}, Body: ${bulkRes.body}`);
    }
  } else {
    // GET metrics
    const metricsQuery = generateMetricRequest('last_hour');
    const metricsUrl = `${baseUrl}/metrics?${metricsQuery}`;
    
    const startTime = Date.now();
    const metricsRes = http.get(metricsUrl, {
      tags: { name: 'GetMetrics' },
    });
    const duration = Date.now() - startTime;
    
    metricsGetDuration.add(duration);
    metricsGetCount.add(1);
    
    const success = check(metricsRes, {
      'metrics: status is 200': (r) => r.status === 200,
      'metrics: response has success field': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.hasOwnProperty('success');
        } catch {
          return false;
        }
      },
    });
    
    metricsGetErrors.add(!success || metricsRes.status !== 200);
    
    // Log errors for debugging
    if (!success || metricsRes.status !== 200) {
      console.error(`[ERROR] GET /metrics failed - Status: ${metricsRes.status}, Body: ${metricsRes.body}, Query: ${metricsQuery}`);
    }
  }
  
  // Think time: 100ms to 500ms between requests
  sleep(0.1 + Math.random() * 0.4);
}

// Teardown phase - runs once after the test completes
export function teardown(data) {
  console.log('Smoke test completed!');
}
