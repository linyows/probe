# エラーハンドリング

エラーハンドリングは、失敗や予期せぬ状況を適切に管理し、回復力のあるワークフローを構築するために重要です。このガイドではエラーハンドリング戦略、復旧パターン、耐障害性のある自動化を構築するための技術について詳しく説明します。

## エラーハンドリングの基礎

Probe は、ワークフローの異なるレベルでエラーを処理するためのいくつかのメカニズムを提供します：

1. **ステップレベル**: 個々のステップが失敗にどう対応するかを制御
2. **ジョブレベル**: ジョブの失敗動作と復旧を管理
3. **ワークフローレベル**: 全体的なワークフロー失敗シナリオを処理
4. **条件付き実行**: 成功/失敗状態に基づいて実行をルーティング

### Probe でのエラータイプ

Probe は複数の種類のエラーを認識します：

```yaml
# テスト失敗 - アサーションが失敗
- name: API Health Check
  uses: http
  with:
    url: "{{vars.API_URL}}/health"
  test: res.code == 200  # これが失敗する可能性がある

# アクション失敗 - HTTP タイムアウト、接続エラーなど
- name: Timeout Example
  uses: http
  with:
    url: "{{vars.SLOW_SERVICE}}"
    timeout: 5s  # タイムアウトする可能性がある

# 設定エラー - 無効な URL、環境変数の不足
- name: Configuration Error
  uses: http
  with:
    url: "{{vars.MISSING_VAR}}/endpoint"  # 未定義の場合がある
```

## ステップレベルエラーハンドリング

### エラー時継続

ステップが失敗したときにワークフロー実行を継続するかどうかを制御します：

```yaml
steps:
  - name: Critical Database Check
    uses: http
    with:
      url: "{{vars.DB_API}}/health"
    test: res.code == 200
    continue_on_error: false  # デフォルト: 失敗時にワークフローを停止

  - name: Optional Analytics Update
    uses: http
    with:
      url: "{{vars.ANALYTICS_API}}/update"
    test: res.code == 200
    continue_on_error: true   # これが失敗しても継続

  - name: This runs only if analytics succeeds
    if: steps.previous.success
    echo: "Analytics updated successfully"

  - name: This runs regardless of analytics result
    echo: "Workflow continues..."
```

### グレースフル・デグラデーション

重要でない障害を適切に処理します：

```yaml
jobs:
  monitoring-with-fallbacks:
    name: Monitoring with Graceful Degradation
    steps:
      - name: Primary Service Check
        id: primary
        uses: http
        with:
          url: "{{vars.PRIMARY_SERVICE_URL}}/health"
        test: res.code == 200
        continue_on_error: true
        outputs:
          primary_healthy: res.code == 200
          primary_response_time: res.time

      - name: Secondary Service Check
        if: "!outputs.primary.primary_healthy"
        id: secondary
        uses: http
        with:
          url: "{{vars.SECONDARY_SERVICE_URL}}/health"
        test: res.code == 200
        continue_on_error: true
        outputs:
          secondary_healthy: res.code == 200
          secondary_response_time: res.time

      - name: Cache Service Check
        id: cache
        uses: http
        with:
          url: "{{vars.CACHE_SERVICE_URL}}/health"
        test: res.code == 200
        continue_on_error: true
        outputs:
          cache_healthy: res.code == 200

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

## ジョブレベルエラーハンドリング

### ジョブ依存関係と失敗の伝播

ジョブの失敗が依存ジョブにどう影響するかを制御します：

```yaml
jobs:
  infrastructure-check:
    name: Infrastructure Validation
    steps:
      - name: Database Connectivity
        uses: http
        with:
          url: "{{vars.DB_URL}}/ping"
        test: res.code == 200

      - name: Message Queue Health
        uses: http
        with:
          url: "{{vars.QUEUE_URL}}/health"
        test: res.code == 200

  application-test:
    name: Application Tests
    needs: [infrastructure-check]  # インフラが正常な場合のみ実行
    steps:
      - name: API Tests
        uses: http
        with:
          url: "{{vars.API_URL}}/test"
        test: res.code == 200

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

### 条件付きジョブ実行

失敗シナリオに基づいて異なるジョブを実行します：

