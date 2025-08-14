# 環境管理

このガイドでは、設定の構成、環境固有の設定、およびデプロイメント戦略を使用して、複数の環境（開発、ステージング、プロダクション）間でProbeワークフローを管理する方法について説明します。

## 基本的な環境設定

### 単一ワークフロー、複数環境

複数の環境で動作するベースワークフローを作成します：

**base-workflow.yml:**
```yaml
name: Multi-Environment API Test
description: API testing workflow that adapts to different environments

vars:
  api_base_url: "{{API_BASE_URL}}"
  environment: "{{ENVIRONMENT ?? 'Unknown'}}"
  default_timeout: "{{DEFAULT_TIMEOUT ?? '30s'}}"

jobs:
- name: API Health Check
  defaults:
    http:
      timeout: "{{vars.default_timeout}}"
      headers:
        User-Agent: "Probe Test Agent"
        Accept: "application/json"
  steps:
    - name: Health Endpoint Test
      uses: http
      with:
        url: "{{vars.api_base_url}}/health"
      test: res.code == 200
      outputs:
        api_healthy: res.code == 200
        response_time: res.time
        api_version: res.body.json.version

    - name: Database Health Check
      uses: http
      with:
        url: "{{vars.api_base_url}}/health/database"
      test: res.code == 200
      outputs:
        database_healthy: res.code == 200
        db_response_time: res.time

- name: Environment Report
  needs: [api-health-check]
  steps:
    - name: Environment Summary
      echo: |
        🌍 Environment Test Report
        =========================
        
        Environment: {{vars.environment}}
        API Base URL: {{vars.api_base_url}}
        
        Health Check Results:
        API Health: {{outputs.api-health-check.api_healthy ? "✅ Healthy" : "❌ Down"}} ({{outputs.api-health-check.response_time}}ms)
        Database: {{outputs.api-health-check.database_healthy ? "✅ Healthy" : "❌ Down"}} ({{outputs.api-health-check.db_response_time}}ms)
        API Version: {{outputs.api-health-check.api_version}}
        
        Environment-Specific Notes:
        {{vars.environment == "development" ? "• Development environment - extended timeouts enabled" : ""}}
        {{vars.environment == "staging" ? "• Staging environment - production-like testing" : ""}}
        {{vars.environment == "production" ? "• Production environment - strict validation" : ""}}
```

**development.yml:**
```yaml
vars:
  ENVIRONMENT: development
  API_BASE_URL: http://localhost:3000
  DEFAULT_TIMEOUT: 60s

# Development-specific additional checks
jobs:
- name: Development Environment Checks
  id: dev-specific-checks
  needs: [api-health-check]
  defaults:
    http:
      timeout: 60s  # More lenient for development
  steps:
    - name: Hot Reload Check
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/dev/hot-reload-status"
      test: res.code == 200
      continue_on_error: true
      outputs:
        hot_reload_enabled: res.body.json.enabled

    - name: Debug Endpoints Check
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/debug/info"
      test: res.code == 200
      continue_on_error: true
      outputs:
        debug_info_available: res.code == 200

    - name: Development Summary
      echo: |
        🛠️ Development Environment Status:
        Hot Reload: {{outputs.hot_reload_enabled ? "✅ Enabled" : "❌ Disabled"}}
        Debug Info: {{outputs.debug_info_available ? "✅ Available" : "❌ Not Available"}}
```

**staging.yml:**
```yaml
vars:
  ENVIRONMENT: staging
  API_BASE_URL: https://api.staging.yourcompany.com
  DEFAULT_TIMEOUT: 30s

# Staging-specific additional checks
jobs:
- name: Staging Environment Checks
  id: staging-specific-checks
  needs: [api-health-check]
  defaults:
    http:
      timeout: 30s
      headers:
        X-Environment: staging
  steps:
    - name: Load Balancer Check
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/health/load-balancer"
      test: res.code == 200
      outputs:
        load_balancer_healthy: res.code == 200
        backend_count: res.body.json.active_backends

    - name: Cache Layer Check
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/health/cache"
      test: res.code == 200
      outputs:
        cache_healthy: res.code == 200
        cache_hit_rate: res.body.json.hit_rate

    - name: Integration Tests
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/test/integration"
      test: res.code == 200 && res.body.json.all_tests_passed == true
      outputs:
        integration_tests_passed: res.body.json.all_tests_passed

    - name: Staging Summary
      echo: |
        🧪 Staging Environment Status:
        Load Balancer: {{outputs.load_balancer_healthy ? "✅ Healthy" : "❌ Issues"}} ({{outputs.backend_count}} backends)
        Cache Layer: {{outputs.cache_healthy ? "✅ Healthy" : "❌ Issues"}} ({{(outputs.cache_hit_rate * 100)}}% hit rate)
        Integration Tests: {{outputs.integration_tests_passed ? "✅ Passed" : "❌ Failed"}}
```

