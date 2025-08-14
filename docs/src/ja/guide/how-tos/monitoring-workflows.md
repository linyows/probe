# モニタリングワークフロー

このガイドでは、Probeを使用して包括的な監視システムを構築する方法を説明します。サービスヘルスのチェック、パフォーマンスの監視、問題のアラートを行うワークフローの作成方法を学びます。

## 基本的なサービス監視

### シンプルなヘルスチェック

基本的なヘルスチェックワークフローから始めます：

```yaml
name: Basic Service Health Check
description: 必須サービスエンドポイントを監視

vars:
  API_BASE_URL: https://api.yourcompany.com
  HEALTH_ENDPOINT: /health
  TIMEOUT: 30s

jobs:
- name: Service Health Check
  defaults:
    http:
      timeout: "{{vars.TIMEOUT}}"
      headers:
        User-Agent: "Probe Monitor v1.0"
  steps:
    - name: API Health Check
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}{{vars.HEALTH_ENDPOINT}}"
      test: res.code == 200
      outputs:
        api_healthy: res.code == 200
        response_time: res.time
        api_version: res.body.json.version

    - name: Health Status Report
      echo: |
        🏥 Health Check Results:
        
        API Status: {{outputs.api_healthy ? "✅ Healthy" : "❌ Down"}}
        Response Time: {{outputs.response_time}}ms
        API Version: {{outputs.api_version}}
        Timestamp: {{unixtime()}}
```

**使用方法:**
```bash
probe health-check.yml
```

### マルチサービスヘルス監視

複数のサービスを並列で監視します：

```yaml
name: Multi-Service Health Monitor
description: すべての重要なサービスのヘルスをチェック

vars:
  USER_SERVICE_URL: https://users.api.yourcompany.com
  ORDER_SERVICE_URL: https://orders.api.yourcompany.com
  PAYMENT_SERVICE_URL: https://payments.api.yourcompany.com
  NOTIFICATION_SERVICE_URL: https://notifications.api.yourcompany.com

jobs:
- name: User Service Health
  steps:
    - name: User Service Check
      uses: http
      with:
        url: "{{vars.USER_SERVICE_URL}}/health"
      test: res.code == 200
      outputs:
        healthy: res.code == 200
        response_time: res.time
        user_count: res.body.json.active_users

- name: Order Service Health
  steps:
    - name: Order Service Check
      uses: http
      with:
        url: "{{vars.ORDER_SERVICE_URL}}/health"
      test: res.code == 200
      outputs:
        healthy: res.code == 200
        response_time: res.time
        pending_orders: res.body.json.pending_orders

- name: Payment Service Health
  steps:
    - name: Payment Service Check
      uses: http
      with:
        url: "{{vars.PAYMENT_SERVICE_URL}}/health"
      test: res.code == 200
      outputs:
        healthy: res.code == 200
        response_time: res.time
        transaction_queue: res.body.json.queue_length

- name: Notification Service Health
  steps:
    - name: Notification Service Check
      uses: http
      with:
        url: "{{vars.NOTIFICATION_SERVICE_URL}}/health"
      test: res.code == 200
      outputs:
        healthy: res.code == 200
        response_time: res.time
        queue_size: res.body.json.notification_queue

- name: Health Summary
  needs: [user-service-health, order-service-health, payment-service-health, notification-service-health]
  steps:
    - name: Generate Health Report
      echo: |
        🎯 Multi-Service Health Report
        ===============================
        
        User Service: {{outputs.user-service-health.healthy ? "✅" : "❌"}} ({{outputs.user-service-health.response_time}}ms)
          Active Users: {{outputs.user-service-health.user_count}}
        
        Order Service: {{outputs.order-service-health.healthy ? "✅" : "❌"}} ({{outputs.order-service-health.response_time}}ms)
          Pending Orders: {{outputs.order-service-health.pending_orders}}
        
        Payment Service: {{outputs.payment-service-health.healthy ? "✅" : "❌"}} ({{outputs.payment-service-health.response_time}}ms)
          Transaction Queue: {{outputs.payment-service-health.transaction_queue}}
        
        Notification Service: {{outputs.notification-service-health.healthy ? "✅" : "❌"}} ({{outputs.notification-service-health.response_time}}ms)
          Notification Queue: {{outputs.notification-service-health.queue_size}}
        
        Overall System Status: {{
          outputs.user-service-health.healthy && 
          outputs.order-service-health.healthy && 
          outputs.payment-service-health.healthy && 
          outputs.notification-service-health.healthy ? 
          "🟢 ALL SYSTEMS OPERATIONAL" : "🔴 ISSUES DETECTED"
        }}
        
        Timestamp: {{unixtime()}}
```