```yaml
jobs:
  health-check:
    name: Primary Health Check
    steps:
      - name: Service Health Check
        id: health
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/health"
        test: res.code == 200
        continue_on_error: true
        outputs:
          service_healthy: res.code == 200
          error_details: res.code != 200 ? res.body : null

  recovery-procedures:
    name: Recovery Procedures
    if: jobs.health-check.failed
    steps:
      - name: Attempt Service Restart
        id: restart
        uses: http
        with:
          url: "{{vars.ADMIN_API}}/restart"
          method: POST
        test: res.code == 200
        continue_on_error: true
        outputs:
          restart_successful: res.code == 200

      - name: Verify Recovery
        if: outputs.restart.restart_successful
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/health"
        test: res.code == 200
        outputs:
          recovery_successful: res.code == 200

  escalation:
    name: Escalation Procedures
    if: jobs.recovery-procedures.executed && jobs.recovery-procedures.failed
    steps:
      - name: Critical Alert
        uses: smtp
        with:
          host: "{{vars.SMTP_HOST}}"
          port: 587
          username: "{{vars.SMTP_USER}}"
          password: "{{vars.SMTP_PASS}}"
          from: "alerts@company.com"
          to: ["ops-team@company.com", "management@company.com"]
          subject: "CRITICAL: Service Down - Manual Intervention Required"
          body: |
            CRITICAL ALERT: Service Recovery Failed
            
            Service: {{vars.SERVICE_NAME}}
            Health Check Failed: {{jobs.health-check.failed}}
            Automatic Recovery Attempted: {{jobs.recovery-procedures.executed}}
            Recovery Successful: {{jobs.recovery-procedures.success}}
            
            Error Details: {{outputs.health-check.error_details}}
            
            Manual intervention is required immediately.
            
            Time: {{unixtime()}}
            Environment: {{vars.NODE_ENV}}

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

## リトライと回復力パターン

### フォールバック付き暗黙的リトライ

条件付き実行を使用してリトライロジックを実装します：

```yaml
jobs:
  resilient-api-test:
    name: Resilient API Testing
    steps:
      - name: Primary Attempt
        id: attempt1
        uses: http
        with:
          url: "{{vars.API_URL}}/endpoint"
          timeout: 10s
        test: res.code == 200
        continue_on_error: true
        outputs:
          success: res.code == 200
          response_time: res.time

      - name: Retry After Brief Delay
        if: "!outputs.attempt1.success"
        id: attempt2
        uses: http
        with:
          url: "{{vars.API_URL}}/endpoint"
          timeout: 15s
        test: res.code == 200
        continue_on_error: true
        outputs:
          success: res.code == 200
          response_time: res.time

      - name: Final Attempt with Extended Timeout
        if: "!outputs.attempt1.success && !outputs.attempt2.success"
        id: attempt3
        uses: http
        with:
          url: "{{vars.API_URL}}/endpoint"
          timeout: 30s
        test: res.code == 200
        continue_on_error: true
        outputs:
          success: res.code == 200
          response_time: res.time

      - name: Fallback to Alternative Endpoint
        if: "!outputs.attempt1.success && !outputs.attempt2.success && !outputs.attempt3.success"
        id: fallback
        uses: http
        with:
          url: "{{vars.FALLBACK_API_URL}}/endpoint"
          timeout: 20s
        test: res.code == 200
        continue_on_error: true
        outputs:
          success: res.code == 200
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

### サーキットブレーカーパターン

カスケード障害を防ぐためのサーキットブレーカーロジックを実装します：

```yaml
jobs:
  circuit-breaker-test:
    name: Circuit Breaker Pattern
    steps:
      - name: Check Service Health History
        id: health-history
        uses: http
        with:
          url: "{{vars.MONITORING_API}}/service/{{vars.SERVICE_NAME}}/health-history"
        test: res.code == 200
        outputs:
          recent_failures: res.body.json.failures_last_5_minutes
          error_rate: res.body.json.error_rate_percentage
          last_success: res.body.json.last_success_timestamp

      - name: Circuit Breaker Decision
        id: circuit-decision
        echo: "Evaluating circuit breaker state"
        outputs:
          circuit_open: "{{outputs.health-history.error_rate > 50 || outputs.health-history.recent_failures > 10}}"
          should_test: "{{!outputs.health-history.circuit_open || (unixtime() - outputs.health-history.last_success) > 300}}"

      - name: Service Test (Circuit Closed)
        if: "!outputs.circuit-decision.circuit_open"
        id: normal-test
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/test"
        test: res.code == 200
        continue_on_error: true
        outputs:
          test_successful: res.code == 200

      - name: Probe Test (Circuit Open)
        if: outputs.circuit-decision.circuit_open && outputs.circuit-decision.should_test
        id: probe-test
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/health"  # より軽量なプローブ
          timeout: 5s
        test: res.code == 200
        continue_on_error: true
        outputs:
          probe_successful: res.code == 200

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

## エラーコンテキストとデバッグ

### 包括的なエラー情報

デバッグのために詳細なエラーコンテキストをキャプチャします：

```yaml
- name: Detailed Error Capture
  id: api-test
  uses: http
  with:
    url: "{{vars.API_URL}}/complex-operation"
    method: POST
    body: |
      {
        "operation": "test",
        "timestamp": {{unixtime()}},
        "user_id": "{{vars.TEST_USER_ID}}"
      }
  test: res.code == 200 && res.body.json.success == true
  continue_on_error: true
  outputs:
    success: res.code == 200
    status_code: res.code
    response_time: res.time
    response_size: res.body_size
    error_message: res.code != 200 ? res.body : null
    response_headers: res.headers
    partial_response: res.code != 200 ? res.body.substring(0, 500) : null