**production.yml:**
```yaml
vars:
  ENVIRONMENT: production
  API_BASE_URL: https://api.yourcompany.com
  DEFAULT_TIMEOUT: 10s

# Production-specific additional checks
jobs:
- name: Production Environment Checks
  id: production-specific-checks
  needs: [api-health-check]
  defaults:
    http:
      timeout: 10s  # Strict timeouts for production
      headers:
        X-Environment: production
  steps:
    - name: SSL Certificate Check
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/health/ssl"
      test: |
        res.code == 200 &&
        res.body.json.certificate_valid == true &&
        res.body.json.days_until_expiry > 30
      outputs:
        ssl_valid: res.body.json.certificate_valid
        ssl_days_remaining: res.body.json.days_until_expiry

    - name: Performance SLA Check
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/health/performance"
      test: |
        res.code == 200 &&
        res.body.json.avg_response_time < 500 &&
        res.body.json.success_rate > 0.999
      outputs:
        sla_met: res.body.json.avg_response_time < 500 && res.body.json.success_rate > 0.999
        avg_response_time: res.body.json.avg_response_time
        success_rate: res.body.json.success_rate

    - name: Security Compliance Check
      uses: http
      with:
        url: "{{vars.API_BASE_URL}}/health/security"
      test: |
        res.code == 200 &&
        res.body.json.security_score >= 0.95
      outputs:
        security_compliant: res.body.json.security_score >= 0.95
        security_score: res.body.json.security_score

    - name: Production Summary
      echo: |
        🏭 Production Environment Status:
        SSL Certificate: {{outputs.ssl_valid ? "✅ Valid" : "❌ Invalid"}} ({{outputs.ssl_days_remaining}} days remaining)
        Performance SLA: {{outputs.sla_met ? "✅ Met" : "❌ Violated"}}
          - Avg Response: {{outputs.avg_response_time}}ms
          - Success Rate: {{(outputs.success_rate * 100)}}%
        Security Compliance: {{outputs.security_compliant ? "✅ Compliant" : "❌ Non-Compliant"}} ({{(outputs.security_score * 100)}}%)
```

**使用方法:**
```bash
# 開発環境
probe base-workflow.yml,development.yml

# ステージング環境  
probe base-workflow.yml,staging.yml

# プロダクション環境
probe base-workflow.yml,production.yml
```

## 高度な環境管理

### 環境固有の機能フラグ

異なる環境でテストされる機能を制御します：

