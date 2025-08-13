# Environment Management

This guide shows you how to manage Probe workflows across multiple environments (development, staging, production) using configuration composition, environment-specific settings, and deployment strategies.

## Basic Environment Configuration

### Single Workflow, Multiple Environments

Create a base workflow that works across environments:

**base-workflow.yml:**
```yaml
name: Multi-Environment API Test
description: API testing workflow that adapts to different environments

vars:
  api_base_url: "{{API_BASE_URL}}"
  environment: "{{ENVIRONMENT ?? 'Unknown'}}"
  default_timeout: "{{DEFAULT_TIMEOUT ?? '30s'}}"

defaults:
  http:
    timeout: "{{vars.default_timeout}}"
    headers:
      User-Agent: "Probe Test Agent"
      Accept: "application/json"

jobs:
  api-health-check:
    name: API Health Check
    steps:
      - name: Health Endpoint Test
        action: http
        with:
          url: "{{vars.api_base_url}}/health"
        test: res.status == 200
        outputs:
          api_healthy: res.status == 200
          response_time: res.time
          api_version: res.json.version

      - name: Database Health Check
        action: http
        with:
          url: "{{vars.api_base_url}}/health/database"
        test: res.status == 200
        outputs:
          database_healthy: res.status == 200
          db_response_time: res.time

  environment-report:
    name: Environment Report
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
env:
  ENVIRONMENT: development
  API_BASE_URL: http://localhost:3000
  DEFAULT_TIMEOUT: 60s

defaults:
  http:
    timeout: 60s  # More lenient for development

# Development-specific additional checks
vars:
  api_base_url: "{{API_BASE_URL}}"

jobs:
  dev-specific-checks:
    name: Development Environment Checks
    needs: [api-health-check]
    steps:
      - name: Hot Reload Check
        action: http
        with:
          url: "{{vars.api_base_url}}/dev/hot-reload-status"
        test: res.status == 200
        continue_on_error: true
        outputs:
          hot_reload_enabled: res.json.enabled

      - name: Debug Endpoints Check
        action: http
        with:
          url: "{{vars.api_base_url}}/debug/info"
        test: res.status == 200
        continue_on_error: true
        outputs:
          debug_info_available: res.status == 200

      - name: Development Summary
        echo: |
          🛠️ Development Environment Status:
          Hot Reload: {{outputs.hot_reload_enabled ? "✅ Enabled" : "❌ Disabled"}}
          Debug Info: {{outputs.debug_info_available ? "✅ Available" : "❌ Not Available"}}
```

**staging.yml:**
```yaml
env:
  ENVIRONMENT: staging
  API_BASE_URL: https://api.staging.yourcompany.com
  DEFAULT_TIMEOUT: 30s

defaults:
  http:
    timeout: 30s
    headers:
      X-Environment: staging

# Staging-specific additional checks
vars:
  api_base_url: "{{API_BASE_URL}}"

jobs:
  staging-specific-checks:
    name: Staging Environment Checks
    needs: [api-health-check]
    steps:
      - name: Load Balancer Check
        action: http
        with:
          url: "{{vars.api_base_url}}/health/load-balancer"
        test: res.status == 200
        outputs:
          load_balancer_healthy: res.status == 200
          backend_count: res.json.active_backends

      - name: Cache Layer Check
        action: http
        with:
          url: "{{vars.api_base_url}}/health/cache"
        test: res.status == 200
        outputs:
          cache_healthy: res.status == 200
          cache_hit_rate: res.json.hit_rate

      - name: Integration Tests
        action: http
        with:
          url: "{{vars.api_base_url}}/test/integration"
        test: res.status == 200 && res.json.all_tests_passed == true
        outputs:
          integration_tests_passed: res.json.all_tests_passed

      - name: Staging Summary
        echo: |
          🧪 Staging Environment Status:
          Load Balancer: {{outputs.load_balancer_healthy ? "✅ Healthy" : "❌ Issues"}} ({{outputs.backend_count}} backends)
          Cache Layer: {{outputs.cache_healthy ? "✅ Healthy" : "❌ Issues"}} ({{(outputs.cache_hit_rate * 100)}}% hit rate)
          Integration Tests: {{outputs.integration_tests_passed ? "✅ Passed" : "❌ Failed"}}
```