## データベースとインフラストラクチャ監視

### データベースヘルス監視

```yaml
name: Database Health Monitor
description: データベース接続とパフォーマンスを監視

vars:
  DB_HOST: db.yourcompany.com
  DB_PORT: 5432
  DB_NAME: production
  REDIS_HOST: redis.yourcompany.com
  REDIS_PORT: 6379

jobs:
- name: Database Connectivity
  steps:
    - name: PostgreSQL Connection Test
      uses: http
      with:
        url: "{{vars.DB_API_URL}}/ping"
      test: res.code == 200
      outputs:
        db_connected: res.code == 200
        connection_time: res.time
        active_connections: res.body.json.active_connections
        max_connections: res.body.json.max_connections

    - name: Database Performance Check
      uses: http
      with:
        url: "{{vars.DB_API_URL}}/stats"
      test: res.code == 200 && res.body.json.query_performance.avg_ms < 100
      outputs:
        avg_query_time: res.body.json.query_performance.avg_ms
        slow_queries: res.body.json.slow_queries.count
        db_size_mb: res.body.json.database_size_mb

- name: Cache System Health
  steps:
    - name: Redis Connection Test
      uses: http
      with:
        url: "{{vars.CACHE_API_URL}}/ping"
      test: res.code == 200
      outputs:
        cache_connected: res.code == 200
        cache_response_time: res.time
        memory_usage_percent: res.body.json.memory.usage_percent
        keys_count: res.body.json.keys.total

    - name: Cache Performance Check
      uses: http
      with:
        url: "{{vars.CACHE_API_URL}}/stats"
      test: res.code == 200 && res.body.json.hit_rate > 0.8
      outputs:
        hit_rate: res.body.json.hit_rate
        miss_rate: res.body.json.miss_rate
        evicted_keys: res.body.json.evicted_keys

- name: Infrastructure Report
  needs: [database-connectivity, cache-system-health]
  steps:
    - name: Infrastructure Health Summary
      echo: |
        🏗️ Infrastructure Health Report
        =================================
        
        Database Status:
        Connection: {{outputs.database-connectivity.db_connected ? "✅ Connected" : "❌ Failed"}}
        Response Time: {{outputs.database-connectivity.connection_time}}ms
        Active Connections: {{outputs.database-connectivity.active_connections}}/{{outputs.database-connectivity.max_connections}}
        Average Query Time: {{outputs.database-connectivity.avg_query_time}}ms
        Slow Queries: {{outputs.database-connectivity.slow_queries}}
        Database Size: {{outputs.database-connectivity.db_size_mb}}MB
        
        Cache Status:
        Connection: {{outputs.cache-system-health.cache_connected ? "✅ Connected" : "❌ Failed"}}
        Response Time: {{outputs.cache-system-health.cache_response_time}}ms
        Memory Usage: {{outputs.cache-system-health.memory_usage_percent}}%
        Total Keys: {{outputs.cache-system-health.keys_count}}
        Hit Rate: {{(outputs.cache-system-health.hit_rate * 100)}}%
        Miss Rate: {{(outputs.cache-system-health.miss_rate * 100)}}%
        
        Performance Alerts:
        {{outputs.database-connectivity.avg_query_time > 100 ? "⚠️ Database queries are slow (>" + outputs.database-connectivity.avg_query_time + "ms)" : ""}}
        {{outputs.database-connectivity.slow_queries > 10 ? "⚠️ High number of slow queries (" + outputs.database-connectivity.slow_queries + ")" : ""}}
        {{outputs.cache-system-health.memory_usage_percent > 80 ? "⚠️ Cache memory usage high (" + outputs.cache-system-health.memory_usage_percent + "%)" : ""}}
        {{outputs.cache-system-health.hit_rate < 0.8 ? "⚠️ Cache hit rate low (" + (outputs.cache-system-health.hit_rate * 100) + "%)" : ""}}
```

