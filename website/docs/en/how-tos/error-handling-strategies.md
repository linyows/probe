---
title: Error Handling Strategies
description: Implement robust error handling patterns for resilient workflows
weight: 30
---

# Error Handling Strategies

This guide shows you how to implement robust error handling in Probe workflows. You'll learn to handle failures gracefully, implement recovery patterns, and build resilient automation that can cope with unexpected conditions.

## Basic Error Handling Patterns

### Fail Fast vs Continue

Choose the right error handling strategy based on the criticality of operations:

```yaml
name: Error Handling Strategy Examples
description: Demonstrate different error handling approaches

env:
  CRITICAL_SERVICE_URL: https://critical.yourcompany.com
  OPTIONAL_SERVICE_URL: https://optional.yourcompany.com
  NOTIFICATION_URL: https://notifications.yourcompany.com

jobs:
  critical-operations:
    name: Critical Operations
    steps:
      # Fail fast for critical operations
      - name: Critical Database Check
        action: http
        with:
          url: "{{env.CRITICAL_SERVICE_URL}}/database/health"
        test: res.status == 200
        continue_on_error: false  # Default: stop workflow on failure
        outputs:
          database_healthy: res.status == 200

      # This step only runs if database check passes
      - name: Critical API Check
        action: http
        with:
          url: "{{env.CRITICAL_SERVICE_URL}}/api/health"
        test: res.status == 200
        outputs:
          api_healthy: res.status == 200

  resilient-operations:
    name: Resilient Operations
    steps:
      # Continue on error for optional services
      - name: Optional Analytics Service
        action: http
        with:
          url: "{{env.OPTIONAL_SERVICE_URL}}/analytics"
        test: res.status == 200
        continue_on_error: true   # Continue even if this fails
        outputs:
          analytics_available: res.status == 200
          analytics_error: res.status != 200 ? res.status : null

      # This step always runs regardless of previous step
      - name: Optional Notification Service
        action: http
        with:
          url: "{{env.NOTIFICATION_URL}}/health"
        test: res.status == 200
        continue_on_error: true
        outputs:
          notifications_available: res.status == 200

      # Conditional logic based on service availability
      - name: Service Availability Report
        echo: |
          🔧 Service Availability Report:
          
          Analytics Service: {{outputs.analytics_available ? "✅ Available" : "❌ Unavailable"}}
          {{outputs.analytics_error ? "Error Code: " + outputs.analytics_error : ""}}
          
          Notification Service: {{outputs.notifications_available ? "✅ Available" : "❌ Unavailable"}}
          
          Impact Assessment:
          {{!outputs.analytics_available ? "• Analytics features may be limited" : ""}}
          {{!outputs.notifications_available ? "• User notifications may be delayed" : ""}}
          {{outputs.analytics_available && outputs.notifications_available ? "• All optional services operational" : ""}}
```

### Graceful Degradation

Implement fallback mechanisms when primary services fail:

```yaml
name: Graceful Degradation Pattern
description: Implement fallback services and graceful degradation

env:
  PRIMARY_API_URL: https://primary.api.yourcompany.com
  SECONDARY_API_URL: https://secondary.api.yourcompany.com
  CACHE_API_URL: https://cache.yourcompany.com
  FALLBACK_API_URL: https://fallback.api.yourcompany.com

jobs:
  service-with-fallbacks:
    name: Service with Multiple Fallbacks
    steps:
      # Try primary service first
      - name: Primary Service Attempt
        id: primary
        action: http
        with:
          url: "{{env.PRIMARY_API_URL}}/data"
          timeout: 10s
        test: res.status == 200 && res.time < 5000
        continue_on_error: true
        outputs:
          success: res.status == 200 && res.time < 5000
          response_time: res.time
          data: res.json

      # Try secondary service if primary fails or is slow
      - name: Secondary Service Attempt
        if: "!outputs.primary.success"
        id: secondary
        action: http
        with:
          url: "{{env.SECONDARY_API_URL}}/data"
          timeout: 15s
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          response_time: res.time
          data: res.json

      # Try cache if both primary and secondary fail
      - name: Cache Fallback
        if: "!outputs.primary.success && !outputs.secondary.success"
        id: cache
        action: http
        with:
          url: "{{env.CACHE_API_URL}}/cached-data"
          timeout: 5s
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          response_time: res.time
          data: res.json
          cached_data: true

      # Final fallback to static data
      - name: Static Fallback
        if: "!outputs.primary.success && !outputs.secondary.success && !outputs.cache.success"
        id: fallback
        action: http
        with:
          url: "{{env.FALLBACK_API_URL}}/static-data"
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          response_time: res.time
          data: res.json
          static_data: true

      - name: Service Resolution Summary
        echo: |
          🎯 Service Resolution Summary:
          
          Resolution Path:
          {{outputs.primary.success ? "✅ Primary Service (optimal)" : "❌ Primary Service failed/slow (" + outputs.primary.response_time + "ms)"}}
          {{outputs.secondary.success ? "✅ Secondary Service (backup)" : (!outputs.primary.success ? "❌ Secondary Service failed" : "")}}
          {{outputs.cache.success ? "✅ Cache Service (degraded)" : (!outputs.primary.success && !outputs.secondary.success ? "❌ Cache Service failed" : "")}}
          {{outputs.fallback.success ? "✅ Static Fallback (minimal)" : (!outputs.primary.success && !outputs.secondary.success && !outputs.cache.success ? "❌ All services failed" : "")}}
          
          Final Status: {{
            outputs.primary.success ? "🟢 Optimal Performance" :
            outputs.secondary.success ? "🟡 Backup Service Active" :
            outputs.cache.success ? "🟠 Degraded Mode (cached data)" :
            outputs.fallback.success ? "🔴 Minimal Functionality (static data)" :
            "🚨 Total Service Failure"
          }}
          
          Data Source: {{
            outputs.primary.success ? "Live Primary" :
            outputs.secondary.success ? "Live Secondary" :
            outputs.cache.success ? "Cached (may be stale)" :
            outputs.fallback.success ? "Static Fallback" :
            "None Available"
          }}
```

## Retry Patterns

### Exponential Backoff Retry

Implement retry logic with increasing delays:

```yaml
name: Retry with Exponential Backoff
description: Implement retry patterns for transient failures

env:
  UNRELIABLE_SERVICE_URL: https://api.unreliable.service.com
  MAX_RETRIES: 3

jobs:
  retry-pattern:
    name: Exponential Backoff Retry Pattern
    steps:
      # First attempt
      - name: Initial Attempt
        id: attempt1
        action: http
        with:
          url: "{{env.UNRELIABLE_SERVICE_URL}}/data"
          timeout: 10s
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          attempt_number: 1
          response_time: res.time
          error_code: res.status != 200 ? res.status : null

      # Second attempt (2-second delay)
      - name: Retry Attempt 1 (2s delay)
        if: "!outputs.attempt1.success"
        id: attempt2
        action: http
        with:
          url: "{{env.UNRELIABLE_SERVICE_URL}}/data"
          timeout: 15s
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          attempt_number: 2
          response_time: res.time
          error_code: res.status != 200 ? res.status : null

      # Third attempt (4-second delay)
      - name: Retry Attempt 2 (4s delay)
        if: "!outputs.attempt1.success && !outputs.attempt2.success"
        id: attempt3
        action: http
        with:
          url: "{{env.UNRELIABLE_SERVICE_URL}}/data"
          timeout: 20s
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          attempt_number: 3
          response_time: res.time
          error_code: res.status != 200 ? res.status : null

      # Final attempt (8-second delay)
      - name: Final Attempt (8s delay)
        if: "!outputs.attempt1.success && !outputs.attempt2.success && !outputs.attempt3.success"
        id: attempt4
        action: http
        with:
          url: "{{env.UNRELIABLE_SERVICE_URL}}/data"
          timeout: 30s
        test: res.status == 200
        continue_on_error: true
        outputs:
          success: res.status == 200
          attempt_number: 4
          response_time: res.time
          error_code: res.status != 200 ? res.status : null

      - name: Retry Summary
        echo: |
          🔄 Retry Pattern Results:
          
          Attempt History:
          1. Initial: {{outputs.attempt1.success ? "✅ Success (" + outputs.attempt1.response_time + "ms)" : "❌ Failed (HTTP " + outputs.attempt1.error_code + ")"}}
          {{outputs.attempt2 ? "2. Retry 1: " + (outputs.attempt2.success ? "✅ Success (" + outputs.attempt2.response_time + "ms)" : "❌ Failed (HTTP " + outputs.attempt2.error_code + ")") : ""}}
          {{outputs.attempt3 ? "3. Retry 2: " + (outputs.attempt3.success ? "✅ Success (" + outputs.attempt3.response_time + "ms)" : "❌ Failed (HTTP " + outputs.attempt3.error_code + ")") : ""}}
          {{outputs.attempt4 ? "4. Final: " + (outputs.attempt4.success ? "✅ Success (" + outputs.attempt4.response_time + "ms)" : "❌ Failed (HTTP " + outputs.attempt4.error_code + ")") : ""}}
          
          Final Result: {{
            outputs.attempt1.success ? "✅ Success on first attempt" :
            outputs.attempt2.success ? "✅ Success on retry 1" :
            outputs.attempt3.success ? "✅ Success on retry 2" :
            outputs.attempt4.success ? "✅ Success on final attempt" :
            "❌ All attempts failed"
          }}
          
          {{
            outputs.attempt1.success ? "" :
            outputs.attempt2.success ? "Service recovered after transient failure" :
            outputs.attempt3.success ? "Service required multiple retries" :
            outputs.attempt4.success ? "Service barely recoverable" :
            "Service appears to be down"
          }}
```

