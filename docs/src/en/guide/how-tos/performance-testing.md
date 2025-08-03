---
title: Performance Testing
description: Implement load testing and performance validation with Probe
weight: 40
---

# Performance Testing

This guide shows you how to implement performance testing and load testing with Probe. You'll learn to measure response times, validate performance thresholds, simulate load, and build comprehensive performance validation workflows.

## Basic Performance Testing

### Response Time Measurement

Start with simple response time measurements:

```yaml
name: Basic Performance Testing
description: Measure and validate API response times

env:
  API_BASE_URL: https://api.yourcompany.com
  PERFORMANCE_THRESHOLD_MS: 1000
  EXCELLENT_THRESHOLD_MS: 500

defaults:
  http:
    timeout: 30s
    headers:
      User-Agent: "Probe Performance Tester v1.0"

jobs:
  response-time-baseline:
    name: Response Time Baseline
    steps:
      - name: Lightweight Endpoint
        id: ping
        action: http
        with:
          url: "{{env.API_BASE_URL}}/ping"
        test: |
          res.status == 200 &&
          res.time < 200
        outputs:
          ping_time: res.time
          ping_performance: |
            {{res.time < 100 ? "excellent" :
              res.time < 200 ? "good" :
              res.time < 500 ? "acceptable" : "poor"}}

      - name: Database Query Endpoint
        id: query
        action: http
        with:
          url: "{{env.API_BASE_URL}}/users?limit=50"
          headers:
            Authorization: "Bearer {{env.API_TOKEN}}"
        test: |
          res.status == 200 &&
          res.time < {{env.PERFORMANCE_THRESHOLD_MS}}
        outputs:
          query_time: res.time
          query_performance: |
            {{res.time < env.EXCELLENT_THRESHOLD_MS ? "excellent" :
              res.time < env.PERFORMANCE_THRESHOLD_MS ? "good" :
              res.time < 2000 ? "acceptable" : "poor"}}

      - name: Complex Computation Endpoint
        id: computation
        action: http
        with:
          url: "{{env.API_BASE_URL}}/analytics/summary"
          headers:
            Authorization: "Bearer {{env.API_TOKEN}}"
        test: |
          res.status == 200 &&
          res.time < 5000
        outputs:
          computation_time: res.time
          computation_performance: |
            {{res.time < 2000 ? "excellent" :
              res.time < 5000 ? "good" :
              res.time < 10000 ? "acceptable" : "poor"}}

      - name: File Upload Test
        id: upload
        action: http
        with:
          url: "{{env.API_BASE_URL}}/files/upload"
          method: POST
          headers:
            Authorization: "Bearer {{env.API_TOKEN}}"
            Content-Type: "multipart/form-data"
          body: |
            --boundary123
            Content-Disposition: form-data; name="file"; filename="test.txt"
            Content-Type: text/plain
            
            This is a test file for performance testing.
            It contains sample data to measure upload performance.
            The file size is designed to be moderate for consistent testing.
            --boundary123--
        test: |
          res.status == 201 &&
          res.time < 10000
        outputs:
          upload_time: res.time
          upload_throughput: "{{1024 / (res.time / 1000)}} bytes/sec"  # Approximate

      - name: Performance Baseline Summary
        echo: |
          📊 Performance Baseline Results:
          
          ENDPOINT PERFORMANCE:
          🏓 Ping: {{outputs.ping_time}}ms ({{outputs.ping_performance}})
          🔍 Query: {{outputs.query_time}}ms ({{outputs.query_performance}})
          🧮 Computation: {{outputs.computation_time}}ms ({{outputs.computation_performance}})
          📁 Upload: {{outputs.upload_time}}ms ({{outputs.upload_performance || "measured"}})
          
          PERFORMANCE CLASSIFICATION:
          {{outputs.ping_time < 100 && outputs.query_time < 500 && outputs.computation_time < 2000 ? "🟢 EXCELLENT - All endpoints performing optimally" : ""}}
          {{outputs.ping_time < 200 && outputs.query_time < 1000 && outputs.computation_time < 5000 ? "🟡 GOOD - Performance within acceptable ranges" : ""}}
          {{outputs.computation_time > 5000 || outputs.upload_time > 10000 ? "🔴 NEEDS ATTENTION - Some endpoints are slow" : ""}}
          
          RECOMMENDATIONS:
          {{outputs.query_time > 800 ? "• Consider database query optimization" : ""}}
          {{outputs.computation_time > 3000 ? "• Review computation algorithm efficiency" : ""}}
          {{outputs.upload_time > 8000 ? "• Optimize file upload handling" : ""}}
          {{outputs.ping_time > 150 ? "• Check network latency and infrastructure" : ""}}
```

### Performance Thresholds Validation

Define and validate performance Service Level Objectives (SLOs):