**feature-flags.yml:**
```yaml
# 機能フラグ設定
vars:
  api_base_url: "{{API_BASE_URL}}"
  admin_token: "{{ADMIN_TOKEN}}"
  environment: "{{ENVIRONMENT}}"
  # コア機能（常に有効）
  feature_user_management: true
  feature_basic_api: true
  # 環境固有の機能
  feature_beta_api: "{{vars.environment != 'production'}}"
  feature_admin_tools: "{{vars.environment == 'development'}}"
  feature_performance_testing: "{{vars.environment != 'development'}}"
  feature_security_scanning: "{{vars.environment == 'production'}}"

jobs:
- name: Core Feature Tests
  steps:
    - name: User Management Test
      if: vars.feature_user_management == true
      uses: http
      with:
        url: "{{vars.api_base_url}}/users"
      test: res.code == 200
      outputs:
        user_management_working: res.code == 200

    - name: Basic API Test
      if: vars.feature_basic_api == true
      uses: http
      with:
        url: "{{vars.api_base_url}}/api/basic"
      test: res.code == 200
      outputs:
        basic_api_working: res.code == 200

- name: Beta Feature Tests
  steps:
    - name: Beta API Test
      if: vars.feature_beta_api == true
      uses: http
      with:
        url: "{{vars.api_base_url}}/api/beta"
      test: res.code == 200
      continue_on_error: true
      outputs:
        beta_api_working: res.code == 200

- name: Admin Feature Tests
  steps:
    - name: Admin Tools Test
      if: vars.feature_admin_tools == true
      uses: http
      with:
        url: "{{vars.api_base_url}}/admin/tools"
        headers:
          Authorization: "Bearer {{vars.admin_token}}"
      test: res.code == 200
      continue_on_error: true
      outputs:
        admin_tools_working: res.code == 200

- name: Performance Tests
  if: vars.feature_performance_testing == true
  steps:
    - name: Load Test
      uses: http
      with:
        url: "{{vars.api_base_url}}/test/load"
        method: POST
        body: |
          {
            "concurrent_users": {{vars.environment == "staging" ? 10 : 50}},
            "duration_seconds": {{vars.environment == "staging" ? 60 : 300}}
          }
      test: res.code == 200
      outputs:
        load_test_passed: res.body.json.success

- name: Security Tests
  if: vars.feature_security_scanning == true
  steps:
    - name: Security Scan
      uses: http
      with:
        url: "{{vars.api_base_url}}/security/scan"
        method: POST
      test: res.code == 200 && res.body.json.vulnerabilities_found == 0
      outputs:
        security_scan_clean: res.body.json.vulnerabilities_found == 0

- name: Feature Test Summary
  needs: [core-feature-tests, beta-feature-tests, admin-feature-tests, performance-tests, security-tests]
  steps:
    - name: Environment Feature Report
      echo: |
        🚀 Feature Test Summary for {{vars.environment}}:
        ================================================
        
        CORE FEATURES:
        {{vars.feature_user_management == true ? "User Management: " + (outputs.core-feature-tests.user_management_working ? "✅ Working" : "❌ Failed") : "User Management: ⏸️ Disabled"}}
        {{vars.feature_basic_api == true ? "Basic API: " + (outputs.core-feature-tests.basic_api_working ? "✅ Working" : "❌ Failed") : "Basic API: ⏸️ Disabled"}}
        
        BETA FEATURES:
        {{vars.feature_beta_api == true ? "Beta API: " + (outputs.beta-feature-tests.beta_api_working ? "✅ Working" : "❌ Failed") : "Beta API: ⏸️ Disabled"}}
        
        ADMIN FEATURES:
        {{vars.feature_admin_tools == true ? "Admin Tools: " + (outputs.admin-feature-tests.admin_tools_working ? "✅ Working" : "❌ Failed") : "Admin Tools: ⏸️ Disabled"}}
        
        PERFORMANCE TESTING:
        {{vars.feature_performance_testing == true ? "Load Testing: " + (outputs.performance-tests.load_test_passed ? "✅ Passed" : "❌ Failed") : "Performance Testing: ⏸️ Disabled"}}
        
        SECURITY TESTING:
        {{vars.feature_security_scanning == true ? "Security Scan: " + (outputs.security-tests.security_scan_clean ? "✅ Clean" : "❌ Vulnerabilities Found") : "Security Testing: ⏸️ Disabled"}}
        
        Environment Configuration:
        Features enabled: {{
          (vars.feature_user_management == true ? 1 : 0) +
          (vars.feature_basic_api == true ? 1 : 0) +
          (vars.feature_beta_api == true ? 1 : 0) +
          (vars.feature_admin_tools == true ? 1 : 0) +
          (vars.feature_performance_testing == true ? 1 : 0) +
          (vars.feature_security_scanning == true ? 1 : 0)
        }} / 6
```

### 認証情報とシークレット管理

環境固有の認証情報を安全に管理します：

**credentials-development.yml:**
```yaml
vars:
  # 開発認証情報（機密性が低い）
  API_TOKEN: dev_token_12345
  DB_PASSWORD: dev_password
  ADMIN_TOKEN: dev_admin_token
  
  # 開発サービスURL
  API_BASE_URL: http://localhost:3000
  DB_URL: localhost:5432
  CACHE_URL: localhost:6379
  
  # 開発固有設定
  LOG_LEVEL: debug
  RATE_LIMIT_DISABLED: true
  SECURITY_CHECKS_RELAXED: true
```