### Circuit Breaker Pattern

Implement circuit breaker to prevent cascading failures:

```yaml
name: Circuit Breaker Pattern
description: Implement circuit breaker for fault isolation

env:
  MONITORED_SERVICE_URL: https://api.monitored.service.com
  CIRCUIT_BREAKER_THRESHOLD: 5
  CIRCUIT_RECOVERY_TIME: 300  # 5 minutes

jobs:
  circuit-breaker-check:
    name: Circuit Breaker Health Check
    steps:
      # Check current circuit breaker state
      - name: Check Circuit Breaker Status
        id: circuit-status
        action: http
        with:
          url: "{{env.MONITORING_API_URL}}/circuit-breaker/{{env.SERVICE_NAME}}"
        test: res.status == 200
        outputs:
          circuit_state: res.json.state
          failure_count: res.json.failure_count
          last_failure_time: res.json.last_failure_time
          last_success_time: res.json.last_success_time

      # Evaluate circuit breaker state
      - name: Circuit Breaker Decision
        id: decision
        echo: "Evaluating circuit breaker state"
        outputs:
          # Circuit is open if too many recent failures
          circuit_open: "{{outputs.circuit-status.failure_count >= env.CIRCUIT_BREAKER_THRESHOLD}}"
          # Allow probe if circuit has been open long enough
          time_since_failure: "{{unixtime() - outputs.circuit-status.last_failure_time}}"
          should_probe: "{{(unixtime() - outputs.circuit-status.last_failure_time) > env.CIRCUIT_RECOVERY_TIME}}"

  service-test:
    name: Service Test with Circuit Breaker
    needs: [circuit-breaker-check]
    steps:
      # Normal operation when circuit is closed
      - name: Normal Service Test
        if: "!outputs.circuit-breaker-check.circuit_open"
        id: normal-test
        action: http
        with:
          url: "{{env.MONITORED_SERVICE_URL}}/health"
          timeout: 10s
        test: res.status == 200
        continue_on_error: true
        outputs:
          test_successful: res.status == 200
          response_time: res.time
          error_code: res.status != 200 ? res.status : null

      # Probe test when circuit is open but recovery time has passed
      - name: Circuit Recovery Probe
        if: outputs.circuit-breaker-check.circuit_open && outputs.circuit-breaker-check.should_probe
        id: probe-test
        action: http
        with:
          url: "{{env.MONITORED_SERVICE_URL}}/ping"  # Lighter probe
          timeout: 5s
        test: res.status == 200
        continue_on_error: true
        outputs:
          probe_successful: res.status == 200
          response_time: res.time

      # Update circuit breaker state
      - name: Update Circuit Breaker
        action: http
        with:
          url: "{{env.MONITORING_API_URL}}/circuit-breaker/{{env.SERVICE_NAME}}/update"
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "test_result": {{
                outputs.normal-test ? outputs.normal-test.test_successful :
                outputs.probe-test ? outputs.probe-test.probe_successful : false
              }},
              "response_time": {{
                outputs.normal-test ? outputs.normal-test.response_time :
                outputs.probe-test ? outputs.probe-test.response_time : null
              }},
              "timestamp": {{unixtime()}}
            }
        test: res.status == 200
        continue_on_error: true

      - name: Circuit Breaker Status Report
        echo: |
          ⚡ Circuit Breaker Status Report:
          
          Previous State:
          Circuit State: {{outputs.circuit-breaker-check.circuit_state}}
          Failure Count: {{outputs.circuit-breaker-check.failure_count}}
          Time Since Last Failure: {{outputs.circuit-breaker-check.time_since_failure}} seconds
          
          Current Test:
          {{outputs.normal-test ? "Normal Test: " + (outputs.normal-test.test_successful ? "✅ Passed" : "❌ Failed (HTTP " + outputs.normal-test.error_code + ")") : ""}}
          {{outputs.probe-test ? "Recovery Probe: " + (outputs.probe-test.probe_successful ? "✅ Passed" : "❌ Failed") : ""}}
          {{outputs.circuit-breaker-check.circuit_open && !outputs.circuit-breaker-check.should_probe ? "⏸️ Circuit Open - Skipping test (recovery time not reached)" : ""}}
          
          Circuit Action: {{
            outputs.normal-test.test_successful ? "✅ Circuit remains closed" :
            outputs.probe-test.probe_successful ? "🟢 Circuit should close (service recovered)" :
            outputs.probe-test && !outputs.probe-test.probe_successful ? "🔴 Circuit remains open (service still failing)" :
            outputs.normal-test && !outputs.normal-test.test_successful ? "🔴 Circuit should open (service failing)" :
            "⏸️ No test performed"
          }}
```

