# Performance Testing

This guide shows you how to implement performance testing and load testing with Probe. You'll learn to measure response times, validate performance thresholds, simulate load, and build comprehensive performance validation workflows.

## Basic Performance Testing

### Response Time Measurement

Start with simple response time measurements:

```yaml
name: Basic Performance Testing
description: Measure and validate API response times

vars:
  api_base_url: "{{API_BASE_URL ?? 'https://api.yourcompany.com'}}"
  api_token: "{{API_TOKEN}}"
  performance_threshold_ms: "{{PERFORMANCE_THRESHOLD_MS ?? '1000'}}"
  excellent_threshold_ms: "{{EXCELLENT_THRESHOLD_MS ?? '500'}}"

jobs:
- id: response-time-baseline
  name: Response Time Baseline
  defaults:
    http:
      headers:
        User-Agent: "Probe Performance Tester v1.0"
  steps:
    - name: Lightweight Endpoint
      id: ping
      uses: http
      with:
        method: GET
        url: "{{vars.api_base_url}}/ping"
      test: |
        res.code == 200 &&
        (rt.sec * 1000) < 200
      outputs:
        ping_time: (rt.sec * 1000)
        ping_performance: |
          {{(rt.sec * 1000) < 100 ? "excellent" :
            (rt.sec * 1000) < 200 ? "good" :
            (rt.sec * 1000) < 500 ? "acceptable" : "poor"}}

    - name: Database Query Endpoint
      id: query
      uses: http
      with:
        method: GET
        url: "{{vars.api_base_url}}/users?limit=50"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: |
        res.code == 200 &&
        (rt.sec * 1000) < {{vars.performance_threshold_ms}}
      outputs:
        query_time: (rt.sec * 1000)
        query_performance: |
          {{(rt.sec * 1000) < vars.excellent_threshold_ms ? "excellent" :
            (rt.sec * 1000) < vars.performance_threshold_ms ? "good" :
            (rt.sec * 1000) < 2000 ? "acceptable" : "poor"}}

    - name: Complex Computation Endpoint
      id: computation
      uses: http
      with:
        method: GET
        url: "{{vars.api_base_url}}/analytics/summary"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: |
        res.code == 200 &&
        (rt.sec * 1000) < 5000
      outputs:
        computation_time: (rt.sec * 1000)
        computation_performance: |
          {{(rt.sec * 1000) < 2000 ? "excellent" :
            (rt.sec * 1000) < 5000 ? "good" :
            (rt.sec * 1000) < 10000 ? "acceptable" : "poor"}}

    - name: File Upload Test
      id: upload
      uses: http
      with:
        url: "{{vars.api_base_url}}/files/upload"
        method: POST
        headers:
          Authorization: "Bearer {{vars.api_token}}"
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
        res.code == 201 &&
        (rt.sec * 1000) < 10000
      outputs:
        upload_time: (rt.sec * 1000)
        upload_throughput: "{{1024 / ((rt.sec * 1000) / 1000)}} bytes/sec"  # Approximate

    - name: Performance Baseline Summary
      uses: hello
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

vars:
  api_base_url: "{{API_BASE_URL ?? 'https://api.yourcompany.com'}}"
  api_token: "{{API_TOKEN}}"

  # Performance SLOs (Service Level Objectives)
  slo_p50_ms: "{{SLO_P50_MS ?? 500}}"       # 50th percentile
  slo_p99_ms: "{{SLO_P99_MS ?? 2000}}"      # 99th percentile
  slo_error_rate: "{{SLO_ERROR_RATE ?? 0.01}}"        # 1% error rate
  slo_availability: "{{SLO_AVAILABILITY ?? 0.999}}"  # 99.9% availability

jobs:
- id: performance-slo-validation
  name: Performance SLO Validation
  steps:
    # Simulate multiple requests to get performance distribution
    - name: Performance Sample 1
      id: sample1
      uses: http
      with:
        method: GET
        url: "{{vars.api_base_url}}/users"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: res.code == 200
      outputs:
        response_time: (rt.sec * 1000)
        success: res.code == 200

    - name: Performance Sample 2
      id: sample2
      uses: http
      with:
        method: GET
        url: "{{vars.api_base_url}}/orders"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: res.code == 200
      outputs:
        response_time: (rt.sec * 1000)
        success: res.code == 200

    - name: Performance Sample 3
      id: sample3
      uses: http
      with:
        method: GET
        url: "{{vars.api_base_url}}/products"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: res.code == 200
      outputs:
        response_time: (rt.sec * 1000)
        success: res.code == 200

    - name: Performance Sample 4
      id: sample4
      uses: http
      with:
        method: GET
        url: "{{vars.api_base_url}}/analytics"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: res.code == 200
      outputs:
        response_time: (rt.sec * 1000)
        success: res.code == 200

    - name: Performance Sample 5
      id: sample5
      uses: http
      with:
        method: GET
        url: "{{vars.api_base_url}}/reports"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: res.code == 200
      outputs:
        response_time: (rt.sec * 1000)
        success: res.code == 200

    - name: SLO Analysis
      uses: hello
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
          {{max([outputs.sample1.response_time,
             outputs.sample2.response_time,
             outputs.sample3.response_time,
             outputs.sample4.response_time,
             outputs.sample5.response_time])}}
          
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
      uses: hello
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
        Average Response Time: {{outputs['slo-analysis'].avg_response_time}}ms
        Max Response Time: {{outputs['slo-analysis'].max_response_time}}ms
        Success Rate: {{(outputs['slo-analysis'].availability * 100)}}%
        Error Rate: {{(outputs['slo-analysis'].error_rate * 100)}}%
          
        SLO COMPLIANCE:
        P50 (Average): {{outputs['slo-analysis'].avg_response_time <= vars.slo_p50_ms ? "✅ PASS" : "❌ FAIL"}} ({{outputs['slo-analysis'].avg_response_time}}ms ≤ {{vars.slo_p50_ms}}ms)
        P99 (Max): {{outputs['slo-analysis'].max_response_time <= vars.slo_p99_ms ? "✅ PASS" : "❌ FAIL"}} ({{outputs['slo-analysis'].max_response_time}}ms ≤ {{vars.slo_p99_ms}}ms)
        Error Rate: {{outputs['slo-analysis'].error_rate <= vars.slo_error_rate ? "✅ PASS" : "❌ FAIL"}} ({{(outputs['slo-analysis'].error_rate * 100)}}% ≤ {{(vars.slo_error_rate * 100)}}%)
        Availability: {{outputs['slo-analysis'].availability >= vars.slo_availability ? "✅ PASS" : "❌ FAIL"}} ({{(outputs['slo-analysis'].availability * 100)}}% ≥ {{(vars.slo_availability * 100)}}%)
          
        OVERALL SLO STATUS: {{
          outputs['slo-analysis'].avg_response_time <= vars.slo_p50_ms &&
          outputs['slo-analysis'].max_response_time <= vars.slo_p99_ms &&
          outputs['slo-analysis'].error_rate <= vars.slo_error_rate &&
          outputs['slo-analysis'].availability >= vars.slo_availability
          ? "🟢 ALL SLOs MET" : "🔴 SLO VIOLATIONS DETECTED"
        }}
```

