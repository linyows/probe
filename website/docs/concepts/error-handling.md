---
title: Error Handling
description: Master error handling strategies, recovery patterns, and resilient workflow design
weight: 70
---

# Error Handling

Error handling is crucial for building resilient workflows that can gracefully manage failures and unexpected conditions. This guide explores error handling strategies, recovery patterns, and techniques for building fault-tolerant automation.

## Error Handling Fundamentals

Probe provides several mechanisms for handling errors at different levels of your workflow:

1. **Step-level**: Control how individual steps respond to failures
2. **Job-level**: Manage job failure behavior and recovery
3. **Workflow-level**: Handle overall workflow failure scenarios
4. **Conditional execution**: Route execution based on success/failure states

### Error Types in Probe

Probe recognizes several types of errors:

```yaml
# Test failure - assertion fails
- name: API Health Check
  action: http
  with:
    url: "{{env.API_URL}}/health"
  test: res.status == 200  # This can fail

# Action failure - HTTP timeout, connection error, etc.
- name: Timeout Example
  action: http
  with:
    url: "{{env.SLOW_SERVICE}}"
    timeout: 5s  # May timeout

# Configuration error - invalid URL, missing environment variable
- name: Configuration Error
  action: http
  with:
    url: "{{env.MISSING_VAR}}/endpoint"  # May be undefined
```

## Step-Level Error Handling

### Continue on Error

Control whether workflow execution continues when a step fails:

```yaml
steps:
  - name: Critical Database Check
    action: http
    with:
      url: "{{env.DB_API}}/health"
    test: res.status == 200
    continue_on_error: false  # Default: stop workflow on failure

  - name: Optional Analytics Update
    action: http
    with:
      url: "{{env.ANALYTICS_API}}/update"
    test: res.status == 200
    continue_on_error: true   # Continue even if this fails

  - name: This runs only if analytics succeeds
    if: steps.previous.success
    echo: "Analytics updated successfully"

  - name: This runs regardless of analytics result
    echo: "Workflow continues..."
```

### Graceful Degradation

Handle non-critical failures gracefully:

```yaml
jobs:
  monitoring-with-fallbacks:
    name: Monitoring with Graceful Degradation
    steps:
      - name: Primary Service Check
        id: primary
        action: http
        with:
          url: "{{env.PRIMARY_SERVICE_URL}}/health"
        test: res.status == 200
        continue_on_error: true
        outputs:
          primary_healthy: res.status == 200
          primary_response_time: res.time

      - name: Secondary Service Check
        if: "!outputs.primary.primary_healthy"
        id: secondary
        action: http
        with:
          url: "{{env.SECONDARY_SERVICE_URL}}/health"
        test: res.status == 200
        continue_on_error: true
        outputs:
          secondary_healthy: res.status == 200
          secondary_response_time: res.time

      - name: Cache Service Check
        id: cache
        action: http
        with:
          url: "{{env.CACHE_SERVICE_URL}}/health"
        test: res.status == 200
        continue_on_error: true
        outputs:
          cache_healthy: res.status == 200

      - name: Service Status Summary
        echo: |
          Service Health Summary:
          
          Primary Service: {{outputs.primary.primary_healthy ? "✅ Healthy" : "❌ Down"}}
          {{outputs.primary.primary_healthy ? "Response Time: " + outputs.primary.primary_response_time + "ms" : ""}}
          
          {{outputs.secondary ? "Fallback Service: " + (outputs.secondary.secondary_healthy ? "✅ Healthy" : "❌ Down") : ""}}
          {{outputs.secondary.secondary_healthy ? "Response Time: " + outputs.secondary.secondary_response_time + "ms" : ""}}
          
          Cache Service: {{outputs.cache.cache_healthy ? "✅ Healthy" : "❌ Down"}}
          
          Overall Status: {{
            outputs.primary.primary_healthy || outputs.secondary.secondary_healthy ? 
            "✅ Operational" : "❌ Service Unavailable"
          }}
```

## Job-Level Error Handling

### Job Dependencies and Failure Propagation

Control how job failures affect dependent jobs:

```yaml
jobs:
  infrastructure-check:
    name: Infrastructure Validation
    steps:
      - name: Database Connectivity
        action: http
        with:
          url: "{{env.DB_URL}}/ping"
        test: res.status == 200

      - name: Message Queue Health
        action: http
        with:
          url: "{{env.QUEUE_URL}}/health"
        test: res.status == 200

  application-test:
    name: Application Tests
    needs: [infrastructure-check]  # Only runs if infrastructure is healthy
    steps:
      - name: API Tests
        action: http
        with:
          url: "{{env.API_URL}}/test"
        test: res.status == 200

  notification:
    name: Send Notifications
    needs: [infrastructure-check, application-test]
    if: jobs.infrastructure-check.failed || jobs.application-test.failed
    steps:
      - name: Alert on Failures
        echo: |
          Monitoring Alert:
          
          Infrastructure Check: {{jobs.infrastructure-check.success ? "✅ Passed" : "❌ Failed"}}
          Application Test: {{jobs.application-test.executed ? (jobs.application-test.success ? "✅ Passed" : "❌ Failed") : "⏸️ Skipped"}}
          
          {{jobs.infrastructure-check.failed ? "⚠️ Infrastructure issues detected - investigate immediately" : ""}}
          {{jobs.application-test.failed ? "⚠️ Application tests failed - check application health" : ""}}
```

### Conditional Job Execution

Execute different jobs based on failure scenarios:

```yaml
jobs:
  health-check:
    name: Primary Health Check
    steps:
      - name: Service Health Check
        id: health
        action: http
        with:
          url: "{{env.SERVICE_URL}}/health"
        test: res.status == 200
        continue_on_error: true
        outputs:
          service_healthy: res.status == 200
          error_details: res.status != 200 ? res.text : null

  recovery-procedures:
    name: Recovery Procedures
    if: jobs.health-check.failed
    steps:
      - name: Attempt Service Restart
        id: restart
        action: http
        with:
          url: "{{env.ADMIN_API}}/restart"
          method: POST
        test: res.status == 200
        continue_on_error: true
        outputs:
          restart_successful: res.status == 200

      - name: Verify Recovery
        if: outputs.restart.restart_successful
        action: http
        with:
          url: "{{env.SERVICE_URL}}/health"
        test: res.status == 200
        outputs:
          recovery_successful: res.status == 200

  escalation:
    name: Escalation Procedures
    if: jobs.recovery-procedures.executed && jobs.recovery-procedures.failed
    steps:
      - name: Critical Alert
        action: smtp
        with:
          host: "{{env.SMTP_HOST}}"
          port: 587
          username: "{{env.SMTP_USER}}"
          password: "{{env.SMTP_PASS}}"
          from: "alerts@company.com"
          to: ["ops-team@company.com", "management@company.com"]
          subject: "CRITICAL: Service Down - Manual Intervention Required"
          body: |
            CRITICAL ALERT: Service Recovery Failed
            
            Service: {{env.SERVICE_NAME}}
            Health Check Failed: {{jobs.health-check.failed}}
            Automatic Recovery Attempted: {{jobs.recovery-procedures.executed}}
            Recovery Successful: {{jobs.recovery-procedures.success}}
            
            Error Details: {{outputs.health-check.error_details}}
            
            Manual intervention is required immediately.
            
            Time: {{unixtime()}}
            Environment: {{env.NODE_ENV}}

  success-notification:
    name: Success Notification
    if: jobs.health-check.success || (jobs.recovery-procedures.success && jobs.recovery-procedures.outputs.recovery_successful)
    steps:
      - name: Success Message
        echo: |
          ✅ Service Status: Healthy
          
          {{jobs.health-check.success ? "Primary health check passed" : ""}}
          {{jobs.recovery-procedures.success ? "Service recovered after restart" : ""}}
          
          All systems operational.
```

## Retry and Resilience Patterns

### Implicit Retries with Fallbacks

Implement retry logic using conditional execution:

```yaml
jobs:
  resilient-api-test:
    name: Resilient API Testing
    steps:
      - name: Primary Attempt
        id: attempt1
        action: http
        with:
          url: "{{env.API_URL}}/endpoint"
          timeout: 10s
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          response_time: res.time

      - name: Retry After Brief Delay
        if: "!outputs.attempt1.success"
        id: attempt2
        action: http
        with:
          url: "{{env.API_URL}}/endpoint"
          timeout: 15s
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          response_time: res.time

      - name: Final Attempt with Extended Timeout
        if: "!outputs.attempt1.success && !outputs.attempt2.success"
        id: attempt3
        action: http
        with:
          url: "{{env.API_URL}}/endpoint"
          timeout: 30s
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          response_time: res.time

      - name: Fallback to Alternative Endpoint
        if: "!outputs.attempt1.success && !outputs.attempt2.success && !outputs.attempt3.success"
        id: fallback
        action: http
        with:
          url: "{{env.FALLBACK_API_URL}}/endpoint"
          timeout: 20s
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          response_time: res.time

      - name: Results Summary
        echo: |
          API Test Results:
          
          Attempt 1: {{outputs.attempt1.success ? "✅ Success (" + outputs.attempt1.response_time + "ms)" : "❌ Failed"}}
          {{outputs.attempt2 ? "Attempt 2: " + (outputs.attempt2.success ? "✅ Success (" + outputs.attempt2.response_time + "ms)" : "❌ Failed") : ""}}
          {{outputs.attempt3 ? "Attempt 3: " + (outputs.attempt3.success ? "✅ Success (" + outputs.attempt3.response_time + "ms)" : "❌ Failed") : ""}}
          {{outputs.fallback ? "Fallback: " + (outputs.fallback.success ? "✅ Success (" + outputs.fallback.response_time + "ms)" : "❌ Failed") : ""}}
          
          Final Result: {{
            outputs.attempt1.success || outputs.attempt2.success || outputs.attempt3.success || outputs.fallback.success ?
            "✅ API Accessible" : "❌ All attempts failed"
          }}
```

### Circuit Breaker Pattern

Implement circuit breaker logic to prevent cascading failures:

```yaml
jobs:
  circuit-breaker-test:
    name: Circuit Breaker Pattern
    steps:
      - name: Check Service Health History
        id: health-history
        action: http
        with:
          url: "{{env.MONITORING_API}}/service/{{env.SERVICE_NAME}}/health-history"
        test: res.status == 200
        outputs:
          recent_failures: res.json.failures_last_5_minutes
          error_rate: res.json.error_rate_percentage
          last_success: res.json.last_success_timestamp

      - name: Circuit Breaker Decision
        id: circuit-decision
        echo: "Evaluating circuit breaker state"
        outputs:
          circuit_open: "{{outputs.health-history.error_rate > 50 || outputs.health-history.recent_failures > 10}}"
          should_test: "{{!outputs.health-history.circuit_open || (unixtime() - outputs.health-history.last_success) > 300}}"

      - name: Service Test (Circuit Closed)
        if: "!outputs.circuit-decision.circuit_open"
        id: normal-test
        action: http
        with:
          url: "{{env.SERVICE_URL}}/test"
        test: res.status == 200
        continue_on_error: true
        outputs:
          test_successful: res.status == 200

      - name: Probe Test (Circuit Open)
        if: outputs.circuit-decision.circuit_open && outputs.circuit-decision.should_test
        id: probe-test
        action: http
        with:
          url: "{{env.SERVICE_URL}}/health"  # Lighter probe
          timeout: 5s
        test: res.status == 200
        continue_on_error: true
        outputs:
          probe_successful: res.status == 200

      - name: Circuit Breaker Status
        echo: |
          Circuit Breaker Status Report:
          
          Error Rate: {{outputs.health-history.error_rate}}%
          Recent Failures: {{outputs.health-history.recent_failures}}
          Circuit State: {{outputs.circuit-decision.circuit_open ? "🔴 OPEN (Service Degraded)" : "🟢 CLOSED (Normal Operation)"}}
          
          {{outputs.normal-test ? "Normal Test: " + (outputs.normal-test.test_successful ? "✅ Passed" : "❌ Failed") : ""}}
          {{outputs.probe-test ? "Probe Test: " + (outputs.probe-test.probe_successful ? "✅ Passed" : "❌ Failed") : ""}}
          
          {{outputs.circuit-decision.circuit_open && !outputs.circuit-decision.should_test ? "⏸️ Skipping tests - circuit breaker active" : ""}}
          {{outputs.probe-test.probe_successful ? "🟢 Service recovery detected - circuit may close" : ""}}
```

## Error Context and Debugging

### Comprehensive Error Information

Capture detailed error context for debugging:

```yaml
- name: Detailed Error Capture
  id: api-test
  action: http
  with:
    url: "{{env.API_URL}}/complex-operation"
    method: POST
    body: |
      {
        "operation": "test",
        "timestamp": {{unixtime()}},
        "user_id": "{{env.TEST_USER_ID}}"
      }
  test: res.status == 200 && res.json.success == true
  continue_on_error: true
  outputs:
    success: res.status == 200
    status_code: res.status
    response_time: res.time
    response_size: res.body_size
    error_message: res.status != 200 ? res.text : null
    response_headers: res.headers
    partial_response: res.status != 200 ? res.text.substring(0, 500) : null

- name: Error Analysis and Reporting
  if: "!outputs.api-test.success"
  echo: |
    🚨 API Test Failure Analysis:
    
    Request Details:
    - URL: {{env.API_URL}}/complex-operation
    - Method: POST
    - User ID: {{env.TEST_USER_ID}}
    - Timestamp: {{unixtime()}}
    
    Response Details:
    - Status Code: {{outputs.api-test.status_code}}
    - Response Time: {{outputs.api-test.response_time}}ms
    - Response Size: {{outputs.api-test.response_size}} bytes
    - Content Type: {{outputs.api-test.response_headers["content-type"]}}
    
    Error Information:
    {{outputs.api-test.error_message ? outputs.api-test.error_message : "No error message available"}}
    
    Response Preview:
    {{outputs.api-test.partial_response ? outputs.api-test.partial_response : "No response content"}}
    
    Troubleshooting Suggestions:
    {{outputs.api-test.status_code == 404 ? "- Check if the endpoint URL is correct" : ""}}
    {{outputs.api-test.status_code == 401 ? "- Verify authentication credentials" : ""}}
    {{outputs.api-test.status_code == 403 ? "- Check user permissions" : ""}}
    {{outputs.api-test.status_code == 500 ? "- Server error - check application logs" : ""}}
    {{outputs.api-test.response_time > 10000 ? "- Request timed out - check network connectivity" : ""}}
```

### Error Correlation and Tracking

Track errors across multiple steps and jobs:

```yaml
jobs:
  distributed-test:
    name: Distributed System Test
    steps:
      - name: Initialize Correlation ID
        id: init
        echo: "Starting distributed test"
        outputs:
          correlation_id: "test_{{unixtime()}}_{{random_str(8)}}"
          start_time: "{{unixtime()}}"

      - name: Service A Test
        id: service-a
        action: http
        with:
          url: "{{env.SERVICE_A_URL}}/test"
          headers:
            X-Correlation-ID: "{{outputs.init.correlation_id}}"
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          error_code: res.status != 200 ? res.status : null
          trace_id: res.headers["x-trace-id"]

      - name: Service B Test
        id: service-b
        action: http
        with:
          url: "{{env.SERVICE_B_URL}}/test"
          headers:
            X-Correlation-ID: "{{outputs.init.correlation_id}}"
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          error_code: res.status != 200 ? res.status : null
          trace_id: res.headers["x-trace-id"]

      - name: Service C Test
        id: service-c
        action: http
        with:
          url: "{{env.SERVICE_C_URL}}/test"
          headers:
            X-Correlation-ID: "{{outputs.init.correlation_id}}"
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          error_code: res.status != 200 ? res.status : null
          trace_id: res.headers["x-trace-id"]

      - name: Error Correlation Report
        if: "!outputs.service-a.success || !outputs.service-b.success || !outputs.service-c.success"
        echo: |
          🔍 Distributed Test Error Report
          
          Correlation ID: {{outputs.init.correlation_id}}
          Test Duration: {{unixtime() - outputs.init.start_time}} seconds
          
          Service Status Summary:
          - Service A: {{outputs.service-a.success ? "✅ Healthy" : "❌ Failed (HTTP " + outputs.service-a.error_code + ")"}}
            {{outputs.service-a.trace_id ? "Trace ID: " + outputs.service-a.trace_id : ""}}
          
          - Service B: {{outputs.service-b.success ? "✅ Healthy" : "❌ Failed (HTTP " + outputs.service-b.error_code + ")"}}
            {{outputs.service-b.trace_id ? "Trace ID: " + outputs.service-b.trace_id : ""}}
          
          - Service C: {{outputs.service-c.success ? "✅ Healthy" : "❌ Failed (HTTP " + outputs.service-c.error_code + ")"}}
            {{outputs.service-c.trace_id ? "Trace ID: " + outputs.service-c.trace_id : ""}}
          
          Error Pattern Analysis:
          {{!outputs.service-a.success && !outputs.service-b.success && !outputs.service-c.success ? "🚨 Total system failure - check infrastructure" : ""}}
          {{!outputs.service-a.success && outputs.service-b.success && outputs.service-c.success ? "⚠️ Service A isolated failure" : ""}}
          {{outputs.service-a.success && !outputs.service-b.success && outputs.service-c.success ? "⚠️ Service B isolated failure" : ""}}
          {{outputs.service-a.success && outputs.service-b.success && !outputs.service-c.success ? "⚠️ Service C isolated failure" : ""}}
          
          Investigation Steps:
          1. Check application logs with correlation ID: {{outputs.init.correlation_id}}
          2. Review distributed traces using trace IDs above
          3. Monitor system metrics for the test time window
          4. Verify network connectivity between services
```