```yaml
name: Performance SLO Validation
description: Validate system performance against defined SLOs

env:
  API_BASE_URL: https://api.yourcompany.com
  
  # Performance SLOs (Service Level Objectives)
  SLO_P50_MS: 500    # 50th percentile
  SLO_P95_MS: 1000   # 95th percentile  
  SLO_P99_MS: 2000   # 99th percentile
  SLO_ERROR_RATE: 0.01  # 1% error rate
  SLO_AVAILABILITY: 0.999  # 99.9% availability

jobs:
  performance-slo-validation:
    name: Performance SLO Validation
    steps:
      # Simulate multiple requests to get performance distribution
      - name: Performance Sample 1
        id: sample1
        action: http
        with:
          url: "{{env.API_BASE_URL}}/users"
          headers:
            Authorization: "Bearer {{env.API_TOKEN}}"
        test: res.status == 200
        continue_on_error: true
        outputs:
          response_time: res.time
          success: res.status == 200

      - name: Performance Sample 2
        id: sample2
        action: http
        with:
          url: "{{env.API_BASE_URL}}/orders"
          headers:
            Authorization: "Bearer {{env.API_TOKEN}}"
        test: res.status == 200
        continue_on_error: true
        outputs:
          response_time: res.time
          success: res.status == 200

      - name: Performance Sample 3
        id: sample3
        action: http
        with:
          url: "{{env.API_BASE_URL}}/products"
          headers:
            Authorization: "Bearer {{env.API_TOKEN}}"
        test: res.status == 200
        continue_on_error: true
        outputs:
          response_time: res.time
          success: res.status == 200

      - name: Performance Sample 4
        id: sample4
        action: http
        with:
          url: "{{env.API_BASE_URL}}/analytics"
          headers:
            Authorization: "Bearer {{env.API_TOKEN}}"
        test: res.status == 200
        continue_on_error: true
        outputs:
          response_time: res.time
          success: res.status == 200

      - name: Performance Sample 5
        id: sample5
        action: http
        with:
          url: "{{env.API_BASE_URL}}/reports"
          headers:
            Authorization: "Bearer {{env.API_TOKEN}}"
        test: res.status == 200
        continue_on_error: true
        outputs:
          response_time: res.time
          success: res.status == 200

      - name: SLO Analysis
        id: slo-analysis
        echo: "Analyzing performance against SLOs"
        outputs:
          # Calculate basic statistics
          total_requests: 5
          successful_requests: |
            {{(outputs.sample1.success ? 1 : 0) +
              (outputs.sample2.success ? 1 : 0) +
              (outputs.sample3.success ? 1 : 0) +
              (outputs.sample4.success ? 1 : 0) +
              (outputs.sample5.success ? 1 : 0)}}
          
          # Calculate average response time
          avg_response_time: |
            {{(outputs.sample1.response_time +
               outputs.sample2.response_time +
               outputs.sample3.response_time +
               outputs.sample4.response_time +
               outputs.sample5.response_time) / 5}}
          
          # Find max response time (approximates 99th percentile for small sample)
          max_response_time: |
            {{[outputs.sample1.response_time,
               outputs.sample2.response_time,
               outputs.sample3.response_time,
               outputs.sample4.response_time,
               outputs.sample5.response_time].max()}}
          
          # Calculate error rate
          error_rate: |
            {{1 - (((outputs.sample1.success ? 1 : 0) +
                    (outputs.sample2.success ? 1 : 0) +
                    (outputs.sample3.success ? 1 : 0) +
                    (outputs.sample4.success ? 1 : 0) +
                    (outputs.sample5.success ? 1 : 0)) / 5)}}
          
          # Calculate availability
          availability: |
            {{((outputs.sample1.success ? 1 : 0) +
               (outputs.sample2.success ? 1 : 0) +
               (outputs.sample3.success ? 1 : 0) +
               (outputs.sample4.success ? 1 : 0) +
               (outputs.sample5.success ? 1 : 0)) / 5}}

      - name: SLO Compliance Check
        echo: |
          📈 Performance SLO Validation Results:
          ======================================
          
          SAMPLE MEASUREMENTS:
          Sample 1: {{outputs.sample1.response_time}}ms {{outputs.sample1.success ? "✅" : "❌"}}
          Sample 2: {{outputs.sample2.response_time}}ms {{outputs.sample2.success ? "✅" : "❌"}}
          Sample 3: {{outputs.sample3.response_time}}ms {{outputs.sample3.success ? "✅" : "❌"}}
          Sample 4: {{outputs.sample4.response_time}}ms {{outputs.sample4.success ? "✅" : "❌"}}
          Sample 5: {{outputs.sample5.response_time}}ms {{outputs.sample5.success ? "✅" : "❌"}}
          
          PERFORMANCE METRICS:
          Average Response Time: {{outputs.slo-analysis.avg_response_time}}ms
          Max Response Time: {{outputs.slo-analysis.max_response_time}}ms
          Success Rate: {{(outputs.slo-analysis.availability * 100)}}%
          Error Rate: {{(outputs.slo-analysis.error_rate * 100)}}%
          
          SLO COMPLIANCE:
          P50 (Average): {{outputs.slo-analysis.avg_response_time <= env.SLO_P50_MS ? "✅ PASS" : "❌ FAIL"}} ({{outputs.slo-analysis.avg_response_time}}ms ≤ {{env.SLO_P50_MS}}ms)
          P99 (Max): {{outputs.slo-analysis.max_response_time <= env.SLO_P99_MS ? "✅ PASS" : "❌ FAIL"}} ({{outputs.slo-analysis.max_response_time}}ms ≤ {{env.SLO_P99_MS}}ms)
          Error Rate: {{outputs.slo-analysis.error_rate <= env.SLO_ERROR_RATE ? "✅ PASS" : "❌ FAIL"}} ({{(outputs.slo-analysis.error_rate * 100)}}% ≤ {{(env.SLO_ERROR_RATE * 100)}}%)
          Availability: {{outputs.slo-analysis.availability >= env.SLO_AVAILABILITY ? "✅ PASS" : "❌ FAIL"}} ({{(outputs.slo-analysis.availability * 100)}}% ≥ {{(env.SLO_AVAILABILITY * 100)}}%)
          
          OVERALL SLO STATUS: {{
            outputs.slo-analysis.avg_response_time <= env.SLO_P50_MS &&
            outputs.slo-analysis.max_response_time <= env.SLO_P99_MS &&
            outputs.slo-analysis.error_rate <= env.SLO_ERROR_RATE &&
            outputs.slo-analysis.availability >= env.SLO_AVAILABILITY
            ? "🟢 ALL SLOs MET" : "🔴 SLO VIOLATIONS DETECTED"
          }}
```