**credentials-staging.yml:**
```yaml
vars:
  # ステージング認証情報（環境変数から）
  api_token: "{{STAGING_API_TOKEN}}"
  db_password: "{{STAGING_DB_PASSWORD}}"
  admin_token: "{{STAGING_ADMIN_TOKEN}}"
  
  # ステージングサービスURL
  API_BASE_URL: https://api.staging.yourcompany.com
  DB_URL: staging-db.yourcompany.com:5432
  CACHE_URL: staging-cache.yourcompany.com:6379
  
  # ステージング固有設定
  LOG_LEVEL: info
  RATE_LIMIT_DISABLED: false
  SECURITY_CHECKS_RELAXED: false
```

**credentials-production.yml:**
```yaml
vars:
  # プロダクション認証情報（セキュアな環境変数から）
  api_token: "{{PROD_API_TOKEN}}"
  db_password: "{{PROD_DB_PASSWORD}}"
  admin_token: "{{PROD_ADMIN_TOKEN}}"
  
  # プロダクションサービスURL
  API_BASE_URL: https://api.yourcompany.com
  DB_URL: prod-db.yourcompany.com:5432
  CACHE_URL: prod-cache.yourcompany.com:6379
  
  # プロダクション固有設定
  LOG_LEVEL: warn
  RATE_LIMIT_DISABLED: false
  SECURITY_CHECKS_RELAXED: false
  
  # プロダクション専用設定
  MONITORING_ENABLED: true
  ALERTS_ENABLED: true
  AUDIT_LOGGING: true
```

### 環境検証ワークフロー

テスト実行前に環境設定を検証します：

**environment-validation.yml:**
```yaml
name: Environment Validation
description: Validate environment configuration and prerequisites

vars:
  environment: "{{ENVIRONMENT}}"
  api_base_url: "{{API_BASE_URL}}"
  api_token: "{{API_TOKEN}}"
  log_level: "{{LOG_LEVEL}}"
  default_timeout: "{{DEFAULT_TIMEOUT}}"

jobs:
- name: Environment Configuration Validation
  steps:
    - name: Required Environment Variables Check
      echo: |
        🔍 Environment Variables Validation:
        
        Required Variables:
        ENVIRONMENT: {{vars.environment ? "✅ Set (" + vars.environment + ")" : "❌ Missing"}}
        API_BASE_URL: {{vars.api_base_url ? "✅ Set (" + vars.api_base_url + ")" : "❌ Missing"}}
        API_TOKEN: {{vars.api_token ? "✅ Set (***)" : "❌ Missing"}}
        
        Optional Variables:
        LOG_LEVEL: {{vars.log_level ? "✅ Set (" + vars.log_level + ")" : "⚠️ Using default"}}
        DEFAULT_TIMEOUT: {{vars.default_timeout ? "✅ Set (" + vars.default_timeout + ")" : "⚠️ Using default"}}
        
        Validation Status: {{
          vars.environment && vars.api_base_url && vars.api_token ? "✅ Valid" : "❌ Invalid"
        }}

    - name: Environment-Specific Validation
      echo: |
        📋 Environment-Specific Validation:
        
        {{vars.environment == "development" ? "Development Environment:" : ""}}
        {{vars.environment == "development" ? "• Extended timeouts enabled" : ""}}
        {{vars.environment == "development" ? "• Debug features available" : ""}}
        {{vars.environment == "development" ? "• Security checks relaxed" : ""}}
        
        {{vars.environment == "staging" ? "Staging Environment:" : ""}}
        {{vars.environment == "staging" ? "• Production-like configuration" : ""}}
        {{vars.environment == "staging" ? "• Integration testing enabled" : ""}}
        {{vars.environment == "staging" ? "• Performance testing included" : ""}}
        
        {{vars.environment == "production" ? "Production Environment:" : ""}}
        {{vars.environment == "production" ? "• Strict timeouts enforced" : ""}}
        {{vars.environment == "production" ? "• Security scanning enabled" : ""}}
        {{vars.environment == "production" ? "• Full monitoring active" : ""}}

    - name: Service Connectivity Pre-Check
      uses: http
      with:
        url: "{{vars.api_base_url}}/health"
        timeout: 10s
      test: res.code == 200
      outputs:
        connectivity_ok: res.code == 200
        api_version: res.body.json.version
        environment_confirmed: res.body.json.environment

    - name: Authentication Pre-Check
      uses: http
      with:
        url: "{{vars.api_base_url}}/auth/validate"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: res.code == 200
      outputs:
        auth_valid: res.code == 200
        token_expires_in: res.body.json.expires_in

    - name: Validation Summary
      echo: |
        ✅ Environment Validation Results:
        
        Connectivity: {{outputs.connectivity_ok ? "✅ Connected" : "❌ Failed"}}
        API Version: {{outputs.api_version}}
        Environment Match: {{outputs.environment_confirmed == vars.environment ? "✅ Confirmed" : "⚠️ Mismatch"}}
        Authentication: {{outputs.auth_valid ? "✅ Valid" : "❌ Invalid"}}
        {{outputs.auth_valid ? "Token Expires In: " + outputs.token_expires_in + " seconds" : ""}}
        
        Environment Ready: {{
          outputs.connectivity_ok && 
          outputs.auth_valid && 
          outputs.environment_confirmed == vars.environment
          ? "🟢 YES" : "🔴 NO"
        }}
```

