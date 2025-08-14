# Multi-Environment Deployment Testing

In this tutorial, you'll learn how to create a comprehensive multi-environment testing strategy using Probe. You'll build workflows that automatically validate deployments across development, staging, and production environments, ensuring consistency and reliability at every stage of your deployment pipeline.

## What You'll Build

A complete multi-environment testing system featuring:

- **Environment-Specific Configurations** - Tailored settings for each environment
- **Progressive Deployment Testing** - Validate each deployment stage
- **Cross-Environment Consistency Checks** - Ensure feature parity
- **Environment Health Monitoring** - Continuous environment validation
- **Deployment Validation Pipelines** - Automated deployment verification
- **Rollback Detection** - Identify when environments diverge
- **Configuration Drift Detection** - Monitor environment consistency

## Prerequisites

- Probe installed ([Installation Guide](../get-started/installation/))
- Access to multiple environments (dev, staging, production)
- Understanding of deployment pipelines
- Basic knowledge of environment management

## Tutorial Overview

We'll create testing workflows for a typical three-tier environment setup:

- **Development** - Latest features, rapid iteration
- **Staging** - Production-like environment for final validation
- **Production** - Live user-facing environment

## Step 1: Environment Configuration Structure

Create a well-organized configuration structure:

```bash
deployment-tests/
├── config/
│   ├── base.yml              # Common configuration
│   ├── environments/
│   │   ├── development.yml   # Dev-specific config
│   │   ├── staging.yml       # Staging-specific config
│   │   └── production.yml    # Prod-specific config
│   └── features/
│       ├── feature-flags.yml # Feature flag configurations
│       └── experiments.yml   # A/B test configurations
├── workflows/
│   ├── health-check.yml      # Basic health validation
│   ├── deployment-validation.yml # Deployment verification
│   ├── consistency-check.yml # Cross-environment validation
│   └── rollback-detection.yml # Rollback identification
└── scripts/
    ├── setup-environment.sh  # Environment setup
    └── deploy-validation.sh  # Deployment validation script
```

**config/base.yml:**
```yaml
name: "Multi-Environment Deployment Testing"
description: "Comprehensive testing across development, staging, and production environments"

env:
  # Common Configuration
  API_VERSION: "v2"
  USER_AGENT: "Probe Multi-Env Tester v1.0"
  
  # Test User Configuration
  TEST_USER_EMAIL: "test@example.com"
  TEST_USER_PASSWORD: "TestPassword123!"
  
  # Performance Thresholds (Base - will be overridden per environment)
  RESPONSE_TIME_THRESHOLD: 2000
  CRITICAL_RESPONSE_TIME: 5000
  
  # Health Check Configuration
  HEALTH_CHECK_INTERVAL: 30
  MAX_CONSECUTIVE_FAILURES: 3
  
  # Feature Testing
  FEATURE_VALIDATION_ENABLED: true
  A_B_TEST_VALIDATION_ENABLED: false

defaults:
  http:
    timeout: "15s"
    headers:
      User-Agent: "{{env.USER_AGENT}}"
      Accept: "application/json"
    follow_redirects: true
    verify_ssl: true
```

**config/environments/development.yml:**
```yaml
env:
  # Environment Identification
  ENVIRONMENT: "development"
  ENVIRONMENT_COLOR: "🟡"
  
  # API Configuration
  API_BASE_URL: "https://api-dev.example.com"
  WEB_BASE_URL: "https://web-dev.example.com"
  
  # Relaxed Performance Thresholds
  RESPONSE_TIME_THRESHOLD: 5000    # 5 seconds (more lenient)
  CRITICAL_RESPONSE_TIME: 10000    # 10 seconds
  
  # Development-Specific Settings
  DEBUG_MODE: true
  DETAILED_LOGGING: true
  SKIP_PERFORMANCE_TESTS: false   # Still test performance in dev
  SKIP_LOAD_TESTS: true           # Skip heavy load testing
  
  # Feature Flags (Development gets all features)
  ENABLE_NEW_FEATURES: true
  ENABLE_EXPERIMENTAL_FEATURES: true
  ENABLE_DEBUG_ENDPOINTS: true
  
  # Development Data
  USE_TEST_DATA: true
  RESET_DATA_BEFORE_TESTS: true

defaults:
  http:
    timeout: "30s"         # Longer timeout for development
    verify_ssl: false      # Allow self-signed certificates
```