## Load Testing Patterns

### Sequential Load Testing

Gradually increase load to find performance breaking points:

```yaml
name: Sequential Load Testing
description: Gradually increase load to identify performance limits

env:
  API_BASE_URL: https://api.yourcompany.com
  LOAD_TEST_ENDPOINT: /api/stress-test

jobs:
  sequential-load-test:
    name: Sequential Load Testing
    steps:
      # Light Load Test (1 concurrent user)
      - name: Light Load Test
        id: light-load
        action: http
        with:
          url: "{{env.API_BASE_URL}}{{env.LOAD_TEST_ENDPOINT}}"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{env.API_TOKEN}}"
          body: |
            {
              "test_type": "light_load",
              "concurrent_users": 1,
              "duration_seconds": 30,
              "requests_per_second": 5
            }
        test: |
          res.status == 200 &&
          res.json.test_completed == true &&
          res.json.success_rate >= 0.99
        outputs:
          success_rate: res.json.success_rate
          avg_response_time: res.json.avg_response_time
          max_response_time: res.json.max_response_time
          throughput: res.json.requests_per_second
          errors: res.json.error_count

      # Medium Load Test (5 concurrent users)
      - name: Medium Load Test
        id: medium-load
        action: http
        with:
          url: "{{env.API_BASE_URL}}{{env.LOAD_TEST_ENDPOINT}}"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{env.API_TOKEN}}"
          body: |
            {
              "test_type": "medium_load",
              "concurrent_users": 5,
              "duration_seconds": 60,
              "requests_per_second": 25
            }
        test: |
          res.status == 200 &&
          res.json.test_completed == true &&
          res.json.success_rate >= 0.95
        outputs:
          success_rate: res.json.success_rate
          avg_response_time: res.json.avg_response_time
          max_response_time: res.json.max_response_time
          throughput: res.json.requests_per_second
          errors: res.json.error_count

      # Heavy Load Test (20 concurrent users)
      - name: Heavy Load Test
        id: heavy-load
        action: http
        with:
          url: "{{env.API_BASE_URL}}{{env.LOAD_TEST_ENDPOINT}}"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{env.API_TOKEN}}"
          body: |
            {
              "test_type": "heavy_load",
              "concurrent_users": 20,
              "duration_seconds": 120,
              "requests_per_second": 100
            }
        test: |
          res.status == 200 &&
          res.json.test_completed == true &&
          res.json.success_rate >= 0.90
        continue_on_error: true
        outputs:
          success_rate: res.json.success_rate
          avg_response_time: res.json.avg_response_time
          max_response_time: res.json.max_response_time
          throughput: res.json.requests_per_second
          errors: res.json.error_count
          load_test_passed: res.json.success_rate >= 0.90

      # Peak Load Test (50 concurrent users)
      - name: Peak Load Test
        id: peak-load
        action: http
        with:
          url: "{{env.API_BASE_URL}}{{env.LOAD_TEST_ENDPOINT}}"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{env.API_TOKEN}}"
          body: |
            {
              "test_type": "peak_load",
              "concurrent_users": 50,
              "duration_seconds": 180,
              "requests_per_second": 250
            }
        test: |
          res.status == 200 &&
          res.json.test_completed == true
        continue_on_error: true
        outputs:
          success_rate: res.json.success_rate
          avg_response_time: res.json.avg_response_time
          max_response_time: res.json.max_response_time
          throughput: res.json.requests_per_second
          errors: res.json.error_count
          peak_test_passed: res.json.success_rate >= 0.80

      - name: Load Testing Analysis
        echo: |
          🔥 Sequential Load Testing Results:
          ===================================
          
          LIGHT LOAD (1 user, 5 RPS):
          Success Rate: {{(outputs.light-load.success_rate * 100)}}%
          Avg Response: {{outputs.light-load.avg_response_time}}ms
          Max Response: {{outputs.light-load.max_response_time}}ms
          Throughput: {{outputs.light-load.throughput}} RPS
          Errors: {{outputs.light-load.errors}}
          
          MEDIUM LOAD (5 users, 25 RPS):
          Success Rate: {{(outputs.medium-load.success_rate * 100)}}%
          Avg Response: {{outputs.medium-load.avg_response_time}}ms
          Max Response: {{outputs.medium-load.max_response_time}}ms
          Throughput: {{outputs.medium-load.throughput}} RPS
          Errors: {{outputs.medium-load.errors}}
          
          HEAVY LOAD (20 users, 100 RPS):
          Success Rate: {{(outputs.heavy-load.success_rate * 100)}}%
          Avg Response: {{outputs.heavy-load.avg_response_time}}ms
          Max Response: {{outputs.heavy-load.max_response_time}}ms
          Throughput: {{outputs.heavy-load.throughput}} RPS
          Errors: {{outputs.heavy-load.errors}}
          Status: {{outputs.heavy-load.load_test_passed ? "✅ PASSED" : "❌ FAILED"}}
          
          PEAK LOAD (50 users, 250 RPS):
          Success Rate: {{(outputs.peak-load.success_rate * 100)}}%
          Avg Response: {{outputs.peak-load.avg_response_time}}ms
          Max Response: {{outputs.peak-load.max_response_time}}ms
          Throughput: {{outputs.peak-load.throughput}} RPS
          Errors: {{outputs.peak-load.errors}}
          Status: {{outputs.peak-load.peak_test_passed ? "✅ PASSED" : "❌ FAILED"}}
          
          PERFORMANCE ANALYSIS:
          {{outputs.light-load.avg_response_time < outputs.medium-load.avg_response_time ? "✅ Response time increases under load (expected)" : "⚠️ Unexpected response time pattern"}}
          {{outputs.heavy-load.load_test_passed ? "✅ System handles heavy load well" : "⚠️ System shows stress under heavy load"}}
          {{outputs.peak-load.peak_test_passed ? "✅ System survives peak load" : "⚠️ System struggles under peak load"}}
          
          CAPACITY RECOMMENDATIONS:
          Maximum Recommended Load: {{
            outputs.peak-load.peak_test_passed ? "50+ concurrent users" :
            outputs.heavy-load.load_test_passed ? "20-50 concurrent users" :
            "Under 20 concurrent users"
          }}
          
          Performance Optimization Needed: {{
            outputs.heavy-load.avg_response_time > 2000 || outputs.peak-load.success_rate < 0.8 ? "YES" : "NO"
          }}
```

