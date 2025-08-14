# マルチ環境デプロイテスト

このチュートリアルでは、Probeを使用してマルチ環境テスト戦略を作成する方法を学びます。開発、ステージング、プロダクション環境でのデプロイメントを自動的に検証し、デプロイメントパイプラインのすべての段階で一貫性と信頼性を確保するワークフローを構築します。

## 構築する内容

以下の機能を持つ完全なマルチ環境テストシステム：

- **環境固有の設定** - 各環境に合わせた設定
- **段階的デプロイテスト** - 各デプロイ段階の検証
- **環境間一貫性チェック** - 機能パリティの確保
- **環境ヘルス監視** - 継続的な環境検証
- **デプロイ検証パイプライン** - 自動デプロイ確認
- **ロールバック検出** - 環境の乖離を特定
- **設定ドリフト検出** - 環境一貫性の監視

## 前提条件

- Probeがインストール済み（[インストールガイド](../get-started/installation/)）
- 複数環境へのアクセス（dev、staging、production）
- デプロイメントパイプラインの理解
- 環境管理の基本知識

## チュートリアル概要

典型的な3層環境セットアップのテストワークフローを作成します：

- **開発環境** - 最新機能、迅速な反復
- **ステージング環境** - 最終検証用のプロダクション類似環境
- **プロダクション環境** - ライブユーザー向け環境

## ステップ1: 環境設定構造

整理された設定構造を作成します：

```bash
deployment-tests/
├── config/
│   ├── base.yml              # 共通設定
│   ├── environments/
│   │   ├── development.yml   # 開発環境固有設定
│   │   ├── staging.yml       # ステージング環境固有設定
│   │   └── production.yml    # プロダクション環境固有設定
│   └── features/
│       ├── feature-flags.yml # 機能フラグ設定
│       └── experiments.yml   # A/Bテスト設定
├── workflows/
│   ├── health-check.yml      # 基本ヘルス検証
│   ├── deployment-validation.yml # デプロイ確認
│   ├── consistency-check.yml # 環境間検証
│   └── rollback-detection.yml # ロールバック識別
└── scripts/
    ├── setup-environment.sh  # 環境セットアップ
    └── deploy-validation.sh  # デプロイ検証スクリプト
```

**config/base.yml:**
```yaml
name: "Multi-Environment Deployment Testing"
description: "開発、ステージング、プロダクション環境での包括的テスト"

vars:
  # 共通設定
  API_VERSION: "v2"
  USER_AGENT: "Probe Multi-Env Tester v1.0"
  
  # テストユーザー設定
  TEST_USER_EMAIL: "test@example.com"
  TEST_USER_PASSWORD: "TestPassword123!"
  
  # パフォーマンスしきい値（ベース - 環境ごとに上書きされる）
  RESPONSE_TIME_THRESHOLD: 2000
  CRITICAL_RESPONSE_TIME: 5000
  
  # ヘルスチェック設定
  HEALTH_CHECK_INTERVAL: 30
  MAX_CONSECUTIVE_FAILURES: 3
  
  # 機能テスト
  FEATURE_VALIDATION_ENABLED: true
  A_B_TEST_VALIDATION_ENABLED: false

jobs:
- name: default
  defaults:
    http:
      timeout: "15s"
      headers:
        User-Agent: "{{vars.USER_AGENT}}"
        Accept: "application/json"
      follow_redirects: true
      verify_ssl: true
```

**config/environments/development.yml:**
```yaml
vars:
  # 環境識別
  ENVIRONMENT: "development"
  ENVIRONMENT_COLOR: "🟡"
  
  # API設定
  API_BASE_URL: "https://api-dev.example.com"
  WEB_BASE_URL: "https://web-dev.example.com"
  
  # 寛容なパフォーマンスしきい値
  RESPONSE_TIME_THRESHOLD: 5000    # 5秒（より寛容）
  CRITICAL_RESPONSE_TIME: 10000    # 10秒
  
  # 開発環境固有設定
  DEBUG_MODE: true
  DETAILED_LOGGING: true
  SKIP_PERFORMANCE_TESTS: false   # 開発環境でもパフォーマンステスト
  SKIP_LOAD_TESTS: true           # 重い負荷テストはスキップ
  
  # 機能フラグ（開発環境は全機能を取得）
  ENABLE_NEW_FEATURES: true
  ENABLE_EXPERIMENTAL_FEATURES: true
  ENABLE_DEBUG_ENDPOINTS: true
  
  # 開発データ
  USE_TEST_DATA: true
  RESET_DATA_BEFORE_TESTS: true

jobs:
- name: default
  defaults:
    http:
      timeout: "30s"         # 開発環境ではより長いタイムアウト
      verify_ssl: false      # 自己署名証明書を許可
```