**config/environments/staging.yml:**
```yaml
env:
  # Environment Identification
  ENVIRONMENT: "staging"
  ENVIRONMENT_COLOR: "🟠"
  
  # API Configuration
  API_BASE_URL: "https://api-staging.example.com"
  WEB_BASE_URL: "https://web-staging.example.com"
  
  # Production-Like Performance Thresholds
  RESPONSE_TIME_THRESHOLD: 2000    # 2 seconds
  CRITICAL_RESPONSE_TIME: 5000     # 5 seconds
  
  # Staging-Specific Settings
  DEBUG_MODE: false
  DETAILED_LOGGING: true
  SKIP_PERFORMANCE_TESTS: false
  SKIP_LOAD_TESTS: false
  
  # Feature Flags (Staging mirrors production + approved features)
  ENABLE_NEW_FEATURES: true
  ENABLE_EXPERIMENTAL_FEATURES: false  # Only stable features
  ENABLE_DEBUG_ENDPOINTS: false
  
  # Staging Data
  USE_TEST_DATA: false
  USE_PRODUCTION_LIKE_DATA: true
  RESET_DATA_BEFORE_TESTS: false

defaults:
  http:
    timeout: "15s"
    verify_ssl: true
```

**config/environments/production.yml:**
```yaml
env:
  # Environment Identification
  ENVIRONMENT: "production"
  ENVIRONMENT_COLOR: "🔴"
  
  # API Configuration
  API_BASE_URL: "https://api.example.com"
  WEB_BASE_URL: "https://web.example.com"
  
  # Strict Performance Thresholds
  RESPONSE_TIME_THRESHOLD: 1500    # 1.5 seconds
  CRITICAL_RESPONSE_TIME: 3000     # 3 seconds
  
  # Production Settings
  DEBUG_MODE: false
  DETAILED_LOGGING: false
  SKIP_PERFORMANCE_TESTS: false
  SKIP_LOAD_TESTS: true            # Avoid impacting production
  
  # Feature Flags (Production only has stable, approved features)
  ENABLE_NEW_FEATURES: false
  ENABLE_EXPERIMENTAL_FEATURES: false
  ENABLE_DEBUG_ENDPOINTS: false
  
  # Production Data
  USE_TEST_DATA: false
  USE_PRODUCTION_DATA: true
  RESET_DATA_BEFORE_TESTS: false
  
  # Production-Specific Monitoring
  ENABLE_REAL_USER_MONITORING: true
  ENABLE_ERROR_TRACKING: true

defaults:
  http:
    timeout: "10s"
    verify_ssl: true
```

## Step 2: Basic Environment Health Checks

Create comprehensive health validation:

**workflows/health-check.yml:**
```yaml
name: "{{env.ENVIRONMENT | upper}} Environment Health Check"
description: "Comprehensive health validation for {{env.ENVIRONMENT}} environment"

jobs:
  infrastructure-health:
    name: "Infrastructure Health Check"
    steps:
      - name: "API Server Health"
        id: api-health
        action: http
        with:
          url: "{{env.API_BASE_URL}}/health"
        test: |
          res.status == 200 &&
          res.time < env.RESPONSE_TIME_THRESHOLD &&
          res.json.status == "healthy"
        outputs:
          api_status: res.json.status
          api_version: res.json.version
          api_response_time: res.time
          api_uptime: res.json.uptime

      - name: "Web Application Health"
        id: web-health
        action: http
        with:
          url: "{{env.WEB_BASE_URL}}/health"
        test: |
          res.status == 200 &&
          res.time < env.RESPONSE_TIME_THRESHOLD
        continue_on_error: true
        outputs:
          web_status: res.status == 200 ? "healthy" : "unhealthy"
          web_response_time: res.time

      - name: "Database Connectivity"
        id: db-health
        action: http
        with:
          url: "{{env.API_BASE_URL}}/health/database"
        test: |
          res.status == 200 &&
          res.json.database.connected == true &&
          res.json.database.responseTime < 1000
        outputs:
          db_status: res.json.database.connected ? "connected" : "disconnected"
          db_response_time: res.json.database.responseTime
          db_pool_size: res.json.database.poolSize

      - name: "Cache System Health"
        id: cache-health
        action: http
        with:
          url: "{{env.API_BASE_URL}}/health/cache"
        test: |
          res.status == 200 &&
          res.json.cache.connected == true
        continue_on_error: true
        outputs:
          cache_status: res.json.cache.connected ? "connected" : "disconnected"
          cache_hit_rate: res.json.cache.hitRate

  service-dependencies:
    name: "External Service Dependencies"
    needs: [infrastructure-health]
    steps:
      - name: "Payment Service Health"
        id: payment-health
        action: http
        with:
          url: "{{env.API_BASE_URL}}/health/payment"
        test: |
          res.status == 200 &&
          res.json.paymentService.available == true
        continue_on_error: true
        outputs:
          payment_status: res.json.paymentService.available ? "available" : "unavailable"

      - name: "Email Service Health"
        id: email-health
        action: http
        with:
          url: "{{env.API_BASE_URL}}/health/email"
        test: |
          res.status == 200 &&
          res.json.emailService.available == true
        continue_on_error: true
        outputs:
          email_status: res.json.emailService.available ? "available" : "unavailable"

      - name: "Search Service Health"
        id: search-health
        action: http
        with:
          url: "{{env.API_BASE_URL}}/health/search"
        test: |
          res.status == 200 &&
          res.json.searchService.available == true
        continue_on_error: true
        outputs:
          search_status: res.json.searchService.available ? "available" : "unavailable"

  environment-report:
    name: "Environment Health Report"
    needs: [infrastructure-health, service-dependencies]
    steps:
      - name: "Generate Health Report"
        echo: |
          {{env.ENVIRONMENT_COLOR}} === {{env.ENVIRONMENT | upper}} ENVIRONMENT HEALTH REPORT ===
          
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
          
          Generated: {{iso8601()}}
          Environment: {{env.ENVIRONMENT}}

      - name: "Health Check Validation"
        test: |
          outputs.api-health.api_status == "healthy" &&
          outputs.db-health.db_status == "connected"
```

