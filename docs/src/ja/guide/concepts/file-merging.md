# ファイルマージ

ファイルマージは、複数の設定ファイルからワークフローを構成する Probe の強力な機能です。設定の再利用、環境固有のカスタマイズ、モジュラーなワークフロー設計を可能にします。このガイドでは、マージ戦略、パターン、ベストプラクティスについて詳しく説明します。

## ファイルマージの基礎

Probe はカンマ区切りのファイルパスを使用して複数の YAML ファイルのマージをサポートします：

```bash
probe base.yml,environment.yml,overrides.yml
```

ファイルは左から右の順序でマージされ、後のファイルが前のファイルを上書きします。

### 基本的なマージ例

**base.yml:**
```yaml
name: API Health Check
description: Basic API monitoring

vars:
  API_TIMEOUT: 30s
  RETRY_COUNT: 3

defaults:
  http:
    timeout: 30s
    headers:
      User-Agent: "Probe Monitor"

jobs:
  health-check:
    name: Health Check
    steps:
      - name: API Health
        uses: http
        with:
          url: "{{vars.API_URL}}/health"
        test: res.code == 200
```

**production.yml:**
```yaml
vars:
  API_URL: https://api.production.company.com
  API_TIMEOUT: 10s

defaults:
  http:
    timeout: 10s
    headers:
      Authorization: "Bearer {{vars.PROD_API_TOKEN}}"
```

**`probe base.yml,production.yml` の結果:**
```yaml
name: API Health Check
description: Basic API monitoring

vars:
  API_URL: https://api.production.company.com
  API_TIMEOUT: 10s                                  # 上書きされた
  RETRY_COUNT: 3                                    # 保持された

defaults:
  http:
    timeout: 10s                                    # 上書きされた
    headers:
      User-Agent: "Probe Monitor"                   # 保持された
      Authorization: "Bearer {{vars.PROD_API_TOKEN}}" # 追加された

jobs:
  health-check:                                     # base から保持
    name: Health Check
    steps:
      - name: API Health
        uses: http
        with:
          url: "{{vars.API_URL}}/health"             # マージされた env.API_URL を使用
        test: res.code == 200
```

## マージ戦略

### 1. 環境ベース設定

基本設定を環境固有設定から分離します：

**base-monitoring.yml:**
```yaml
name: Service Monitoring
description: Monitor critical services

defaults:
  http:
    headers:
      User-Agent: "Probe Monitor v1.0"

jobs:
  api-health:
    name: API Health Check
    steps:
      - name: Health Endpoint
        uses: http
        with:
          url: "{{vars.API_BASE_URL}}/health"
        test: res.code == 200

      - name: Metrics Endpoint
        uses: http
        with:
          url: "{{vars.API_BASE_URL}}/metrics"
        test: res.code == 200

  database-health:
    name: Database Health
    steps:
      - name: Database Ping
        uses: http
        with:
          url: "{{vars.DB_API_URL}}/ping"
        test: res.code == 200
```

**development.yml:**
```yaml
vars:
  API_BASE_URL: http://localhost:3000
  DB_API_URL: http://localhost:5432

defaults:
  http:
    timeout: 60s  # 開発環境では緩い設定
```

**staging.yml:**
```yaml
vars:
  API_BASE_URL: https://api.staging.company.com
  DB_API_URL: https://db.staging.company.com

defaults:
  http:
    timeout: 30s
    headers:
      X-Environment: staging
```

**production.yml:**
```yaml
vars:
  API_BASE_URL: https://api.company.com
  DB_API_URL: https://db.company.com

defaults:
  http:
    timeout: 10s  # プロダクション用の厳格なタイムアウト
    headers:
      X-Environment: production
      Authorization: "Bearer {{vars.PROD_API_TOKEN}}"

jobs:
  # プロダクション固有の監視を追加
  security-scan:
    name: Security Scan
    steps:
      - name: Security Health Check
        uses: http
        with:
          url: "{{vars.API_BASE_URL}}/security/health"
        test: res.code == 200
```