## 包括的システム監視

### フルスタック監視ワークフロー

```yaml
name: Full-Stack System Monitor
description: すべてのシステムコンポーネントの包括的監視

vars:
  # サービスURL
  FRONTEND_URL: https://app.yourcompany.com
  API_GATEWAY_URL: https://api.yourcompany.com
  
  # 監視しきい値
  MAX_RESPONSE_TIME: 2000
  MIN_SUCCESS_RATE: 0.95
  MAX_ERROR_RATE: 0.05

jobs:
  # Tier 1: インフラストラクチャ層
- name: Infrastructure Health Check
  steps:
    - name: Load Balancer Health
      uses: http
      with:
        url: "{{vars.LOAD_BALANCER_URL}}/health"
      test: res.code == 200
      outputs:
        lb_healthy: res.code == 200
        active_backends: res.body.json.active_backends
        total_backends: res.body.json.total_backends

    - name: CDN Performance
      uses: http
      with:
        url: "{{vars.CDN_URL}}/health"
      test: res.code == 200 && res.time < 500
      outputs:
        cdn_healthy: res.code == 200
        cdn_response_time: res.time
        cache_hit_ratio: res.body.json.cache_hit_ratio

  # Tier 2: アプリケーション層
- name: Application Health Check
  needs: [infrastructure-health-check]
  steps:
    - name: Frontend Health
      uses: http
      with:
        url: "{{vars.FRONTEND_URL}}/health"
      test: res.code == 200
      outputs:
        frontend_healthy: res.code == 200
        frontend_version: res.body.json.version
        frontend_build: res.body.json.build

    - name: API Gateway Health
      uses: http
      with:
        url: "{{vars.API_GATEWAY_URL}}/health"
      test: res.code == 200
      outputs:
        gateway_healthy: res.code == 200
        gateway_version: res.body.json.version
        registered_services: res.body.json.services.length

  # Tier 3: ビジネスロジック層
- name: Business Logic Health
  needs: [application-health-check]
  steps:
    - name: User Service Functional Test
      uses: http
      with:
        url: "{{vars.API_GATEWAY_URL}}/users/health-check"
      test: res.code == 200 && res.body.json.functional_test_passed == true
      outputs:
        user_service_functional: res.body.json.functional_test_passed
        active_sessions: res.body.json.active_sessions

    - name: Order Service Functional Test
      uses: http
      with:
        url: "{{vars.API_GATEWAY_URL}}/orders/health-check"
      test: res.code == 200 && res.body.json.functional_test_passed == true
      outputs:
        order_service_functional: res.body.json.functional_test_passed
        processing_queue_length: res.body.json.queue_length

  # Tier 4: パフォーマンス検証
- name: Performance Validation
  needs: [business-logic-health]
  steps:
    - name: End-to-End Performance Test
      uses: http
      with:
        url: "{{vars.API_GATEWAY_URL}}/performance/e2e-test"
        method: POST
        body: |
          {
            "test_type": "quick_validation",
            "max_duration_seconds": 30
          }
      test: |
        res.code == 200 && 
        res.body.json.success_rate >= {{vars.MIN_SUCCESS_RATE}} &&
        res.body.json.avg_response_time <= {{vars.MAX_RESPONSE_TIME}}
      outputs:
        success_rate: res.body.json.success_rate
        avg_response_time: res.body.json.avg_response_time
        p95_response_time: res.body.json.p95_response_time
        error_rate: res.body.json.error_rate

  # Tier 5: セキュリティとコンプライアンス
- name: Security Health Checks
  needs: [performance-validation]
  steps:
    - name: SSL Certificate Check
      uses: http
      with:
        url: "{{vars.SECURITY_API_URL}}/ssl-check"
        method: POST
        body: |
          {
            "domains": [
              "{{vars.FRONTEND_URL}}",
              "{{vars.API_GATEWAY_URL}}"
            ]
          }
      test: res.code == 200 && res.body.json.all_certificates_valid == true
      outputs:
        ssl_valid: res.body.json.all_certificates_valid
        cert_expiry_days: res.body.json.min_days_to_expiry

    - name: Security Headers Check
      uses: http
      with:
        url: "{{vars.SECURITY_API_URL}}/headers-check"
        method: POST
        body: |
          {
            "url": "{{vars.FRONTEND_URL}}"
          }
      test: res.code == 200 && res.body.json.security_score >= 0.8
      outputs:
        security_score: res.body.json.security_score
        missing_headers: res.body.json.missing_headers

  # 最終レポート
- name: System Health Report
  needs: [infrastructure-health-check, application-health-check, business-logic-health, performance-validation, security-health-checks]
  steps:
    - name: Generate Comprehensive Report
      echo: |
        🌐 Full-Stack System Health Report
        ===================================
        Generated: {{unixtime()}}
        
        📊 INFRASTRUCTURE LAYER
        Load Balancer: {{outputs.infrastructure-health-check.lb_healthy ? "✅ Healthy" : "❌ Issues"}}
          Backends: {{outputs.infrastructure-health-check.active_backends}}/{{outputs.infrastructure-health-check.total_backends}} active
        CDN: {{outputs.infrastructure-health-check.cdn_healthy ? "✅ Healthy" : "❌ Issues"}} ({{outputs.infrastructure-health-check.cdn_response_time}}ms)
          Cache Hit Ratio: {{(outputs.infrastructure-health-check.cache_hit_ratio * 100)}}%
        
        🖥️ APPLICATION LAYER
        Frontend: {{outputs.application-health-check.frontend_healthy ? "✅ Healthy" : "❌ Issues"}}
          Version: {{outputs.application-health-check.frontend_version}} (Build: {{outputs.application-health-check.frontend_build}})
        API Gateway: {{outputs.application-health-check.gateway_healthy ? "✅ Healthy" : "❌ Issues"}}
          Version: {{outputs.application-health-check.gateway_version}}
          Services: {{outputs.application-health-check.registered_services}} registered
        
        🏢 BUSINESS LOGIC LAYER
        User Service: {{outputs.business-logic-health.user_service_functional ? "✅ Functional" : "❌ Issues"}}
          Active Sessions: {{outputs.business-logic-health.active_sessions}}
        Order Service: {{outputs.business-logic-health.order_service_functional ? "✅ Functional" : "❌ Issues"}}
          Processing Queue: {{outputs.business-logic-health.processing_queue_length}} items
        
        ⚡ PERFORMANCE METRICS
        Success Rate: {{(outputs.performance-validation.success_rate * 100)}}%
        Average Response Time: {{outputs.performance-validation.avg_response_time}}ms
        95th Percentile: {{outputs.performance-validation.p95_response_time}}ms
        Error Rate: {{(outputs.performance-validation.error_rate * 100)}}%
        
        🔒 SECURITY STATUS
        SSL Certificates: {{outputs.security-health-checks.ssl_valid ? "✅ Valid" : "❌ Issues"}}
          Expiry: {{outputs.security-health-checks.cert_expiry_days}} days minimum
        Security Headers: Score {{(outputs.security-health-checks.security_score * 100)}}%
        {{outputs.security-health-checks.missing_headers ? "Missing Headers: " + outputs.security-health-checks.missing_headers : ""}}
        
        🎯 OVERALL SYSTEM STATUS
        {{
          outputs.infrastructure-health-check.lb_healthy &&
          outputs.infrastructure-health-check.cdn_healthy &&
          outputs.application-health-check.frontend_healthy &&
          outputs.application-health-check.gateway_healthy &&
          outputs.business-logic-health.user_service_functional &&
          outputs.business-logic-health.order_service_functional &&
          outputs.performance-validation.success_rate >= vars.MIN_SUCCESS_RATE &&
          outputs.performance-validation.avg_response_time <= vars.MAX_RESPONSE_TIME &&
          outputs.security-health-checks.ssl_valid &&
          outputs.security-health-checks.security_score >= 0.8
          ? "🟢 ALL SYSTEMS OPERATIONAL" 
          : "🔴 ISSUES REQUIRE ATTENTION"
        }}
        
        ⚠️ ALERTS
        {{outputs.infrastructure-health-check.active_backends != outputs.infrastructure-health-check.total_backends ? "• Load balancer has inactive backends" : ""}}
        {{outputs.infrastructure-health-check.cdn_response_time > 1000 ? "• CDN response time is high" : ""}}
        {{outputs.performance-validation.success_rate < vars.MIN_SUCCESS_RATE ? "• Success rate below threshold" : ""}}
        {{outputs.performance-validation.avg_response_time > vars.MAX_RESPONSE_TIME ? "• Average response time exceeds threshold" : ""}}
        {{outputs.security-health-checks.cert_expiry_days < 30 ? "• SSL certificates expiring soon" : ""}}
        {{outputs.security-health-checks.security_score < 0.8 ? "• Security headers need improvement" : ""}}
```