**config/environments/staging.yml:**
```yaml
vars:
  # 環境識別
  ENVIRONMENT: "staging"
  ENVIRONMENT_COLOR: "🟠"
  
  # API設定
  API_BASE_URL: "https://api-staging.example.com"
  WEB_BASE_URL: "https://web-staging.example.com"
  
  # プロダクション類似のパフォーマンスしきい値
  RESPONSE_TIME_THRESHOLD: 2000    # 2秒
  CRITICAL_RESPONSE_TIME: 5000     # 5秒
  
  # ステージング環境固有設定
  DEBUG_MODE: false
  DETAILED_LOGGING: true
  SKIP_PERFORMANCE_TESTS: false
  SKIP_LOAD_TESTS: false
  
  # 機能フラグ（ステージングはプロダクション + 承認済み機能をミラー）
  ENABLE_NEW_FEATURES: true
  ENABLE_EXPERIMENTAL_FEATURES: false  # 安定機能のみ
  ENABLE_DEBUG_ENDPOINTS: false
  
  # ステージングデータ
  USE_TEST_DATA: false
  USE_PRODUCTION_LIKE_DATA: true
  RESET_DATA_BEFORE_TESTS: false

jobs:
- name: default
  defaults:
    http:
      timeout: "15s"
      verify_ssl: true
```

**config/environments/production.yml:**
```yaml
vars:
  # 環境識別
  ENVIRONMENT: "production"
  ENVIRONMENT_COLOR: "🔴"
  
  # API設定
  API_BASE_URL: "https://api.example.com"
  WEB_BASE_URL: "https://web.example.com"
  
  # 厳密なパフォーマンスしきい値
  RESPONSE_TIME_THRESHOLD: 1500    # 1.5秒
  CRITICAL_RESPONSE_TIME: 3000     # 3秒
  
  # プロダクション設定
  DEBUG_MODE: false
  DETAILED_LOGGING: false
  SKIP_PERFORMANCE_TESTS: false
  SKIP_LOAD_TESTS: true            # プロダクションへの影響を避ける
  
  # 機能フラグ（プロダクションは安定・承認済み機能のみ）
  ENABLE_NEW_FEATURES: false
  ENABLE_EXPERIMENTAL_FEATURES: false
  ENABLE_DEBUG_ENDPOINTS: false
  
  # プロダクションデータ
  USE_TEST_DATA: false
  USE_PRODUCTION_DATA: true
  RESET_DATA_BEFORE_TESTS: false
  
  # プロダクション固有監視
  ENABLE_REAL_USER_MONITORING: true
  ENABLE_ERROR_TRACKING: true

jobs:
- name: default
  defaults:
    http:
      timeout: "10s"
      verify_ssl: true
```

## ステップ2: 基本環境ヘルスチェック

包括的なヘルス検証を作成します：