### Stress Testing

Push the system beyond normal operating conditions:

```yaml
name: Stress Testing
description: Test system behavior under extreme load conditions

env:
  API_BASE_URL: https://api.yourcompany.com
  STRESS_TEST_ENDPOINT: /api/stress-test

jobs:
  stress-testing:
    name: System Stress Testing
    steps:
      # Baseline measurement
      - name: Baseline Performance
        id: baseline
        action: http
        with:
          url: "{{env.API_BASE_URL}}{{env.STRESS_TEST_ENDPOINT}}"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{env.API_TOKEN}}"
          body: |
            {
              "test_type": "baseline",
              "concurrent_users": 1,
              "duration_seconds": 30,
              "requests_per_second": 1
            }
        test: res.status == 200
        outputs:
          baseline_response_time: res.json.avg_response_time
          baseline_success_rate: res.json.success_rate

      # Stress Test - High Concurrency
      - name: High Concurrency Stress Test
        id: concurrency-stress
        action: http
        with:
          url: "{{env.API_BASE_URL}}{{env.STRESS_TEST_ENDPOINT}}"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{env.API_TOKEN}}"
          body: |
            {
              "test_type": "concurrency_stress",
              "concurrent_users": 100,
              "duration_seconds": 300,
              "requests_per_second": 500
            }
        test: res.status == 200
        continue_on_error: true
        outputs:
          concurrency_success_rate: res.json.success_rate
          concurrency_avg_response: res.json.avg_response_time
          concurrency_max_response: res.json.max_response_time
          concurrency_throughput: res.json.actual_throughput
          concurrency_errors: res.json.error_count
          concurrency_survived: res.json.success_rate > 0.5

      # Stress Test - High Request Rate
      - name: High Request Rate Stress Test
        id: rate-stress
        action: http
        with:
          url: "{{env.API_BASE_URL}}{{env.STRESS_TEST_ENDPOINT}}"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{env.API_TOKEN}}"
          body: |
            {
              "test_type": "rate_stress",
              "concurrent_users": 20,
              "duration_seconds": 600,
              "requests_per_second": 1000
            }
        test: res.status == 200
        continue_on_error: true
        outputs:
          rate_success_rate: res.json.success_rate
          rate_avg_response: res.json.avg_response_time
          rate_max_response: res.json.max_response_time
          rate_throughput: res.json.actual_throughput
          rate_errors: res.json.error_count
          rate_survived: res.json.success_rate > 0.3

      # Memory Stress Test
      - name: Memory Stress Test
        id: memory-stress
        action: http
        with:
          url: "{{env.API_BASE_URL}}{{env.STRESS_TEST_ENDPOINT}}"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{env.API_TOKEN}}"
          body: |
            {
              "test_type": "memory_stress",
              "concurrent_users": 10,
              "duration_seconds": 240,
              "requests_per_second": 50,
              "large_payload": true,
              "payload_size_mb": 10
            }
        test: res.status == 200
        continue_on_error: true
        outputs:
          memory_success_rate: res.json.success_rate
          memory_avg_response: res.json.avg_response_time
          memory_max_response: res.json.max_response_time
          memory_errors: res.json.error_count
          memory_survived: res.json.success_rate > 0.7

      - name: Stress Test Analysis
        echo: |
          💥 Stress Testing Analysis:
          ===========================
          
          BASELINE PERFORMANCE:
          Response Time: {{outputs.baseline.baseline_response_time}}ms
          Success Rate: {{(outputs.baseline.baseline_success_rate * 100)}}%
          
          HIGH CONCURRENCY STRESS (100 users, 500 RPS):
          Success Rate: {{(outputs.concurrency-stress.concurrency_success_rate * 100)}}%
          Avg Response: {{outputs.concurrency-stress.concurrency_avg_response}}ms
          Max Response: {{outputs.concurrency-stress.concurrency_max_response}}ms
          Actual Throughput: {{outputs.concurrency-stress.concurrency_throughput}} RPS
          Total Errors: {{outputs.concurrency-stress.concurrency_errors}}
          Survival Status: {{outputs.concurrency-stress.concurrency_survived ? "✅ SURVIVED" : "❌ FAILED"}}
          
          HIGH REQUEST RATE STRESS (20 users, 1000 RPS):
          Success Rate: {{(outputs.rate-stress.rate_success_rate * 100)}}%
          Avg Response: {{outputs.rate-stress.rate_avg_response}}ms
          Max Response: {{outputs.rate-stress.rate_max_response}}ms
          Actual Throughput: {{outputs.rate-stress.rate_throughput}} RPS
          Total Errors: {{outputs.rate-stress.rate_errors}}
          Survival Status: {{outputs.rate-stress.rate_survived ? "✅ SURVIVED" : "❌ FAILED"}}
          
          MEMORY STRESS (10MB payloads):
          Success Rate: {{(outputs.memory-stress.memory_success_rate * 100)}}%
          Avg Response: {{outputs.memory-stress.memory_avg_response}}ms
          Max Response: {{outputs.memory-stress.memory_max_response}}ms
          Total Errors: {{outputs.memory-stress.memory_errors}}
          Survival Status: {{outputs.memory-stress.memory_survived ? "✅ SURVIVED" : "❌ FAILED"}}
          
          DEGRADATION ANALYSIS:
          Response Time Degradation: {{((outputs.concurrency-stress.concurrency_avg_response / outputs.baseline.baseline_response_time) * 100)}}% of baseline
          Throughput vs Target: {{(outputs.concurrency-stress.concurrency_throughput / 500 * 100)}}%
          
          SYSTEM RESILIENCE:
          {{outputs.concurrency-stress.concurrency_survived && outputs.rate-stress.rate_survived && outputs.memory-stress.memory_survived ? "🟢 EXCELLENT - System handles all stress scenarios" : ""}}
          {{outputs.concurrency-stress.concurrency_survived && outputs.rate-stress.rate_survived ? "🟡 GOOD - System handles most stress scenarios" : ""}}
          {{!outputs.concurrency-stress.concurrency_survived || !outputs.rate-stress.rate_survived ? "🔴 NEEDS IMPROVEMENT - System struggles under stress" : ""}}
          
          BREAKING POINTS IDENTIFIED:
          {{!outputs.concurrency-stress.concurrency_survived ? "• High concurrency breaks the system" : ""}}
          {{!outputs.rate-stress.rate_survived ? "• High request rate overwhelms the system" : ""}}
          {{!outputs.memory-stress.memory_survived ? "• Large payloads cause memory issues" : ""}}
          {{outputs.concurrency-stress.concurrency_avg_response > (outputs.baseline.baseline_response_time * 10) ? "• Severe response time degradation under load" : ""}}
```