**使用方法:**
```bash
# 開発環境
probe base-monitoring.yml,development.yml

# ステージング環境
probe base-monitoring.yml,staging.yml

# プロダクション環境（追加のセキュリティチェックを含む）
probe base-monitoring.yml,production.yml
```

### 2. レイヤード設定アーキテクチャ

最大限の柔軟性のために設定をレイヤーで構築します：

**レイヤー1 - 基盤 (foundation.yml):**
```yaml
name: Multi-Service Health Check
description: Foundation configuration for service monitoring

defaults:
  http:
    headers:
      User-Agent: "Probe Health Monitor"
      Accept: "application/json"

jobs:
  connectivity:
    name: Basic Connectivity
    steps:
      - name: Network Check
        echo: "Checking network connectivity"
```

**レイヤー2 - コアサービス (core-services.yml):**
```yaml
jobs:
  user-service:
    name: User Service Health
    needs: [connectivity]
    steps:
      - name: User API Health
        uses: http
        with:
          url: "{{vars.USER_SERVICE_URL}}/health"
        test: res.code == 200

  order-service:
    name: Order Service Health
    needs: [connectivity]
    steps:
      - name: Order API Health
        uses: http
        with:
          url: "{{vars.ORDER_SERVICE_URL}}/health"
        test: res.code == 200
```

**レイヤー3 - 拡張サービス (extended-services.yml):**
```yaml
jobs:
  notification-service:
    name: Notification Service Health
    needs: [connectivity]
    steps:
      - name: Notification API Health
        uses: http
        with:
          url: "{{vars.NOTIFICATION_SERVICE_URL}}/health"
        test: res.code == 200

  analytics-service:
    name: Analytics Service Health
    needs: [connectivity]
    steps:
      - name: Analytics API Health
        uses: http
        with:
          url: "{{vars.ANALYTICS_SERVICE_URL}}/health"
        test: res.code == 200
```

**レイヤー4 - 統合テスト (integration.yml):**
```yaml
jobs:
  integration-tests:
    name: Integration Tests
    needs: [user-service, order-service]
    steps:
      - name: User-Order Integration
        uses: http
        with:
          url: "{{vars.API_GATEWAY_URL}}/integration/user-order"
        test: res.code == 200

  end-to-end:
    name: End-to-End Tests
    needs: [integration-tests, notification-service]
    steps:
      - name: Complete Workflow Test
        uses: http
        with:
          url: "{{vars.API_GATEWAY_URL}}/e2e/complete"
        test: res.code == 200
```

**レイヤー5 - 環境 (environment.yml):**
```yaml
vars:
  USER_SERVICE_URL: https://users.api.company.com
  ORDER_SERVICE_URL: https://orders.api.company.com
  NOTIFICATION_SERVICE_URL: https://notifications.api.company.com
  ANALYTICS_SERVICE_URL: https://analytics.api.company.com
  API_GATEWAY_URL: https://gateway.api.company.com

defaults:
  http:
    timeout: 15s
```

**組み合わせの使用例:**
```bash
# コアサービスのみ
probe foundation.yml,core-services.yml,environment.yml

# 統合なしの拡張サービス
probe foundation.yml,core-services.yml,extended-services.yml,environment.yml

# 統合テスト付きの完全スイート
probe foundation.yml,core-services.yml,extended-services.yml,integration.yml,environment.yml
```

### 3. 機能ベースの構成

混合して組み合わせることができる機能でワークフローを構成します：

**base-workflow.yml:**
```yaml
name: Modular Service Test
description: Base workflow with modular features

jobs:
  setup:
    name: Test Setup
    steps:
      - name: Initialize Test Environment
        echo: "Test environment initialized"
        outputs:
          test_session_id: "{{random_str(16)}}"
          start_time: "{{unixtime()}}"
```