**workflows/health-check.yml:**
```yaml
name: "{{vars.ENVIRONMENT | upper}} Environment Health Check"
description: "{{vars.ENVIRONMENT}}環境の包括的ヘルス検証"

jobs:
- name: Infrastructure Health Check
  steps:
    - name: "API Server Health"
      id: api-health
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/health"
      test: |
        res.code == 200 &&
        res.time < vars.RESPONSE_TIME_THRESHOLD &&
        res.body.json.status == "healthy"
      outputs:
        api_status: res.body.json.status
        api_version: res.body.json.version
        api_response_time: res.time
        api_uptime: res.body.json.uptime

    - name: "Web Application Health"
      id: web-health
      uses: http
      with:
        url: "{{vars.WEB_BASE_URL}}/health"
      test: |
        res.code == 200 &&
        res.time < vars.RESPONSE_TIME_THRESHOLD
      continue_on_error: true
      outputs:
        web_status: res.code == 200 ? "healthy" : "unhealthy"
        web_response_time: res.time

    - name: "Database Connectivity"
      id: db-health
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/health/database"
      test: |
        res.code == 200 &&
        res.body.json.database.connected == true &&
        res.body.json.database.responseTime < 1000
      outputs:
        db_status: res.body.json.database.connected ? "connected" : "disconnected"
        db_response_time: res.body.json.database.responseTime
        db_pool_size: res.body.json.database.poolSize

    - name: "Cache System Health"
      id: cache-health
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/health/cache"
      test: |
        res.code == 200 &&
        res.body.json.cache.connected == true
      continue_on_error: true
      outputs:
        cache_status: res.body.json.cache.connected ? "connected" : "disconnected"
        cache_hit_rate: res.body.json.cache.hitRate

- name: External Service Dependencies
  needs: [infrastructure-health-check]
  steps:
    - name: "Payment Service Health"
      id: payment-health
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/health/payment"
      test: |
        res.code == 200 &&
        res.body.json.paymentService.available == true
      continue_on_error: true
      outputs:
        payment_status: res.body.json.paymentService.available ? "available" : "unavailable"

    - name: "Email Service Health"
      id: email-health
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/health/email"
      test: |
        res.code == 200 &&
        res.body.json.emailService.available == true
      continue_on_error: true
      outputs:
        email_status: res.body.json.emailService.available ? "available" : "unavailable"

    - name: "Search Service Health"
      id: search-health
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/health/search"
      test: |
        res.code == 200 &&
        res.body.json.searchService.available == true
      continue_on_error: true
      outputs:
        search_status: res.body.json.searchService.available ? "available" : "unavailable"

- name: Environment Health Report
  needs: [infrastructure-health-check, external-service-dependencies]
  steps:
    - name: "Generate Health Report"
      echo: |
        {{vars.ENVIRONMENT_COLOR}} === {{vars.ENVIRONMENT | upper}} ENVIRONMENT HEALTH REPORT ===
        
        Infrastructure Status:
        • API Server: {{outputs.api-health.api_status | upper}} ({{outputs.api-health.api_response_time}}ms)
          Version: {{outputs.api-health.api_version}}
          Uptime: {{outputs.api-health.api_uptime}}
        
        • Web Application: {{outputs.web-health.web_status | upper}} ({{outputs.web-health.web_response_time}}ms)
        
        • Database: {{outputs.db-health.db_status | upper}} ({{outputs.db-health.db_response_time}}ms)
          Pool Size: {{outputs.db-health.db_pool_size}}
        
        • Cache: {{outputs.cache-health.cache_status | upper}}
          Hit Rate: {{outputs.cache-health.cache_hit_rate}}%
        
        External Services:
        • Payment Service: {{outputs.payment-health.payment_status | upper}}
        • Email Service: {{outputs.email-health.email_status | upper}}
        • Search Service: {{outputs.search-health.search_status | upper}}
        
        Overall Status: {{
          outputs.api-health.api_status == "healthy" &&
          outputs.web-health.web_status == "healthy" &&
          outputs.db-health.db_status == "connected" ? "✅ HEALTHY" : "⚠️ DEGRADED"
        }}
        
        Generated: {{unixtime()}}
        Environment: {{vars.ENVIRONMENT}}

    - name: "Health Check Validation"
      test: |
        outputs.api-health.api_status == "healthy" &&
        outputs.db-health.db_status == "connected"
```

## ステップ3: デプロイ検証ワークフロー

包括的なデプロイ確認を作成します：