**production.yml:**
```yaml
env:
  ENVIRONMENT: production
  API_BASE_URL: https://api.yourcompany.com
  DEFAULT_TIMEOUT: 10s

defaults:
  http:
    timeout: 10s  # Strict timeouts for production
    headers:
      X-Environment: production

# Production-specific additional checks
vars:
  api_base_url: "{{API_BASE_URL}}"

jobs:
  production-specific-checks:
    name: Production Environment Checks
    needs: [api-health-check]
    steps:
      - name: SSL Certificate Check
        action: http
        with:
          url: "{{vars.api_base_url}}/health/ssl"
        test: |
          res.status == 200 &&
          res.json.certificate_valid == true &&
          res.json.days_until_expiry > 30
        outputs:
          ssl_valid: res.json.certificate_valid
          ssl_days_remaining: res.json.days_until_expiry

      - name: Performance SLA Check
        action: http
        with:
          url: "{{vars.api_base_url}}/health/performance"
        test: |
          res.status == 200 &&
          res.json.avg_response_time < 500 &&
          res.json.success_rate > 0.999
        outputs:
          sla_met: res.json.avg_response_time < 500 && res.json.success_rate > 0.999
          avg_response_time: res.json.avg_response_time
          success_rate: res.json.success_rate

      - name: Security Compliance Check
        action: http
        with:
          url: "{{vars.api_base_url}}/health/security"
        test: |
          res.status == 200 &&
          res.json.security_score >= 0.95
        outputs:
          security_compliant: res.json.security_score >= 0.95
          security_score: res.json.security_score

      - name: Production Summary
        echo: |
          🏭 Production Environment Status:
          SSL Certificate: {{outputs.ssl_valid ? "✅ Valid" : "❌ Invalid"}} ({{outputs.ssl_days_remaining}} days remaining)
          Performance SLA: {{outputs.sla_met ? "✅ Met" : "❌ Violated"}}
            - Avg Response: {{outputs.avg_response_time}}ms
            - Success Rate: {{(outputs.success_rate * 100)}}%
          Security Compliance: {{outputs.security_compliant ? "✅ Compliant" : "❌ Non-Compliant"}} ({{(outputs.security_score * 100)}}%)
```

**Usage:**
```bash
# Development environment
probe base-workflow.yml,development.yml

# Staging environment  
probe base-workflow.yml,staging.yml

# Production environment
probe base-workflow.yml,production.yml
```

## Advanced Environment Management

### Environment-Specific Feature Flags

Control which features are tested in different environments:

**feature-flags.yml:**
```yaml
# Feature flags configuration
vars:
  api_base_url: "{{API_BASE_URL}}"
  admin_token: "{{ADMIN_TOKEN}}"
  environment: "{{ENVIRONMENT}}"
  # Core features (always enabled)
  feature_user_management: true
  feature_basic_api: true
  # Environment-specific features
  feature_beta_api: "{{vars.environment != 'production'}}"
  feature_admin_tools: "{{vars.environment == 'development'}}"
  feature_performance_testing: "{{vars.environment != 'development'}}"
  feature_security_scanning: "{{vars.environment == 'production'}}"

jobs:
  core-feature-tests:
    name: Core Feature Tests
    steps:
      - name: User Management Test
        if: vars.feature_user_management == true
        action: http
        with:
          url: "{{vars.api_base_url}}/users"
        test: res.status == 200
        outputs:
          user_management_working: res.status == 200

      - name: Basic API Test
        if: vars.feature_basic_api == true
        action: http
        with:
          url: "{{vars.api_base_url}}/api/basic"
        test: res.status == 200
        outputs:
          basic_api_working: res.status == 200

  beta-feature-tests:
    name: Beta Feature Tests
    steps:
      - name: Beta API Test
        if: vars.feature_beta_api == true
        action: http
        with:
          url: "{{vars.api_base_url}}/api/beta"
        test: res.status == 200
        continue_on_error: true
        outputs:
          beta_api_working: res.status == 200

  admin-feature-tests:
    name: Admin Feature Tests
    steps:
      - name: Admin Tools Test
        if: vars.feature_admin_tools == true
        action: http
        with:
          url: "{{vars.api_base_url}}/admin/tools"
          headers:
            Authorization: "Bearer {{vars.admin_token}}"
        test: res.status == 200
        continue_on_error: true
        outputs:
          admin_tools_working: res.status == 200

  performance-tests:
    name: Performance Tests
    if: vars.feature_performance_testing == true
    steps:
      - name: Load Test
        action: http
        with:
          url: "{{vars.api_base_url}}/test/load"
          method: POST
          body: |
            {
              "concurrent_users": {{vars.environment == "staging" ? 10 : 50}},
              "duration_seconds": {{vars.environment == "staging" ? 60 : 300}}
            }
        test: res.status == 200
        outputs:
          load_test_passed: res.json.success

  security-tests:
    name: Security Tests
    if: vars.feature_security_scanning == true
    steps:
      - name: Security Scan
        action: http
        with:
          url: "{{vars.api_base_url}}/security/scan"
          method: POST
        test: res.status == 200 && res.json.vulnerabilities_found == 0
        outputs:
          security_scan_clean: res.json.vulnerabilities_found == 0

  feature-summary:
    name: Feature Test Summary
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

### Credential and Secret Management

Manage environment-specific credentials securely:

**credentials-development.yml:**
```yaml
env:
  # Development credentials (less sensitive)
  API_TOKEN: dev_token_12345
  DB_PASSWORD: dev_password
  ADMIN_TOKEN: dev_admin_token
  
  # Development service URLs
  API_BASE_URL: http://localhost:3000
  DB_URL: localhost:5432
  CACHE_URL: localhost:6379
  
  # Development-specific settings
  LOG_LEVEL: debug
  RATE_LIMIT_DISABLED: true
  SECURITY_CHECKS_RELAXED: true
