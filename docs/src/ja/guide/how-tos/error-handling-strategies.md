# エラーハンドリング戦略

このガイドでは、Probeワークフローで堅牢なエラーハンドリングを実装する方法を説明します。失敗を適切に処理し、回復パターンを実装し、予期しない状況に対処できる耐性のある自動化を構築する方法を学びます。

## 基本的なエラーハンドリングパターン

### フェイルファースト vs 続行

操作の重要度に基づいて適切なエラーハンドリング戦略を選択します：

```yaml
name: Error Handling Strategy Examples
description: 異なるエラーハンドリングアプローチを実演

vars:
  CRITICAL_SERVICE_URL: https://critical.yourcompany.com
  OPTIONAL_SERVICE_URL: https://optional.yourcompany.com
  NOTIFICATION_URL: https://notifications.yourcompany.com

jobs:
- name: Critical Operations
  steps:
    # 重要な操作では高速失敗
    - name: Critical Database Check
      uses: http
      with:
        url: "{{vars.CRITICAL_SERVICE_URL}}/database/health"
      test: res.code == 200
      continue_on_error: false  # デフォルト: 失敗時にワークフローを停止
      outputs:
        database_healthy: res.code == 200

    # このステップはデータベースチェックが成功した場合のみ実行
    - name: Critical API Check
      uses: http
      with:
        url: "{{vars.CRITICAL_SERVICE_URL}}/api/health"
      test: res.code == 200
      outputs:
        api_healthy: res.code == 200

- name: Resilient Operations
  steps:
    # オプショナルサービスではエラー時に続行
    - name: Optional Analytics Service
      uses: http
      with:
        url: "{{vars.OPTIONAL_SERVICE_URL}}/analytics"
      test: res.code == 200
      continue_on_error: true   # これが失敗しても続行
      outputs:
        analytics_available: res.code == 200
        analytics_error: res.code != 200 ? res.code : null

    # このステップは前のステップに関係なく常に実行
    - name: Optional Notification Service
      uses: http
      with:
        url: "{{vars.NOTIFICATION_URL}}/health"
      test: res.code == 200
      continue_on_error: true
      outputs:
        notifications_available: res.code == 200

    # サービス可用性に基づく条件付きロジック
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

### グレースフルデグラデーション

プライマリサービスが失敗した際のフォールバックメカニズムを実装します：

```yaml
name: Graceful Degradation Pattern
description: フォールバックサービスとグレースフルデグラデーションを実装

vars:
  PRIMARY_API_URL: https://primary.api.yourcompany.com
  SECONDARY_API_URL: https://secondary.api.yourcompany.com
  CACHE_API_URL: https://cache.yourcompany.com
  FALLBACK_API_URL: https://fallback.api.yourcompany.com

jobs:
- name: Service with Multiple Fallbacks
  steps:
    # まずプライマリサービスを試行
    - name: Primary Service Attempt
      id: primary
      uses: http
      with:
        url: "{{vars.PRIMARY_API_URL}}/data"
        timeout: 10s
      test: res.code == 200 && res.time < 5000
      continue_on_error: true
      outputs:
        success: res.code == 200 && res.time < 5000
        response_time: res.time
        data: res.body.json

    # プライマリが失敗または遅い場合はセカンダリサービスを試行
    - name: Secondary Service Attempt
      if: "!outputs.primary.success"
      id: secondary
      uses: http
      with:
        url: "{{vars.SECONDARY_API_URL}}/data"
        timeout: 15s
      test: res.code == 200
      continue_on_error: true
      outputs:
        success: res.code == 200
        response_time: res.time
        data: res.body.json

    # プライマリとセカンダリの両方が失敗した場合はキャッシュを試行
    - name: Cache Fallback
      if: "!outputs.primary.success && !outputs.secondary.success"
      id: cache
      uses: http
      with:
        url: "{{vars.CACHE_API_URL}}/cached-data"
        timeout: 5s
      test: res.code == 200
      continue_on_error: true
      outputs:
        success: res.code == 200
        response_time: res.time
        data: res.body.json
        cached_data: true

    # 静的データへの最終フォールバック
    - name: Static Fallback
      if: "!outputs.primary.success && !outputs.secondary.success && !outputs.cache.success"
      id: fallback
      uses: http
      with:
        url: "{{vars.FALLBACK_API_URL}}/static-data"
      test: res.code == 200
      continue_on_error: true
      outputs:
        success: res.code == 200
        response_time: res.time
        data: res.body.json
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