**workflows/deployment-validation.yml:**
```yaml
name: "{{vars.ENVIRONMENT | upper}} Deployment Validation"
description: "{{vars.ENVIRONMENT}}環境でのデプロイ成功を検証"

vars:
  # デプロイメタデータ（CI/CDから渡される）
  DEPLOYMENT_ID: "{{DEPLOYMENT_ID ?? unixtime()}}"
  BUILD_VERSION: "{{BUILD_VERSION ?? 'unknown'}}"
  DEPLOYMENT_TIMESTAMP: "{{DEPLOYMENT_TIMESTAMP ?? unixtime()}}"

jobs:
- name: Pre-Deployment Validation
  steps:
    - name: "Verify Deployment Prerequisites"
      id: prerequisites
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/deployment/prerequisites"
        headers:
          X-Deployment-ID: "{{vars.DEPLOYMENT_ID}}"
      test: |
        res.code == 200 &&
        res.body.json.readyForDeployment == true
      outputs:
        deployment_ready: res.body.json.readyForDeployment
        current_version: res.body.json.currentVersion
        target_version: res.body.json.targetVersion

    - name: "Database Migration Status"
      id: migration-status
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/deployment/migrations"
      test: |
        res.code == 200 &&
        res.body.json.pendingMigrations == 0
      outputs:
        pending_migrations: res.body.json.pendingMigrations
        last_migration: res.body.json.lastMigration

- name: Deployment Verification
  needs: [pre-deployment-validation]
  steps:
    - name: "Verify New Version Deployment"
      id: version-check
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/version"
      test: |
        res.code == 200 &&
        res.body.json.version == vars.BUILD_VERSION
      outputs:
        deployed_version: res.body.json.version
        deployment_time: res.body.json.deploymentTime
        build_hash: res.body.json.buildHash

    - name: "Feature Flag Validation"
      id: feature-flags
      if: vars.ENABLE_NEW_FEATURES == "true"
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/features"
      test: |
        res.code == 200 &&
        res.body.json.features != null
      outputs:
        active_features: res.body.json.features
        feature_count: res.body.json.features.length

    - name: "Configuration Validation"
      id: config-validation
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/configuration"
      test: |
        res.code == 200 &&
        res.body.json.environment == vars.ENVIRONMENT &&
        res.body.json.configVersion != null
      outputs:
        config_environment: res.body.json.environment
        config_version: res.body.json.configVersion
        config_valid: res.body.json.valid

- name: Post-Deployment Functional Tests
  needs: [deployment-verification]
  steps:
    - name: "Critical Path Test - User Authentication"
      id: auth-test
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/api/{{vars.API_VERSION}}/auth/login"
        method: "POST"
        body: |
          {
            "email": "{{vars.TEST_USER_EMAIL}}",
            "password": "{{vars.TEST_USER_PASSWORD}}"
          }
      test: |
        res.code == 200 &&
        res.body.json.token != null &&
        res.time < vars.RESPONSE_TIME_THRESHOLD
      outputs:
        auth_token: res.body.json.token
        auth_response_time: res.time

    - name: "Critical Path Test - Data Retrieval"
      id: data-test
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/api/{{vars.API_VERSION}}/products?limit=5"
        headers:
          Authorization: "Bearer {{outputs.auth-test.auth_token}}"
      test: |
        res.code == 200 &&
        res.body.json.products != null &&
        res.body.json.products.length > 0 &&
        res.time < vars.RESPONSE_TIME_THRESHOLD
      outputs:
        product_count: res.body.json.products.length
        data_response_time: res.time

    - name: "Critical Path Test - Data Mutation"
      id: mutation-test
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/api/{{vars.API_VERSION}}/products"
        method: "POST"
        headers:
          Authorization: "Bearer {{outputs.auth-test.auth_token}}"
        body: |
          {
            "name": "Deployment Test Product {{vars.DEPLOYMENT_ID}}",
            "price": 99.99,
            "category": "Test"
          }
      test: |
        res.code == 201 &&
        res.body.json.id != null &&
        res.time < vars.RESPONSE_TIME_THRESHOLD
      outputs:
        created_product_id: res.body.json.id
        mutation_response_time: res.time

    - name: "Cleanup Test Data"
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/api/{{vars.API_VERSION}}/products/{{outputs.mutation-test.created_product_id}}"
        method: "DELETE"
        headers:
          Authorization: "Bearer {{outputs.auth-test.auth_token}}"
      test: res.code == 204 || res.code == 200
      continue_on_error: true

- name: Performance Validation
  needs: [post-deployment-functional-tests]
  if: vars.SKIP_PERFORMANCE_TESTS != "true"
  steps:
    - name: "Response Time Validation"
      id: perf-test
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/api/{{vars.API_VERSION}}/products"
      test: |
        res.code == 200 &&
        res.time < vars.RESPONSE_TIME_THRESHOLD
      outputs:
        response_time: res.time
        performance_grade: |
          {{res.time < 500 ? "A" :
            res.time < 1000 ? "B" :
            res.time < 2000 ? "C" : "D"}}

    - name: "Load Test Simulation"
      if: vars.SKIP_LOAD_TESTS != "true"
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/health"
      test: res.code == 200 && res.time < (vars.RESPONSE_TIME_THRESHOLD * 2)
      # 実際のシナリオでは、これが同時リクエストをトリガーする

- name: Deployment Validation Report
  needs: [pre-deployment-validation, deployment-verification, post-deployment-functional-tests, performance-validation]
  steps:
    - name: "Generate Deployment Report"
      echo: |
        {{vars.ENVIRONMENT_COLOR}} === {{vars.ENVIRONMENT | upper}} DEPLOYMENT VALIDATION REPORT ===
        
        Deployment Information:
        • Deployment ID: {{vars.DEPLOYMENT_ID}}
        • Target Version: {{vars.BUILD_VERSION}}
        • Deployed Version: {{outputs.version-check.deployed_version}}
        • Deployment Time: {{outputs.version-check.deployment_time}}
        • Environment: {{vars.ENVIRONMENT}}
        
        Pre-Deployment Status:
        • Prerequisites: {{outputs.prerequisites.deployment_ready ? "✅ Ready" : "❌ Not Ready"}}
        • Current Version: {{outputs.prerequisites.current_version}}
        • Target Version: {{outputs.prerequisites.target_version}}
        • Pending Migrations: {{outputs.migration-status.pending_migrations}}
        
        Deployment Verification:
        • Version Match: {{outputs.version-check.deployed_version == vars.BUILD_VERSION ? "✅ Correct" : "❌ Mismatch"}}
        • Configuration: {{outputs.config-validation.config_valid ? "✅ Valid" : "❌ Invalid"}}
        • Feature Flags: {{vars.ENABLE_NEW_FEATURES == "true" ? outputs.feature-flags.feature_count + " features active" : "Default features"}}
        
        Functional Tests:
        • Authentication: {{outputs.auth-test.auth_response_time}}ms {{outputs.auth-test.auth_response_time < vars.RESPONSE_TIME_THRESHOLD ? "✅" : "⚠️"}}
        • Data Retrieval: {{outputs.data-test.data_response_time}}ms {{outputs.data-test.data_response_time < vars.RESPONSE_TIME_THRESHOLD ? "✅" : "⚠️"}}
        • Data Mutation: {{outputs.mutation-test.mutation_response_time}}ms {{outputs.mutation-test.mutation_response_time < vars.RESPONSE_TIME_THRESHOLD ? "✅" : "⚠️"}}
        
        Performance Validation:
        {{vars.SKIP_PERFORMANCE_TESTS != "true" ? "• Response Time: " + outputs.perf-test.response_time + "ms (Grade: " + outputs.perf-test.performance_grade + ")" : "• Performance Tests: ⏭️ Skipped"}}
        {{vars.SKIP_PERFORMANCE_TESTS != "true" ? "• Performance Status: " + (outputs.perf-test.response_time < vars.RESPONSE_TIME_THRESHOLD ? "✅ Acceptable" : "⚠️ Slow") : ""}}
        
        Overall Deployment Status: {{
          outputs.version-check.deployed_version == vars.BUILD_VERSION &&
          outputs.config-validation.config_valid &&
          outputs.auth-test.auth_response_time < vars.RESPONSE_TIME_THRESHOLD &&
          outputs.data-test.data_response_time < vars.RESPONSE_TIME_THRESHOLD &&
          outputs.mutation-test.mutation_response_time < vars.RESPONSE_TIME_THRESHOLD ? 
          "✅ DEPLOYMENT SUCCESSFUL" : "❌ DEPLOYMENT ISSUES DETECTED"
        }}
        
        Generated: {{unixtime()}}

    - name: "Validate Deployment Success"
      test: |
        outputs.version-check.deployed_version == vars.BUILD_VERSION &&
        outputs.config-validation.config_valid &&
        outputs.auth-test.auth_response_time < vars.RESPONSE_TIME_THRESHOLD &&
        outputs.data-test.data_response_time < vars.RESPONSE_TIME_THRESHOLD
```