- name: Error Analysis and Reporting
  if: "!outputs.api-test.success"
  echo: |
    🚨 API Test Failure Analysis:
    
    Request Details:
    - URL: {{vars.API_URL}}/complex-operation
    - Method: POST
    - User ID: {{vars.TEST_USER_ID}}
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

### エラー関連性と追跡

複数のステップとジョブ間でエラーを追跡します：

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
        uses: http
        with:
          url: "{{vars.SERVICE_A_URL}}/test"
          headers:
            X-Correlation-ID: "{{outputs.init.correlation_id}}"
        test: res.code == 200
        continue_on_error: true
        outputs:
          success: res.code == 200
          error_code: res.code != 200 ? res.code : null
          trace_id: res.headers["x-trace-id"]

      - name: Service B Test
        id: service-b
        uses: http
        with:
          url: "{{vars.SERVICE_B_URL}}/test"
          headers:
            X-Correlation-ID: "{{outputs.init.correlation_id}}"
        test: res.code == 200
        continue_on_error: true
        outputs:
          success: res.code == 200
          error_code: res.code != 200 ? res.code : null
          trace_id: res.headers["x-trace-id"]

      - name: Service C Test
        id: service-c
        uses: http
        with:
          url: "{{vars.SERVICE_C_URL}}/test"
          headers:
            X-Correlation-ID: "{{outputs.init.correlation_id}}"
        test: res.code == 200
        continue_on_error: true
        outputs:
          success: res.code == 200
          error_code: res.code != 200 ? res.code : null
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

## エラー回復戦略

### 自動回復手順

一般的な失敗シナリオに対して自動回復を実装します：

```yaml
jobs:
  self-healing-monitor:
    name: Self-Healing Monitoring
    steps:
      - name: Service Health Check
        id: health-check
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/health"
        test: res.code == 200
        continue_on_error: true
        outputs:
          healthy: res.code == 200
          status_code: res.code

      - name: Memory Usage Check
        if: outputs.health-check.healthy
        id: memory-check
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/metrics"
        test: res.code == 200 && res.body.json.memory_usage_percent < 90
        continue_on_error: true
        outputs:
          memory_ok: res.code == 200 && res.body.json.memory_usage_percent < 90
          memory_usage: res.body.json.memory_usage_percent

      - name: Restart Service (Health Failure)
        if: "!outputs.health-check.healthy"
        id: restart-health
        uses: http
        with:
          url: "{{vars.ADMIN_API}}/services/{{vars.SERVICE_NAME}}/restart"
          method: POST
        test: res.code == 200
        continue_on_error: true
        outputs:
          restart_initiated: res.code == 200
          restart_reason: "health_check_failed"

      - name: Restart Service (Memory Issue)
        if: outputs.health-check.healthy && outputs.memory-check && !outputs.memory-check.memory_ok
        id: restart-memory
        uses: http
        with:
          url: "{{vars.ADMIN_API}}/services/{{vars.SERVICE_NAME}}/restart"
          method: POST
        test: res.code == 200
        continue_on_error: true
        outputs:
          restart_initiated: res.code == 200
          restart_reason: "high_memory_usage"

      - name: Wait for Service Recovery
        if: outputs.restart-health.restart_initiated || outputs.restart-memory.restart_initiated
        id: recovery-wait
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/health"
          timeout: 60s
        test: res.code == 200
        continue_on_error: true
        outputs:
          recovery_successful: res.code == 200

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

### 段階的回復テスト

回復中のシステムに過負荷をかけないよう段階的にテストします：

```yaml
jobs:
  gradual-recovery-test:
    name: Gradual Recovery Testing
    steps:
      - name: Basic Connectivity Test
        id: ping
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/ping"
          timeout: 5s
        test: res.code == 200
        continue_on_error: true
        outputs:
          connectivity: res.code == 200

      - name: Health Endpoint Test
        if: outputs.ping.connectivity
        id: health
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/health"
          timeout: 10s
        test: res.code == 200
        continue_on_error: true
        outputs:
          health_ok: res.code == 200

      - name: Light Functional Test
        if: outputs.health.health_ok
        id: light-test
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/api/status"
          timeout: 15s
        test: res.code == 200 && res.body.json.status == "ready"
        continue_on_error: true
        outputs:
          light_test_passed: res.code == 200 && res.body.json.status == "ready"

      - name: Standard Load Test
        if: outputs.light-test.light_test_passed
        id: standard-test
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/api/users"
          timeout: 30s
        test: res.code == 200 && res.body.json.users != null
        continue_on_error: true
        outputs:
          standard_test_passed: res.code == 200

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