## アラートと通知の統合

### メールアラート付き監視

```yaml
name: Monitoring with Email Alerts
description: 自動メール通知付きヘルス監視

vars:
  # SMTP設定
  SMTP_HOST: smtp.gmail.com
  SMTP_PORT: 587
  SMTP_USERNAME: alerts@yourcompany.com
  ALERT_RECIPIENTS: ["ops@yourcompany.com", "dev-team@yourcompany.com"]
  
  # 監視設定
  CRITICAL_SERVICES: ["user-service", "payment-service", "order-service"]

jobs:
- name: Health Monitoring
  steps:
    - name: User Service Check
      id: user-service
      uses: http
      with:
        url: "{{vars.USER_SERVICE_URL}}/health"
      test: res.code == 200
      continue_on_error: true
      outputs:
        healthy: res.code == 200
        status_code: res.code
        response_time: res.time

    - name: Payment Service Check
      id: payment-service
      uses: http
      with:
        url: "{{vars.PAYMENT_SERVICE_URL}}/health"
      test: res.code == 200
      continue_on_error: true
      outputs:
        healthy: res.code == 200
        status_code: res.code
        response_time: res.time

    - name: Order Service Check
      id: order-service
      uses: http
      with:
        url: "{{vars.ORDER_SERVICE_URL}}/health"
      test: res.code == 200
      continue_on_error: true
      outputs:
        healthy: res.code == 200
        status_code: res.code
        response_time: res.time

- name: Alert Processing
  needs: [health-monitoring]
  steps:
    - name: Critical Service Alert
      if: "!outputs.user-service.healthy || !outputs.payment-service.healthy || !outputs.order-service.healthy"
      uses: smtp
      with:
        host: "{{vars.SMTP_HOST}}"
        port: "{{vars.SMTP_PORT}}"
        username: "{{vars.SMTP_USERNAME}}"
        password: "{{vars.SMTP_PASSWORD}}"
        from: "{{vars.SMTP_USERNAME}}"
        to: "{{vars.ALERT_RECIPIENTS}}"
        subject: "🚨 CRITICAL: Service Health Alert - {{unixtime()}}"
        body: |
          CRITICAL SERVICE HEALTH ALERT
          =============================
          
          Time: {{unixtime()}}
          Environment: {{vars.ENVIRONMENT || "Production"}}
          
          Service Status:
          User Service: {{outputs.user-service.healthy ? "✅ Healthy" : "❌ DOWN (HTTP " + outputs.user-service.status_code + ")"}}
          Payment Service: {{outputs.payment-service.healthy ? "✅ Healthy" : "❌ DOWN (HTTP " + outputs.payment-service.status_code + ")"}}
          Order Service: {{outputs.order-service.healthy ? "✅ Healthy" : "❌ DOWN (HTTP " + outputs.order-service.status_code + ")"}}
          
          Response Times:
          User Service: {{outputs.user-service.response_time}}ms
          Payment Service: {{outputs.payment-service.response_time}}ms
          Order Service: {{outputs.order-service.response_time}}ms
          
          IMMEDIATE ACTION REQUIRED
          
          Please investigate the failing services immediately.
          
          Monitoring Dashboard: {{vars.DASHBOARD_URL}}
          Incident Management: {{vars.INCIDENT_URL}}

    - name: Performance Warning Alert
      if: |
        (outputs.user-service.healthy && outputs.user-service.response_time > 2000) ||
        (outputs.payment-service.healthy && outputs.payment-service.response_time > 2000) ||
        (outputs.order-service.healthy && outputs.order-service.response_time > 2000)
      uses: smtp
      with:
        host: "{{vars.SMTP_HOST}}"
        port: "{{vars.SMTP_PORT}}"
        username: "{{vars.SMTP_USERNAME}}"
        password: "{{vars.SMTP_PASSWORD}}"
        from: "{{vars.SMTP_USERNAME}}"
        to: "{{vars.ALERT_RECIPIENTS}}"
        subject: "⚠️ WARNING: Performance Degradation Detected"
        body: |
          PERFORMANCE WARNING
          ===================
          
          Time: {{unixtime()}}
          Environment: {{vars.ENVIRONMENT || "Production"}}
          
          Performance Issues Detected:
          {{outputs.user-service.response_time > 2000 ? "• User Service: " + outputs.user-service.response_time + "ms (threshold: 2000ms)" : ""}}
          {{outputs.payment-service.response_time > 2000 ? "• Payment Service: " + outputs.payment-service.response_time + "ms (threshold: 2000ms)" : ""}}
          {{outputs.order-service.response_time > 2000 ? "• Order Service: " + outputs.order-service.response_time + "ms (threshold: 2000ms)" : ""}}
          
          While services are responding, performance degradation may impact user experience.
          Please investigate at your earliest convenience.

    - name: All Clear Notification
      if: outputs.user-service.healthy && outputs.payment-service.healthy && outputs.order-service.healthy && outputs.user-service.response_time <= 2000 && outputs.payment-service.response_time <= 2000 && outputs.order-service.response_time <= 2000
      echo: |
        ✅ All Services Healthy
        
        All critical services are operating normally:
        • User Service: {{outputs.user-service.response_time}}ms
        • Payment Service: {{outputs.payment-service.response_time}}ms  
        • Order Service: {{outputs.order-service.response_time}}ms
        
        No alerts sent - system is healthy.
```