## ステップ4: 環境間一貫性チェック

環境間の一貫性を検証するワークフローを作成します：

**workflows/consistency-check.yml:**
```yaml
name: "Cross-Environment Consistency Check"
description: "環境間の一貫性を検証"

vars:
  # 比較する環境を定義
  PRIMARY_ENVIRONMENT: "{{PRIMARY_ENV ?? 'production'}}"
  COMPARISON_ENVIRONMENTS: "{{COMPARISON_ENVS ?? 'staging,development'}}"

jobs:
- name: Version Consistency Check
  steps:
    - name: "Get Primary Environment Version"
      id: primary-version
      uses: http
      with:
        url: "{{PRIMARY_ENV == 'production' ? 'https://api.example.com' : 
                PRIMARY_ENV == 'staging' ? 'https://api-staging.example.com' :
                'https://api-dev.example.com'}}/version"
      test: res.code == 200
      outputs:
        primary_version: res.body.json.version
        primary_env: vars.PRIMARY_ENVIRONMENT
        primary_config_version: res.body.json.configVersion

    - name: "Get Staging Environment Version"
      id: staging-version
      if: vars.PRIMARY_ENVIRONMENT != "staging"
      uses: http
      with:
        url: "https://api-staging.example.com/version"
      test: res.code == 200
      continue_on_error: true
      outputs:
        staging_version: res.body.json.version
        staging_config_version: res.body.json.configVersion

    - name: "Get Development Environment Version"
      id: dev-version
      if: vars.PRIMARY_ENVIRONMENT != "development"
      uses: http
      with:
        url: "https://api-dev.example.com/version"
      test: res.code == 200
      continue_on_error: true
      outputs:
        dev_version: res.body.json.version
        dev_config_version: res.body.json.configVersion

- name: Feature Flag Consistency
  needs: [version-consistency-check]
  steps:
    - name: "Compare Production vs Staging Features"
      id: prod-staging-features
      uses: http
      with:
        url: "https://api.example.com/features"
      test: res.code == 200
      outputs:
        prod_features: res.body.json.features ? Object.keys(res.body.json.features) : []
        prod_feature_count: res.body.json.features ? Object.keys(res.body.json.features).length : 0

    - name: "Get Staging Features"
      id: staging-features
      uses: http
      with:
        url: "https://api-staging.example.com/features"
      test: res.code == 200
      outputs:
        staging_features: res.body.json.features ? Object.keys(res.body.json.features) : []
        staging_feature_count: res.body.json.features ? Object.keys(res.body.json.features).length : 0

    - name: "Feature Drift Analysis"
      echo: |
        === FEATURE FLAG CONSISTENCY ANALYSIS ===
        
        Production Features: {{outputs.prod-staging-features.prod_feature_count}}
        Staging Features: {{outputs.staging-features.staging_feature_count}}
        
        Feature Drift: {{Math.abs(outputs.prod-staging-features.prod_feature_count - outputs.staging-features.staging_feature_count)}} features different
        
        Status: {{outputs.prod-staging-features.prod_feature_count == outputs.staging-features.staging_feature_count ? "✅ Consistent" : "⚠️ Drift Detected"}}

- name: Configuration Consistency
  needs: [version-consistency-check]
  steps:
    - name: "Database Schema Consistency"
      id: schema-consistency
      uses: http
      with:
        url: "https://api.example.com/schema/version"
      test: res.code == 200
      outputs:
        prod_schema_version: res.body.json.schemaVersion
        prod_migration_count: res.body.json.migrationCount

    - name: "Staging Schema Version"
      id: staging-schema
      uses: http
      with:
        url: "https://api-staging.example.com/schema/version"
      test: res.code == 200
      outputs:
        staging_schema_version: res.body.json.schemaVersion
        staging_migration_count: res.body.json.migrationCount

    - name: "Schema Consistency Report"
      echo: |
        === DATABASE SCHEMA CONSISTENCY ===
        
        Production Schema: v{{outputs.schema-consistency.prod_schema_version}} ({{outputs.schema-consistency.prod_migration_count}} migrations)
        Staging Schema: v{{outputs.staging-schema.staging_schema_version}} ({{outputs.staging-schema.staging_migration_count}} migrations)
        
        Schema Status: {{outputs.schema-consistency.prod_schema_version == outputs.staging-schema.staging_schema_version ? "✅ Synchronized" : "⚠️ Version Mismatch"}}
        Migration Status: {{outputs.schema-consistency.prod_migration_count == outputs.staging-schema.staging_migration_count ? "✅ Synchronized" : "⚠️ Migration Count Mismatch"}}

- name: Overall Consistency Report
  needs: [version-consistency-check, feature-flag-consistency, configuration-consistency]
  steps:
    - name: "Generate Consistency Report"
      echo: |
        🔍 === CROSS-ENVIRONMENT CONSISTENCY REPORT ===
        
        Environment Versions:
        • Production: {{outputs.primary-version.primary_version}} (config: {{outputs.primary-version.primary_config_version}})
        • Staging: {{outputs.staging-version.staging_version ?? "N/A"}} (config: {{outputs.staging-version.staging_config_version ?? "N/A"}})
        • Development: {{outputs.dev-version.dev_version ?? "N/A"}} (config: {{outputs.dev-version.dev_config_version ?? "N/A"}})
        
        Consistency Checks:
        • Version Alignment: {{outputs.primary-version.primary_version == outputs.staging-version.staging_version ? "✅ Aligned" : "⚠️ Misaligned"}}
        • Feature Flags: {{outputs.prod-staging-features.prod_feature_count == outputs.staging-features.staging_feature_count ? "✅ Consistent" : "⚠️ Drift Detected"}}
        • Database Schema: {{outputs.schema-consistency.prod_schema_version == outputs.staging-schema.staging_schema_version ? "✅ Synchronized" : "⚠️ Mismatch"}}
        
        Recommendations:
        {{outputs.primary-version.primary_version != outputs.staging-version.staging_version ? "⚠️ Version mismatch detected - consider updating staging environment" : ""}}
        {{outputs.schema-consistency.prod_schema_version != outputs.staging-schema.staging_schema_version ? "⚠️ Schema version mismatch - verify migration deployment" : ""}}
        {{outputs.prod-staging-features.prod_feature_count != outputs.staging-features.staging_feature_count ? "⚠️ Feature flag drift - review feature flag synchronization" : ""}}
        
        Overall Status: {{
          outputs.primary-version.primary_version == outputs.staging-version.staging_version &&
          outputs.schema-consistency.prod_schema_version == outputs.staging-schema.staging_schema_version ?
          "✅ ENVIRONMENTS CONSISTENT" : "⚠️ CONSISTENCY ISSUES DETECTED"
        }}
        
        Generated: {{unixtime()}}
```