## リトライパターン

### 指数バックオフリトライ

遅延を増加させるリトライロジックを実装します：

```yaml
name: Retry with Exponential Backoff
description: 一時的な失敗に対するリトライパターンを実装

vars:
  UNRELIABLE_SERVICE_URL: https://api.unreliable.service.com
  MAX_RETRIES: 3

jobs:
- name: Exponential Backoff Retry Pattern
  steps:
    # 最初の試行
    - name: Initial Attempt
      id: attempt1
      uses: http
      with:
        url: "{{vars.UNRELIABLE_SERVICE_URL}}/data"
        timeout: 10s
      test: res.code == 200
      continue_on_error: true
      outputs:
        success: res.code == 200
        attempt_number: 1
        response_time: res.time
        error_code: res.code != 200 ? res.code : null

    # 2回目の試行（2秒遅延）
    - name: Retry Attempt 1 (2s delay)
      if: "!outputs.attempt1.success"
      id: attempt2
      uses: http
      with:
        url: "{{vars.UNRELIABLE_SERVICE_URL}}/data"
        timeout: 15s
      test: res.code == 200
      continue_on_error: true
      outputs:
        success: res.code == 200
        attempt_number: 2
        response_time: res.time
        error_code: res.code != 200 ? res.code : null

    # 3回目の試行（4秒遅延）
    - name: Retry Attempt 2 (4s delay)
      if: "!outputs.attempt1.success && !outputs.attempt2.success"
      id: attempt3
      uses: http
      with:
        url: "{{vars.UNRELIABLE_SERVICE_URL}}/data"
        timeout: 20s
      test: res.code == 200
      continue_on_error: true
      outputs:
        success: res.code == 200
        attempt_number: 3
        response_time: res.time
        error_code: res.code != 200 ? res.code : null

    # 最終試行（8秒遅延）
    - name: Final Attempt (8s delay)
      if: "!outputs.attempt1.success && !outputs.attempt2.success && !outputs.attempt3.success"
      id: attempt4
      uses: http
      with:
        url: "{{vars.UNRELIABLE_SERVICE_URL}}/data"
        timeout: 30s
      test: res.code == 200
      continue_on_error: true
      outputs:
        success: res.code == 200
        attempt_number: 4
        response_time: res.time
        error_code: res.code != 200 ? res.code : null

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

### サーキットブレーカーパターン

障害の連鎖を防ぐためのサーキットブレーカーを実装します：

```yaml
name: Circuit Breaker Pattern
description: 障害の分離のためのサーキットブレーカーを実装

vars:
  MONITORED_SERVICE_URL: https://api.monitored.service.com
  CIRCUIT_BREAKER_THRESHOLD: 5
  CIRCUIT_RECOVERY_TIME: 300  # 5分