## Performance Monitoring and Alerting

### Continuous Performance Monitoring

Monitor performance continuously and alert on degradation:

```yaml
name: Continuous Performance Monitoring
description: Monitor system performance and detect degradation

env:
  API_BASE_URL: https://api.yourcompany.com
  BASELINE_RESPONSE_TIME: 500
  WARNING_THRESHOLD: 1000
  CRITICAL_THRESHOLD: 2000
  DEGRADATION_THRESHOLD: 2.0  # 2x baseline

jobs:
  performance-monitoring:
    name: Performance Monitoring
    steps:
      # Monitor critical user journey
      - name: User Login Performance
        id: login
        action: http
        with:
          url: "{{env.API_BASE_URL}}/auth/login"
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "username": "{{env.PERF_TEST_USERNAME}}",
              "password": "{{env.PERF_TEST_PASSWORD}}"
            }
        test: res.status == 200
        outputs:
          login_time: res.time
          login_success: res.status == 200
          auth_token: res.json.access_token

      # Monitor data retrieval performance
      - name: Data Retrieval Performance
        id: data-retrieval
        action: http
        with:
          url: "{{env.API_BASE_URL}}/dashboard/data"
          headers:
            Authorization: "Bearer {{outputs.login.auth_token}}"
        test: res.status == 200
        outputs:
          retrieval_time: res.time
          retrieval_success: res.status == 200
          data_size: res.body_size

      # Monitor search performance
      - name: Search Performance
        id: search
        action: http
        with:
          url: "{{env.API_BASE_URL}}/search?q=test&limit=50"
          headers:
            Authorization: "Bearer {{outputs.login.auth_token}}"
        test: res.status == 200
        outputs:
          search_time: res.time
          search_success: res.status == 200
          results_count: res.json.results.length

      # Monitor transaction performance
      - name: Transaction Performance
        id: transaction
        action: http
        with:
          url: "{{env.API_BASE_URL}}/transactions"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{outputs.login.auth_token}}"
          body: |
            {
              "type": "test_transaction",
              "amount": 10.00,
              "currency": "USD"
            }
        test: res.status == 201
        outputs:
          transaction_time: res.time
          transaction_success: res.status == 201

      - name: Performance Analysis
        id: analysis
        echo: "Analyzing performance metrics"
        outputs:
          # Calculate overall health
          all_operations_healthy: |
            {{outputs.login.login_success &&
              outputs.data-retrieval.retrieval_success &&
              outputs.search.search_success &&
              outputs.transaction.transaction_success}}
          
          # Calculate average response time
          avg_response_time: |
            {{(outputs.login.login_time +
               outputs.data-retrieval.retrieval_time +
               outputs.search.search_time +
               outputs.transaction.transaction_time) / 4}}
          
          # Check for performance degradation
          degradation_detected: |
            {{outputs.login.login_time > (env.BASELINE_RESPONSE_TIME * env.DEGRADATION_THRESHOLD) ||
              outputs.data-retrieval.retrieval_time > (env.BASELINE_RESPONSE_TIME * env.DEGRADATION_THRESHOLD) ||
              outputs.search.search_time > (env.BASELINE_RESPONSE_TIME * env.DEGRADATION_THRESHOLD) ||
              outputs.transaction.transaction_time > (env.BASELINE_RESPONSE_TIME * env.DEGRADATION_THRESHOLD)}}
          
          # Determine alert level
          alert_level: |
            {{outputs.login.login_time > env.CRITICAL_THRESHOLD ||
              outputs.data-retrieval.retrieval_time > env.CRITICAL_THRESHOLD ||
              outputs.search.search_time > env.CRITICAL_THRESHOLD ||
              outputs.transaction.transaction_time > env.CRITICAL_THRESHOLD ? "critical" :
              outputs.login.login_time > env.WARNING_THRESHOLD ||
              outputs.data-retrieval.retrieval_time > env.WARNING_THRESHOLD ||
              outputs.search.search_time > env.WARNING_THRESHOLD ||
              outputs.transaction.transaction_time > env.WARNING_THRESHOLD ? "warning" : "ok"}}

  performance-alerting:
    name: Performance Alerting
    needs: [performance-monitoring]
    steps:
      # Critical performance alert
      - name: Critical Performance Alert
        if: outputs.performance-monitoring.alert_level == "critical"
        action: smtp
        with:
          host: "{{env.SMTP_HOST}}"
          port: 587
          username: "{{env.SMTP_USERNAME}}"
          password: "{{env.SMTP_PASSWORD}}"
          from: "performance-alerts@yourcompany.com"
          to: ["oncall@yourcompany.com", "performance-team@yourcompany.com"]
          subject: "🚨 CRITICAL: Performance Degradation Detected"
          body: |
            CRITICAL PERFORMANCE ALERT
            =========================
            
            Time: {{unixtime()}}
            Environment: {{env.ENVIRONMENT}}
            
            Performance Metrics:
            Login: {{outputs.performance-monitoring.login_time}}ms (threshold: {{env.CRITICAL_THRESHOLD}}ms)
            Data Retrieval: {{outputs.performance-monitoring.retrieval_time}}ms
            Search: {{outputs.performance-monitoring.search_time}}ms
            Transaction: {{outputs.performance-monitoring.transaction_time}}ms
            
            Average Response Time: {{outputs.performance-monitoring.avg_response_time}}ms
            Baseline: {{env.BASELINE_RESPONSE_TIME}}ms
            
            Impact: User experience severely degraded
            Action Required: Immediate investigation
            
            Dashboard: {{env.PERFORMANCE_DASHBOARD_URL}}
            Runbook: {{env.PERFORMANCE_RUNBOOK_URL}}

      # Warning performance alert
      - name: Warning Performance Alert
        if: outputs.performance-monitoring.alert_level == "warning"
        action: smtp
        with:
          host: "{{env.SMTP_HOST}}"
          port: 587
          username: "{{env.SMTP_USERNAME}}"
          password: "{{env.SMTP_PASSWORD}}"
          from: "performance-alerts@yourcompany.com"
          to: ["performance-team@yourcompany.com"]
          subject: "⚠️ WARNING: Performance Degradation Detected"
          body: |
            PERFORMANCE WARNING
            ==================
            
            Time: {{unixtime()}}
            Environment: {{env.ENVIRONMENT}}
            
            Performance Metrics:
            Login: {{outputs.performance-monitoring.login_time}}ms
            Data Retrieval: {{outputs.performance-monitoring.retrieval_time}}ms
            Search: {{outputs.performance-monitoring.search_time}}ms
            Transaction: {{outputs.performance-monitoring.transaction_time}}ms
            
            Average Response Time: {{outputs.performance-monitoring.avg_response_time}}ms
            Warning Threshold: {{env.WARNING_THRESHOLD}}ms
            
            Action: Monitor closely and investigate if degradation continues

      # All clear status
      - name: Performance Status Report
        if: outputs.performance-monitoring.alert_level == "ok"
        echo: |
          ✅ Performance Monitoring - All Systems Normal
          
          Performance Metrics:
          Login: {{outputs.performance-monitoring.login_time}}ms
          Data Retrieval: {{outputs.performance-monitoring.retrieval_time}}ms
          Search: {{outputs.performance-monitoring.search_time}}ms
          Transaction: {{outputs.performance-monitoring.transaction_time}}ms
          
          Average Response Time: {{outputs.performance-monitoring.avg_response_time}}ms
          Status: All operations within acceptable performance ranges
```