## ステップ5: 自動デプロイパイプライン統合

CI/CDパイプラインと統合するスクリプトを作成します：

**scripts/deploy-validation.sh:**
```bash
#!/bin/bash

set -e

# 設定
ENVIRONMENT=${1:-staging}
BUILD_VERSION=${2:-$(git rev-parse --short HEAD)}
DEPLOYMENT_ID=${3:-$(date +%s)}

# パラメータ検証
if [[ ! "$ENVIRONMENT" =~ ^(development|staging|production)$ ]]; then
    echo "❌ Invalid environment: $ENVIRONMENT"
    echo "Usage: $0 <environment> [build_version] [deployment_id]"
    exit 1
fi

echo "🚀 Starting deployment validation for $ENVIRONMENT environment"
echo "📦 Build Version: $BUILD_VERSION"
echo "🆔 Deployment ID: $DEPLOYMENT_ID"

# 環境変数をエクスポート
export ENVIRONMENT=$ENVIRONMENT
export BUILD_VERSION=$BUILD_VERSION
export DEPLOYMENT_ID=$DEPLOYMENT_ID
export DEPLOYMENT_TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# 環境固有の変数を設定
case $ENVIRONMENT in
    "development")
        export API_AUTH_TOKEN=$DEV_API_TOKEN
        export SMTP_USERNAME=$DEV_SMTP_USERNAME
        export SMTP_PASSWORD=$DEV_SMTP_PASSWORD
        ;;
    "staging")
        export API_AUTH_TOKEN=$STAGING_API_TOKEN
        export SMTP_USERNAME=$STAGING_SMTP_USERNAME
        export SMTP_PASSWORD=$STAGING_SMTP_PASSWORD
        ;;
    "production")
        export API_AUTH_TOKEN=$PROD_API_TOKEN
        export SMTP_USERNAME=$PROD_SMTP_USERNAME
        export SMTP_PASSWORD=$PROD_SMTP_PASSWORD
        ;;
esac

echo "🏥 Running health check..."
if probe config/base.yml,config/environments/${ENVIRONMENT}.yml,workflows/health-check.yml; then
    echo "✅ Health check passed"
else
    echo "❌ Health check failed"
    exit 1
fi

echo "🔍 Running deployment validation..."
if probe config/base.yml,config/environments/${ENVIRONMENT}.yml,workflows/deployment-validation.yml; then
    echo "✅ Deployment validation passed"
else
    echo "❌ Deployment validation failed"
    exit 1
fi

echo "🔄 Running consistency checks..."
if [[ "$ENVIRONMENT" != "development" ]]; then
    if probe config/base.yml,config/environments/${ENVIRONMENT}.yml,workflows/consistency-check.yml; then
        echo "✅ Consistency checks passed"
    else
        echo "⚠️ Consistency issues detected (non-blocking)"
    fi
else
    echo "⏭️ Skipping consistency checks for development environment"
fi

echo "🎉 Deployment validation completed successfully for $ENVIRONMENT"
echo "📊 Deployment ID: $DEPLOYMENT_ID"
echo "⏰ Completed at: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
```