## 通知とアラート

### エラー駆動通知

エラーの重要度とコンテキストに基づいて通知を送信します：

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
          
          # エラー重要度計算
          severity_level: |
            {{jobs.health-check.failed ? "CRITICAL" : 
              (jobs.security-scan.failed ? "HIGH" : 
                (jobs.performance-test.failed ? "MEDIUM" : "LOW"))}}

      - name: Critical Alert
        if: outputs.classification.critical_errors
        uses: smtp
        with:
          host: "{{vars.SMTP_HOST}}"
          port: 587
          username: "{{vars.SMTP_USER}}"
          password: "{{vars.SMTP_PASS}}"
          from: "critical-alerts@company.com"
          to: ["oncall@company.com", "management@company.com"]
          subject: "🚨 CRITICAL: {{vars.SERVICE_NAME}} Service Down"
          body: |
            CRITICAL SERVICE ALERT
            
            Service: {{vars.SERVICE_NAME}}
            Environment: {{vars.NODE_ENV}}
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
        uses: smtp
        with:
          host: "{{vars.SMTP_HOST}}"
          port: 587
          username: "{{vars.SMTP_USER}}"
          password: "{{vars.SMTP_PASS}}"
          from: "monitoring@company.com"
          to: ["dev-team@company.com"]
          subject: "⚠️ {{outputs.classification.severity_level}}: {{vars.SERVICE_NAME}} Issues Detected"
          body: |
            Service Monitoring Alert
            
            Service: {{vars.SERVICE_NAME}}
            Environment: {{vars.NODE_ENV}}
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
          
          Service: {{vars.SERVICE_NAME}}
          Environment: {{vars.NODE_ENV}}
          Monitoring Status: All checks passed
          
          No alerts sent - system is healthy.
```

## ベストプラクティス

### 1. フェイルファストと回復力のバランス

```yaml
# 重要パス - フェイルファスト
- name: Database Connection
  uses: http
  with:
    url: "{{vars.DB_URL}}/ping"
  test: res.code == 200
  continue_on_error: false  # データベースが停止している場合はすぐに停止

# 非重要パス - 回復力を持つ
- name: Analytics Tracking
  uses: http
  with:
    url: "{{vars.ANALYTICS_URL}}/track"
  test: res.code == 200
  continue_on_error: true   # 分析が失敗しても継続
```

### 2. エラーコンテキストの保持

```yaml
# 良い例: エラーコンテキストを保持
outputs:
  success: res.code == 200
  error_details: |
    {{res.code != 200 ? 
      "HTTP " + res.code + ": " + res.body.substring(0, 200) : 
      null}}
  debug_info: |
    {{res.code != 200 ? 
      "URL: " + vars.API_URL + ", Time: " + res.time + "ms" : 
      null}}
```

### 3. 段階的応答

```yaml
# 良い例: 異なるエラータイプに対する異なる応答
- name: Error Response Strategy
  if: steps.api-test.failed
  echo: |
    Error Response:
    {{outputs.api-test.status_code == 500 ? "🚨 Server error - escalate immediately" : ""}}
    {{outputs.api-test.status_code == 404 ? "⚠️ Endpoint not found - check configuration" : ""}}
    {{outputs.api-test.status_code == 401 ? "🔐 Authentication failed - refresh credentials" : ""}}
```

### 4. エラー回復ドキュメント

```yaml
# ワークフロー内で回復手順を文書化
- name: Recovery Instructions
  if: jobs.health-check.failed
  echo: |
    🛠️ Recovery Procedures for {{vars.SERVICE_NAME}}:
    
    1. Check service logs: kubectl logs -l app={{vars.SERVICE_NAME}}
    2. Verify configuration: check {{vars.CONFIG_PATH}}
    3. Restart service: kubectl rollout restart deployment/{{vars.SERVICE_NAME}}
    4. Monitor recovery: watch kubectl get pods -l app={{vars.SERVICE_NAME}}
    
    Escalation: If service doesn't recover in 10 minutes, contact on-call team.
```

## 次のステップ

エラーハンドリングを理解したら、以下を探索してください：

1. **[実行モデル](../execution-model/)** - Probe がワークフローを実行する方法を学ぶ
2. **[ファイルマージ](../file-merging/)** - 設定の構成を理解する
3. **[ハウツー](../../how-tos/)** - 実用的なエラーハンドリングパターンを見る

エラーハンドリングはあなたの安全ネットです。これらのパターンをマスターして、予期しないことを適切に処理し、可能な場合は自動的に回復するワークフローを構築しましょう。