## Error Recovery Strategies

### Self-Healing Workflows

Implement workflows that can automatically recover from failures:

```yaml
name: Self-Healing Service Monitor
description: Monitor services and automatically attempt recovery

env:
  SERVICE_NAME: user-service
  SERVICE_HEALTH_URL: https://user-service.yourcompany.com/health
  ADMIN_API_URL: https://admin.yourcompany.com/api
  RECOVERY_ATTEMPTS: 3

jobs:
  health-monitoring:
    name: Health Monitoring and Recovery
    steps:
      # Step 1: Check service health
      - name: Service Health Check
        id: health-check
        action: http
        with:
          url: "{{env.SERVICE_HEALTH_URL}}"
          timeout: 30s
        test: res.status == 200
        continue_on_error: true
        outputs:
          healthy: res.status == 200
          status_code: res.status
          response_time: res.time
          error_details: res.status != 200 ? res.text : null

      # Step 2: Detailed diagnostics if unhealthy
      - name: Service Diagnostics
        if: "!outputs.health-check.healthy"
        id: diagnostics
        action: http
        with:
          url: "{{env.SERVICE_HEALTH_URL}}/diagnostics"
          timeout: 45s
        test: res.status == 200
        continue_on_error: true
        outputs:
          diagnostics_available: res.status == 200
          memory_usage: res.json.memory_usage_percent
          cpu_usage: res.json.cpu_usage_percent
          active_connections: res.json.active_connections
          error_rate: res.json.error_rate_1min

  automated-recovery:
    name: Automated Recovery Procedures
    needs: [health-monitoring]
    if: jobs.health-monitoring.failed
    steps:
      # Recovery Attempt 1: Graceful restart
      - name: Graceful Service Restart
        id: restart-attempt-1
        action: http
        with:
          url: "{{env.ADMIN_API_URL}}/services/{{env.SERVICE_NAME}}/restart"
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "restart_type": "graceful",
              "drain_connections": true,
              "timeout_seconds": 60
            }
        test: res.status == 200
        continue_on_error: true
        outputs:
          restart_initiated: res.status == 200
          restart_id: res.json.restart_id

      # Wait and verify first restart
      - name: Verify Graceful Restart
        if: outputs.restart-attempt-1.restart_initiated
        action: http
        with:
          url: "{{env.SERVICE_HEALTH_URL}}"
          timeout: 60s
        test: res.status == 200
        continue_on_error: true
        outputs:
          restart_successful: res.status == 200

      # Recovery Attempt 2: Force restart if graceful failed
      - name: Force Service Restart
        if: outputs.restart-attempt-1.restart_initiated && !outputs.restart_successful
        id: restart-attempt-2
        action: http
        with:
          url: "{{env.ADMIN_API_URL}}/services/{{env.SERVICE_NAME}}/restart"
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "restart_type": "force",
              "timeout_seconds": 30
            }
        test: res.status == 200
        continue_on_error: true
        outputs:
          force_restart_initiated: res.status == 200

      # Verify force restart
      - name: Verify Force Restart
        if: outputs.restart-attempt-2.force_restart_initiated
        action: http
        with:
          url: "{{env.SERVICE_HEALTH_URL}}"
          timeout: 60s
        test: res.status == 200
        continue_on_error: true
        outputs:
          force_restart_successful: res.status == 200

      # Recovery Attempt 3: Scale up new instances
      - name: Scale Up Service
        if: "!outputs.restart_successful && !outputs.force_restart_successful"
        id: scale-up
        action: http
        with:
          url: "{{env.ADMIN_API_URL}}/services/{{env.SERVICE_NAME}}/scale"
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "action": "scale_up",
              "additional_instances": 2,
              "health_check_grace_period": 120
            }
        test: res.status == 200
        continue_on_error: true
        outputs:
          scale_up_initiated: res.status == 200

      # Final health check
      - name: Final Health Verification
        action: http
        with:
          url: "{{env.SERVICE_HEALTH_URL}}"
          timeout: 120s
        test: res.status == 200
        continue_on_error: true
        outputs:
          final_health_status: res.status == 200

  recovery-reporting:
    name: Recovery Status Reporting
    needs: [health-monitoring, automated-recovery]
    steps:
      - name: Recovery Status Report
        echo: |
          🏥 Service Recovery Report for {{env.SERVICE_NAME}}:
          ================================================
          
          INITIAL HEALTH CHECK:
          Status: {{outputs.health-monitoring.healthy ? "✅ Healthy" : "❌ Unhealthy (HTTP " + outputs.health-monitoring.status_code + ")"}}
          Response Time: {{outputs.health-monitoring.response_time}}ms
          {{outputs.health-monitoring.error_details ? "Error Details: " + outputs.health-monitoring.error_details : ""}}
          
          {{outputs.health-monitoring.diagnostics_available ? "DIAGNOSTICS:" : ""}}
          {{outputs.health-monitoring.diagnostics_available ? "Memory Usage: " + outputs.health-monitoring.memory_usage + "%" : ""}}
          {{outputs.health-monitoring.diagnostics_available ? "CPU Usage: " + outputs.health-monitoring.cpu_usage + "%" : ""}}
          {{outputs.health-monitoring.diagnostics_available ? "Active Connections: " + outputs.health-monitoring.active_connections : ""}}
          {{outputs.health-monitoring.diagnostics_available ? "Error Rate: " + outputs.health-monitoring.error_rate + "/min" : ""}}
          
          RECOVERY ACTIONS:
          {{outputs.automated-recovery.restart_initiated ? "1. Graceful Restart: " + (outputs.automated-recovery.restart_successful ? "✅ Successful" : "❌ Failed") : "1. Graceful Restart: ⏸️ Not attempted"}}
          {{outputs.automated-recovery.force_restart_initiated ? "2. Force Restart: " + (outputs.automated-recovery.force_restart_successful ? "✅ Successful" : "❌ Failed") : "2. Force Restart: ⏸️ Not attempted"}}
          {{outputs.automated-recovery.scale_up_initiated ? "3. Scale Up: ✅ Initiated" : "3. Scale Up: ⏸️ Not attempted"}}
          
          FINAL STATUS:
          Service Health: {{outputs.automated-recovery.final_health_status ? "✅ Healthy" : "❌ Still Unhealthy"}}
          
          RECOVERY RESULT: {{
            outputs.health-monitoring.healthy ? "ℹ️ No recovery needed - service was healthy" :
            outputs.automated-recovery.restart_successful ? "🟢 Recovered via graceful restart" :
            outputs.automated-recovery.force_restart_successful ? "🟡 Recovered via force restart" :
            outputs.automated-recovery.final_health_status ? "🟢 Recovered via scaling" :
            "🔴 Recovery failed - manual intervention required"
          }}
          
          {{!outputs.automated-recovery.final_health_status && !outputs.health-monitoring.healthy ? "🚨 ALERT: Service recovery failed - escalating to on-call team" : ""}}

      # Escalation notification if recovery failed
      - name: Escalation Alert
        if: "!outputs.health-monitoring.healthy && !outputs.automated-recovery.final_health_status"
        action: smtp
        with:
          host: "{{env.SMTP_HOST}}"
          port: 587
          username: "{{env.SMTP_USERNAME}}"
          password: "{{env.SMTP_PASSWORD}}"
          from: "alerts@yourcompany.com"
          to: ["oncall@yourcompany.com", "devops@yourcompany.com"]
          subject: "🚨 CRITICAL: Service Recovery Failed - {{env.SERVICE_NAME}}"
          body: |
            CRITICAL SERVICE RECOVERY FAILURE
            =================================
            
            Service: {{env.SERVICE_NAME}}
            Time: {{unixtime()}}
            Environment: {{env.ENVIRONMENT}}
            
            Initial Problem:
            - Health Check: Failed (HTTP {{outputs.health-monitoring.status_code}})
            - Response Time: {{outputs.health-monitoring.response_time}}ms
            
            Recovery Attempts:
            {{outputs.automated-recovery.restart_initiated ? "- Graceful Restart: " + (outputs.automated-recovery.restart_successful ? "Success" : "Failed") : "- Graceful Restart: Not attempted"}}
            {{outputs.automated-recovery.force_restart_initiated ? "- Force Restart: " + (outputs.automated-recovery.force_restart_successful ? "Success" : "Failed") : "- Force Restart: Not attempted"}}
            {{outputs.automated-recovery.scale_up_initiated ? "- Scale Up: Initiated" : "- Scale Up: Not attempted"}}
            
            Current Status: Service remains unhealthy
            
            MANUAL INTERVENTION REQUIRED
            
            Please investigate immediately:
            1. Check service logs
            2. Verify infrastructure status  
            3. Consider emergency rollback
            4. Update incident status
            
            Dashboard: {{env.DASHBOARD_URL}}
            Runbook: {{env.RUNBOOK_URL}}
```