## CI/CD統合

### GitHub Actions統合

自動環境テスト用にCI/CDパイプラインと統合します：

**.github/workflows/probe-tests.yml:**
```yaml
name: Probe Environment Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  development-tests:
    name: Development Environment Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Probe
        run: |
          curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o probe
          chmod +x probe
          sudo mv probe /usr/local/bin/
      
      - name: Run Development Tests
        env:
          DEV_API_TOKEN: ${{ secrets.DEV_API_TOKEN }}
        run: |
          probe workflows/base-workflow.yml,environments/development.yml,credentials/development.yml

  staging-tests:
    name: Staging Environment Tests
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Probe
        run: |
          curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o probe
          chmod +x probe
          sudo mv probe /usr/local/bin/
      
      - name: Validate Staging Environment
        env:
          STAGING_API_TOKEN: ${{ secrets.STAGING_API_TOKEN }}
          STAGING_DB_PASSWORD: ${{ secrets.STAGING_DB_PASSWORD }}
        run: |
          probe workflows/environment-validation.yml,environments/staging.yml,credentials/staging.yml
      
      - name: Run Staging Tests
        env:
          STAGING_API_TOKEN: ${{ secrets.STAGING_API_TOKEN }}
          STAGING_DB_PASSWORD: ${{ secrets.STAGING_DB_PASSWORD }}
        run: |
          probe workflows/base-workflow.yml,environments/staging.yml,credentials/staging.yml,features/feature-flags.yml

  production-smoke-tests:
    name: Production Smoke Tests
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Probe
        run: |
          curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o probe
          chmod +x probe
          sudo mv probe /usr/local/bin/
      
      - name: Production Smoke Tests
        env:
          PROD_API_TOKEN: ${{ secrets.PROD_API_TOKEN }}
          PROD_DB_PASSWORD: ${{ secrets.PROD_DB_PASSWORD }}
        run: |
          probe workflows/smoke-test.yml,environments/production.yml,credentials/production.yml
```

### 環境固有のテストスイート

異なる環境に対応する様々なテストスイートを作成します：

**smoke-test.yml（プロダクション用）:**
```yaml
name: Production Smoke Test
description: Minimal smoke test for production environment

vars:
  api_base_url: "{{API_BASE_URL}}"

jobs:
- name: Critical Endpoints Check
  steps:
    - name: Health Check
      uses: http
      with:
        url: "{{vars.api_base_url}}/health"
      test: res.code == 200
      outputs:
        api_healthy: res.code == 200

    - name: Authentication Check
      uses: http
      with:
        url: "{{vars.api_base_url}}/auth/health"
      test: res.code == 200
      outputs:
        auth_healthy: res.code == 200

    - name: Database Check
      uses: http
      with:
        url: "{{vars.api_base_url}}/health/database"
      test: res.code == 200
      outputs:
        db_healthy: res.code == 200

- name: Smoke Test Summary
  needs: [critical-endpoints-check]
  steps:
    - name: Production Health Summary
      echo: |
        🏭 Production Smoke Test Results:
        
        Critical Systems:
        API: {{outputs.critical-endpoints-check.api_healthy ? "✅ Healthy" : "🚨 DOWN"}}
        Authentication: {{outputs.critical-endpoints-check.auth_healthy ? "✅ Healthy" : "🚨 DOWN"}}
        Database: {{outputs.critical-endpoints-check.db_healthy ? "✅ Healthy" : "🚨 DOWN"}}
        
        Overall Status: {{
          outputs.critical-endpoints-check.api_healthy &&
          outputs.critical-endpoints-check.auth_healthy &&
          outputs.critical-endpoints-check.db_healthy
          ? "🟢 ALL SYSTEMS OPERATIONAL" : "🔴 CRITICAL ISSUES DETECTED"
        }}
```

