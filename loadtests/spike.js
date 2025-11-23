/**
 * k6 Spike Test
 * Purpose: Test system behavior under sudden traffic bursts
 * Load: Rapid spikes from 10 to 500 VUs and back
 * Use case: Validate auto-scaling, circuit breakers, and recovery behavior
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
const spikePhase = new Counter('spike_phase');

// Test configuration
export const options = {
  stages: [
    { duration: '10s', target: 10 },      // Baseline: 10 VUs
    { duration: '10s', target: 500 },     // SPIKE 1: Sudden jump to 500 VUs
    { duration: '30s', target: 500 },     // Sustain spike
    { duration: '10s', target: 10 },      // Drop back to baseline
    { duration: '30s', target: 10 },      // Recovery period
    { duration: '10s', target: 700 },     // SPIKE 2: Even larger spike to 700 VUs
    { duration: '30s', target: 700 },     // Sustain spike
    { duration: '10s', target: 10 },      // Drop back to baseline
    { duration: '30s', target: 10 },      // Recovery period
    { duration: '10s', target: 300 },     // SPIKE 3: Medium spike to 300 VUs
    { duration: '20s', target: 300 },     // Sustain spike
    { duration: '10s', target: 10 },      // Drop back to baseline
    { duration: '20s', target: 10 },      // Final recovery
  ],
  
  // Reduce metric volume
  discardResponseBodies: true,
  
  thresholds: {
    http_req_failed: ['rate<0.03'],         // Less than 3% errors (some expected during spikes)
    http_req_duration: ['p(95)<1500'],      // 95% under 1.5s
    http_req_duration: ['p(99)<3000'],      // 99% under 3s
  },
  
  summaryTrendStats: ['min', 'avg', 'med', 'max', 'p(90)', 'p(95)', 'p(99)', 'p(99.9)'],
};

// Setup phase
export function setup() {
  console.log('=== Spike Test Configuration ===');
  
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
  
  console.log('Spike Pattern: 10 → 500 → 10 → 700 → 10 → 300 → 10');
  console.log('Total Duration: ~4 minutes');
  console.log('================================');
  
  return { baseUrl, timestampMode };
}

// Main test function
export default function (data) {
  const baseUrl = data.baseUrl;
  const timestampMode = data.timestampMode;
  
  // Track which phase we're in based on VUs
  const currentVUs = __VU;
  if (currentVUs > 400) {
    spikePhase.add(1); // High spike phase
  }
  
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
      timeout: '10s',
    });
    const endTime = new Date().getTime();
    
    const duration = endTime - startTime;
    eventLatency.add(duration);
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
      // Log errors during spike phases - reduced sampling to 1%
      if (currentVUs > 400 && Math.random() < 0.01) {
        console.error(`[SPIKE] Event POST failed at ${currentVUs} VUs: ${eventsRes.status}`);
      }
    } else {
      errorRate.add(0);
    }
    
  } else if (action < eventPct + bulkPct) {
    // POST bulk events
    const bulkRequest = generateBulkEvents(150, timestampMode); // 150 events per bulk during spikes
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
      if (currentVUs > 400 && Math.random() < 0.05) {
        console.error(`[SPIKE] Bulk POST failed at ${currentVUs} VUs: ${bulkRes.status}`);
      }
    } else {
      errorRate.add(0);
    }
    
  } else {
    // GET metrics
    const rangeType = Math.random() < 0.7 ? 'last_hour' : 'last_day';
    const metricsQuery = generateMetricRequest(rangeType);
    const metricsUrl = `${baseUrl}/metrics?${metricsQuery}`;
    
    const startTime = new Date().getTime();
    const metricsRes = http.get(metricsUrl, {
      tags: { name: 'GetMetrics' },
      timeout: '15s',
    });
    const endTime = new Date().getTime();
    
    const duration = endTime - startTime;
    metricsLatency.add(duration);
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
      if (currentVUs > 400 && Math.random() < 0.05) {
        console.error(`[SPIKE] Metrics GET failed at ${currentVUs} VUs: ${metricsRes.status}`);
      }
    } else {
      errorRate.add(0);
    }
  }
  
  // Very short think time to maximize spike impact
  sleep(0.01 + Math.random() * 0.05);
}

// Teardown phase
export function teardown(data) {
  console.log('=== Spike Test Complete ===');
  console.log('Analyze the results to identify:');
  console.log('- Did the system handle sudden traffic spikes?');
  console.log('- How quickly did it recover after spikes?');
  console.log('- Were there cascading failures or circuit breaker activations?');
  console.log('- Did error rates spike during transitions?');
}