## Comprehensive Error Context

### Error Information Collection

Collect comprehensive error information for debugging:

```yaml
name: Comprehensive Error Context Collection
description: Collect detailed error information for effective debugging

env:
  API_BASE_URL: https://api.yourservice.com
  CORRELATION_ID: "{{random_str(32)}}"

jobs:
  error-context-collection:
    name: Error Context Collection
    steps:
      - name: API Test with Error Context
        id: api-test
        action: http
        with:
          url: "{{env.API_BASE_URL}}/complex-operation"
          method: POST
          headers:
            Content-Type: "application/json"
            X-Correlation-ID: "{{env.CORRELATION_ID}}"
            Authorization: "Bearer {{env.API_TOKEN}}"
          body: |
            {
              "operation": "test_operation",
              "parameters": {
                "user_id": {{env.TEST_USER_ID}},
                "data_size": "large",
                "timeout": 30
              },
              "metadata": {
                "test_run_id": "{{env.CORRELATION_ID}}",
                "timestamp": {{unixtime()}}
              }
            }
        test: res.status == 200 && res.json.success == true
        continue_on_error: true
        outputs:
          # Success indicators
          operation_successful: res.status == 200 && res.json.success == true
          
          # Response metadata
          status_code: res.status
          response_time: res.time
          response_size: res.body_size
          content_type: res.headers["content-type"]
          
          # Error context (only populated on failure)
          error_message: res.status != 200 ? res.json.error.message : null
          error_code: res.status != 200 ? res.json.error.code : null
          error_details: res.status != 200 ? res.json.error.details : null
          trace_id: res.headers["x-trace-id"]
          request_id: res.headers["x-request-id"]
          
          # Performance context
          server_response_time: res.headers["x-response-time"]
          database_time: res.json.debug ? res.json.debug.database_time_ms : null
          cache_hit: res.json.debug ? res.json.debug.cache_hit : null
          
          # Business context
          affected_user: res.json.error ? res.json.error.affected_user : null
          operation_id: res.json.operation_id
          retry_after: res.headers["retry-after"]

      - name: Error Analysis and Enrichment
        if: "!outputs.api-test.operation_successful"
        id: error-analysis
        echo: "Analyzing error context"
        outputs:
          # Classify error type
          error_category: |
            {{outputs.api-test.status_code >= 500 ? "server_error" :
              outputs.api-test.status_code == 429 ? "rate_limit" :
              outputs.api-test.status_code >= 400 && outputs.api-test.status_code < 500 ? "client_error" :
              outputs.api-test.status_code == 0 ? "network_error" : "unknown"}}
          
          # Determine severity
          severity_level: |
            {{outputs.api-test.status_code >= 500 ? "high" :
              outputs.api-test.status_code == 429 ? "medium" :
              outputs.api-test.status_code >= 400 && outputs.api-test.status_code < 500 ? "low" :
              "critical"}}
          
          # Generate troubleshooting hints
          troubleshooting_hints: |
            {{outputs.api-test.status_code == 401 ? "Check authentication token expiry and permissions" :
              outputs.api-test.status_code == 403 ? "Verify user has required permissions for this operation" :
              outputs.api-test.status_code == 404 ? "Confirm API endpoint exists and user/resource exists" :
              outputs.api-test.status_code == 409 ? "Resource conflict - check for duplicate operations" :
              outputs.api-test.status_code == 429 ? "Rate limit exceeded - implement backoff or check quota" :
              outputs.api-test.status_code >= 500 ? "Server error - check application logs and infrastructure" :
              "Network or timeout issue - verify connectivity and service availability"}}
          
          # Context for debugging
          debug_context: |
            Correlation ID: {{env.CORRELATION_ID}}
            Test User ID: {{env.TEST_USER_ID}}
            Request Timestamp: {{unixtime()}}
            Environment: {{env.ENVIRONMENT}}

      - name: Detailed Error Report
        if: "!outputs.api-test.operation_successful"
        echo: |
          🔍 Comprehensive Error Analysis Report
          =====================================
          
          ERROR OVERVIEW:
          Correlation ID: {{env.CORRELATION_ID}}
          Error Category: {{outputs.error-analysis.error_category}}
          Severity Level: {{outputs.error-analysis.severity_level}}
          Timestamp: {{unixtime()}}
          
          REQUEST DETAILS:
          URL: {{env.API_BASE_URL}}/complex-operation
          Method: POST
          User ID: {{env.TEST_USER_ID}}
          Content Type: {{outputs.api-test.content_type}}
          
          RESPONSE DETAILS:
          Status Code: {{outputs.api-test.status_code}}
          Response Time: {{outputs.api-test.response_time}}ms
          Response Size: {{outputs.api-test.response_size}} bytes
          Server Response Time: {{outputs.api-test.server_response_time}}ms
          
          ERROR INFORMATION:
          Error Code: {{outputs.api-test.error_code}}
          Error Message: {{outputs.api-test.error_message}}
          Error Details: {{outputs.api-test.error_details}}
          
          TRACING INFORMATION:
          Trace ID: {{outputs.api-test.trace_id}}
          Request ID: {{outputs.api-test.request_id}}
          Operation ID: {{outputs.api-test.operation_id}}
          
          PERFORMANCE CONTEXT:
          {{outputs.api-test.database_time ? "Database Time: " + outputs.api-test.database_time + "ms" : ""}}
          {{outputs.api-test.cache_hit ? "Cache Hit: " + outputs.api-test.cache_hit : ""}}
          {{outputs.api-test.retry_after ? "Retry After: " + outputs.api-test.retry_after + " seconds" : ""}}
          
          BUSINESS CONTEXT:
          {{outputs.api-test.affected_user ? "Affected User: " + outputs.api-test.affected_user : ""}}
          
          TROUBLESHOOTING:
          {{outputs.error-analysis.troubleshooting_hints}}
          
          DEBUG CONTEXT:
          {{outputs.error-analysis.debug_context}}
          
          NEXT STEPS:
          1. Review application logs with Trace ID: {{outputs.api-test.trace_id}}
          2. Check infrastructure metrics around {{unixtime()}}
          3. Validate request parameters and authentication
          {{outputs.api-test.retry_after ? "4. Retry after " + outputs.api-test.retry_after + " seconds" : ""}}
          5. Escalate to development team if issue persists

      - name: Success Report
        if: outputs.api-test.operation_successful
        echo: |
          ✅ Operation Completed Successfully
          
          Correlation ID: {{env.CORRELATION_ID}}
          Response Time: {{outputs.api-test.response_time}}ms
          Operation ID: {{outputs.api-test.operation_id}}
          
          Performance Metrics:
          Server Response Time: {{outputs.api-test.server_response_time}}ms
          {{outputs.api-test.database_time ? "Database Time: " + outputs.api-test.database_time + "ms" : ""}}
          {{outputs.api-test.cache_hit ? "Cache Hit: " + outputs.api-test.cache_hit : ""}}
```