## Load Testing Patterns

### Sequential Load Testing

Gradually increase load to find performance breaking points:

```yaml
name: Sequential Load Testing
description: Gradually increase load to identify performance limits

vars:
  api_base_url: "{{API_BASE_URL ?? 'https://api.yourcompany.com'}}"
  load_test_endpoint: "{{LOAD_TEST_ENDPOINT ?? '/api/stress-test'}}"
  api_token: "{{API_TOKEN}}"

jobs:
- id: sequential-load-test
  name: Sequential Load Testing
  steps:
    # Light Load Test (1 concurrent user)
    - name: Light Load Test
      id: light-load
      uses: http
      with:
        url: "{{vars.api_base_url}}{{vars.load_test_endpoint}}"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{vars.api_token}}"
        body: |
          {
            "test_type": "light_load",
            "concurrent_users": 1,
            "duration_seconds": 30,
            "requests_per_second": 5
          }
      test: |
        res.code == 200 &&
        res.body.test_completed == true &&
        res.body.success_rate >= 0.99
      outputs:
        success_rate: res.body.success_rate
        avg_response_time: res.body.avg_response_time
        max_response_time: res.body.max_response_time
        throughput: res.body.requests_per_second
        errors: res.body.error_count

    # Medium Load Test (5 concurrent users)
    - name: Medium Load Test
      id: medium-load
      uses: http
      with:
        url: "{{vars.api_base_url}}{{vars.load_test_endpoint}}"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{vars.api_token}}"
        body: |
          {
            "test_type": "medium_load",
            "concurrent_users": 5,
            "duration_seconds": 60,
            "requests_per_second": 25
          }
      test: |
        res.code == 200 &&
        res.body.test_completed == true &&
        res.body.success_rate >= 0.95
      outputs:
        success_rate: res.body.success_rate
        avg_response_time: res.body.avg_response_time
        max_response_time: res.body.max_response_time
        throughput: res.body.requests_per_second
        errors: res.body.error_count

    # Heavy Load Test (20 concurrent users)
    - name: Heavy Load Test
      id: heavy-load
      uses: http
      with:
        url: "{{vars.api_base_url}}{{vars.load_test_endpoint}}"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{vars.api_token}}"
        body: |
          {
            "test_type": "heavy_load",
            "concurrent_users": 20,
            "duration_seconds": 120,
            "requests_per_second": 100
          }
      test: |
        res.code == 200 &&
        res.body.test_completed == true &&
        res.body.success_rate >= 0.90
      outputs:
        success_rate: res.body.success_rate
        avg_response_time: res.body.avg_response_time
        max_response_time: res.body.max_response_time
        throughput: res.body.requests_per_second
        errors: res.body.error_count
        load_test_passed: res.body.success_rate >= 0.90

    # Peak Load Test (50 concurrent users)
    - name: Peak Load Test
      id: peak-load
      uses: http
      with:
        url: "{{vars.api_base_url}}{{vars.load_test_endpoint}}"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{vars.api_token}}"
        body: |
          {
            "test_type": "peak_load",
            "concurrent_users": 50,
            "duration_seconds": 180,
            "requests_per_second": 250
          }
      test: |
        res.code == 200 &&
        res.body.test_completed == true
      outputs:
        success_rate: res.body.success_rate
        avg_response_time: res.body.avg_response_time
        max_response_time: res.body.max_response_time
        throughput: res.body.requests_per_second
        errors: res.body.error_count
        peak_test_passed: res.body.success_rate >= 0.80

    - name: Load Testing Analysis
      uses: hello
      echo: |
        🔥 Sequential Load Testing Results:
        ===================================
          
        LIGHT LOAD (1 user, 5 RPS):
        Success Rate: {{(outputs['light-load'].success_rate * 100)}}%
        Avg Response: {{outputs['light-load'].avg_response_time}}ms
        Max Response: {{outputs['light-load'].max_response_time}}ms
        Throughput: {{outputs['light-load'].throughput}} RPS
        Errors: {{outputs['light-load'].errors}}
          
        MEDIUM LOAD (5 users, 25 RPS):
        Success Rate: {{(outputs['medium-load'].success_rate * 100)}}%
        Avg Response: {{outputs['medium-load'].avg_response_time}}ms
        Max Response: {{outputs['medium-load'].max_response_time}}ms
        Throughput: {{outputs['medium-load'].throughput}} RPS
        Errors: {{outputs['medium-load'].errors}}
          
        HEAVY LOAD (20 users, 100 RPS):
        Success Rate: {{(outputs['heavy-load'].success_rate * 100)}}%
        Avg Response: {{outputs['heavy-load'].avg_response_time}}ms
        Max Response: {{outputs['heavy-load'].max_response_time}}ms
        Throughput: {{outputs['heavy-load'].throughput}} RPS
        Errors: {{outputs['heavy-load'].errors}}
        Status: {{outputs['heavy-load'].load_test_passed ? "✅ PASSED" : "❌ FAILED"}}
          
        PEAK LOAD (50 users, 250 RPS):
        Success Rate: {{(outputs['peak-load'].success_rate * 100)}}%
        Avg Response: {{outputs['peak-load'].avg_response_time}}ms
        Max Response: {{outputs['peak-load'].max_response_time}}ms
        Throughput: {{outputs['peak-load'].throughput}} RPS
        Errors: {{outputs['peak-load'].errors}}
        Status: {{outputs['peak-load'].peak_test_passed ? "✅ PASSED" : "❌ FAILED"}}
          
        PERFORMANCE ANALYSIS:
        {{outputs['light-load'].avg_response_time < outputs['medium-load'].avg_response_time ? "✅ Response time increases under load (expected)" : "⚠️ Unexpected response time pattern"}}
        {{outputs['heavy-load'].load_test_passed ? "✅ System handles heavy load well" : "⚠️ System shows stress under heavy load"}}
        {{outputs['peak-load'].peak_test_passed ? "✅ System survives peak load" : "⚠️ System struggles under peak load"}}
          
        CAPACITY RECOMMENDATIONS:
        Maximum Recommended Load: {{
          outputs['peak-load'].peak_test_passed ? "50+ concurrent users" :
          outputs['heavy-load'].load_test_passed ? "20-50 concurrent users" :
          "Under 20 concurrent users"
        }}
          
        Performance Optimization Needed: {{
          outputs['heavy-load'].avg_response_time > 2000 || outputs['peak-load'].success_rate < 0.8 ? "YES" : "NO"
        }}
```