```

**credentials-staging.yml:**
```yaml
vars:
  # Staging credentials (from environment variables)
  api_token: "{{STAGING_API_TOKEN}}"
  db_password: "{{STAGING_DB_PASSWORD}}"
  admin_token: "{{STAGING_ADMIN_TOKEN}}"
  
  # Staging service URLs
  API_BASE_URL: https://api.staging.yourcompany.com
  DB_URL: staging-db.yourcompany.com:5432
  CACHE_URL: staging-cache.yourcompany.com:6379
  
  # Staging-specific settings
  LOG_LEVEL: info
  RATE_LIMIT_DISABLED: false
  SECURITY_CHECKS_RELAXED: false
```

**credentials-production.yml:**
```yaml
vars:
  # Production credentials (from secure environment variables)
  api_token: "{{PROD_API_TOKEN}}"
  db_password: "{{PROD_DB_PASSWORD}}"
  admin_token: "{{PROD_ADMIN_TOKEN}}"
  
  # Production service URLs
  API_BASE_URL: https://api.yourcompany.com
  DB_URL: prod-db.yourcompany.com:5432
  CACHE_URL: prod-cache.yourcompany.com:6379
  
  # Production-specific settings
  LOG_LEVEL: warn
  RATE_LIMIT_DISABLED: false
  SECURITY_CHECKS_RELAXED: false
  
  # Production-only settings
  MONITORING_ENABLED: true
  ALERTS_ENABLED: true
  AUDIT_LOGGING: true
```

### Environment Validation Workflows

Validate environment configuration before running tests:

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
  environment-validation:
    name: Environment Configuration Validation
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
        action: http
        with:
          url: "{{vars.api_base_url}}/health"
          timeout: 10s
        test: res.status == 200
        outputs:
          connectivity_ok: res.status == 200
          api_version: res.json.version
          environment_confirmed: res.json.environment

      - name: Authentication Pre-Check
        action: http
        with:
          url: "{{vars.api_base_url}}/auth/validate"
          headers:
            Authorization: "Bearer {{vars.api_token}}"
        test: res.status == 200
        outputs:
          auth_valid: res.status == 200
          token_expires_in: res.json.expires_in

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

## CI/CD Integration

### GitHub Actions Integration

Integrate with CI/CD pipelines for automated environment testing:

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

### Environment-Specific Test Suites

Create different test suites for different environments:

**smoke-test.yml (for production):**
```yaml
name: Production Smoke Test
description: Minimal smoke test for production environment

vars:
  api_base_url: "{{API_BASE_URL}}"

jobs:
  critical-endpoints:
    name: Critical Endpoints Check
    steps:
      - name: Health Check
        action: http
        with:
          url: "{{vars.api_base_url}}/health"
        test: res.status == 200
        outputs:
          api_healthy: res.status == 200

      - name: Authentication Check
        action: http
        with:
          url: "{{vars.api_base_url}}/auth/health"
        test: res.status == 200
        outputs:
          auth_healthy: res.status == 200

      - name: Database Check
        action: http
        with:
          url: "{{vars.api_base_url}}/health/database"
        test: res.status == 200
        outputs:
          db_healthy: res.status == 200

  smoke-test-summary:
    name: Smoke Test Summary
    needs: [critical-endpoints]
    steps:
      - name: Production Health Summary
        echo: |
          🏭 Production Smoke Test Results:
          
          Critical Systems:
          API: {{outputs.critical-endpoints.api_healthy ? "✅ Healthy" : "🚨 DOWN"}}
          Authentication: {{outputs.critical-endpoints.auth_healthy ? "✅ Healthy" : "🚨 DOWN"}}
          Database: {{outputs.critical-endpoints.db_healthy ? "✅ Healthy" : "🚨 DOWN"}}
          
          Overall Status: {{
            outputs.critical-endpoints.api_healthy &&
            outputs.critical-endpoints.auth_healthy &&
            outputs.critical-endpoints.db_healthy
            ? "🟢 ALL SYSTEMS OPERATIONAL" : "🔴 CRITICAL ISSUES DETECTED"
          }}