**feature-authentication.yml:**
```yaml
jobs:
  authentication-tests:
    name: Authentication Tests
    needs: [setup]
    steps:
      - name: Login Test
        id: login
        uses: http
        with:
          url: "{{vars.API_URL}}/auth/login"
          method: POST
          body: |
            {
              "username": "{{vars.TEST_USERNAME}}",
              "password": "{{vars.TEST_PASSWORD}}"
            }
        test: res.code == 200
        outputs:
          access_token: res.body.json.access_token

      - name: Token Validation
        uses: http
        with:
          url: "{{vars.API_URL}}/auth/validate"
          headers:
            Authorization: "Bearer {{outputs.login.access_token}}"
        test: res.code == 200
```

**feature-user-management.yml:**
```yaml
jobs:
  user-management-tests:
    name: User Management Tests
    needs: [authentication-tests]
    steps:
      - name: Create User
        id: create-user
        uses: http
        with:
          url: "{{vars.API_URL}}/users"
          method: POST
          headers:
            Authorization: "Bearer {{outputs.authentication-tests.access_token}}"
          body: |
            {
              "name": "Test User {{random_str(6)}}",
              "email": "test{{random_str(8)}}@example.com"
            }
        test: res.code == 201
        outputs:
          user_id: res.body.json.user.id

      - name: Get User
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.create-user.user_id}}"
          headers:
            Authorization: "Bearer {{outputs.authentication-tests.access_token}}"
        test: res.code == 200

      - name: Delete User
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.create-user.user_id}}"
          method: DELETE
          headers:
            Authorization: "Bearer {{outputs.authentication-tests.access_token}}"
        test: res.code == 204
```

**feature-performance.yml:**
```yaml
jobs:
  performance-tests:
    name: Performance Tests
    needs: [setup]
    steps:
      - name: Response Time Test
        uses: http
        with:
          url: "{{vars.API_URL}}/performance/test"
        test: res.code == 200 && res.time < 1000
        outputs:
          response_time: res.time

      - name: Load Test
        uses: http
        with:
          url: "{{vars.API_URL}}/performance/load"
          method: POST
          body: |
            {
              "concurrent_users": 10,
              "duration": 30
            }
        test: res.code == 200
        outputs:
          load_test_passed: res.body.json.success
```

**feature-reporting.yml:**
```yaml
jobs:
  test-reporting:
    name: Test Reporting
    needs: [setup]  # 他のすべてのテスト完了後に実行
    steps:
      - name: Generate Report
        echo: |
          Test Execution Report
          ====================
          
          Session ID: {{outputs.setup.test_session_id}}
          Execution Time: {{unixtime() - outputs.setup.start_time}} seconds
          
          Test Results:
          {{jobs.authentication-tests ? "Authentication: " + (jobs.authentication-tests.success ? "✅ Passed" : "❌ Failed") : "Authentication: ⏸️ Not Run"}}
          {{jobs.user-management-tests ? "User Management: " + (jobs.user-management-tests.success ? "✅ Passed" : "❌ Failed") : "User Management: ⏸️ Not Run"}}
          {{jobs.performance-tests ? "Performance: " + (jobs.performance-tests.success ? "✅ Passed" : "❌ Failed") : "Performance: ⏸️ Not Run"}}
          
          {{jobs.performance-tests ? "Performance Metrics:" : ""}}
          {{jobs.performance-tests ? "- Response Time: " + outputs.performance-tests.response_time + "ms" : ""}}
          {{jobs.performance-tests ? "- Load Test: " + (outputs.performance-tests.load_test_passed ? "✅ Passed" : "❌ Failed") : ""}}
```

**機能の組み合わせ:**
```bash
# 認証のみ
probe base-workflow.yml,feature-authentication.yml,feature-reporting.yml,env.yml

# ユーザー管理（認証を含む）
probe base-workflow.yml,feature-authentication.yml,feature-user-management.yml,feature-reporting.yml,env.yml

# パフォーマンステストのみ
probe base-workflow.yml,feature-performance.yml,feature-reporting.yml,env.yml

# 完全な機能スイート
probe base-workflow.yml,feature-authentication.yml,feature-user-management.yml,feature-performance.yml,feature-reporting.yml,env.yml
```

## 高度なマージパターン

### 1. オーバーライドパターン

特別なケースには特定のオーバーライドファイルを使用します：