### Stress Testing

Push the system beyond normal operating conditions:

```yaml
name: Stress Testing
description: Test system behavior under extreme load conditions

vars:
  api_base_url: "{{API_BASE_URL ?? 'https://api.yourcompany.com'}}"
  stress_test_endpoint: "{{STRESS_TEST_ENDPOINT ?? '/api/stress-test'}}"
  api_token: "{{API_TOKEN}}"

jobs:
- id: stress-testing
  name: System Stress Testing
  steps:
    # Baseline measurement
    - name: Baseline Performance
      id: baseline
      uses: http
      with:
        url: "{{vars.api_base_url}}{{vars.stress_test_endpoint}}"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{vars.api_token}}"
        body: |
          {
            "test_type": "baseline",
            "concurrent_users": 1,
            "duration_seconds": 30,
            "requests_per_second": 1
          }
      test: res.code == 200
      outputs:
        baseline_response_time: res.body.avg_response_time
        baseline_success_rate: res.body.success_rate

    # Stress Test - High Concurrency
    - name: High Concurrency Stress Test
      id: concurrency-stress
      uses: http
      with:
        url: "{{vars.api_base_url}}{{vars.stress_test_endpoint}}"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{vars.api_token}}"
        body: |
          {
            "test_type": "concurrency_stress",
            "concurrent_users": 100,
            "duration_seconds": 300,
            "requests_per_second": 500
          }
      test: res.code == 200
      outputs:
        concurrency_success_rate: res.body.success_rate
        concurrency_avg_response: res.body.avg_response_time
        concurrency_max_response: res.body.max_response_time
        concurrency_throughput: res.body.actual_throughput
        concurrency_errors: res.body.error_count
        concurrency_survived: res.body.success_rate > 0.5

    # Stress Test - High Request Rate
    - name: High Request Rate Stress Test
      id: rate-stress
      uses: http
      with:
        url: "{{vars.api_base_url}}{{vars.stress_test_endpoint}}"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{vars.api_token}}"
        body: |
          {
            "test_type": "rate_stress",
            "concurrent_users": 20,
            "duration_seconds": 600,
            "requests_per_second": 1000
          }
      test: res.code == 200
      outputs:
        rate_success_rate: res.body.success_rate
        rate_avg_response: res.body.avg_response_time
        rate_max_response: res.body.max_response_time
        rate_throughput: res.body.actual_throughput
        rate_errors: res.body.error_count
        rate_survived: res.body.success_rate > 0.3

    # Memory Stress Test
    - name: Memory Stress Test
      id: memory-stress
      uses: http
      with:
        url: "{{vars.api_base_url}}{{vars.stress_test_endpoint}}"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{vars.api_token}}"
        body: |
          {
            "test_type": "memory_stress",
            "concurrent_users": 10,
            "duration_seconds": 240,
            "requests_per_second": 50,
            "large_payload": true,
            "payload_size_mb": 10
          }
      test: res.code == 200
      outputs:
        memory_success_rate: res.body.success_rate
        memory_avg_response: res.body.avg_response_time
        memory_max_response: res.body.max_response_time
        memory_errors: res.body.error_count
        memory_survived: res.body.success_rate > 0.7

    - name: Stress Test Analysis
      uses: hello
      echo: |
        💥 Stress Testing Analysis:
        ===========================
          
        BASELINE PERFORMANCE:
        Response Time: {{outputs.baseline.baseline_response_time}}ms
        Success Rate: {{(outputs.baseline.baseline_success_rate * 100)}}%
          
        HIGH CONCURRENCY STRESS (100 users, 500 RPS):
        Success Rate: {{(outputs['concurrency-stress'].concurrency_success_rate * 100)}}%
        Avg Response: {{outputs['concurrency-stress'].concurrency_avg_response}}ms
        Max Response: {{outputs['concurrency-stress'].concurrency_max_response}}ms
        Actual Throughput: {{outputs['concurrency-stress'].concurrency_throughput}} RPS
        Total Errors: {{outputs['concurrency-stress'].concurrency_errors}}
        Survival Status: {{outputs['concurrency-stress'].concurrency_survived ? "✅ SURVIVED" : "❌ FAILED"}}
          
        HIGH REQUEST RATE STRESS (20 users, 1000 RPS):
        Success Rate: {{(outputs['rate-stress'].rate_success_rate * 100)}}%
        Avg Response: {{outputs['rate-stress'].rate_avg_response}}ms
        Max Response: {{outputs['rate-stress'].rate_max_response}}ms
        Actual Throughput: {{outputs['rate-stress'].rate_throughput}} RPS
        Total Errors: {{outputs['rate-stress'].rate_errors}}
        Survival Status: {{outputs['rate-stress'].rate_survived ? "✅ SURVIVED" : "❌ FAILED"}}
          
        MEMORY STRESS (10MB payloads):
        Success Rate: {{(outputs['memory-stress'].memory_success_rate * 100)}}%
        Avg Response: {{outputs['memory-stress'].memory_avg_response}}ms
        Max Response: {{outputs['memory-stress'].memory_max_response}}ms
        Total Errors: {{outputs['memory-stress'].memory_errors}}
        Survival Status: {{outputs['memory-stress'].memory_survived ? "✅ SURVIVED" : "❌ FAILED"}}
          
        DEGRADATION ANALYSIS:
        Response Time Degradation: {{((outputs['concurrency-stress'].concurrency_avg_response / outputs.baseline.baseline_response_time) * 100)}}% of baseline
        Throughput vs Target: {{(outputs['concurrency-stress'].concurrency_throughput / 500 * 100)}}%
          
        SYSTEM RESILIENCE:
        {{outputs['concurrency-stress'].concurrency_survived && outputs['rate-stress'].rate_survived && outputs['memory-stress'].memory_survived ? "🟢 EXCELLENT - System handles all stress scenarios" : ""}}
        {{outputs['concurrency-stress'].concurrency_survived && outputs['rate-stress'].rate_survived ? "🟡 GOOD - System handles most stress scenarios" : ""}}
        {{!outputs['concurrency-stress'].concurrency_survived || !outputs['rate-stress'].rate_survived ? "🔴 NEEDS IMPROVEMENT - System struggles under stress" : ""}}
          
        BREAKING POINTS IDENTIFIED:
        {{!outputs['concurrency-stress'].concurrency_survived ? "• High concurrency breaks the system" : ""}}
        {{!outputs['rate-stress'].rate_survived ? "• High request rate overwhelms the system" : ""}}
        {{!outputs['memory-stress'].memory_survived ? "• Large payloads cause memory issues" : ""}}
        {{outputs['concurrency-stress'].concurrency_avg_response > (outputs.baseline.baseline_response_time * 10) ? "• Severe response time degradation under load" : ""}}
```

