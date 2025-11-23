/**
 * Data Generator for k6 Load Tests
 * Generates realistic event data using faker patterns and pre-initialized pools
 */

// Shared data pools - initialized once at script startup
let userPool = [];
let campaignPool = [];
let eventTypePool = [];
let channelPool = [];
let tagPool = [];

const USER_POOL_SIZE = 1000;
const CAMPAIGN_POOL_SIZE = 50;

/**
 * Initialize all shared data pools
 * Call this once in the setup() phase of your k6 script
 */
export function initPools() {
  // Generate user IDs
  for (let i = 0; i < USER_POOL_SIZE; i++) {
    userPool.push(`user_${String(i).padStart(6, '0')}`);
  }

  // Generate campaign IDs
  const campaignTypes = ['summer_sale', 'winter_promo', 'black_friday', 'cyber_monday', 'flash_deal', 'loyalty_reward', 'new_year', 'back_to_school', 'spring_clearance', 'holiday_special'];
  for (let i = 0; i < CAMPAIGN_POOL_SIZE; i++) {
    const type = campaignTypes[i % campaignTypes.length];
    campaignPool.push(`${type}_2025_${String(i).padStart(3, '0')}`);
  }

  // Event types
  eventTypePool = [
    'page_view', 'button_click', 'form_submit', 'purchase', 'add_to_cart',
    'remove_from_cart', 'checkout_start', 'checkout_complete', 'search',
    'filter_apply', 'product_view', 'category_view', 'login', 'logout',
    'signup', 'password_reset', 'profile_update', 'wishlist_add',
    'review_submit', 'share_product', 'video_play', 'video_pause',
    'download_start', 'download_complete', 'error_occurred'
  ];

  // Channels
  channelPool = ['web', 'mobile', 'ios', 'android', 'tablet', 'desktop', 'api', 'email', 'sms', 'push'];

  // Tags
  tagPool = [
    'premium', 'free', 'trial', 'paid', 'mobile', 'desktop', 'new_user',
    'returning_user', 'high_value', 'low_value', 'promotion', 'organic',
    'referral', 'social', 'direct', 'search', 'email_campaign', 'retargeting'
  ];

  console.log(`Initialized pools: ${USER_POOL_SIZE} users, ${CAMPAIGN_POOL_SIZE} campaigns, ${eventTypePool.length} event types, ${channelPool.length} channels, ${tagPool.length} tags`);
}

/**
 * Get a random element from an array
 */