**base-config.yml:**
```yaml
name: Service Monitor
vars:
  API_TIMEOUT: 30s
  RETRY_COUNT: 3
  LOG_LEVEL: info

defaults:
  http:
    timeout: 30s

jobs:
  health-check:
    name: Health Check
    steps:
      - name: API Health
        uses: http
        with:
          url: "{{vars.API_URL}}/health"
        test: res.code == 200
```

**debug-overrides.yml:**
```yaml
vars:
  LOG_LEVEL: debug
  API_TIMEOUT: 120s  # デバッグ用の長いタイムアウト

defaults:
  http:
    timeout: 120s

jobs:
  health-check:
    steps:
      - name: API Health
        uses: http
        with:
          url: "{{vars.API_URL}}/health"
        test: res.code == 200
        outputs:
          # デバッグ出力を追加
          debug_response_headers: res.headers
          debug_response_time: res.time
          debug_response_size: res.body_size

      # デバッグステップを追加
      - name: Debug Information
        echo: |
          Debug Information:
          Response Time: {{outputs.debug_response_time}}ms
          Response Size: {{outputs.debug_response_size}} bytes
          Content Type: {{outputs.debug_response_headers["content-type"]}}
```

**load-test-overrides.yml:**
```yaml
vars:
  CONCURRENT_REQUESTS: 100
  TEST_DURATION: 300s

jobs:
  # health-check をロードテストでオーバーライド
  health-check:
    name: Load Test
    steps:
      - name: Load Test Execution
        uses: http
        with:
          url: "{{vars.API_URL}}/load-test"
          method: POST
          body: |
            {
              "concurrent_users": {{vars.CONCURRENT_REQUESTS}},
              "duration_seconds": {{vars.TEST_DURATION}}
            }
        test: res.code == 200 && res.body.json.success_rate > 0.95
        outputs:
          success_rate: res.body.json.success_rate
          avg_response_time: res.body.json.avg_response_time
```

**使用方法:**
```bash
# 通常の監視
probe base-config.yml,production-env.yml

# デバッグモード
probe base-config.yml,production-env.yml,debug-overrides.yml

# ロードテスト
probe base-config.yml,production-env.yml,load-test-overrides.yml
```

### 2. 環境変数を使った条件付きマージ

環境変数を使用してどのファイルをマージするかを制御します：

**conditional-merge.sh:**
```bash
#!/bin/bash

BASE_FILES="foundation.yml,core-services.yml"
ENV_FILE="${ENVIRONMENT:-development}.yml"
FEATURE_FILES=""

# 環境変数に基づいて機能を追加
if [ "$INCLUDE_SECURITY" = "true" ]; then
    FEATURE_FILES="${FEATURE_FILES},security-tests.yml"
fi

if [ "$INCLUDE_PERFORMANCE" = "true" ]; then
    FEATURE_FILES="${FEATURE_FILES},performance-tests.yml"
fi

if [ "$INCLUDE_INTEGRATION" = "true" ]; then
    FEATURE_FILES="${FEATURE_FILES},integration-tests.yml"
fi

# 最終ファイルリストを構築
FILES="${BASE_FILES},${ENV_FILE}${FEATURE_FILES}"

echo "Executing: probe $FILES"
probe $FILES
```

**使用方法:**
```bash
# 基本的な監視
ENVIRONMENT=production ./conditional-merge.sh

# セキュリティテスト付き
ENVIRONMENT=production INCLUDE_SECURITY=true ./conditional-merge.sh

# 完全スイート
ENVIRONMENT=production INCLUDE_SECURITY=true INCLUDE_PERFORMANCE=true INCLUDE_INTEGRATION=true ./conditional-merge.sh
```

### 3. テンプレートベースの設定

マージによって完成されるテンプレートを使用します：