jobs:
- name: Circuit Breaker Health Check
  steps:
    # 現在のサーキットブレーカー状態をチェック
    - name: Check Circuit Breaker Status
      id: circuit-status
      uses: http
      with:
        url: "{{vars.MONITORING_API_URL}}/circuit-breaker/{{vars.SERVICE_NAME}}"
      test: res.code == 200
      outputs:
        circuit_state: res.body.json.state
        failure_count: res.body.json.failure_count
        last_failure_time: res.body.json.last_failure_time
        last_success_time: res.body.json.last_success_time

    # サーキットブレーカー状態を評価
    - name: Circuit Breaker Decision
      id: decision
      echo: "Evaluating circuit breaker state"
      outputs:
        # 最近の失敗が多すぎる場合はサーキットを開く
        circuit_open: "{{outputs.circuit-status.failure_count >= vars.CIRCUIT_BREAKER_THRESHOLD}}"
        # サーキットが十分な時間開いていればプローブを許可
        time_since_failure: "{{unixtime() - outputs.circuit-status.last_failure_time}}"
        should_probe: "{{(unixtime() - outputs.circuit-status.last_failure_time) > vars.CIRCUIT_RECOVERY_TIME}}"

- name: Service Test with Circuit Breaker
  needs: [circuit-breaker-check]
  steps:
    # サーキットが閉じている場合の通常動作
    - name: Normal Service Test
      if: "!outputs.circuit-breaker-check.circuit_open"
      id: normal-test
      uses: http
      with:
        url: "{{vars.MONITORED_SERVICE_URL}}/health"
        timeout: 10s
      test: res.code == 200
      continue_on_error: true
      outputs:
        test_successful: res.code == 200
        response_time: res.time
        error_code: res.code != 200 ? res.code : null

    # サーキットが開いているが回復時間が経過した場合のプローブテスト
    - name: Circuit Recovery Probe
      if: outputs.circuit-breaker-check.circuit_open && outputs.circuit-breaker-check.should_probe
      id: probe-test
      uses: http
      with:
        url: "{{vars.MONITORED_SERVICE_URL}}/ping"  # より軽いプローブ
        timeout: 5s
      test: res.code == 200
      continue_on_error: true
      outputs:
        probe_successful: res.code == 200
        response_time: res.time

    # サーキットブレーカー状態を更新
    - name: Update Circuit Breaker
      uses: http
      with:
        url: "{{vars.MONITORING_API_URL}}/circuit-breaker/{{vars.SERVICE_NAME}}/update"
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
      test: res.code == 200
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

## エラー回復戦略

### セルフヒーリングワークフロー

失敗から自動的に回復できるワークフローを実装します：