## Step 3: Deployment Validation Workflow

Create comprehensive deployment verification:

**workflows/deployment-validation.yml:**
```yaml
name: "{{env.ENVIRONMENT | upper}} Deployment Validation"
description: "Validate deployment success in {{env.ENVIRONMENT}} environment"

env:
  # Deployment metadata (passed from CI/CD)
  DEPLOYMENT_ID: "{{env.DEPLOYMENT_ID || unixtime()}}"
  BUILD_VERSION: "{{env.BUILD_VERSION || 'unknown'}}"
  DEPLOYMENT_TIMESTAMP: "{{env.DEPLOYMENT_TIMESTAMP || iso8601()}}"

jobs:
  pre-deployment-validation:
    name: "Pre-Deployment Validation"
    steps:
      - name: "Verify Deployment Prerequisites"
        id: prerequisites
        action: http
        with:
          url: "{{env.API_BASE_URL}}/deployment/prerequisites"
          headers:
            X-Deployment-ID: "{{env.DEPLOYMENT_ID}}"
        test: |
          res.status == 200 &&
          res.json.readyForDeployment == true
        outputs:
          deployment_ready: res.json.readyForDeployment
          current_version: res.json.currentVersion
          target_version: res.json.targetVersion

      - name: "Database Migration Status"
        id: migration-status
        action: http
        with:
          url: "{{env.API_BASE_URL}}/deployment/migrations"
        test: |
          res.status == 200 &&
          res.json.pendingMigrations == 0
        outputs:
          pending_migrations: res.json.pendingMigrations
          last_migration: res.json.lastMigration

  deployment-verification:
    name: "Deployment Verification"
    needs: [pre-deployment-validation]
    steps:
      - name: "Verify New Version Deployment"
        id: version-check
        action: http
        with:
          url: "{{env.API_BASE_URL}}/version"
        test: |
          res.status == 200 &&
          res.json.version == env.BUILD_VERSION
        outputs:
          deployed_version: res.json.version
          deployment_time: res.json.deploymentTime
          build_hash: res.json.buildHash

      - name: "Feature Flag Validation"
        id: feature-flags
        if: env.ENABLE_NEW_FEATURES == "true"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/features"
        test: |
          res.status == 200 &&
          res.json.features != null
        outputs:
          active_features: res.json.features
          feature_count: len(res.json.features)

      - name: "Configuration Validation"
        id: config-validation
        action: http
        with:
          url: "{{env.API_BASE_URL}}/configuration"
        test: |
          res.status == 200 &&
          res.json.environment == env.ENVIRONMENT &&
          res.json.configVersion != null
        outputs:
          config_environment: res.json.environment
          config_version: res.json.configVersion
          config_valid: res.json.valid

  functional-validation:
    name: "Post-Deployment Functional Tests"
    needs: [deployment-verification]
    steps:
      - name: "Critical Path Test - User Authentication"
        id: auth-test
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/login"
          method: "POST"
          body: |
            {
              "email": "{{env.TEST_USER_EMAIL}}",
              "password": "{{env.TEST_USER_PASSWORD}}"
            }
        test: |
          res.status == 200 &&
          res.json.token != null &&
          res.time < env.RESPONSE_TIME_THRESHOLD
        outputs:
          auth_token: res.json.token
          auth_response_time: res.time

      - name: "Critical Path Test - Data Retrieval"
        id: data-test
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products?limit=5"
          headers:
            Authorization: "Bearer {{outputs.auth-test.auth_token}}"
        test: |
          res.status == 200 &&
          res.json.products != null &&
          len(res.json.products) > 0 &&
          res.time < env.RESPONSE_TIME_THRESHOLD
        outputs:
          product_count: len(res.json.products)
          data_response_time: res.time

      - name: "Critical Path Test - Data Mutation"
        id: mutation-test
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products"
          method: "POST"
          headers:
            Authorization: "Bearer {{outputs.auth-test.auth_token}}"
          body: |
            {
              "name": "Deployment Test Product {{env.DEPLOYMENT_ID}}",
              "price": 99.99,
              "category": "Test"
            }
        test: |
          res.status == 201 &&
          res.json.id != null &&
          res.time < env.RESPONSE_TIME_THRESHOLD
        outputs:
          created_product_id: res.json.id
          mutation_response_time: res.time

      - name: "Cleanup Test Data"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products/{{outputs.mutation-test.created_product_id}}"
          method: "DELETE"
          headers:
            Authorization: "Bearer {{outputs.auth-test.auth_token}}"
        test: res.status == 204 || res.status == 200
        continue_on_error: true

  performance-validation:
    name: "Performance Validation"
    needs: [functional-validation]
    if: env.SKIP_PERFORMANCE_TESTS != "true"
    steps:
      - name: "Response Time Validation"
        id: perf-test
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products"
        test: |
          res.status == 200 &&
          res.time < env.RESPONSE_TIME_THRESHOLD
        outputs:
          response_time: res.time
          performance_grade: |
            {{res.time < 500 ? "A" :
              res.time < 1000 ? "B" :
              res.time < 2000 ? "C" : "D"}}

      - name: "Load Test Simulation"
        if: env.SKIP_LOAD_TESTS != "true"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/health"
        test: res.status == 200 && res.time < mul(env.RESPONSE_TIME_THRESHOLD, 2)
        # In real scenario, this would trigger concurrent requests

  deployment-report:
    name: "Deployment Validation Report"
    needs: [pre-deployment-validation, deployment-verification, functional-validation, performance-validation]
    steps:
      - name: "Generate Deployment Report"
        echo: |
          {{env.ENVIRONMENT_COLOR}} === {{env.ENVIRONMENT | upper}} DEPLOYMENT VALIDATION REPORT ===
          
          Deployment Information:
          • Deployment ID: {{env.DEPLOYMENT_ID}}
          • Target Version: {{env.BUILD_VERSION}}
          • Deployed Version: {{outputs.version-check.deployed_version}}
          • Deployment Time: {{outputs.version-check.deployment_time}}
          • Environment: {{env.ENVIRONMENT}}
          
          Pre-Deployment Status:
          • Prerequisites: {{outputs.prerequisites.deployment_ready ? "✅ Ready" : "❌ Not Ready"}}
          • Current Version: {{outputs.prerequisites.current_version}}
          • Target Version: {{outputs.prerequisites.target_version}}
          • Pending Migrations: {{outputs.migration-status.pending_migrations}}
          
          Deployment Verification:
          • Version Match: {{outputs.version-check.deployed_version == env.BUILD_VERSION ? "✅ Correct" : "❌ Mismatch"}}
          • Configuration: {{outputs.config-validation.config_valid ? "✅ Valid" : "❌ Invalid"}}
          • Feature Flags: {{env.ENABLE_NEW_FEATURES == "true" ? outputs.feature-flags.feature_count + " features active" : "Default features"}}
          
          Functional Tests:
          • Authentication: {{outputs.auth-test.auth_response_time}}ms {{outputs.auth-test.auth_response_time < env.RESPONSE_TIME_THRESHOLD ? "✅" : "⚠️"}}
          • Data Retrieval: {{outputs.data-test.data_response_time}}ms {{outputs.data-test.data_response_time < env.RESPONSE_TIME_THRESHOLD ? "✅" : "⚠️"}}
          • Data Mutation: {{outputs.mutation-test.mutation_response_time}}ms {{outputs.mutation-test.mutation_response_time < env.RESPONSE_TIME_THRESHOLD ? "✅" : "⚠️"}}
          
          Performance Validation:
          {{if env.SKIP_PERFORMANCE_TESTS != "true"}}
          • Response Time: {{outputs.perf-test.response_time}}ms (Grade: {{outputs.perf-test.performance_grade}})
          • Performance Status: {{outputs.perf-test.response_time < env.RESPONSE_TIME_THRESHOLD ? "✅ Acceptable" : "⚠️ Slow"}}
          {{else}}
          • Performance Tests: ⏭️ Skipped
          {{end}}
          
          Overall Deployment Status: {{
            outputs.version-check.deployed_version == env.BUILD_VERSION &&
            outputs.config-validation.config_valid &&
            outputs.auth-test.auth_response_time < env.RESPONSE_TIME_THRESHOLD &&
            outputs.data-test.data_response_time < env.RESPONSE_TIME_THRESHOLD &&
            outputs.mutation-test.mutation_response_time < env.RESPONSE_TIME_THRESHOLD ? 
            "✅ DEPLOYMENT SUCCESSFUL" : "❌ DEPLOYMENT ISSUES DETECTED"
          }}
          
          Generated: {{iso8601()}}

      - name: "Validate Deployment Success"
        test: |
          outputs.version-check.deployed_version == env.BUILD_VERSION &&
          outputs.config-validation.config_valid &&
          outputs.auth-test.auth_response_time < env.RESPONSE_TIME_THRESHOLD &&
          outputs.data-test.data_response_time < env.RESPONSE_TIME_THRESHOLD
```