**comprehensive-test.yml（ステージング用）:**
```yaml
name: Comprehensive Staging Test
description: Full test suite for staging environment validation

vars:
  api_base_url: "{{API_BASE_URL}}"

jobs:
- name: API Test Suite
  steps:
    - name: User Management API
      uses: http
      with:
        url: "{{vars.api_base_url}}/users"
      test: res.code == 200

    - name: Order Management API
      uses: http
      with:
        url: "{{vars.api_base_url}}/orders"
      test: res.code == 200

    - name: Product Catalog API
      uses: http
      with:
        url: "{{vars.api_base_url}}/products"
      test: res.code == 200

- name: Integration Tests
  needs: [api-test-suite]
  steps:
    - name: User-Order Integration
      uses: http
      with:
        url: "{{vars.api_base_url}}/test/user-order-flow"
      test: res.code == 200 && res.body.json.test_passed == true

    - name: Payment Integration
      uses: http
      with:
        url: "{{vars.api_base_url}}/test/payment-flow"
      test: res.code == 200 && res.body.json.test_passed == true

- name: Performance Validation
  needs: [integration-tests]
  steps:
    - name: Load Test
      uses: http
      with:
        url: "{{vars.api_base_url}}/test/load"
        method: POST
        body: |
          {
            "concurrent_users": 10,
            "duration_seconds": 60
          }
      test: res.code == 200 && res.body.json.success_rate > 0.95
```

## 環境モニタリングとアラート

### 環境ヘルスモニタリング

各環境のヘルスを継続的に監視します：