```yaml
name: Self-Healing Service Monitor
description: サービスを監視し、自動的に回復を試行

vars:
  SERVICE_NAME: user-service
  SERVICE_HEALTH_URL: https://user-service.yourcompany.com/health
  ADMIN_API_URL: https://admin.yourcompany.com/api
  RECOVERY_ATTEMPTS: 3

jobs:
- name: Health Monitoring and Recovery
  steps:
    # ステップ1: サービスヘルスチェック
    - name: Service Health Check
      id: health-check
      uses: http
      with:
        url: "{{vars.SERVICE_HEALTH_URL}}"
        timeout: 30s
      test: res.code == 200
      continue_on_error: true
      outputs:
        healthy: res.code == 200
        status_code: res.code
        response_time: res.time
        error_details: res.code != 200 ? res.body.text : null

    # ステップ2: 異常な場合の詳細診断
    - name: Service Diagnostics
      if: "!outputs.health-check.healthy"
      id: diagnostics
      uses: http
      with:
        url: "{{vars.SERVICE_HEALTH_URL}}/diagnostics"
        timeout: 45s
      test: res.code == 200
      continue_on_error: true
      outputs:
        diagnostics_available: res.code == 200
        memory_usage: res.body.json.memory_usage_percent
        cpu_usage: res.body.json.cpu_usage_percent
        active_connections: res.body.json.active_connections
        error_rate: res.body.json.error_rate_1min

- name: Automated Recovery Procedures
  needs: [health-monitoring]
  if: jobs.health-monitoring.failed
  steps:
    # 回復試行1: グレースフル再起動
    - name: Graceful Service Restart
      id: restart-attempt-1
      uses: http
      with:
        url: "{{vars.ADMIN_API_URL}}/services/{{vars.SERVICE_NAME}}/restart"
        method: POST
        headers:
          Content-Type: "application/json"
        body: |
          {
            "restart_type": "graceful",
            "drain_connections": true,
            "timeout_seconds": 60
          }
      test: res.code == 200
      continue_on_error: true
      outputs:
        restart_initiated: res.code == 200
        restart_id: res.body.json.restart_id

    # 最初の再起動を待機・検証
    - name: Verify Graceful Restart
      if: outputs.restart-attempt-1.restart_initiated
      uses: http
      with:
        url: "{{vars.SERVICE_HEALTH_URL}}"
        timeout: 60s
      test: res.code == 200
      continue_on_error: true
      outputs:
        restart_successful: res.code == 200

    # 回復試行2: グレースフルが失敗した場合は強制再起動
    - name: Force Service Restart
      if: outputs.restart-attempt-1.restart_initiated && !outputs.restart_successful
      id: restart-attempt-2
      uses: http
      with:
        url: "{{vars.ADMIN_API_URL}}/services/{{vars.SERVICE_NAME}}/restart"
        method: POST
        headers:
          Content-Type: "application/json"
        body: |
          {
            "restart_type": "force",
            "timeout_seconds": 30
          }
      test: res.code == 200
      continue_on_error: true
      outputs:
        force_restart_initiated: res.code == 200

    # 強制再起動を検証
    - name: Verify Force Restart
      if: outputs.restart-attempt-2.force_restart_initiated
      uses: http
      with:
        url: "{{vars.SERVICE_HEALTH_URL}}"
        timeout: 60s
      test: res.code == 200
      continue_on_error: true
      outputs:
        force_restart_successful: res.code == 200

    # 回復試行3: 新しいインスタンスをスケールアップ
    - name: Scale Up Service
      if: "!outputs.restart_successful && !outputs.force_restart_successful"
      id: scale-up
      uses: http
      with:
        url: "{{vars.ADMIN_API_URL}}/services/{{vars.SERVICE_NAME}}/scale"
        method: POST
        headers:
          Content-Type: "application/json"
        body: |
          {
            "action": "scale_up",
            "additional_instances": 2,
            "health_check_grace_period": 120
          }
      test: res.code == 200
      continue_on_error: true
      outputs:
        scale_up_initiated: res.code == 200

    # 最終ヘルスチェック
    - name: Final Health Verification
      uses: http
      with:
        url: "{{vars.SERVICE_HEALTH_URL}}"
        timeout: 120s
      test: res.code == 200
      continue_on_error: true
      outputs:
        final_health_status: res.code == 200

- name: Recovery Status Reporting
  needs: [health-monitoring, automated-recovery]
  steps:
    - name: Recovery Status Report
      echo: |
        🏥 Service Recovery Report for {{vars.SERVICE_NAME}}:
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

    # 回復が失敗した場合のエスカレーション通知
    - name: Escalation Alert
      if: "!outputs.health-monitoring.healthy && !outputs.automated-recovery.final_health_status"
      uses: smtp
      with:
        host: "{{vars.SMTP_HOST}}"
        port: 587
        username: "{{vars.SMTP_USERNAME}}"
        password: "{{vars.SMTP_PASSWORD}}"
        from: "alerts@yourcompany.com"
        to: ["oncall@yourcompany.com", "devops@yourcompany.com"]
        subject: "🚨 CRITICAL: Service Recovery Failed - {{vars.SERVICE_NAME}}"
        body: |
          CRITICAL SERVICE RECOVERY FAILURE
          =================================
          
          Service: {{vars.SERVICE_NAME}}
          Time: {{unixtime()}}
          Environment: {{vars.ENVIRONMENT}}
          
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
          
          Dashboard: {{vars.DASHBOARD_URL}}
          Runbook: {{vars.RUNBOOK_URL}}
```