**template-workflow.yml:**
```yaml
name: "{{WORKFLOW_NAME || 'Default Workflow'}}"
description: "{{WORKFLOW_DESCRIPTION || 'Template-based workflow'}}"

vars:
  SERVICE_NAME: "{{SERVICE_NAME}}"
  ENVIRONMENT: "{{ENVIRONMENT}}"

defaults:
  http:
    timeout: "{{DEFAULT_TIMEOUT || '30s'}}"
    headers:
      X-Service: "{{SERVICE_NAME}}"
      X-Environment: "{{ENVIRONMENT}}"

jobs:
  health-check:
    name: "{{SERVICE_NAME}} Health Check"
    steps:
      - name: Health Endpoint
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/health"
        test: res.code == 200
```

**service-config.yml:**
```yaml
vars:
  WORKFLOW_NAME: User Service Monitoring
  WORKFLOW_DESCRIPTION: Comprehensive monitoring for the user service
  SERVICE_NAME: user-service
  SERVICE_URL: https://users.api.company.com
  ENVIRONMENT: production
  DEFAULT_TIMEOUT: 15s

jobs:
  # サービス固有のテストを追加
  user-specific-tests:
    name: User Service Specific Tests
    needs: [health-check]
    steps:
      - name: User Count Check
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/users/count"
        test: res.code == 200 && res.body.json.count >= 0

      - name: User Authentication Test
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/auth/health"
        test: res.code == 200
```

## 設定管理パターン

### 1. 階層設定

継承のための階層で設定を整理します：

```
configs/
├── base/
│   ├── foundation.yml
│   └── common-defaults.yml
├── services/
│   ├── user-service.yml
│   ├── order-service.yml
│   └── notification-service.yml
├── environments/
│   ├── development.yml
│   ├── staging.yml
│   └── production.yml
└── features/
    ├── security.yml
    ├── performance.yml
    └── integration.yml
```

**設定管理用 Makefile:**
```makefile
# 基本設定
BASE_CONFIG = configs/base/foundation.yml,configs/base/common-defaults.yml

# 環境固有設定
dev: $(BASE_CONFIG),configs/environments/development.yml
	probe $^

staging: $(BASE_CONFIG),configs/environments/staging.yml,configs/features/security.yml
	probe $^

prod: $(BASE_CONFIG),configs/environments/production.yml,configs/features/security.yml,configs/features/performance.yml
	probe $^

# サービス固有テスト
test-user-service: $(BASE_CONFIG),configs/services/user-service.yml,configs/environments/$(ENV).yml
	probe $^

test-all-services: $(BASE_CONFIG),configs/services/user-service.yml,configs/services/order-service.yml,configs/services/notification-service.yml,configs/environments/$(ENV).yml
	probe $^
```

### 2. 設定検証

実行前にマージ済み設定を検証します：

**validation-config.yml:**
```yaml
name: Configuration Validation
description: Validate that required configuration is present

jobs:
  config-validation:
    name: Configuration Validation
    steps:
      - name: Validate Required Environment Variables
        echo: |
          Configuration Validation:
          
          Required Variables:
          API_URL: {{vars.API_URL ? "✅ Set" : "❌ Missing"}}
          DB_URL: {{vars.DB_URL ? "✅ Set" : "❌ Missing"}}
          ENVIRONMENT: {{vars.ENVIRONMENT ? "✅ Set (" + env.ENVIRONMENT + ")" : "❌ Missing"}}
          
          Optional Variables:
          SMTP_HOST: {{vars.SMTP_HOST ? "✅ Set" : "⚠️ Not Set"}}
          SLACK_WEBHOOK: {{vars.SLACK_WEBHOOK ? "✅ Set" : "⚠️ Not Set"}}
          
          Configuration Status: {{
            env.API_URL && env.DB_URL && env.ENVIRONMENT ? "✅ Valid" : "❌ Invalid"
          }}

      - name: Validate Configuration Consistency
        echo: |
          Configuration Consistency Check:
          
          Environment Alignment:
          {{vars.ENVIRONMENT == "production" && env.API_URL.contains("prod") ? "✅ Production URLs match environment" : ""}}
          {{vars.ENVIRONMENT == "staging" && env.API_URL.contains("staging") ? "✅ Staging URLs match environment" : ""}}
          {{vars.ENVIRONMENT == "development" && (env.API_URL.contains("localhost") || env.API_URL.contains("dev")) ? "✅ Development URLs match environment" : ""}}
          
          Security Check:
          {{vars.ENVIRONMENT == "production" && env.API_URL.startsWith("https://") ? "✅ Production uses HTTPS" : ""}}
          {{vars.ENVIRONMENT != "production" || env.API_URL.startsWith("https://") ? "" : "⚠️ Production should use HTTPS"}}
```