## Step 4: Cross-Environment Consistency Checks

Create workflows to validate consistency across environments:

**workflows/consistency-check.yml:**
```yaml
name: "Cross-Environment Consistency Check"
description: "Validate consistency between environments"

env:
  # Define environments to compare
  PRIMARY_ENVIRONMENT: "{{env.PRIMARY_ENV || 'production'}}"
  COMPARISON_ENVIRONMENTS: "{{env.COMPARISON_ENVS || 'staging,development'}}"

jobs:
  version-consistency:
    name: "Version Consistency Check"
    steps:
      - name: "Get Primary Environment Version"
        id: primary-version
        action: http
        with:
          url: "{{env.PRIMARY_ENV == 'production' ? 'https://api.example.com' : 
                  env.PRIMARY_ENV == 'staging' ? 'https://api-staging.example.com' :
                  'https://api-dev.example.com'}}/version"
        test: res.status == 200
        outputs:
          primary_version: res.json.version
          primary_env: env.PRIMARY_ENVIRONMENT
          primary_config_version: res.json.configVersion

      - name: "Get Staging Environment Version"
        id: staging-version
        if: env.PRIMARY_ENVIRONMENT != "staging"
        action: http
        with:
          url: "https://api-staging.example.com/version"
        test: res.status == 200
        continue_on_error: true
        outputs:
          staging_version: res.json.version
          staging_config_version: res.json.configVersion

      - name: "Get Development Environment Version"
        id: dev-version
        if: env.PRIMARY_ENVIRONMENT != "development"
        action: http
        with:
          url: "https://api-dev.example.com/version"
        test: res.status == 200
        continue_on_error: true
        outputs:
          dev_version: res.json.version
          dev_config_version: res.json.configVersion

  feature-consistency:
    name: "Feature Flag Consistency"
    needs: [version-consistency]
    steps:
      - name: "Compare Production vs Staging Features"
        id: prod-staging-features
        action: http
        with:
          url: "https://api.example.com/features"
        test: res.status == 200
        outputs:
          prod_features: keys(res.json.features)
          prod_feature_count: len(res.json.features)

      - name: "Get Staging Features"
        id: staging-features
        action: http
        with:
          url: "https://api-staging.example.com/features"
        test: res.status == 200
        outputs:
          staging_features: keys(res.json.features)
          staging_feature_count: len(res.json.features)

      - name: "Feature Drift Analysis"
        echo: |
          === FEATURE FLAG CONSISTENCY ANALYSIS ===
          
          Production Features: {{outputs.prod-staging-features.prod_feature_count}}
          Staging Features: {{outputs.staging-features.staging_feature_count}}
          
          Feature Drift: {{abs(sub(outputs.prod-staging-features.prod_feature_count, outputs.staging-features.staging_feature_count))}} features different
          
          Status: {{outputs.prod-staging-features.prod_feature_count == outputs.staging-features.staging_feature_count ? "✅ Consistent" : "⚠️ Drift Detected"}}

  configuration-consistency:
    name: "Configuration Consistency"
    needs: [version-consistency]
    steps:
      - name: "Database Schema Consistency"
        id: schema-consistency
        action: http
        with:
          url: "https://api.example.com/schema/version"
        test: res.status == 200
        outputs:
          prod_schema_version: res.json.schemaVersion
          prod_migration_count: res.json.migrationCount

      - name: "Staging Schema Version"
        id: staging-schema
        action: http
        with:
          url: "https://api-staging.example.com/schema/version"
        test: res.status == 200
        outputs:
          staging_schema_version: res.json.schemaVersion
          staging_migration_count: res.json.migrationCount

      - name: "Schema Consistency Report"
        echo: |
          === DATABASE SCHEMA CONSISTENCY ===
          
          Production Schema: v{{outputs.schema-consistency.prod_schema_version}} ({{outputs.schema-consistency.prod_migration_count}} migrations)
          Staging Schema: v{{outputs.staging-schema.staging_schema_version}} ({{outputs.staging-schema.staging_migration_count}} migrations)
          
          Schema Status: {{outputs.schema-consistency.prod_schema_version == outputs.staging-schema.staging_schema_version ? "✅ Synchronized" : "⚠️ Version Mismatch"}}
          Migration Status: {{outputs.schema-consistency.prod_migration_count == outputs.staging-schema.staging_migration_count ? "✅ Synchronized" : "⚠️ Migration Count Mismatch"}}

  api-consistency:
    name: "API Endpoint Consistency"
    needs: [version-consistency]
    steps:
      - name: "Get Production API Endpoints"
        id: prod-endpoints
        action: http
        with:
          url: "https://api.example.com/api/endpoints"
        test: res.status == 200
        outputs:
          prod_endpoints: keys(res.json.endpoints)
          prod_endpoint_count: len(res.json.endpoints)

      - name: "Get Staging API Endpoints"
        id: staging-endpoints
        action: http
        with:
          url: "https://api-staging.example.com/api/endpoints"
        test: res.status == 200
        outputs:
          staging_endpoints: keys(res.json.endpoints)
          staging_endpoint_count: len(res.json.endpoints)

      - name: "API Consistency Analysis"
        echo: |
          === API ENDPOINT CONSISTENCY ===
          
          Production Endpoints: {{outputs.prod-endpoints.prod_endpoint_count}}
          Staging Endpoints: {{outputs.staging-endpoints.staging_endpoint_count}}
          
          Endpoint Drift: {{abs(sub(outputs.prod-endpoints.prod_endpoint_count, outputs.staging-endpoints.staging_endpoint_count))}} endpoints different
          
          API Status: {{outputs.prod-endpoints.prod_endpoint_count == outputs.staging-endpoints.staging_endpoint_count ? "✅ Consistent" : "⚠️ Endpoint Drift"}}

  consistency-report:
    name: "Overall Consistency Report"
    needs: [version-consistency, feature-consistency, configuration-consistency, api-consistency]
    steps:
      - name: "Generate Consistency Report"
        echo: |
          🔍 === CROSS-ENVIRONMENT CONSISTENCY REPORT ===
          
          Environment Versions:
          • Production: {{outputs.primary-version.primary_version}} (config: {{outputs.primary-version.primary_config_version}})
          • Staging: {{outputs.staging-version.staging_version || "N/A"}} (config: {{outputs.staging-version.staging_config_version || "N/A"}})
          • Development: {{outputs.dev-version.dev_version || "N/A"}} (config: {{outputs.dev-version.dev_config_version || "N/A"}})
          
          Consistency Checks:
          • Version Alignment: {{outputs.primary-version.primary_version == outputs.staging-version.staging_version ? "✅ Aligned" : "⚠️ Misaligned"}}
          • Feature Flags: {{outputs.prod-staging-features.prod_feature_count == outputs.staging-features.staging_feature_count ? "✅ Consistent" : "⚠️ Drift Detected"}}
          • Database Schema: {{outputs.schema-consistency.prod_schema_version == outputs.staging-schema.staging_schema_version ? "✅ Synchronized" : "⚠️ Mismatch"}}
          • API Endpoints: {{outputs.prod-endpoints.prod_endpoint_count == outputs.staging-endpoints.staging_endpoint_count ? "✅ Consistent" : "⚠️ Drift"}}
          
          Recommendations:
          {{if outputs.primary-version.primary_version != outputs.staging-version.staging_version}}
          ⚠️ Version mismatch detected - consider updating staging environment
          {{end}}
          {{if outputs.schema-consistency.prod_schema_version != outputs.staging-schema.staging_schema_version}}
          ⚠️ Schema version mismatch - verify migration deployment
          {{end}}
          {{if outputs.prod-staging-features.prod_feature_count != outputs.staging-features.staging_feature_count}}
          ⚠️ Feature flag drift - review feature flag synchronization
          {{end}}
          
          Overall Status: {{
            outputs.primary-version.primary_version == outputs.staging-version.staging_version &&
            outputs.schema-consistency.prod_schema_version == outputs.staging-schema.staging_schema_version ?
            "✅ ENVIRONMENTS CONSISTENT" : "⚠️ CONSISTENCY ISSUES DETECTED"
          }}
          
          Generated: {{iso8601()}}
```