## 環境固有の監視

### マルチ環境設定

**base-monitoring.yml:**
```yaml
name: Service Health Monitor
description: すべての環境用のベース監視ワークフロー

jobs:
- name: Service Health Check
  defaults:
    http:
      headers:
        User-Agent: "Probe Monitor"
        Accept: "application/json"
  steps:
    - name: API Health
      uses: http
      with:
        url: "{{vars.API_URL}}/health"
      test: res.code == 200
      outputs:
        api_healthy: res.code == 200
        response_time: res.time

    - name: Database Health
      uses: http
      with:
        url: "{{vars.DB_API_URL}}/ping"
      test: res.code == 200
      outputs:
        db_healthy: res.code == 200
        db_response_time: res.time

- name: Monitoring Report
  needs: [service-health-check]
  steps:
    - name: Status Report
      echo: |
        Environment: {{vars.ENVIRONMENT}}
        API: {{outputs.service-health-check.api_healthy ? "✅" : "❌"}} ({{outputs.service-health-check.response_time}}ms)
        Database: {{outputs.service-health-check.db_healthy ? "✅" : "❌"}} ({{outputs.service-health-check.db_response_time}}ms)
```

**development.yml:**
```yaml
vars:
  ENVIRONMENT: development
  API_URL: http://localhost:3000
  DB_API_URL: http://localhost:5432

jobs:
- name: Service Health Check
  defaults:
    http:
      timeout: 60s  # 開発環境ではより寛容
```