## Error Recovery Strategies

### Automated Recovery Procedures

Implement automated recovery for common failure scenarios:

```yaml
jobs:
  self-healing-monitor:
    name: Self-Healing Monitoring
    steps:
      - name: Service Health Check
        id: health-check
        action: http
        with:
          url: "{{env.SERVICE_URL}}/health"
        test: res.status == 200
        continue_on_error: true
        outputs:
          healthy: res.status == 200
          status_code: res.status

      - name: Memory Usage Check
        if: outputs.health-check.healthy
        id: memory-check
        action: http
        with:
          url: "{{env.SERVICE_URL}}/metrics"
        test: res.status == 200 && res.json.memory_usage_percent < 90
        continue_on_error: true
        outputs:
          memory_ok: res.status == 200 && res.json.memory_usage_percent < 90
          memory_usage: res.json.memory_usage_percent

      - name: Restart Service (Health Failure)
        if: "!outputs.health-check.healthy"
        id: restart-health
        action: http
        with:
          url: "{{env.ADMIN_API}}/services/{{env.SERVICE_NAME}}/restart"
          method: POST
        test: res.status == 200
        continue_on_error: true
        outputs:
          restart_initiated: res.status == 200
          restart_reason: "health_check_failed"

      - name: Restart Service (Memory Issue)
        if: outputs.health-check.healthy && outputs.memory-check && !outputs.memory-check.memory_ok
        id: restart-memory
        action: http
        with:
          url: "{{env.ADMIN_API}}/services/{{env.SERVICE_NAME}}/restart"
          method: POST
        test: res.status == 200
        continue_on_error: true
        outputs:
          restart_initiated: res.status == 200
          restart_reason: "high_memory_usage"

      - name: Wait for Service Recovery
        if: outputs.restart-health.restart_initiated || outputs.restart-memory.restart_initiated
        id: recovery-wait
        action: http
        with:
          url: "{{env.SERVICE_URL}}/health"
          timeout: 60s
        test: res.status == 200
        continue_on_error: true
        outputs:
          recovery_successful: res.status == 200

      - name: Recovery Status Report
        echo: |
          🔄 Self-Healing Monitor Report
          
          Initial Health Check: {{outputs.health-check.healthy ? "✅ Healthy" : "❌ Failed (" + outputs.health-check.status_code + ")"}}
          {{outputs.memory-check ? "Memory Usage: " + outputs.memory-check.memory_usage + "% " + (outputs.memory-check.memory_ok ? "✅ Normal" : "⚠️ High") : ""}}
          
          Recovery Actions:
          {{outputs.restart-health.restart_initiated ? "🔄 Service restarted due to health check failure" : ""}}
          {{outputs.restart-memory.restart_initiated ? "🔄 Service restarted due to high memory usage (" + outputs.memory-check.memory_usage + "%)" : ""}}
          
          Recovery Result:
          {{outputs.recovery-wait ? (outputs.recovery-wait.recovery_successful ? "✅ Service recovered successfully" : "❌ Service failed to recover") : "ℹ️ No recovery action needed"}}
          
          Next Actions:
          {{outputs.recovery-wait && !outputs.recovery-wait.recovery_successful ? "🚨 Manual intervention required - service did not recover" : ""}}
          {{outputs.health-check.healthy && (!outputs.memory-check || outputs.memory-check.memory_ok) ? "✅ All systems normal - monitoring continues" : ""}}
```

### Gradual Recovery Testing

Test recovery gradually to avoid overwhelming recovering systems:

```yaml
jobs:
  gradual-recovery-test:
    name: Gradual Recovery Testing
    steps:
      - name: Basic Connectivity Test
        id: ping
        action: http
        with:
          url: "{{env.SERVICE_URL}}/ping"
          timeout: 5s
        test: res.status == 200
        continue_on_error: true
        outputs:
          connectivity: res.status == 200

      - name: Health Endpoint Test
        if: outputs.ping.connectivity
        id: health
        action: http
        with:
          url: "{{env.SERVICE_URL}}/health"
          timeout: 10s
        test: res.status == 200
        continue_on_error: true
        outputs:
          health_ok: res.status == 200

      - name: Light Functional Test
        if: outputs.health.health_ok
        id: light-test
        action: http
        with:
          url: "{{env.SERVICE_URL}}/api/status"
          timeout: 15s
        test: res.status == 200 && res.json.status == "ready"
        continue_on_error: true
        outputs:
          light_test_passed: res.status == 200 && res.json.status == "ready"

      - name: Standard Load Test
        if: outputs.light-test.light_test_passed
        id: standard-test
        action: http
        with:
          url: "{{env.SERVICE_URL}}/api/users"
          timeout: 30s
        test: res.status == 200 && res.json.users != null
        continue_on_error: true
        outputs:
          standard_test_passed: res.status == 200

      - name: Recovery Assessment
        echo: |
          🏥 Recovery Assessment Report
          
          Recovery Progress:
          1. Basic Connectivity: {{outputs.ping.connectivity ? "✅ Restored" : "❌ Failed"}}
          {{outputs.ping.connectivity ? "2. Health Check: " + (outputs.health.health_ok ? "✅ Passed" : "❌ Failed") : "2. Health Check: ⏸️ Skipped"}}
          {{outputs.health.health_ok ? "3. Light Function Test: " + (outputs.light-test.light_test_passed ? "✅ Passed" : "❌ Failed") : "3. Light Function Test: ⏸️ Skipped"}}
          {{outputs.light-test.light_test_passed ? "4. Standard Load Test: " + (outputs.standard-test.standard_test_passed ? "✅ Passed" : "❌ Failed") : "4. Standard Load Test: ⏸️ Skipped"}}
          
          Recovery Status: {{
            outputs.standard-test.standard_test_passed ? "🟢 FULLY RECOVERED" :
            outputs.light-test.light_test_passed ? "🟡 PARTIALLY RECOVERED" :
            outputs.health.health_ok ? "🟡 BASIC FUNCTIONALITY RESTORED" :
            outputs.ping.connectivity ? "🟡 CONNECTIVITY RESTORED" :
            "🔴 SERVICE STILL DOWN"
          }}
          
          Recommendations:
          {{!outputs.ping.connectivity ? "- Check network connectivity and service deployment" : ""}}
          {{outputs.ping.connectivity && !outputs.health.health_ok ? "- Service is starting but not ready - wait and retry" : ""}}
          {{outputs.health.health_ok && !outputs.light-test.light_test_passed ? "- Service health OK but functionality impaired - check dependencies" : ""}}
          {{outputs.light-test.light_test_passed && !outputs.standard-test.standard_test_passed ? "- Service functional but may be under load - monitor performance" : ""}}
          {{outputs.standard-test.standard_test_passed ? "- Service fully operational - resume normal monitoring" : ""}}
```

## Notification and Alerting

### Error-Driven Notifications

Send notifications based on error severity and context:

```yaml
jobs:
  error-notification:
    name: Error Notification System
    needs: [health-check, performance-test, security-scan]
    steps:
      - name: Classify Errors
        id: classification
        echo: "Classifying detected errors"
        outputs:
          critical_errors: "{{jobs.health-check.failed}}"
          performance_degraded: "{{jobs.performance-test.failed}}"
          security_issues: "{{jobs.security-scan.failed}}"
          
          # Error severity calculation
          severity_level: |
            {{jobs.health-check.failed ? "CRITICAL" : 
              (jobs.security-scan.failed ? "HIGH" : 
                (jobs.performance-test.failed ? "MEDIUM" : "LOW"))}}

      - name: Critical Alert
        if: outputs.classification.critical_errors
        action: smtp
        with:
          host: "{{env.SMTP_HOST}}"
          port: 587
          username: "{{env.SMTP_USER}}"
          password: "{{env.SMTP_PASS}}"
          from: "critical-alerts@company.com"
          to: ["oncall@company.com", "management@company.com"]
          subject: "🚨 CRITICAL: {{env.SERVICE_NAME}} Service Down"
          body: |
            CRITICAL SERVICE ALERT
            
            Service: {{env.SERVICE_NAME}}
            Environment: {{env.NODE_ENV}}
            Time: {{unixtime()}}
            Severity: {{outputs.classification.severity_level}}
            
            Issues Detected:
            - Health Check: {{jobs.health-check.failed ? "❌ FAILED" : "✅ Passed"}}
            - Performance Test: {{jobs.performance-test.failed ? "❌ FAILED" : "✅ Passed"}}
            - Security Scan: {{jobs.security-scan.failed ? "❌ FAILED" : "✅ Passed"}}
            
            IMMEDIATE ACTION REQUIRED
            
            This is a critical service failure requiring immediate attention.
            Please check the service status and begin recovery procedures.

      - name: Warning Alert
        if: "!outputs.classification.critical_errors && (outputs.classification.performance_degraded || outputs.classification.security_issues)"
        action: smtp
        with:
          host: "{{env.SMTP_HOST}}"
          port: 587
          username: "{{env.SMTP_USER}}"
          password: "{{env.SMTP_PASS}}"
          from: "monitoring@company.com"
          to: ["dev-team@company.com"]
          subject: "⚠️ {{outputs.classification.severity_level}}: {{env.SERVICE_NAME}} Issues Detected"
          body: |
            Service Monitoring Alert
            
            Service: {{env.SERVICE_NAME}}
            Environment: {{env.NODE_ENV}}
            Time: {{unixtime()}}
            Severity: {{outputs.classification.severity_level}}
            
            Issues Detected:
            {{jobs.performance-test.failed ? "⚠️ Performance degradation detected" : ""}}
            {{jobs.security-scan.failed ? "⚠️ Security issues identified" : ""}}
            
            While the service is operational, these issues require attention
            to prevent potential service degradation.

      - name: Recovery Success Notification
        if: jobs.health-check.success && jobs.performance-test.success && jobs.security-scan.success
        echo: |
          ✅ All Systems Operational
          
          Service: {{env.SERVICE_NAME}}
          Environment: {{env.NODE_ENV}}
          Monitoring Status: All checks passed
          
          No alerts sent - system is healthy.
```

## Best Practices

### 1. Fail Fast vs. Resilience Balance

```yaml
# Critical path - fail fast
- name: Database Connection
  action: http
  with:
    url: "{{env.DB_URL}}/ping"
  test: res.status == 200
  continue_on_error: false  # Stop immediately if database is down

# Non-critical path - be resilient
- name: Analytics Tracking
  action: http
  with:
    url: "{{env.ANALYTICS_URL}}/track"
  test: res.status == 200
  continue_on_error: true   # Continue even if analytics fails
```

### 2. Error Context Preservation

```yaml
# Good: Preserve error context
outputs:
  success: res.status == 200
  error_details: |
    {{res.status != 200 ? 
      "HTTP " + res.status + ": " + res.text.substring(0, 200) : 
      null}}
  debug_info: |
    {{res.status != 200 ? 
      "URL: " + env.API_URL + ", Time: " + res.time + "ms" : 
      null}}
```

### 3. Graduated Response

```yaml
# Good: Different responses for different error types
- name: Error Response Strategy
  if: steps.api-test.failed
  echo: |
    Error Response:
    {{outputs.api-test.status_code == 500 ? "🚨 Server error - escalate immediately" : ""}}
    {{outputs.api-test.status_code == 404 ? "⚠️ Endpoint not found - check configuration" : ""}}
    {{outputs.api-test.status_code == 401 ? "🔐 Authentication failed - refresh credentials" : ""}}
```

### 4. Error Recovery Documentation

```yaml
# Document recovery procedures in workflow
- name: Recovery Instructions
  if: jobs.health-check.failed
  echo: |
    🛠️ Recovery Procedures for {{env.SERVICE_NAME}}:
    
    1. Check service logs: kubectl logs -l app={{env.SERVICE_NAME}}
    2. Verify configuration: check {{env.CONFIG_PATH}}
    3. Restart service: kubectl rollout restart deployment/{{env.SERVICE_NAME}}
    4. Monitor recovery: watch kubectl get pods -l app={{env.SERVICE_NAME}}
    
    Escalation: If service doesn't recover in 10 minutes, contact on-call team.
```

## What's Next?

Now that you understand error handling, explore:

1. **[Execution Model](../execution-model/)** - Learn how Probe executes workflows
2. **[File Merging](../file-merging/)** - Understand configuration composition
3. **[How-tos](../../how-tos/)** - See practical error handling patterns

Error handling is your safety net. Master these patterns to build workflows that gracefully handle the unexpected and recover automatically when possible.