## Step 5: Automated Deployment Pipeline Integration

Create scripts to integrate with your CI/CD pipeline:

**scripts/deploy-validation.sh:**
```bash
#!/bin/bash

set -e

# Configuration
ENVIRONMENT=${1:-staging}
BUILD_VERSION=${2:-$(git rev-parse --short HEAD)}
DEPLOYMENT_ID=${3:-$(date +%s)}

# Validate parameters
if [[ ! "$ENVIRONMENT" =~ ^(development|staging|production)$ ]]; then
    echo "❌ Invalid environment: $ENVIRONMENT"
    echo "Usage: $0 <environment> [build_version] [deployment_id]"
    exit 1
fi

echo "🚀 Starting deployment validation for $ENVIRONMENT environment"
echo "📦 Build Version: $BUILD_VERSION"
echo "🆔 Deployment ID: $DEPLOYMENT_ID"

# Export environment variables
export ENVIRONMENT=$ENVIRONMENT
export BUILD_VERSION=$BUILD_VERSION
export DEPLOYMENT_ID=$DEPLOYMENT_ID
export DEPLOYMENT_TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Set environment-specific variables
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

**scripts/setup-environment.sh:**
```bash
#!/bin/bash

ENVIRONMENT=${1:-development}

echo "🔧 Setting up environment variables for: $ENVIRONMENT"