**production.yml:**
```yaml
vars:
  ENVIRONMENT: production
  API_URL: https://api.yourcompany.com
  DB_API_URL: https://db-api.yourcompany.com

jobs:
- name: Service Health Check
  defaults:
    http:
      timeout: 10s  # プロダクション用の厳密なタイムアウト

# プロダクション固有のセキュリティ監視を追加
- name: Security Monitoring
  needs: [service-health-check]
  steps:
    - name: SSL Certificate Check
      uses: http
      with:
        url: "{{vars.SECURITY_API_URL}}/ssl-status"
      test: res.code == 200 && res.body.json.all_valid == true
      outputs:
        ssl_valid: res.body.json.all_valid
        days_to_expiry: res.body.json.min_days_to_expiry
```

**使用方法:**
```bash
# 開発環境監視
probe base-monitoring.yml,development.yml

# プロダクション環境監視（セキュリティチェック含む）
probe base-monitoring.yml,production.yml
```

## ベストプラクティス

### 1. 監視戦略

- **監視を階層化**: インフラストラクチャ → アプリケーション → ビジネスロジック
- **適切なタイムアウトを設定**: プロダクション用は厳密、開発用は寛容
- **continue_on_errorを使用**: 重要でないチェックに対して
- **段階的アラート**: 情報 → 警告 → 重大