**検証付きの使用方法:**
```bash
# メインワークフロー実行前に設定を検証
probe validation-config.yml,base-config.yml,production.yml && \
probe base-workflow.yml,production.yml
```

## ベストプラクティス

### 1. ファイル整理

```yaml
# 良い例: 論理的なファイル整理
# base.yml - コアワークフロー構造
# environment.yml - 環境固有変数
# features.yml - オプション機能追加

# 避ける: すべてを処理しようとするモノリシックファイル
```

### 2. 明確なマージ順序

```bash
# 良い例: 論理的なマージ順序（一般的なものから具体的なものへ）
probe base.yml,environment.yml,team-overrides.yml

# 避ける: 混乱を招くマージ順序
probe overrides.yml,base.yml,environment.yml  # オーバーライドが無視される可能性
```

### 3. 環境変数戦略

```yaml
# 良い例: 動的な値に環境変数を使用
vars:
  API_URL: "{{vars.EXTERNAL_API_URL}}"
  TIMEOUT: "{{vars.REQUEST_TIMEOUT || '30s'}}"

# 良い例: 基本設定でデフォルトを提供
vars:
  DEFAULT_TIMEOUT: 30s
  DEFAULT_RETRY_COUNT: 3
```

### 4. ドキュメント

```yaml
# ワークフローにマージ戦略を文書化
name: Multi-Environment API Test
description: |
  このワークフローはファイルマージによって複数環境をサポートします。
  
  使用方法:
  - 開発環境: probe workflow.yml,development.yml
  - ステージング環境: probe workflow.yml,staging.yml
  - プロダクション環境: probe workflow.yml,production.yml
  
  ベース workflow.yml がコア構造を提供し、環境ファイルが環境固有の
  URL、タイムアウト、認証情報を提供します。
```

### 5. マージ検証

```yaml
jobs:
  pre-execution-check:
    name: Pre-execution Validation
    steps:
      - name: Validate Merged Configuration
        echo: |
          Merged Configuration Summary:
          
          Environment: {{vars.ENVIRONMENT || "Not specified"}}
          API URL: {{vars.API_URL || "Not configured"}}
          Timeout: {{defaults.http.timeout || "Default"}}
          
          Required configurations: {{
            env.API_URL && env.ENVIRONMENT ? "✅ Present" : "❌ Missing"
          }}
```

## よくあるアンチパターン

### 1. 循環依存

```yaml
# 避ける: 相互に依存するファイル
# base.yml は production.yml でのみ動作する参照を含む
# production.yml は base.yml でのみ動作するジョブを含む
```

### 2. 深いオーバーライド階層

```bash
# 避ける: オーバーライドレイヤーが多すぎる
probe base.yml,region.yml,environment.yml,team.yml,user.yml,local.yml
# 最終設定がどうなるかを理解することが困難
```

### 3. 一貫性のないマージ戦略

```yaml
# 避ける: 異なるマージアプローチの混在
# あるファイルは完全に上書き、他のファイルは加算的にマージ
# 予測できない結果を生成
```

## 次のステップ

ファイルマージを理解したら、以下を探索してください：

1. **[ハウツー](../../how-tos/)** - 実用的なファイルマージパターンの実例を見る
2. **[リファレンス](../../reference/)** - 詳細な構文と設定リファレンス
3. **[チュートリアル](../../tutorials/)** - 一般的なシナリオのステップバイステップガイド

ファイルマージは、柔軟で保守しやすいワークフロー設定を構築するための鍵です。これらのパターンをマスターして、ニーズに応じてスケールする再利用可能で環境対応の自動化を作成しましょう。