# Load environment-specific secrets
case $ENVIRONMENT in
    "development")
        export DEV_API_TOKEN=$(cat ~/.probe/dev-api-token 2>/dev/null || echo "dev-token-placeholder")
        export DEV_SMTP_USERNAME=$(cat ~/.probe/dev-smtp-user 2>/dev/null || echo "dev@example.com")
        export DEV_SMTP_PASSWORD=$(cat ~/.probe/dev-smtp-pass 2>/dev/null || echo "dev-password")
        ;;
    "staging")
        export STAGING_API_TOKEN=$(cat ~/.probe/staging-api-token 2>/dev/null || echo "staging-token-placeholder")
        export STAGING_SMTP_USERNAME=$(cat ~/.probe/staging-smtp-user 2>/dev/null || echo "staging@example.com")
        export STAGING_SMTP_PASSWORD=$(cat ~/.probe/staging-smtp-pass 2>/dev/null || echo "staging-password")
        ;;
    "production")
        export PROD_API_TOKEN=$(cat ~/.probe/prod-api-token 2>/dev/null || echo "prod-token-placeholder")
        export PROD_SMTP_USERNAME=$(cat ~/.probe/prod-smtp-user 2>/dev/null || echo "alerts@example.com")
        export PROD_SMTP_PASSWORD=$(cat ~/.probe/prod-smtp-pass 2>/dev/null || echo "prod-password")
        ;;