## ステップ6: マルチ環境テストの実行

包括的なマルチ環境テストを実行します：

```bash
# 個別環境のテスト
probe config/base.yml,config/environments/development.yml,workflows/health-check.yml
probe config/base.yml,config/environments/staging.yml,workflows/deployment-validation.yml
probe config/base.yml,config/environments/production.yml,workflows/health-check.yml

# 完全なデプロイ検証を実行
./scripts/deploy-validation.sh staging v2.1.0
./scripts/deploy-validation.sh production v2.1.0

# 一貫性チェックを実行
probe config/base.yml,workflows/consistency-check.yml

# すべての環境を順次実行
for env in development staging production; do
    echo "Testing $env environment..."
    probe config/base.yml,config/environments/${env}.yml,workflows/health-check.yml
done
```

## ステップ7: 高度なマルチ環境パターン

### Blue-Greenデプロイテスト

Blue-Greenデプロイのテストを作成します：

```yaml
# workflows/blue-green-validation.yml
name: "Blue-Green Deployment Validation"

jobs:
- name: blue-green-test
  steps:
    - name: "Test Blue Environment"
      id: blue-test
      uses: http
      with:
        url: "{{BLUE_URL}}/health"
      test: res.code == 200
      
    - name: "Test Green Environment"
      id: green-test
      uses: http
      with:
        url: "{{GREEN_URL}}/health"
      test: res.code == 200
      
    - name: "Compare Environment Performance"
      echo: |
        Blue Environment: {{outputs.blue-test.time}}ms
        Green Environment: {{outputs.green-test.time}}ms
        Performance Difference: {{Math.abs(outputs.blue-test.time - outputs.green-test.time)}}ms
```