function randomElement(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

/**
 * Get random elements from an array (0 to maxCount)
 */
function randomElements(arr, maxCount = 3) {
  const count = Math.floor(Math.random() * (maxCount + 1));
  const shuffled = [...arr].sort(() => 0.5 - Math.random());
  return shuffled.slice(0, count);
}

/**
 * Generate a random integer between min and max (inclusive)
 */
function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

/**
 * Generate a random float between min and max
 */
function randomFloat(min, max, decimals = 2) {
  const value = Math.random() * (max - min) + min;
  return parseFloat(value.toFixed(decimals));
}

/**
 * Generate timestamp based on mode
 * @param {string} mode - 'recent', 'historical', or 'mixed'
 * @returns {number} Unix timestamp in seconds
 */
function generateTimestamp(mode) {
  const now = Math.floor(Date.now() / 1000);
  
  switch (mode) {
    case 'recent':
      // Last hour: now - 3600 to now - 5 seconds (avoid future timestamps due to clock skew)
      return now - randomInt(5, 3600);
    
    case 'historical':
      // Last 30 days: now - 30 days to now - 1 day
      const thirtyDaysAgo = now - (30 * 24 * 3600);
      const oneDayAgo = now - (24 * 3600);
      return randomInt(thirtyDaysAgo, oneDayAgo);
    
    case 'mixed':
      // 70% recent (last 24 hours), 30% historical (last 30 days)
      if (Math.random() < 0.7) {
        return now - randomInt(5, 24 * 3600);
      } else {
        return now - randomInt(24 * 3600, 30 * 24 * 3600);
      }
    
    default:
      return now - 10; // Default: 10 seconds ago
  }
}

/**
 * Generate realistic metadata based on event type
 */
function generateMetadata(eventName) {
  const metadata = {};

  // Common fields for all events
  metadata.session_id = `session_${randomInt(100000, 999999)}`;
  metadata.user_agent = randomElement([
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36',
    'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15',
    'Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36'
  ]);

  // Event-specific fields (check eventName exists and is a string)
  if (eventName && typeof eventName === 'string') {
    if (eventName.includes('purchase') || eventName.includes('checkout')) {
      metadata.amount = randomFloat(10, 500, 2);
      metadata.currency = randomElement(['USD', 'EUR', 'GBP']);
      metadata.items_count = randomInt(1, 5);
      metadata.payment_method = randomElement(['credit_card', 'paypal', 'apple_pay', 'google_pay']);
    } else if (eventName.includes('cart')) {
      metadata.product_id = `prod_${randomInt(1000, 9999)}`;
      metadata.quantity = randomInt(1, 3);
      metadata.price = randomFloat(10, 200, 2);
    } else if (eventName.includes('view')) {
      metadata.page_url = `https://example.com/${randomElement(['products', 'categories', 'about', 'blog'])}/${randomInt(1, 100)}`;
      metadata.referrer = randomElement(['google.com', 'facebook.com', 'direct', 'twitter.com', '']);
      metadata.duration_seconds = randomInt(5, 300);
    } else if (eventName.includes('search')) {
      metadata.query = randomElement(['laptop', 'phone', 'headphones', 'camera', 'shoes', 'watch']);
      metadata.results_count = randomInt(0, 50);
    } else if (eventName.includes('video')) {
      metadata.video_id = `video_${randomInt(1000, 9999)}`;
      metadata.position_seconds = randomInt(0, 600);
      metadata.duration_seconds = randomInt(60, 3600);
    }
  }

  // Random additional context
  if (Math.random() > 0.5) {
    metadata.ab_test_variant = randomElement(['A', 'B', 'control']);
  }
  if (Math.random() > 0.7) {
    metadata.feature_flag = randomElement(['new_ui', 'beta_feature', 'experimental']);
  }

  return metadata;
}

/**
 * Generate a complete EventRequest object
 * @param {string} timestampMode - 'recent', 'historical', or 'mixed'
 * @returns {object} EventRequest matching the domain schema
 */
export function generateEvent(timestampMode = 'recent') {
  const eventName = randomElement(eventTypePool);
  
  return {
    event_name: eventName,
    channel: randomElement(channelPool),
    campaign_id: randomElement(campaignPool),
    user_id: randomElement(userPool),
    timestamp: generateTimestamp(timestampMode),
    tags: randomElements(tagPool, 3),
    metadata: generateMetadata(eventName)
  };
}

/**
 * Generate a time range for metrics queries
 * Returns from/to timestamps matching recent event insertions
 * @param {string} rangeType - 'last_hour', 'last_day', 'last_week', 'custom'
 * @returns {object} { from: number, to: number }
 */
export function generateTimeRange(rangeType = 'last_hour') {
  const now = Math.floor(Date.now() / 1000);
  
  switch (rangeType) {
    case 'last_hour':
      return {
        from: now - 3600,
        to: now
      };
    
    case 'last_day':
      return {
        from: now - (24 * 3600),
        to: now
      };
    
    case 'last_week':
      return {
        from: now - (7 * 24 * 3600),
        to: now
      };
    
    case 'custom':
      // Random range within last 7 days
      const rangeStart = now - randomInt(3600, 7 * 24 * 3600);
      const rangeEnd = rangeStart + randomInt(3600, 24 * 3600);
      return {
        from: rangeStart,
        to: rangeEnd
      };
    
    default:
      return {
        from: now - 3600,
        to: now
      };
  }
}

/**
 * Generate query parameters for GET /metrics endpoint
 * @param {string} rangeType - 'last_hour', 'last_day', 'last_week', 'custom'
 * @returns {string} URL query string for metrics endpoint
 */
export function generateMetricRequest(rangeType = 'last_hour') {
  const timeRange = generateTimeRange(rangeType);
  const groupByOptions = ['hour', 'day', 'week', 'month', 'year', 'channel', 'campaign_id', 'user_id', 'event_name'];
  
  // 70% of requests filter by event name, 30% query all events
  const eventName = Math.random() < 0.7 ? randomElement(eventTypePool) : null;
  
  // Build query string manually (URLSearchParams not available in k6)
  let params = `from=${timeRange.from}&to=${timeRange.to}&group_by=${randomElement(groupByOptions)}`;

  if (eventName) {
    params += `&event_name=${encodeURIComponent(eventName)}`;
  }

  return params;
}

/**
 * Generate bulk events request
 * @param {number} count - Number of events to generate (1-1000)
 * @param {string} timestampMode - 'recent', 'historical', or 'mixed'
 * @returns {object} BulkEventRequest with array of events
 */
export function generateBulkEvents(count = 100, timestampMode = 'recent') {
  const events = [];
  const actualCount = Math.min(Math.max(1, count), 1000); // Limit to 1-1000
  
  for (let i = 0; i < actualCount; i++) {
    events.push(generateEvent(timestampMode));
  }
  
  return { events };
}

/**
 * Get pool information for debugging
 */
export function getPoolInfo() {
  return {
    users: userPool.length,
    campaigns: campaignPool.length,
    eventTypes: eventTypePool.length,
    channels: channelPool.length,
    tags: tagPool.length
  };
}