esac

# Common environment variables
export PROBE_LOG_LEVEL=${PROBE_LOG_LEVEL:-info}
export PROBE_NO_COLOR=${PROBE_NO_COLOR:-false}

echo "✅ Environment setup complete for: $ENVIRONMENT"
```

## Step 6: GitHub Actions Integration

Create a comprehensive CI/CD workflow:

**.github/workflows/multi-environment-deployment.yml:**
```yaml
name: Multi-Environment Deployment Testing

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]
  workflow_dispatch:
    inputs:
      environment:
        description: 'Target environment'
        required: true
        default: 'staging'
        type: choice
        options:
        - development
        - staging
        - production

jobs:
  determine-environments:
    runs-on: ubuntu-latest
    outputs:
      environments: ${{ steps.set-environments.outputs.environments }}
      build-version: ${{ steps.set-version.outputs.version }}
    steps:
      - name: Determine deployment environments
        id: set-environments
        run: |
          if [[ "${{ github.event_name }}" == "workflow_dispatch" ]]; then
            echo "environments=[\"${{ github.event.inputs.environment }}\"]" >> $GITHUB_OUTPUT
          elif [[ "${{ github.ref }}" == "refs/heads/main" ]]; then
            echo "environments=[\"staging\", \"production\"]" >> $GITHUB_OUTPUT
          elif [[ "${{ github.ref }}" == "refs/heads/develop" ]]; then
            echo "environments=[\"development\", \"staging\"]" >> $GITHUB_OUTPUT
          else
            echo "environments=[\"development\"]" >> $GITHUB_OUTPUT
          fi
      
      - name: Set build version
        id: set-version
        run: |
          echo "version=${{ github.sha }}" >> $GITHUB_OUTPUT

  deploy-and-test:
    needs: determine-environments
    runs-on: ubuntu-latest
    strategy:
      matrix:
        environment: ${{ fromJson(needs.determine-environments.outputs.environments) }}
      fail-fast: false
    
    environment: ${{ matrix.environment }}
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Install Probe
        run: |
          curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o probe
          chmod +x probe
          sudo mv probe /usr/local/bin/
      
      - name: Setup environment
        run: |
          chmod +x scripts/setup-environment.sh
          ./scripts/setup-environment.sh ${{ matrix.environment }}
        env:
          DEV_API_TOKEN: ${{ secrets.DEV_API_TOKEN }}
          STAGING_API_TOKEN: ${{ secrets.STAGING_API_TOKEN }}
          PROD_API_TOKEN: ${{ secrets.PROD_API_TOKEN }}
          DEV_SMTP_USERNAME: ${{ secrets.DEV_SMTP_USERNAME }}
          DEV_SMTP_PASSWORD: ${{ secrets.DEV_SMTP_PASSWORD }}
          STAGING_SMTP_USERNAME: ${{ secrets.STAGING_SMTP_USERNAME }}
          STAGING_SMTP_PASSWORD: ${{ secrets.STAGING_SMTP_PASSWORD }}
          PROD_SMTP_USERNAME: ${{ secrets.PROD_SMTP_USERNAME }}
          PROD_SMTP_PASSWORD: ${{ secrets.PROD_SMTP_PASSWORD }}
      
      - name: Run deployment validation
        run: |
          chmod +x scripts/deploy-validation.sh
          ./scripts/deploy-validation.sh ${{ matrix.environment }} ${{ needs.determine-environments.outputs.build-version }}
        env:
          BUILD_VERSION: ${{ needs.determine-environments.outputs.build-version }}
          DEPLOYMENT_ID: ${{ github.run_id }}-${{ matrix.environment }}
      
      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-results-${{ matrix.environment }}
          path: test-results/
          retention-days: 30

  consistency-check:
    needs: [determine-environments, deploy-and-test]
    runs-on: ubuntu-latest
    if: contains(fromJson(needs.determine-environments.outputs.environments), 'production') && contains(fromJson(needs.determine-environments.outputs.environments), 'staging')
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
      
      - name: Install Probe
        run: |
          curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o probe
          chmod +x probe
          sudo mv probe /usr/local/bin/
      
      - name: Run cross-environment consistency check
        run: |
          probe config/base.yml,workflows/consistency-check.yml
        env:
          PRIMARY_ENV: production
          COMPARISON_ENVS: staging,development

  deployment-report:
    needs: [determine-environments, deploy-and-test, consistency-check]
    runs-on: ubuntu-latest
    if: always()
    
    steps:
      - name: Generate deployment report
        run: |
          echo "## 🚀 Multi-Environment Deployment Report" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY
          echo "**Build Version:** ${{ needs.determine-environments.outputs.build-version }}" >> $GITHUB_STEP_SUMMARY
          echo "**Environments:** ${{ needs.determine-environments.outputs.environments }}" >> $GITHUB_STEP_SUMMARY
          echo "**Timestamp:** $(date -u +"%Y-%m-%dT%H:%M:%SZ")" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY
          
          # Add deployment status for each environment
          echo "### Deployment Status" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY
          
          # This would be populated by the actual test results
          echo "| Environment | Status | Tests | Performance |" >> $GITHUB_STEP_SUMMARY
          echo "|-------------|--------|-------|-------------|" >> $GITHUB_STEP_SUMMARY
          echo "| Development | ✅ Success | ✅ Passed | ✅ Good |" >> $GITHUB_STEP_SUMMARY
          echo "| Staging | ✅ Success | ✅ Passed | ✅ Good |" >> $GITHUB_STEP_SUMMARY
          echo "| Production | ${{ needs.deploy-and-test.result == 'success' && '✅ Success' || '❌ Failed' }} | Results | Performance |" >> $GITHUB_STEP_SUMMARY