### カナリアデプロイテスト

カナリアデプロイを監視します：

```yaml
# workflows/canary-validation.yml
name: "Canary Deployment Validation"

jobs:
- name: canary-monitoring
  steps:
    - name: "Monitor Canary Traffic"
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/metrics/canary"
      test: |
        res.code == 200 &&
        res.body.json.canaryTrafficPercent <= vars.MAX_CANARY_TRAFFIC &&
        res.body.json.canaryErrorRate < vars.MAX_CANARY_ERROR_RATE
```

## トラブルシューティング

### よくある問題

**環境設定のミスマッチ:**
```bash
# デプロイ前に設定を検証
probe config/base.yml,config/environments/staging.yml --validate-only
```

**バージョンドリフト検出:**
```bash
# 環境間のバージョン一貫性をチェック
probe workflows/consistency-check.yml --verbose
```

**パフォーマンス回帰:**
```bash
# 環境間のパフォーマンスを比較
probe config/base.yml,config/environments/production.yml,workflows/performance-validation.yml
```

## 次のステップ

マルチ環境デプロイテストシステムが完成しました！以下の拡張を検討してください：

1. **高度な監視** - ビジネスメトリクス検証の追加
2. **自動ロールバック** - 失敗時のロールバックトリガー
3. **段階的デプロイ** - 段階的ロールアウトテストの実装
4. **コンプライアンステスト** - セキュリティとコンプライアンス検証の追加
5. **パフォーマンスベンチマーク** - 履歴パフォーマンス比較

## 関連リソース

- **[初めての監視システムチュートリアル](../first-monitoring-system/)** - 基本的な監視セットアップ
- **[APIテストパイプラインチュートリアル](../api-testing-pipeline/)** - 包括的なAPIテスト
- **[ハウツー: モニタリングワークフロー](../../how-tos/monitoring-workflows/)** - 高度な監視パターン
- **[ハウツー: 環境管理](../../how-tos/environment-management/)** - 環境設定戦略