## Database Performance Testing

### Database Query Performance

Test database query performance and optimization:

```yaml
name: Database Performance Testing
description: Test database query performance and identify bottlenecks

env:
  DB_API_URL: https://db-api.yourcompany.com
  QUERY_TIMEOUT: 5000

jobs:
  database-performance:
    name: Database Performance Testing
    steps:
      # Simple query performance
      - name: Simple Query Performance
        id: simple-query
        action: http
        with:
          url: "{{env.DB_API_URL}}/query/simple"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{env.DB_API_TOKEN}}"
          body: |
            {
              "query": "SELECT COUNT(*) FROM users WHERE active = true",
              "timeout": {{env.QUERY_TIMEOUT}}
            }
        test: |
          res.status == 200 &&
          res.time < 1000 &&
          res.json.execution_time_ms < 500
        outputs:
          simple_query_time: res.time
          simple_execution_time: res.json.execution_time_ms
          simple_rows_affected: res.json.rows_affected

      # Complex query performance
      - name: Complex Query Performance
        id: complex-query
        action: http
        with:
          url: "{{env.DB_API_URL}}/query/complex"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{env.DB_API_TOKEN}}"
          body: |
            {
              "query": "SELECT u.*, COUNT(o.id) as order_count FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.created_at > DATE_SUB(NOW(), INTERVAL 30 DAY) GROUP BY u.id ORDER BY order_count DESC LIMIT 100",
              "timeout": {{env.QUERY_TIMEOUT}}
            }
        test: |
          res.status == 200 &&
          res.time < 3000 &&
          res.json.execution_time_ms < 2000
        outputs:
          complex_query_time: res.time
          complex_execution_time: res.json.execution_time_ms
          complex_rows_returned: res.json.rows_returned

      # Aggregation query performance
      - name: Aggregation Query Performance
        id: aggregation-query
        action: http
        with:
          url: "{{env.DB_API_URL}}/query/aggregation"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{env.DB_API_TOKEN}}"
          body: |
            {
              "query": "SELECT DATE(created_at) as date, COUNT(*) as daily_orders, SUM(total) as daily_revenue, AVG(total) as avg_order_value FROM orders WHERE created_at >= DATE_SUB(NOW(), INTERVAL 90 DAY) GROUP BY DATE(created_at) ORDER BY date DESC",
              "timeout": {{env.QUERY_TIMEOUT}}
            }
        test: |
          res.status == 200 &&
          res.time < 5000 &&
          res.json.execution_time_ms < 3000
        outputs:
          aggregation_query_time: res.time
          aggregation_execution_time: res.json.execution_time_ms
          aggregation_rows_returned: res.json.rows_returned

      # Index performance test
      - name: Index Performance Test
        id: index-test
        action: http
        with:
          url: "{{env.DB_API_URL}}/performance/indexes"
          headers:
            Authorization: "Bearer {{env.DB_API_TOKEN}}"
        test: res.status == 200
        outputs:
          index_efficiency: res.json.index_efficiency_percent
          slow_queries_count: res.json.slow_queries_last_hour
          missing_indexes: res.json.missing_indexes_count

      - name: Database Performance Analysis
        echo: |
          🗄️ Database Performance Analysis:
          =================================
          
          QUERY PERFORMANCE:
          Simple Query: {{outputs.simple-query.simple_execution_time}}ms ({{outputs.simple-query.simple_rows_affected}} rows)
          Complex Query: {{outputs.complex-query.complex_execution_time}}ms ({{outputs.complex-query.complex_rows_returned}} rows)
          Aggregation Query: {{outputs.aggregation-query.aggregation_execution_time}}ms ({{outputs.aggregation-query.aggregation_rows_returned}} rows)
          
          PERFORMANCE CLASSIFICATION:
          Simple Query: {{outputs.simple-query.simple_execution_time < 100 ? "🟢 Excellent" : outputs.simple-query.simple_execution_time < 500 ? "🟡 Good" : "🔴 Needs Optimization"}}
          Complex Query: {{outputs.complex-query.complex_execution_time < 500 ? "🟢 Excellent" : outputs.complex-query.complex_execution_time < 2000 ? "🟡 Good" : "🔴 Needs Optimization"}}
          Aggregation Query: {{outputs.aggregation-query.aggregation_execution_time < 1000 ? "🟢 Excellent" : outputs.aggregation-query.aggregation_execution_time < 3000 ? "🟡 Good" : "🔴 Needs Optimization"}}
          
          INDEX PERFORMANCE:
          Index Efficiency: {{outputs.index-test.index_efficiency}}%
          Slow Queries (1hr): {{outputs.index-test.slow_queries_count}}
          Missing Indexes: {{outputs.index-test.missing_indexes}}
          
          RECOMMENDATIONS:
          {{outputs.simple-query.simple_execution_time > 500 ? "• Optimize simple query execution - consider indexing" : ""}}
          {{outputs.complex-query.complex_execution_time > 2000 ? "• Complex query needs optimization - review joins and indexes" : ""}}
          {{outputs.aggregation-query.aggregation_execution_time > 3000 ? "• Aggregation query is slow - consider pre-computed summaries" : ""}}
          {{outputs.index-test.index_efficiency < 80 ? "• Index efficiency is low - review and optimize indexes" : ""}}
          {{outputs.index-test.slow_queries_count > 10 ? "• High number of slow queries detected - investigate query patterns" : ""}}
          {{outputs.index-test.missing_indexes > 0 ? "• Missing indexes detected - implement recommended indexes" : ""}}
          
          OVERALL DATABASE HEALTH: {{
            outputs.simple-query.simple_execution_time < 500 &&
            outputs.complex-query.complex_execution_time < 2000 &&
            outputs.aggregation-query.aggregation_execution_time < 3000 &&
            outputs.index-test.index_efficiency > 80
            ? "🟢 EXCELLENT" : "🟡 NEEDS ATTENTION"
          }}
```