### 2. アラート疲れの防止

```yaml
# 良い例: 条件付きアラート
- name: Smart Alerting
  if: errors.count > 5 && duration > 300  # 持続的な問題のみでアラート
  uses: smtp
  # ...

# 避ける: すべての問題でアラート
- name: Noisy Alerting
  if: any_error_detected
  uses: smtp
  # アラート疲れを引き起こす
```

### 3. パフォーマンスの考慮事項

```yaml
# 良い例: 独立したチェックを並列実行
jobs:
- name: service-a-check    # 並列実行
- name: service-b-check    # 並列実行
- name: service-c-check    # 並列実行

# 良い例: 効率的な出力
outputs:
  service_healthy: res.code == 200  # ブール値フラグ
  response_time: res.time            # 特定のメトリクス
  # 避ける: レスポンス全体の保存: full_response: res.body.json
```

### 4. 文書化と保守

```yaml
name: Well-Documented Monitor
description: |
  E-commerceプラットフォーム用の監視ワークフロー。
  
  チェック:
  - ユーザーサービスのヘルスとパフォーマンス
  - 注文処理サービス
  - ペイメントゲートウェイの接続性
  - データベースパフォーマンス
  
  アラート:
  - 重大: サービス完全停止
  - 警告: パフォーマンス劣化
  - 情報: すべてのシステム正常
  
  予想実行時間: 30-60秒
  
  保守:
  - しきい値を月次レビュー
  - サービス移行時にサービスURLを更新
  - アラートチャネルを四半期毎にテスト
```

## 一般的な問題のトラブルシューティング

### 1. サービス発見の問題

```yaml
- name: Service Discovery Check
  uses: http
  with:
    url: "{{vars.SERVICE_REGISTRY_URL}}/services"
  test: res.code == 200 && res.body.json.services.length > 0
  outputs:
    available_services: res.body.json.services.map(s -> s.name)
    service_count: res.body.json.services.length
```

### 2. ネットワーク接続の問題

```yaml
- name: Network Connectivity Test
  uses: http
  with:
    url: "{{vars.EXTERNAL_HEALTH_CHECK_URL}}"
    timeout: 5s
  test: res.code == 200
  continue_on_error: true
  outputs:
    external_connectivity: res.code == 200
```

### 3. 認証の問題

```yaml
- name: Authentication Health Check
  uses: http
  with:
    url: "{{vars.AUTH_SERVICE_URL}}/health"
    headers:
      Authorization: "Bearer {{vars.HEALTH_CHECK_TOKEN}}"
  test: res.code == 200
  continue_on_error: true
  outputs:
    auth_service_healthy: res.code == 200
```

## 次のステップ

監視ワークフローの構築ができるようになったので、次を探索してください：

- **[APIテスト](../api-testing/)** - 包括的なAPIテスト戦略
- **[エラーハンドリング戦略](../error-handling-strategies/)** - 堅牢なエラーハンドリングパターン
- **[パフォーマンステスト](../performance-testing/)** - 負荷テストとパフォーマンス検証

監視は信頼性の高いシステムの基盤です。これらのパターンを使用して、ユーザーに影響を与える前に問題をキャッチする包括的な監視を構築しましょう。