## 包括的なエラーコンテキスト

### エラー情報収集

デバッグのための包括的なエラー情報を収集します：

```yaml
name: Comprehensive Error Context Collection
description: 効果的なデバッグのための詳細なエラー情報を収集

vars:
  API_BASE_URL: https://api.yourservice.com
  CORRELATION_ID: "{{random_str(32)}}"

jobs:
- name: Error Context Collection
  steps:
    - name: API Test with Error Context
      id: api-test
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/complex-operation"
        method: POST
        headers:
          Content-Type: "application/json"
          X-Correlation-ID: "{{vars.CORRELATION_ID}}"
          Authorization: "Bearer {{vars.API_TOKEN}}"
        body: |
          {
            "operation": "test_operation",
            "parameters": {
              "user_id": {{vars.TEST_USER_ID}},
              "data_size": "large",
              "timeout": 30
            },
            "metadata": {
              "test_run_id": "{{vars.CORRELATION_ID}}",
              "timestamp": {{unixtime()}}
            }
          }
      test: res.code == 200 && res.body.json.success == true
      continue_on_error: true
      outputs:
        # 成功指標
        operation_successful: res.code == 200 && res.body.json.success == true
        
        # レスポンスメタデータ
        status_code: res.code
        response_time: res.time
        response_size: res.body_size
        content_type: res.headers["content-type"]
        
        # エラーコンテキスト（失敗時のみ入力）
        error_message: res.code != 200 ? res.body.json.error.message : null
        error_code: res.code != 200 ? res.body.json.error.code : null
        error_details: res.code != 200 ? res.body.json.error.details : null
        trace_id: res.headers["x-trace-id"]
        request_id: res.headers["x-request-id"]
        
        # パフォーマンスコンテキスト
        server_response_time: res.headers["x-response-time"]
        database_time: res.body.json.debug ? res.body.json.debug.database_time_ms : null
        cache_hit: res.body.json.debug ? res.body.json.debug.cache_hit : null
        
        # ビジネスコンテキスト
        affected_user: res.body.json.error ? res.body.json.error.affected_user : null
        operation_id: res.body.json.operation_id
        retry_after: res.headers["retry-after"]

    - name: Error Analysis and Enrichment
      if: "!outputs.api-test.operation_successful"
      id: error-analysis
      echo: "Analyzing error context"
      outputs:
        # エラータイプを分類
        error_category: |
          {{outputs.api-test.status_code >= 500 ? "server_error" :
            outputs.api-test.status_code == 429 ? "rate_limit" :
            outputs.api-test.status_code >= 400 && outputs.api-test.status_code < 500 ? "client_error" :
            outputs.api-test.status_code == 0 ? "network_error" : "unknown"}}
        
        # 深刻度を決定
        severity_level: |
          {{outputs.api-test.status_code >= 500 ? "high" :
            outputs.api-test.status_code == 429 ? "medium" :
            outputs.api-test.status_code >= 400 && outputs.api-test.status_code < 500 ? "low" :
            "critical"}}
        
        # トラブルシューティングヒントを生成
        troubleshooting_hints: |
          {{outputs.api-test.status_code == 401 ? "Check authentication token expiry and permissions" :
            outputs.api-test.status_code == 403 ? "Verify user has required permissions for this operation" :
            outputs.api-test.status_code == 404 ? "Confirm API endpoint exists and user/resource exists" :
            outputs.api-test.status_code == 409 ? "Resource conflict - check for duplicate operations" :
            outputs.api-test.status_code == 429 ? "Rate limit exceeded - implement backoff or check quota" :
            outputs.api-test.status_code >= 500 ? "Server error - check application logs and infrastructure" :
            "Network or timeout issue - verify connectivity and service availability"}}
        
        # デバッグ用のコンテキスト
        debug_context: |
          Correlation ID: {{vars.CORRELATION_ID}}
          Test User ID: {{vars.TEST_USER_ID}}
          Request Timestamp: {{unixtime()}}
          Environment: {{vars.ENVIRONMENT}}

    - name: Detailed Error Report
      if: "!outputs.api-test.operation_successful"
      echo: |
        🔍 Comprehensive Error Analysis Report
        =====================================
        
        ERROR OVERVIEW:
        Correlation ID: {{vars.CORRELATION_ID}}
        Error Category: {{outputs.error-analysis.error_category}}
        Severity Level: {{outputs.error-analysis.severity_level}}
        Timestamp: {{unixtime()}}
        
        REQUEST DETAILS:
        URL: {{vars.API_BASE_URL}}/complex-operation
        Method: POST
        User ID: {{vars.TEST_USER_ID}}
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
        
        Correlation ID: {{vars.CORRELATION_ID}}
        Response Time: {{outputs.api-test.response_time}}ms
        Operation ID: {{outputs.api-test.operation_id}}
        
        Performance Metrics:
        Server Response Time: {{outputs.api-test.server_response_time}}ms
        {{outputs.api-test.database_time ? "Database Time: " + outputs.api-test.database_time + "ms" : ""}}
        {{outputs.api-test.cache_hit ? "Cache Hit: " + outputs.api-test.cache_hit : ""}}
```

