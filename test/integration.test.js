const http = require('http');

// Test configuration
const TEST_HOST = process.env.TEST_HOST || 'localhost';
const TEST_PORT = process.env.TEST_PORT || 8070;
const TEST_TIMEOUT = 30000; // 30 seconds

// Test coordinates (Tallinn, Estonia)
const TEST_LAT = 59.437;
const TEST_LON = 24.7536;

function makeGraphQLRequest(query, variables = {}) {
  return new Promise((resolve, reject) => {
    const postData = JSON.stringify({
      query,
      variables
    });

    const options = {
      hostname: TEST_HOST,
      port: TEST_PORT,
      path: '/graphql',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData)
      },
      timeout: TEST_TIMEOUT
    };

    const req = http.request(options, (res) => {
      let data = '';

      res.on('data', (chunk) => {
        data += chunk;
      });

      res.on('end', () => {
        try {
          const parsed = JSON.parse(data);
          resolve({ statusCode: res.statusCode, body: parsed });
        } catch (e) {
          reject(new Error(`Failed to parse response: ${e.message}`));
        }
      });
    });

    req.on('error', (e) => {
      reject(new Error(`Request failed: ${e.message}`));
    });

    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Request timeout'));
    });

    req.write(postData);
    req.end();
  });
}

async function testHealthCheck() {
  return new Promise((resolve, reject) => {
    const req = http.request({
      hostname: TEST_HOST,
      port: TEST_PORT,
      path: '/health',
      method: 'GET',
      timeout: 5000
    }, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => {
        try {
          const parsed = JSON.parse(data);
          if (parsed.hello === 'world') {
            console.log('✓ Health check passed');
            resolve(true);
          } else {
            reject(new Error('Health check failed: unexpected response'));
          }
        } catch (e) {
          reject(new Error(`Health check failed: ${e.message}`));
        }
      });
    });

    req.on('error', reject);
    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Health check timeout'));
    });
    req.end();
  });
}

async function testWeatherQuery() {
  const query = `
    query GetWeather($lat: String!, $lng: String!) {
      weather(lat: $lat, lng: $lng)
    }
  `;

  const response = await makeGraphQLRequest(query, {
    lat: TEST_LAT.toString(),
    lng: TEST_LON.toString()
  });

  if (response.statusCode !== 200) {
    throw new Error(`Expected status 200, got ${response.statusCode}`);
  }

  if (response.body.errors) {
    throw new Error(`GraphQL errors: ${JSON.stringify(response.body.errors)}`);
  }

  const weather = response.body.data?.weather;
  if (!weather) {
    throw new Error('No weather data returned');
  }

  // Weather returns JSON object
  console.log('✓ Weather query returned data:', JSON.stringify(weather).substring(0, 200) + '...');

  return true;
}

async function testForecastQuery() {
  const query = `
    query GetHistoricalWeather($lat: String!, $lng: String!, $startDate: String!, $endDate: String!) {
      historicalWeather(lat: $lat, lng: $lng, startDate: $startDate, endDate: $endDate) {
        temperature {
          temperature_2m {
            time
            value
          }
        }
      }
    }
  `;

  // Get dates for the last 7 days
  const endDate = new Date();
  const startDate = new Date();
  startDate.setDate(startDate.getDate() - 7);

  const response = await makeGraphQLRequest(query, {
    lat: TEST_LAT.toString(),
    lng: TEST_LON.toString(),
    startDate: startDate.toISOString().split('T')[0],
    endDate: endDate.toISOString().split('T')[0]
  });

  if (response.statusCode !== 200) {
    throw new Error(`Expected status 200, got ${response.statusCode}`);
  }

  if (response.body.errors) {
    throw new Error(`GraphQL errors: ${JSON.stringify(response.body.errors)}`);
  }

  const historical = response.body.data?.historicalWeather;
  if (!historical) {
    throw new Error('No historical weather data returned');
  }

  const tempData = historical.temperature?.temperature_2m;
  if (!tempData || !Array.isArray(tempData) || tempData.length === 0) {
    throw new Error('No temperature data in historical weather');
  }

  console.log(`✓ Historical weather query returned ${tempData.length} temperature readings`);
  
  return true;
}

async function runTests() {
  console.log('Starting integration tests...\n');
  
  const tests = [
    { name: 'Health Check', fn: testHealthCheck },
    { name: 'Weather Query', fn: testWeatherQuery },
    { name: 'Historical Weather Query', fn: testForecastQuery }
  ];

  let passed = 0;
  let failed = 0;

  for (const test of tests) {
    try {
      console.log(`Running: ${test.name}`);
      await test.fn();
      passed++;
    } catch (error) {
      console.error(`✗ ${test.name} failed:`, error.message);
      failed++;
    }
    console.log('');
  }

  console.log('─'.repeat(50));
  console.log(`Tests completed: ${passed} passed, ${failed} failed`);
  console.log('─'.repeat(50));

  if (failed > 0) {
    process.exit(1);
  }

  process.exit(0);
}

// Run tests
runTests().catch((error) => {
  console.error('Test runner failed:', error);
  process.exit(1);
});