```

**comprehensive-test.yml (for staging):**
```yaml
name: Comprehensive Staging Test
description: Full test suite for staging environment validation

vars:
  api_base_url: "{{API_BASE_URL}}"

jobs:
  api-tests:
    name: API Test Suite
    steps:
      - name: User Management API
        action: http
        with:
          url: "{{vars.api_base_url}}/users"
        test: res.status == 200

      - name: Order Management API
        action: http
        with:
          url: "{{vars.api_base_url}}/orders"
        test: res.status == 200

      - name: Product Catalog API
        action: http
        with:
          url: "{{vars.api_base_url}}/products"
        test: res.status == 200

  integration-tests:
    name: Integration Tests
    needs: [api-tests]
    steps:
      - name: User-Order Integration
        action: http
        with:
          url: "{{vars.api_base_url}}/test/user-order-flow"
        test: res.status == 200 && res.json.test_passed == true

      - name: Payment Integration
        action: http
        with:
          url: "{{vars.api_base_url}}/test/payment-flow"
        test: res.status == 200 && res.json.test_passed == true

  performance-tests:
    name: Performance Validation
    needs: [integration-tests]
    steps:
      - name: Load Test
        action: http
        with:
          url: "{{vars.api_base_url}}/test/load"
          method: POST
          body: |
            {
              "concurrent_users": 10,
              "duration_seconds": 60
            }
        test: res.status == 200 && res.json.success_rate > 0.95
```

## Environment Monitoring and Alerting

### Environment Health Monitoring

Monitor the health of each environment continuously:

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
  monitoring_interval: "{{MONITORING_INTERVAL ?? '300'}}"  # 5 minutes
  alert_threshold: "{{ALERT_THRESHOLD ?? '2'}}"        # Alert after 2 consecutive failures

jobs:
  environment-health-check:
    name: Environment Health Check
    steps:
      - name: System Resources Check
        id: resources
        action: http
        with:
          url: "{{vars.api_base_url}}/health/resources"
        test: |
          res.status == 200 &&
          res.json.cpu_usage < 80 &&
          res.json.memory_usage < 80 &&
          res.json.disk_usage < 90
        continue_on_error: true
        outputs:
          resources_healthy: |
            res.status == 200 &&
            res.json.cpu_usage < 80 &&
            res.json.memory_usage < 80 &&
            res.json.disk_usage < 90
          cpu_usage: res.json.cpu_usage
          memory_usage: res.json.memory_usage
          disk_usage: res.json.disk_usage

      - name: Service Dependencies Check
        id: dependencies
        action: http
        with:
          url: "{{vars.api_base_url}}/health/dependencies"
        test: |
          res.status == 200 &&
          res.json.all_dependencies_healthy == true
        continue_on_error: true
        outputs:
          dependencies_healthy: res.json.all_dependencies_healthy
          unhealthy_services: res.json.unhealthy_services

      - name: Environment-Specific Checks
        echo: |
          Environment-specific validation for {{vars.environment}}
        outputs:
          env_specific_checks: |
            {{vars.environment == "production" ? "SSL, Security, Performance" :
              vars.environment == "staging" ? "Integration, Load Testing" :
              "Development Tools, Debug Features"}}

  alerting:
    name: Environment Alerting
    needs: [environment-health-check]
    if: |
      !outputs.environment-health-check.resources_healthy ||
      !outputs.environment-health-check.dependencies_healthy
    steps:
      - name: Environment Alert
        action: smtp
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

  health-summary:
    name: Health Summary
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

## Best Practices

### 1. Environment Isolation

```yaml
# Good: Clear environment separation
environments/
├── development.yml
├── staging.yml
└── production.yml

# Good: Environment-specific credentials
credentials/
├── development.yml
├── staging.yml
└── production.yml (uses env vars only)
```

### 2. Progressive Testing

```yaml
# Good: Test pipeline progression
Development → Unit Tests
Staging → Integration Tests + Performance Tests  
Production → Smoke Tests + Monitoring
```

### 3. Configuration Validation

```yaml
# Good: Validate before running tests
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

### 4. Secure Secret Management

```yaml
# Good: Use environment variables for secrets
vars:
  api_token: "{{PROD_API_TOKEN}}"  # From secure env var

# Avoid: Hardcoded secrets
vars:
  api_token: "hardcoded_secret_123"  # Never do this
```

## What's Next?

Now that you can manage environments effectively, explore:

- **[Monitoring Workflows](../monitoring-workflows/)** - Build comprehensive monitoring for all environments
- **[API Testing](../api-testing/)** - Create environment-specific API test suites
- **[Error Handling Strategies](../error-handling-strategies/)** - Handle environment-specific failures

Environment management is crucial for reliable testing across the software development lifecycle. Use these patterns to create consistent, secure, and maintainable environment configurations.