## Best Practices

### 1. Error Classification

```yaml
# Good: Classify errors by type and severity
outputs:
  error_type: |
    {{res.status >= 500 ? "server_error" :
      res.status == 429 ? "rate_limit" :
      res.status >= 400 ? "client_error" : "network_error"}}
  
  severity: |
    {{res.status >= 500 ? "critical" :
      res.status == 429 ? "warning" : "error"}}
```

### 2. Contextual Information

```yaml
# Good: Capture comprehensive context
outputs:
  error_context: |
    Request ID: {{res.headers["x-request-id"]}}
    Timestamp: {{unixtime()}}
    User: {{env.TEST_USER_ID}}
    Operation: {{operation_name}}
```

### 3. Recovery Strategy Selection

```yaml
# Good: Choose recovery strategy based on error type
- name: Recovery Strategy
  if: error_detected
  echo: |
    Recovery Strategy: {{
      error_type == "rate_limit" ? "Wait and retry" :
      error_type == "server_error" ? "Switch to backup service" :
      error_type == "client_error" ? "Fix request and retry" :
      "Investigate and escalate"
    }}
```

### 4. Progressive Error Handling

```yaml
# Good: Progressive error handling
jobs:
  quick-retry:      # Try immediate retry
  fallback-service: # Try alternative service
  cache-fallback:   # Use cached data
  manual-escalation: # Alert humans
```

## Common Error Scenarios

### Network Connectivity Issues

```yaml
- name: Network Connectivity Test
  action: http
  with:
    url: "{{env.EXTERNAL_SERVICE_URL}}/ping"
    timeout: 5s
  test: res.status == 200
  continue_on_error: true
  outputs:
    connectivity_ok: res.status == 200
    network_error: res.status == 0
```

### Authentication Failures

```yaml
- name: Authentication Error Handler
  if: res.status == 401
  echo: |
    Authentication failed:
    1. Check token expiry
    2. Verify credentials
    3. Refresh authentication
```

### Rate Limiting

```yaml
- name: Rate Limit Handler
  if: res.status == 429
  echo: |
    Rate limit exceeded:
    Retry after: {{res.headers["retry-after"]}} seconds
    Current quota: {{res.headers["x-rate-limit-remaining"]}}
```

## What's Next?

Now that you can handle errors effectively, explore:

- **[Performance Testing](../performance-testing/)** - Test system performance and scalability
- **[Environment Management](../environment-management/)** - Manage configurations across environments
- **[Monitoring Workflows](../monitoring-workflows/)** - Build comprehensive monitoring systems

Error handling is your safety net. Master these patterns to build workflows that gracefully handle the unexpected and recover automatically when possible.