## Performance Monitoring and Alerting

### Continuous Performance Monitoring

Monitor performance continuously and alert on degradation:

```yaml
name: Continuous Performance Monitoring
description: Monitor system performance and detect degradation

vars:
  api_base_url: "{{API_BASE_URL ?? 'https://api.yourcompany.com'}}"
  perf_test_username: "{{PERF_TEST_USERNAME}}"
  perf_test_password: "{{PERF_TEST_PASSWORD}}"
  baseline_response_time: "{{BASELINE_RESPONSE_TIME ?? '500'}}"
  warning_threshold: "{{WARNING_THRESHOLD ?? '1000'}}"
  critical_threshold: "{{CRITICAL_THRESHOLD ?? '2000'}}"
  degradation_threshold: "{{DEGRADATION_THRESHOLD ?? '2.0'}}"  # 2x baseline

jobs:
- id: performance-monitoring
  name: Performance Monitoring
  steps:
    # Monitor critical user journey
    - name: User Login Performance
      id: login
      uses: http
      with:
        url: "{{vars.api_base_url}}/auth/login"
        method: POST
        headers:
          Content-Type: "application/json"
        body: |
          {
            "username": "{{vars.perf_test_username}}",
            "password": "{{vars.perf_test_password}}"
          }
      test: res.code == 200
      outputs:
        login_time: (rt.sec * 1000)
        login_success: res.code == 200
        auth_token: res.body.access_token

    # Monitor data retrieval performance
    - name: Data Retrieval Performance
      id: data-retrieval
      uses: http
      with:
        method: GET
        url: "{{vars.api_base_url}}/dashboard/data"
        headers:
          Authorization: "Bearer {{outputs.login.auth_token}}"
      test: res.code == 200
      outputs:
        retrieval_time: (rt.sec * 1000)
        retrieval_success: res.code == 200
        data_size: res.body_size

    # Monitor search performance
    - name: Search Performance
      id: search
      uses: http
      with:
        method: GET
        url: "{{vars.api_base_url}}/search?q=test&limit=50"
        headers:
          Authorization: "Bearer {{outputs.login.auth_token}}"
      test: res.code == 200
      outputs:
        search_time: (rt.sec * 1000)
        search_success: res.code == 200
        results_count: len(res.body.results)

    # Monitor transaction performance
    - name: Transaction Performance
      id: transaction
      uses: http
      with:
        url: "{{vars.api_base_url}}/transactions"
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
      test: res.code == 201
      outputs:
        transaction_time: (rt.sec * 1000)
        transaction_success: res.code == 201

    - name: Performance Analysis
      uses: hello
      id: analysis
      echo: "Analyzing performance metrics"
      outputs:
        # Calculate overall health
        all_operations_healthy: |
          {{outputs.login.login_success &&
            outputs['data-retrieval'].retrieval_success &&
            outputs.search.search_success &&
            outputs.transaction.transaction_success}}
          
        # Calculate average response time
        avg_response_time: |
          {{(outputs.login.login_time +
             outputs['data-retrieval'].retrieval_time +
             outputs.search.search_time +
             outputs.transaction.transaction_time) / 4}}
          
        # Check for performance degradation
        degradation_detected: |
          {{outputs.login.login_time > (vars.baseline_response_time * vars.degradation_threshold) ||
            outputs['data-retrieval'].retrieval_time > (vars.baseline_response_time * vars.degradation_threshold) ||
            outputs.search.search_time > (vars.baseline_response_time * vars.degradation_threshold) ||
            outputs.transaction.transaction_time > (vars.baseline_response_time * vars.degradation_threshold)}}
          
        # Determine alert level
        alert_level: |
          {{outputs.login.login_time > vars.critical_threshold ||
            outputs['data-retrieval'].retrieval_time > vars.critical_threshold ||
            outputs.search.search_time > vars.critical_threshold ||
            outputs.transaction.transaction_time > vars.critical_threshold ? "critical" :
            outputs.login.login_time > vars.warning_threshold ||
            outputs['data-retrieval'].retrieval_time > vars.warning_threshold ||
            outputs.search.search_time > vars.warning_threshold ||
            outputs.transaction.transaction_time > vars.warning_threshold ? "warning" : "ok"}}

vars:
  smtp_host: "{{SMTP_HOST}}"
  smtp_username: "{{SMTP_USERNAME}}"
  smtp_password: "{{SMTP_PASSWORD}}"
  environment: "{{ENVIRONMENT}}"
  performance_dashboard_url: "{{PERFORMANCE_DASHBOARD_URL}}"
  performance_runbook_url: "{{PERFORMANCE_RUNBOOK_URL}}"

  performance-alerting:
    name: Performance Alerting
    needs: [performance-monitoring]
    steps:
      # Critical performance alert
      - name: Critical Performance Alert
        uses: smtp
        with:
          addr: "{{vars.smtp_host}}:587"
          from: "performance-alerts@yourcompany.com"
          to: "oncall@yourcompany.com"
          subject: "🚨 CRITICAL: Performance Degradation Detected"
          session: 1
          message: 1
          length: 500
        echo: |
          CRITICAL PERFORMANCE ALERT
          =========================
            
          Time: {{unixtime()}}
          Environment: {{vars.environment}}
            
          Performance Metrics:
          Login: {{outputs['performance-monitoring'].login_time}}ms (threshold: {{vars.critical_threshold}}ms)
          Data Retrieval: {{outputs['performance-monitoring'].retrieval_time}}ms
          Search: {{outputs['performance-monitoring'].search_time}}ms
          Transaction: {{outputs['performance-monitoring'].transaction_time}}ms
            
          Average Response Time: {{outputs['performance-monitoring'].avg_response_time}}ms
          Baseline: {{vars.baseline_response_time}}ms
            
          Impact: User experience severely degraded
          Action Required: Immediate investigation
            
          Dashboard: {{vars.performance_dashboard_url}}
          Runbook: {{vars.performance_runbook_url}}

          ing performance alert
      - name: Warning Performance Alert
        uses: smtp
        with:
          addr: "{{vars.smtp_host}}:587"
          from: "performance-alerts@yourcompany.com"
          to: "performance-team@yourcompany.com"
          subject: "⚠️ WARNING: Performance Degradation Detected"
          session: 1
          message: 1
          length: 500
        echo: |
          PERFORMANCE WARNING
          ==================
            
          Time: {{unixtime()}}
          Environment: {{vars.environment}}
            
          Performance Metrics:
          Login: {{outputs['performance-monitoring'].login_time}}ms
          Data Retrieval: {{outputs['performance-monitoring'].retrieval_time}}ms
          Search: {{outputs['performance-monitoring'].search_time}}ms
          Transaction: {{outputs['performance-monitoring'].transaction_time}}ms
            
          Average Response Time: {{outputs['performance-monitoring'].avg_response_time}}ms
          Warning Threshold: {{vars.warning_threshold}}ms
            
          Action: Monitor closely and investigate if degradation continues

          clear status
      - name: Performance Status Report
        uses: hello
        echo: |
          ✅ Performance Monitoring - All Systems Normal
          
          Performance Metrics:
          Login: {{outputs['performance-monitoring'].login_time}}ms
          Data Retrieval: {{outputs['performance-monitoring'].retrieval_time}}ms
          Search: {{outputs['performance-monitoring'].search_time}}ms
          Transaction: {{outputs['performance-monitoring'].transaction_time}}ms
          
          Average Response Time: {{outputs['performance-monitoring'].avg_response_time}}ms
          Status: All operations within acceptable performance ranges
```