**environment-monitor.yml:**
```yaml
name: Environment Health Monitor
description: Continuous monitoring of environment health

vars:
  api_base_url: "{{API_BASE_URL}}"
  environment: "{{ENVIRONMENT}}"
  smtp_host: "{{SMTP_HOST}}"
  smtp_username: "{{SMTP_USERNAME}}"
  smtp_password: "{{SMTP_PASSWORD}}"
  monitoring_interval: "{{MONITORING_INTERVAL ?? '300'}}"  # 5分
  alert_threshold: "{{ALERT_THRESHOLD ?? '2'}}"        # 連続2回失敗後にアラート

jobs:
- name: Environment Health Check
  steps:
    - name: System Resources Check
      id: resources
      uses: http
      with:
        url: "{{vars.api_base_url}}/health/resources"
      test: |
        res.code == 200 &&
        res.body.json.cpu_usage < 80 &&
        res.body.json.memory_usage < 80 &&
        res.body.json.disk_usage < 90
      continue_on_error: true
      outputs:
        resources_healthy: |
          res.code == 200 &&
          res.body.json.cpu_usage < 80 &&
          res.body.json.memory_usage < 80 &&
          res.body.json.disk_usage < 90
        cpu_usage: res.body.json.cpu_usage
        memory_usage: res.body.json.memory_usage
        disk_usage: res.body.json.disk_usage

    - name: Service Dependencies Check
      id: dependencies
      uses: http
      with:
        url: "{{vars.api_base_url}}/health/dependencies"
      test: |
        res.code == 200 &&
        res.body.json.all_dependencies_healthy == true
      continue_on_error: true
      outputs:
        dependencies_healthy: res.body.json.all_dependencies_healthy
        unhealthy_services: res.body.json.unhealthy_services

    - name: Environment-Specific Checks
      echo: |
        Environment-specific validation for {{vars.environment}}
      outputs:
        env_specific_checks: |
          {{vars.environment == "production" ? "SSL, Security, Performance" :
            vars.environment == "staging" ? "Integration, Load Testing" :
            "Development Tools, Debug Features"}}

- name: Environment Alerting
  needs: [environment-health-check]
  if: |
    !outputs.environment-health-check.resources_healthy ||
    !outputs.environment-health-check.dependencies_healthy
  steps:
    - name: Environment Alert
      uses: smtp
      with:
        host: "{{vars.smtp_host}}"
        port: 587
        username: "{{vars.smtp_username}}"
        password: "{{vars.smtp_password}}"
        from: "environment-alerts@yourcompany.com"
        to: ["devops@yourcompany.com"]
        subject: "🚨 Environment Health Alert - {{vars.environment}}"
        body: |
          ENVIRONMENT HEALTH ALERT
          ========================
          
          Environment: {{vars.environment}}
          Time: {{unixtime()}}
          
          RESOURCE STATUS:
          {{outputs.environment-health-check.resources_healthy ? "✅ Resources Healthy" : "❌ Resource Issues"}}
          {{!outputs.environment-health-check.resources_healthy ? "CPU Usage: " + outputs.environment-health-check.cpu_usage + "%" : ""}}
          {{!outputs.environment-health-check.resources_healthy ? "Memory Usage: " + outputs.environment-health-check.memory_usage + "%" : ""}}
          {{!outputs.environment-health-check.resources_healthy ? "Disk Usage: " + outputs.environment-health-check.disk_usage + "%" : ""}}
          
          DEPENDENCIES STATUS:
          {{outputs.environment-health-check.dependencies_healthy ? "✅ All Dependencies Healthy" : "❌ Dependency Issues"}}
          {{!outputs.environment-health-check.dependencies_healthy ? "Unhealthy Services: " + outputs.environment-health-check.unhealthy_services : ""}}
          
          ACTION REQUIRED: Investigate {{vars.environment}} environment immediately

- name: Health Summary
  needs: [environment-health-check]
  if: |
    outputs.environment-health-check.resources_healthy &&
    outputs.environment-health-check.dependencies_healthy
  steps:
    - name: All Systems Healthy
      echo: |
        ✅ {{vars.environment}} Environment Health Check
        
        All systems operational:
        • Resources: CPU {{outputs.environment-health-check.cpu_usage}}%, Memory {{outputs.environment-health-check.memory_usage}}%, Disk {{outputs.environment-health-check.disk_usage}}%
        • Dependencies: All healthy
        • Environment: {{vars.environment}}
        
        Next check in {{vars.monitoring_interval}} seconds
```

## ベストプラクティス

### 1. 環境分離

```yaml
# 良い例: 明確な環境分離
environments/
├── development.yml
├── staging.yml
└── production.yml

# 良い例: 環境固有の認証情報
credentials/
├── development.yml
├── staging.yml
└── production.yml（環境変数のみ使用）
```

### 2. 段階的テスト

```yaml
# 良い例: テストパイプラインの進行
Development → Unit Tests
Staging → Integration Tests + Performance Tests  
Production → Smoke Tests + Monitoring
```

### 3. 設定検証

```yaml
# 良い例: テスト実行前の検証
vars:
  environment: "{{ENVIRONMENT}}"
  api_base_url: "{{API_BASE_URL}}"
  api_token: "{{API_TOKEN}}"

- name: Pre-Test Validation
  echo: |
    Environment: {{vars.environment ? "✅" : "❌"}}
    API URL: {{vars.api_base_url ? "✅" : "❌"}}
    Credentials: {{vars.api_token ? "✅" : "❌"}}
```

### 4. セキュアなシークレット管理

```yaml
# 良い例: シークレットに環境変数を使用
vars:
  api_token: "{{PROD_API_TOKEN}}"  # セキュアな環境変数から

# 避ける: ハードコードされたシークレット
vars:
  api_token: "hardcoded_secret_123"  # これは絶対にしない
```

## 次のステップ

効果的に環境を管理できるようになったので、次を探索してください：

- **[モニタリングワークフロー](../monitoring-workflows/)** - 全環境の包括的モニタリングを構築
- **[APIテスト](../api-testing/)** - 環境固有のAPIテストスイートを作成
- **[エラーハンドリング戦略](../error-handling-strategies/)** - 環境固有の障害を処理

環境管理は、ソフトウェア開発ライフサイクル全体にわたる信頼性の高いテストにとって重要です。これらのパターンを使用して、一貫性があり、セキュアで、保守しやすい環境設定を作成しましょう。