## ベストプラクティス

### 1. エラー分類

```yaml
# 良い例: タイプと深刻度でエラーを分類
outputs:
  error_type: |
    {{res.code >= 500 ? "server_error" :
      res.code == 429 ? "rate_limit" :
      res.code >= 400 ? "client_error" : "network_error"}}
  
  severity: |
    {{res.code >= 500 ? "critical" :
      res.code == 429 ? "warning" : "error"}}
```

### 2. コンテキスト情報

```yaml
# 良い例: 包括的なコンテキストを取得
outputs:
  error_context: |
    Request ID: {{res.headers["x-request-id"]}}
    Timestamp: {{unixtime()}}
    User: {{vars.TEST_USER_ID}}
    Operation: {{operation_name}}
```

### 3. 回復戦略の選択

```yaml
# 良い例: エラータイプに基づいた回復戦略の選択
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

### 4. 段階的エラーハンドリング

```yaml
# 良い例: 段階的エラーハンドリング
jobs:
  quick-retry:      # 即座にリトライを試行
  fallback-service: # 代替サービスを試行
  cache-fallback:   # キャッシュされたデータを使用
  manual-escalation: # 人間にアラート
```

## 一般的なエラーシナリオ

### ネットワーク接続の問題

```yaml
- name: Network Connectivity Test
  uses: http
  with:
    url: "{{vars.EXTERNAL_SERVICE_URL}}/ping"
    timeout: 5s
  test: res.code == 200
  continue_on_error: true
  outputs:
    connectivity_ok: res.code == 200
    network_error: res.code == 0
```

### 認証失敗

```yaml
- name: Authentication Error Handler
  if: res.code == 401
  echo: |
    Authentication failed:
    1. Check token expiry
    2. Verify credentials
    3. Refresh authentication
```

### レート制限

```yaml
- name: Rate Limit Handler
  if: res.code == 429
  echo: |
    Rate limit exceeded:
    Retry after: {{res.headers["retry-after"]}} seconds
    Current quota: {{res.headers["x-rate-limit-remaining"]}}
```

## 次のステップ

効果的にエラーを処理できるようになったので、次を探索してください：

- **[パフォーマンステスト](../performance-testing/)** - システムのパフォーマンスとスケーラビリティをテスト
- **[環境管理](../environment-management/)** - 環境間での設定を管理
- **[モニタリングワークフロー](../monitoring-workflows/)** - 包括的な監視システムを構築

エラーハンドリングはあなたのセーフティネットです。これらのパターンをマスターして、予期しない事態を適切に処理し、可能な場合は自動的に回復するワークフローを構築しましょう。