## Database Performance Testing

### Database Query Performance

Test database query performance and optimization:

```yaml
name: Database Performance Testing
description: Test database query performance and identify bottlenecks

vars:
  db_api_url: "{{DB_API_URL ?? 'https://db-api.yourcompany.com'}}"
  db_api_token: "{{DB_API_TOKEN}}"
  query_timeout: "{{QUERY_TIMEOUT ?? '5000'}}"

jobs:
- id: database-performance
  name: Database Performance Testing
  steps:
    # Simple query performance
    - name: Simple Query Performance
      id: simple-query
      uses: http
      with:
        url: "{{vars.db_api_url}}/query/simple"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{vars.db_api_token}}"
        body: |
          {
            "query": "SELECT COUNT(*) FROM users WHERE active = true",
            "timeout": {{vars.query_timeout}}
          }
      test: |
        res.code == 200 &&
        (rt.sec * 1000) < 1000 &&
        res.body.execution_time_ms < 500
      outputs:
        simple_query_time: (rt.sec * 1000)
        simple_execution_time: res.body.execution_time_ms
        simple_rows_affected: res.body.rows_affected

    # Complex query performance
    - name: Complex Query Performance
      id: complex-query
      uses: http
      with:
        url: "{{vars.db_api_url}}/query/complex"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{vars.db_api_token}}"
        body: |
          {
            "query": "SELECT u.*, COUNT(o.id) as order_count FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.created_at > DATE_SUB(NOW(), INTERVAL 30 DAY) GROUP BY u.id ORDER BY order_count DESC LIMIT 100",
            "timeout": {{vars.query_timeout}}
          }
      test: |
        res.code == 200 &&
        (rt.sec * 1000) < 3000 &&
        res.body.execution_time_ms < 2000
      outputs:
        complex_query_time: (rt.sec * 1000)
        complex_execution_time: res.body.execution_time_ms
        complex_rows_returned: res.body.rows_returned

    # Aggregation query performance
    - name: Aggregation Query Performance
      id: aggregation-query
      uses: http
      with:
        url: "{{vars.db_api_url}}/query/aggregation"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{vars.db_api_token}}"
        body: |
          {
            "query": "SELECT DATE(created_at) as date, COUNT(*) as daily_orders, SUM(total) as daily_revenue, AVG(total) as avg_order_value FROM orders WHERE created_at >= DATE_SUB(NOW(), INTERVAL 90 DAY) GROUP BY DATE(created_at) ORDER BY date DESC",
            "timeout": {{vars.query_timeout}}
          }
      test: |
        res.code == 200 &&
        (rt.sec * 1000) < 5000 &&
        res.body.execution_time_ms < 3000
      outputs:
        aggregation_query_time: (rt.sec * 1000)
        aggregation_execution_time: res.body.execution_time_ms
        aggregation_rows_returned: res.body.rows_returned

    # Index performance test
    - name: Index Performance Test
      id: index-test
      uses: http
      with:
        method: GET
        url: "{{vars.db_api_url}}/performance/indexes"
        headers:
          Authorization: "Bearer {{vars.db_api_token}}"
      test: res.code == 200
      outputs:
        index_efficiency: res.body.index_efficiency_percent
        slow_queries_count: res.body.slow_queries_last_hour
        missing_indexes: res.body.missing_indexes_count

    - name: Database Performance Analysis
      uses: hello
      echo: |
        🗄️ Database Performance Analysis:
        =================================
          
        QUERY PERFORMANCE:
        Simple Query: {{outputs['simple-query'].simple_execution_time}}ms ({{outputs['simple-query'].simple_rows_affected}} rows)
        Complex Query: {{outputs['complex-query'].complex_execution_time}}ms ({{outputs['complex-query'].complex_rows_returned}} rows)
        Aggregation Query: {{outputs['aggregation-query'].aggregation_execution_time}}ms ({{outputs['aggregation-query'].aggregation_rows_returned}} rows)
          
        PERFORMANCE CLASSIFICATION:
        Simple Query: {{outputs['simple-query'].simple_execution_time < 100 ? "🟢 Excellent" : outputs['simple-query'].simple_execution_time < 500 ? "🟡 Good" : "🔴 Needs Optimization"}}
        Complex Query: {{outputs['complex-query'].complex_execution_time < 500 ? "🟢 Excellent" : outputs['complex-query'].complex_execution_time < 2000 ? "🟡 Good" : "🔴 Needs Optimization"}}
        Aggregation Query: {{outputs['aggregation-query'].aggregation_execution_time < 1000 ? "🟢 Excellent" : outputs['aggregation-query'].aggregation_execution_time < 3000 ? "🟡 Good" : "🔴 Needs Optimization"}}
          
        INDEX PERFORMANCE:
        Index Efficiency: {{outputs['index-test'].index_efficiency}}%
        Slow Queries (1hr): {{outputs['index-test'].slow_queries_count}}
        Missing Indexes: {{outputs['index-test'].missing_indexes}}
          
        RECOMMENDATIONS:
        {{outputs['simple-query'].simple_execution_time > 500 ? "• Optimize simple query execution - consider indexing" : ""}}
        {{outputs['complex-query'].complex_execution_time > 2000 ? "• Complex query needs optimization - review joins and indexes" : ""}}
        {{outputs['aggregation-query'].aggregation_execution_time > 3000 ? "• Aggregation query is slow - consider pre-computed summaries" : ""}}
        {{outputs['index-test'].index_efficiency < 80 ? "• Index efficiency is low - review and optimize indexes" : ""}}
        {{outputs['index-test'].slow_queries_count > 10 ? "• High number of slow queries detected - investigate query patterns" : ""}}
        {{outputs['index-test'].missing_indexes > 0 ? "• Missing indexes detected - implement recommended indexes" : ""}}
          
        OVERALL DATABASE HEALTH: {{
          outputs['simple-query'].simple_execution_time < 500 &&
          outputs['complex-query'].complex_execution_time < 2000 &&
          outputs['aggregation-query'].aggregation_execution_time < 3000 &&
          outputs['index-test'].index_efficiency > 80
          ? "🟢 EXCELLENT" : "🟡 NEEDS ATTENTION"
        }}
```