```

## Step 7: Running Multi-Environment Tests

Execute your comprehensive multi-environment testing:

```bash
# Test individual environments
probe config/base.yml,config/environments/development.yml,workflows/health-check.yml
probe config/base.yml,config/environments/staging.yml,workflows/deployment-validation.yml
probe config/base.yml,config/environments/production.yml,workflows/health-check.yml

# Run full deployment validation
./scripts/deploy-validation.sh staging v2.1.0
./scripts/deploy-validation.sh production v2.1.0

# Run consistency checks
probe config/base.yml,workflows/consistency-check.yml

# Run all environments in sequence
for env in development staging production; do
    echo "Testing $env environment..."
    probe config/base.yml,config/environments/${env}.yml,workflows/health-check.yml
done
```

## Step 8: Advanced Multi-Environment Patterns

### Blue-Green Deployment Testing

Create tests for blue-green deployments:

```yaml
# workflows/blue-green-validation.yml
name: "Blue-Green Deployment Validation"

jobs:
  blue-green-test:
    steps:
      - name: "Test Blue Environment"
        id: blue-test
        action: http
        with:
          url: "{{env.BLUE_URL}}/health"
        test: res.status == 200
        
      - name: "Test Green Environment"
        id: green-test
        action: http
        with:
          url: "{{env.GREEN_URL}}/health"
        test: res.status == 200
        
      - name: "Compare Environment Performance"
        echo: |
          Blue Environment: {{outputs.blue-test.time}}ms
          Green Environment: {{outputs.green-test.time}}ms
          Performance Difference: {{abs(sub(outputs.blue-test.time, outputs.green-test.time))}}ms
```

### Canary Deployment Testing

Monitor canary deployments:

```yaml
# workflows/canary-validation.yml
name: "Canary Deployment Validation"

jobs:
  canary-monitoring:
    steps:
      - name: "Monitor Canary Traffic"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/metrics/canary"
        test: |
          res.status == 200 &&
          res.json.canaryTrafficPercent <= env.MAX_CANARY_TRAFFIC &&
          res.json.canaryErrorRate < env.MAX_CANARY_ERROR_RATE
```

## Troubleshooting

### Common Issues

**Environment Configuration Mismatch:**
```bash
# Validate configuration before deployment
probe config/base.yml,config/environments/staging.yml --validate-only
```

**Version Drift Detection:**
```bash
# Check version consistency across environments
probe workflows/consistency-check.yml --verbose
```

**Performance Regression:**
```bash
# Compare performance across environments
probe config/base.yml,config/environments/production.yml,workflows/performance-validation.yml
```

## Next Steps

Your multi-environment deployment testing system is complete! Consider these enhancements:

1. **Advanced Monitoring** - Add business metrics validation
2. **Automated Rollback** - Trigger rollbacks on failure
3. **Progressive Deployment** - Implement progressive rollout testing
4. **Compliance Testing** - Add security and compliance validation
5. **Performance Benchmarking** - Historical performance comparison

## Related Resources

- **[First Monitoring System Tutorial](../first-monitoring-system/)** - Basic monitoring setup
- **[API Testing Pipeline Tutorial](../api-testing-pipeline/)** - Comprehensive API testing
- **[How-tos: Monitoring Workflows](../../how-tos/monitoring-workflows/)** - Advanced monitoring patterns
- **[How-tos: Environment Management](../../how-tos/environment-management/)** - Environment configuration strategies
