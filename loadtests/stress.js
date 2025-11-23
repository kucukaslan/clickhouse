/**
 * k6 Stress Test
 * Purpose: Find system breaking point and test behavior under extreme load
 * Load: Ramp up from 0 to 1000+ VUs over 10 minutes
 * Use case: Identify capacity limits, bottlenecks, and degradation patterns
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
    { duration: '2m', target: 100 },      // Warm up to 100 VUs
    { duration: '2m', target: 300 },      // Ramp to 300 VUs
    { duration: '2m', target: 500 },      // Ramp to 500 VUs
    { duration: '2m', target: 800 },      // Ramp to 800 VUs
    { duration: '2m', target: 1000 },     // Ramp to 1000 VUs - push to limits
    { duration: '3m', target: 1000 },     // Sustain peak load
    { duration: '2m', target: 500 },      // Ramp down
    { duration: '1m', target: 0 },        // Cool down
  ],
  
  // Reduce metric volume
  discardResponseBodies: true,
  
  thresholds: {
    http_req_failed: ['rate<0.05'],         // Less than 5% errors (relaxed for stress)
    http_req_duration: ['p(95)<2000'],      // 95% under 2s (relaxed)
    http_req_duration: ['p(99)<5000'],      // 99% under 5s (relaxed)
    // No minimum RPS requirement - we're testing limits
  },
  
  summaryTrendStats: ['min', 'avg', 'med', 'max', 'p(90)', 'p(95)', 'p(99)', 'p(99.9)'],
};

// Setup phase
export function setup() {
  console.log('=== Stress Test Configuration ===');
  
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
  const eventPct = parseFloat(__ENV.EVENT_PCT || '0.90');
  const bulkPct = parseFloat(__ENV.BULK_PCT || '0.09');
  const metricsPct = parseFloat(__ENV.METRICS_PCT || '0.01');
  console.log(`Load Distribution: Events=${(eventPct*100).toFixed(1)}%, Bulk=${(bulkPct*100).toFixed(1)}%, Metrics=${(metricsPct*100).toFixed(1)}%`);
  
  console.log(`Max VUs: 1000`);
  console.log(`Total Duration: ~16 minutes`);
  console.log('WARNING: This test will push the system to its limits!');
  console.log('==================================');
  
  return { baseUrl, timestampMode };
}

// Main test function
export default function (data) {
  const baseUrl = data.baseUrl;
  const timestampMode = data.timestampMode;

  // Load distribution - configurable via env vars
  const eventPct = parseFloat(__ENV.EVENT_PCT || '0.90');
  const bulkPct = parseFloat(__ENV.BULK_PCT || '0.09');
  const metricsPct = parseFloat(__ENV.METRICS_PCT || '0.01');

  const action = Math.random();

  if (action < eventPct) {
    // POST single event
    const event = generateEvent(timestampMode);
    const eventsUrl = `${baseUrl}/events`;
    
    const startTime = new Date().getTime();
    const eventsRes = http.post(eventsUrl, JSON.stringify(event), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'PostEvent' },
      timeout: '10s', // Increased timeout for stress conditions
    });
    const endTime = new Date().getTime();
    
    eventLatency.add(endTime - startTime);
    eventsPosted.add(1);
    
    const success = check(eventsRes, {
      'events: status is 200': (r) => r.status === 200,
      'events: not a timeout': (r) => r.status !== 0,
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
      if (Math.random() < 0.01) { // Log 1% of errors to avoid spam
        console.error(`Event POST failed: ${eventsRes.status} - VU: ${__VU}, Iter: ${__ITER}`);
      }
    } else {
      errorRate.add(0);
    }
    
  } else if (action < eventPct + bulkPct) {
    // POST bulk events
    const bulkRequest = generateBulkEvents(200, timestampMode); // 200 events per bulk under stress
    const bulkUrl = `${baseUrl}/events/bulk`;
    
    const startTime = new Date().getTime();
    const bulkRes = http.post(bulkUrl, JSON.stringify(bulkRequest), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'PostEventsBulk' },
      timeout: '15s',
    });
    const endTime = new Date().getTime();
    
    bulkLatency.add(endTime - startTime);
    
    const success = check(bulkRes, {
      'bulk: status is 200': (r) => r.status === 200,
      'bulk: not a timeout': (r) => r.status !== 0,
      'bulk: success is true': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.success === true;
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
      if (Math.random() < 0.01) { // Log 1% of errors
        console.error(`Bulk POST failed: ${bulkRes.status} - VU: ${__VU}, Iter: ${__ITER}`);
      }
    } else {
      errorRate.add(0);
    }
    
  } else {
    // GET metrics
    const rangeType = Math.random() < 0.6 ? 'last_hour' : 'last_day';
    const metricsQuery = generateMetricRequest(rangeType);
    const metricsUrl = `${baseUrl}/metrics?${metricsQuery}`;
    
    const startTime = new Date().getTime();
    const metricsRes = http.get(metricsUrl, {
      tags: { name: 'GetMetrics' },
      timeout: '15s', // Higher timeout for metrics queries under stress
    });
    const endTime = new Date().getTime();
    
    metricsLatency.add(endTime - startTime);
    metricsQueried.add(1);
    
    const success = check(metricsRes, {
      'metrics: status is 200': (r) => r.status === 200,
      'metrics: not a timeout': (r) => r.status !== 0,
      'metrics: success is true': (r) => {
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
      if (Math.random() < 0.01) { // Log 1% of errors
        console.error(`Metrics GET failed: ${metricsRes.status} - VU: ${__VU}, Iter: ${__ITER}`);
      }
    } else {
      errorRate.add(0);
    }
  }
  
  // Minimal think time under stress: 10ms to 100ms
  sleep(0.01 + Math.random() * 0.09);
}

// Teardown phase
export function teardown(data) {
  console.log('=== Stress Test Complete ===');
  console.log('Analyze the results to identify:');
  console.log('- At what VU count did errors start increasing?');
  console.log('- What was the maximum sustainable throughput?');
  console.log('- Did latency degrade gracefully or spike suddenly?');
  console.log('- Were there any timeout or connection errors?');
}
