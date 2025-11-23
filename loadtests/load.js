/**
 * k6 Load Test
 * Purpose: Test normal operating capacity and sustained performance
 * Load: 200 VUs for 5 minutes (with 30s ramp-up and ramp-down)
 * Use case: Validate system performance under expected production load
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { initPools, generateEvent, generateMetricRequest, generateBulkEvents } from './lib/data-generator.js';

// Initialize data pools at module level (runs once when script loads)
initPools();

// Custom metrics
const eventsPosted = new Counter('events_posted');
const bulkEventsPosted = new Counter('bulk_events_posted');
const metricsQueried = new Counter('metrics_queried');
const eventLatency = new Trend('event_post_duration');
const bulkLatency = new Trend('bulk_post_duration');
const metricsLatency = new Trend('metrics_query_duration');
const errorRate = new Rate('errors');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 50 },     // Ramp-up to 50 VUs
    { duration: '1m', target: 100 },     // Ramp-up to 100 VUs
    { duration: '1m', target: 200 },     // Ramp-up to 200 VUs
    { duration: '2m', target: 200 },     // Stay at 200 VUs
    { duration: '30s', target: 100 },    // Ramp-down to 100 VUs
    { duration: '30s', target: 0 },      // Ramp-down to 0
  ],
  
  // Reduce metric volume
  discardResponseBodies: true,
  
  thresholds: {
    http_req_failed: ['rate<0.01'],         // Less than 1% errors
    http_req_duration: ['p(95)<500'],       // 95% under 500ms
    http_req_duration: ['p(99)<1000'],      // 99% under 1s
    http_reqs: ['rate>100'],                 // At least 100 RPS
    errors: ['rate<0.01'],                   // Less than 1% custom errors
  },
  
  summaryTrendStats: ['min', 'avg', 'med', 'max', 'p(90)', 'p(95)', 'p(99)'],
};

// Setup phase
export function setup() {
  console.log('=== Load Test Configuration ===');
  
  const baseUrl = __ENV.BASE_URL || 'http://localhost:50051';
  const timestampMode = __ENV.TIMESTAMP_MODE || 'recent';
  
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
  
  console.log(`Target URL: ${baseUrl}`);
  console.log(`Timestamp Mode: ${timestampMode}`);
  
  // Display load distribution
  const eventPct = parseFloat(__ENV.EVENT_PCT || '0.95');
  const bulkPct = parseFloat(__ENV.BULK_PCT || '0.04');
  const metricsPct = parseFloat(__ENV.METRICS_PCT || '0.01');
  console.log(`Load Distribution: Events=${(eventPct*100).toFixed(1)}%, Bulk=${(bulkPct*100).toFixed(1)}%, Metrics=${(metricsPct*100).toFixed(1)}%`);
  
  // Calculate total test duration
  let totalDuration = 0;
  for (const stage of options.stages) {
    if (stage.duration) {
      totalDuration += parseDuration(stage.duration);
    }
  }
  console.log(`Test Duration: ${totalDuration}s`);
  console.log(`Max VUs: 200`);
  console.log('==============================');
  
  return { baseUrl, timestampMode };
}

// Helper to parse duration strings
function parseDuration(duration) {
  if (typeof duration !== 'string') return 0;
  const match = duration.match(/(\d+)([smh])/);
  if (!match) return 0;
  const value = parseInt(match[1]);
  const unit = match[2];
  switch (unit) {
    case 's': return value;
    case 'm': return value * 60;
    case 'h': return value * 3600;
    default: return 0;
  }
}

// Main test function
export default function (data) {
  const baseUrl = data.baseUrl;
  const timestampMode = data.timestampMode;
  
  // Load distribution - configurable via env vars
  const eventPct = parseFloat(__ENV.EVENT_PCT || '0.65');
  const bulkPct = parseFloat(__ENV.BULK_PCT || '0.20');
  const metricsPct = parseFloat(__ENV.METRICS_PCT || '0.15');
  
  const action = Math.random();
  
  if (action < eventPct) {
    // POST single event
    const event = generateEvent(timestampMode);
    const eventsUrl = `${baseUrl}/events`;
    
    const startTime = new Date().getTime();
    const eventsRes = http.post(eventsUrl, JSON.stringify(event), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'PostEvent' },
    });
    const endTime = new Date().getTime();
    
    eventLatency.add(endTime - startTime);
    eventsPosted.add(1);
    
    const success = check(eventsRes, {
      'events: status is 200': (r) => r.status === 200,
      'events: success is true': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.success === true;
        } catch {
          return false;
        }
      },
    });
    
    if (!success) {
      errorRate.add(1);
      console.error(`Event POST failed: ${eventsRes.status} - ${eventsRes.body}`);
    } else {
      errorRate.add(0);
    }
    
  } else if (action < eventPct + bulkPct) {
    // POST bulk events
    const bulkRequest = generateBulkEvents(100, timestampMode); // 100 events per bulk request
    const bulkUrl = `${baseUrl}/events/bulk`;
    
    const startTime = new Date().getTime();
    const bulkRes = http.post(bulkUrl, JSON.stringify(bulkRequest), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'PostEventsBulk' },
    });
    const endTime = new Date().getTime();
    
    bulkLatency.add(endTime - startTime);
    
    const success = check(bulkRes, {
      'bulk: status is 200': (r) => r.status === 200,
      'bulk: success is true': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.success === true;
        } catch {
          return false;
        }
      },
      'bulk: has counts': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.hasOwnProperty('success_count') && body.hasOwnProperty('total_count');
        } catch {
          return false;
        }
      },
    });
    
    if (success) {
      try {
        const body = JSON.parse(bulkRes.body);
        bulkEventsPosted.add(body.success_count || 0);
      } catch {
        bulkEventsPosted.add(0);
      }
    }
    
    if (!success) {
      errorRate.add(1);
      console.error(`Bulk POST failed: ${bulkRes.status} - ${bulkRes.body}`);
    } else {
      errorRate.add(0);
    }
    
  } else {
    // GET metrics
    const rangeType = Math.random() < 0.5 ? 'last_hour' : 'last_day';
    const metricsQuery = generateMetricRequest(rangeType);
    const metricsUrl = `${baseUrl}/metrics?${metricsQuery}`;
    
    const startTime = new Date().getTime();
    const metricsRes = http.get(metricsUrl, {
      tags: { name: 'GetMetrics' },
    });
    const endTime = new Date().getTime();
    
    metricsLatency.add(endTime - startTime);
    metricsQueried.add(1);
    
    const success = check(metricsRes, {
      'metrics: status is 200': (r) => r.status === 200,
      'metrics: success is true': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.success === true;
        } catch {
          return false;
        }
      },
      'metrics: has metrics array': (r) => {
        try {
          const body = JSON.parse(r.body);
          return Array.isArray(body.metrics);
        } catch {
          return false;
        }
      },
    });
    
    if (!success) {
      errorRate.add(1);
      console.error(`Metrics GET failed: ${metricsRes.status} - ${metricsRes.body}`);
    } else {
      errorRate.add(0);
    }
  }
  
  // Variable think time: 50ms to 300ms
  sleep(0.05 + Math.random() * 0.25);
}

// Teardown phase
export function teardown(data) {
  console.log('=== Load Test Summary ===');
  console.log('Test completed successfully!');
  console.log('Check the summary above for detailed metrics.');
}