## Best Practices

### 1. Baseline Establishment

```yaml
# Good: Establish baseline before testing
- name: Establish Baseline
  action: http
  with:
    url: "{{env.API_URL}}/health"
  outputs:
    baseline_response_time: res.time
```

### 2. Gradual Load Increase

```yaml
# Good: Gradually increase load
jobs:
  light-load:      # 1-10 users
  medium-load:     # 10-50 users
  heavy-load:      # 50-100 users
  stress-load:     # 100+ users
```

### 3. Comprehensive Metrics

```yaml
# Good: Capture multiple performance metrics
outputs:
  response_time: res.time
  throughput: res.json.requests_per_second
  success_rate: res.json.success_rate
  error_rate: res.json.error_rate
  cpu_usage: res.json.system.cpu_percent
  memory_usage: res.json.system.memory_percent
```

### 4. Performance Thresholds

```yaml
# Good: Define clear performance thresholds
env:
  EXCELLENT_THRESHOLD: 200ms
  GOOD_THRESHOLD: 500ms
  ACCEPTABLE_THRESHOLD: 1000ms
  CRITICAL_THRESHOLD: 2000ms
```

## What's Next?

Now that you can implement performance testing, explore:

- **[Environment Management](../environment-management/)** - Manage test configurations across environments
- **[Monitoring Workflows](../monitoring-workflows/)** - Build comprehensive monitoring systems
- **[Error Handling Strategies](../error-handling-strategies/)** - Handle performance failures gracefully

Performance testing ensures your system can handle real-world load. Use these patterns to validate performance requirements and identify bottlenecks before they impact users.