## Best Practices

### 1. Baseline Establishment

```yaml
# Good: Establish baseline before testing
vars:
  api_url: "{{API_URL}}"

- name: Establish Baseline
  uses: http
  with:
    method: GET
    url: "{{vars.api_url}}/health"
  outputs:
    baseline_response_time: (rt.sec * 1000)
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
  response_time: (rt.sec * 1000)
  throughput: res.body.requests_per_second
  success_rate: res.body.success_rate
  error_rate: res.body.error_rate
  cpu_usage: res.body.system.cpu_percent
  memory_usage: res.body.system.memory_percent
```

### 4. Performance Thresholds

```yaml
# Good: Define clear performance thresholds
vars:
  EXCELLENT_THRESHOLD: 200ms
  GOOD_THRESHOLD: 500ms
  ACCEPTABLE_THRESHOLD: 1000ms
  CRITICAL_THRESHOLD: 2000ms
```

## What's Next?

Now that you can implement performance testing, explore:

- **[Environment Management](/guide/how-tos/environment-management)** - Manage test configurations across environments
- **[Monitoring Workflows](/guide/how-tos/monitoring-workflows)** - Build comprehensive monitoring systems
- **[Error Handling Strategies](/guide/how-tos/error-handling-strategies)** - Handle performance failures gracefully

Performance testing ensures your system can handle real-world load. Use these patterns to validate performance requirements and identify bottlenecks before they impact users.
