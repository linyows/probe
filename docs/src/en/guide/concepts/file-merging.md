# File Merging

File merging is Probe's powerful feature for composing workflows from multiple configuration files. This enables configuration reuse, environment-specific customization, and modular workflow design. This guide explores merging strategies, patterns, and best practices.

## File Merging Fundamentals

Probe supports merging multiple YAML files using comma-separated file paths:

```bash
probe base.yml,environment.yml,overrides.yml
```

Files are merged in order from left to right, with later files overriding earlier ones.

### Basic Merging Example

**base.yml:**
```yaml
name: API Health Check
description: Basic API monitoring

env:
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
        action: http
        with:
          url: "{{env.API_URL}}/health"
        test: res.status == 200
```

**production.yml:**
```yaml
env:
  API_URL: https://api.production.company.com
  API_TIMEOUT: 10s

defaults:
  http:
    timeout: 10s
    headers:
      Authorization: "Bearer {{env.PROD_API_TOKEN}}"
```

**Result of `probe base.yml,production.yml`:**
```yaml
name: API Health Check
description: Basic API monitoring

env:
  API_URL: https://api.production.company.com
  API_TIMEOUT: 10s                                  # Overridden
  RETRY_COUNT: 3                                    # Preserved

defaults:
  http:
    timeout: 10s                                    # Overridden
    headers:
      User-Agent: "Probe Monitor"                   # Preserved
      Authorization: "Bearer {{env.PROD_API_TOKEN}}" # Added

jobs:
  health-check:                                     # Preserved from base
    name: Health Check
    steps:
      - name: API Health
        action: http
        with:
          url: "{{env.API_URL}}/health"             # Uses merged env.API_URL
        test: res.status == 200
```

## Merging Strategies

### 1. Environment-Based Configuration

Separate base configuration from environment-specific settings:

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
        action: http
        with:
          url: "{{env.API_BASE_URL}}/health"
        test: res.status == 200

      - name: Metrics Endpoint
        action: http
        with:
          url: "{{env.API_BASE_URL}}/metrics"
        test: res.status == 200

  database-health:
    name: Database Health
    steps:
      - name: Database Ping
        action: http
        with:
          url: "{{env.DB_API_URL}}/ping"
        test: res.status == 200
```

**development.yml:**
```yaml
env:
  API_BASE_URL: http://localhost:3000
  DB_API_URL: http://localhost:5432

defaults:
  http:
    timeout: 60s  # More lenient for development
```

**staging.yml:**
```yaml
env:
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
env:
  API_BASE_URL: https://api.company.com
  DB_API_URL: https://db.company.com

defaults:
  http:
    timeout: 10s  # Strict timeouts for production
    headers:
      X-Environment: production
      Authorization: "Bearer {{env.PROD_API_TOKEN}}"

jobs:
  # Add production-specific monitoring
  security-scan:
    name: Security Scan
    steps:
      - name: Security Health Check
        action: http
        with:
          url: "{{env.API_BASE_URL}}/security/health"
        test: res.status == 200
```

**Usage:**
```bash
# Development
probe base-monitoring.yml,development.yml

# Staging
probe base-monitoring.yml,staging.yml

# Production (includes additional security checks)
probe base-monitoring.yml,production.yml
```

### 2. Layered Configuration Architecture

Build configurations in layers for maximum flexibility:

**Layer 1 - Foundation (foundation.yml):**
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

**Layer 2 - Core Services (core-services.yml):**
```yaml
jobs:
  user-service:
    name: User Service Health
    needs: [connectivity]
    steps:
      - name: User API Health
        action: http
        with:
          url: "{{env.USER_SERVICE_URL}}/health"
        test: res.status == 200

  order-service:
    name: Order Service Health
    needs: [connectivity]
    steps:
      - name: Order API Health
        action: http
        with:
          url: "{{env.ORDER_SERVICE_URL}}/health"
        test: res.status == 200
```

**Layer 3 - Extended Services (extended-services.yml):**
```yaml
jobs:
  notification-service:
    name: Notification Service Health
    needs: [connectivity]
    steps:
      - name: Notification API Health
        action: http
        with:
          url: "{{env.NOTIFICATION_SERVICE_URL}}/health"
        test: res.status == 200

  analytics-service:
    name: Analytics Service Health
    needs: [connectivity]
    steps:
      - name: Analytics API Health
        action: http
        with:
          url: "{{env.ANALYTICS_SERVICE_URL}}/health"
        test: res.status == 200
```

**Layer 4 - Integration Tests (integration.yml):**
```yaml
jobs:
  integration-tests:
    name: Integration Tests
    needs: [user-service, order-service]
    steps:
      - name: User-Order Integration
        action: http
        with:
          url: "{{env.API_GATEWAY_URL}}/integration/user-order"
        test: res.status == 200

  end-to-end:
    name: End-to-End Tests
    needs: [integration-tests, notification-service]
    steps:
      - name: Complete Workflow Test
        action: http
        with:
          url: "{{env.API_GATEWAY_URL}}/e2e/complete"
        test: res.status == 200
```

**Layer 5 - Environment (environment.yml):**
```yaml
env:
  USER_SERVICE_URL: https://users.api.company.com
  ORDER_SERVICE_URL: https://orders.api.company.com
  NOTIFICATION_SERVICE_URL: https://notifications.api.company.com
  ANALYTICS_SERVICE_URL: https://analytics.api.company.com
  API_GATEWAY_URL: https://gateway.api.company.com

defaults:
  http:
    timeout: 15s
```

**Usage Combinations:**
```bash
# Basic core services only
probe foundation.yml,core-services.yml,environment.yml

# Extended services without integration
probe foundation.yml,core-services.yml,extended-services.yml,environment.yml

# Full suite with integration tests
probe foundation.yml,core-services.yml,extended-services.yml,integration.yml,environment.yml
```

### 3. Feature-Based Composition

Compose workflows by features that can be mixed and matched:

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
        action: http
        with:
          url: "{{env.API_URL}}/auth/login"
          method: POST
          body: |
            {
              "username": "{{env.TEST_USERNAME}}",
              "password": "{{env.TEST_PASSWORD}}"
            }
        test: res.status == 200
        outputs:
          access_token: res.json.access_token

      - name: Token Validation
        action: http
        with:
          url: "{{env.API_URL}}/auth/validate"
          headers:
            Authorization: "Bearer {{outputs.login.access_token}}"
        test: res.status == 200
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
        action: http
        with:
          url: "{{env.API_URL}}/users"
          method: POST
          headers:
            Authorization: "Bearer {{outputs.authentication-tests.access_token}}"
          body: |
            {
              "name": "Test User {{random_str(6)}}",
              "email": "test{{random_str(8)}}@example.com"
            }
        test: res.status == 201
        outputs:
          user_id: res.json.user.id

      - name: Get User
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.create-user.user_id}}"
          headers:
            Authorization: "Bearer {{outputs.authentication-tests.access_token}}"
        test: res.status == 200

      - name: Delete User
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.create-user.user_id}}"
          method: DELETE
          headers:
            Authorization: "Bearer {{outputs.authentication-tests.access_token}}"
        test: res.status == 204
```

**feature-performance.yml:**
```yaml
jobs:
  performance-tests:
    name: Performance Tests
    needs: [setup]
    steps:
      - name: Response Time Test
        action: http
        with:
          url: "{{env.API_URL}}/performance/test"
        test: res.status == 200 && res.time < 1000
        outputs:
          response_time: res.time

      - name: Load Test
        action: http
        with:
          url: "{{env.API_URL}}/performance/load"
          method: POST
          body: |
            {
              "concurrent_users": 10,
              "duration": 30
            }
        test: res.status == 200
        outputs:
          load_test_passed: res.json.success
```

**feature-reporting.yml:**
```yaml
jobs:
  test-reporting:
    name: Test Reporting
    needs: [setup]  # Runs after all other tests complete
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

**Feature Combinations:**
```bash
# Authentication only
probe base-workflow.yml,feature-authentication.yml,feature-reporting.yml,env.yml

# User management (includes authentication)
probe base-workflow.yml,feature-authentication.yml,feature-user-management.yml,feature-reporting.yml,env.yml

# Performance testing only
probe base-workflow.yml,feature-performance.yml,feature-reporting.yml,env.yml

# Full feature suite
probe base-workflow.yml,feature-authentication.yml,feature-user-management.yml,feature-performance.yml,feature-reporting.yml,env.yml
```

## Advanced Merging Patterns

### 1. Override Patterns

Use specific override files for special cases:

**base-config.yml:**
```yaml
name: Service Monitor
env:
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
        action: http
        with:
          url: "{{env.API_URL}}/health"
        test: res.status == 200
```

**debug-overrides.yml:**
```yaml
env:
  LOG_LEVEL: debug
  API_TIMEOUT: 120s  # Longer timeouts for debugging

defaults:
  http:
    timeout: 120s

jobs:
  health-check:
    steps:
      - name: API Health
        action: http
        with:
          url: "{{env.API_URL}}/health"
        test: res.status == 200
        outputs:
          # Add debug outputs
          debug_response_headers: res.headers
          debug_response_time: res.time
          debug_response_size: res.body_size

      # Add debug step
      - name: Debug Information
        echo: |
          Debug Information:
          Response Time: {{outputs.debug_response_time}}ms
          Response Size: {{outputs.debug_response_size}} bytes
          Content Type: {{outputs.debug_response_headers["content-type"]}}
```

**load-test-overrides.yml:**
```yaml
env:
  CONCURRENT_REQUESTS: 100
  TEST_DURATION: 300s

jobs:
  # Override health-check with load testing
  health-check:
    name: Load Test
    steps:
      - name: Load Test Execution
        action: http
        with:
          url: "{{env.API_URL}}/load-test"
          method: POST
          body: |
            {
              "concurrent_users": {{env.CONCURRENT_REQUESTS}},
              "duration_seconds": {{env.TEST_DURATION}}
            }
        test: res.status == 200 && res.json.success_rate > 0.95
        outputs:
          success_rate: res.json.success_rate
          avg_response_time: res.json.avg_response_time
```

**Usage:**
```bash
# Normal monitoring
probe base-config.yml,production-env.yml

# Debug mode
probe base-config.yml,production-env.yml,debug-overrides.yml

# Load testing
probe base-config.yml,production-env.yml,load-test-overrides.yml
```

### 2. Conditional Merging with Environment Variables

Use environment variables to control which files are merged:

**conditional-merge.sh:**
```bash
#!/bin/bash

BASE_FILES="foundation.yml,core-services.yml"
ENV_FILE="${ENVIRONMENT:-development}.yml"
FEATURE_FILES=""

# Add features based on environment variables
if [ "$INCLUDE_SECURITY" = "true" ]; then
    FEATURE_FILES="${FEATURE_FILES},security-tests.yml"
fi

if [ "$INCLUDE_PERFORMANCE" = "true" ]; then
    FEATURE_FILES="${FEATURE_FILES},performance-tests.yml"
fi

if [ "$INCLUDE_INTEGRATION" = "true" ]; then
    FEATURE_FILES="${FEATURE_FILES},integration-tests.yml"
fi

# Build final file list
FILES="${BASE_FILES},${ENV_FILE}${FEATURE_FILES}"

echo "Executing: probe $FILES"
probe $FILES
```

**Usage:**
```bash
# Basic monitoring
ENVIRONMENT=production ./conditional-merge.sh

# With security tests
ENVIRONMENT=production INCLUDE_SECURITY=true ./conditional-merge.sh

# Full suite
ENVIRONMENT=production INCLUDE_SECURITY=true INCLUDE_PERFORMANCE=true INCLUDE_INTEGRATION=true ./conditional-merge.sh
```

### 3. Template-Based Configuration

Use templates that are completed through merging:

**template-workflow.yml:**
```yaml
name: "{{WORKFLOW_NAME || 'Default Workflow'}}"
description: "{{WORKFLOW_DESCRIPTION || 'Template-based workflow'}}"

env:
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
        action: http
        with:
          url: "{{env.SERVICE_URL}}/health"
        test: res.status == 200
```

**service-config.yml:**
```yaml
env:
  WORKFLOW_NAME: User Service Monitoring
  WORKFLOW_DESCRIPTION: Comprehensive monitoring for the user service
  SERVICE_NAME: user-service
  SERVICE_URL: https://users.api.company.com
  ENVIRONMENT: production
  DEFAULT_TIMEOUT: 15s

jobs:
  # Add service-specific tests
  user-specific-tests:
    name: User Service Specific Tests
    needs: [health-check]
    steps:
      - name: User Count Check
        action: http
        with:
          url: "{{env.SERVICE_URL}}/users/count"
        test: res.status == 200 && res.json.count >= 0

      - name: User Authentication Test
        action: http
        with:
          url: "{{env.SERVICE_URL}}/auth/health"
        test: res.status == 200
```

## Configuration Management Patterns

### 1. Hierarchical Configuration

Organize configurations in a hierarchy for inheritance:

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

**Makefile for configuration management:**
```makefile
# Base configuration
BASE_CONFIG = configs/base/foundation.yml,configs/base/common-defaults.yml

# Environment-specific configurations
dev: $(BASE_CONFIG),configs/environments/development.yml
	probe $^

staging: $(BASE_CONFIG),configs/environments/staging.yml,configs/features/security.yml
	probe $^

prod: $(BASE_CONFIG),configs/environments/production.yml,configs/features/security.yml,configs/features/performance.yml
	probe $^

# Service-specific tests
test-user-service: $(BASE_CONFIG),configs/services/user-service.yml,configs/environments/$(ENV).yml
	probe $^

test-all-services: $(BASE_CONFIG),configs/services/user-service.yml,configs/services/order-service.yml,configs/services/notification-service.yml,configs/environments/$(ENV).yml
	probe $^
```

### 2. Configuration Validation

Validate merged configurations before execution:

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
          API_URL: {{env.API_URL ? "✅ Set" : "❌ Missing"}}
          DB_URL: {{env.DB_URL ? "✅ Set" : "❌ Missing"}}
          ENVIRONMENT: {{env.ENVIRONMENT ? "✅ Set (" + env.ENVIRONMENT + ")" : "❌ Missing"}}
          
          Optional Variables:
          SMTP_HOST: {{env.SMTP_HOST ? "✅ Set" : "⚠️ Not Set"}}
          SLACK_WEBHOOK: {{env.SLACK_WEBHOOK ? "✅ Set" : "⚠️ Not Set"}}
          
          Configuration Status: {{
            env.API_URL && env.DB_URL && env.ENVIRONMENT ? "✅ Valid" : "❌ Invalid"
          }}

      - name: Validate Configuration Consistency
        echo: |
          Configuration Consistency Check:
          
          Environment Alignment:
          {{env.ENVIRONMENT == "production" && env.API_URL.contains("prod") ? "✅ Production URLs match environment" : ""}}
          {{env.ENVIRONMENT == "staging" && env.API_URL.contains("staging") ? "✅ Staging URLs match environment" : ""}}
          {{env.ENVIRONMENT == "development" && (env.API_URL.contains("localhost") || env.API_URL.contains("dev")) ? "✅ Development URLs match environment" : ""}}
          
          Security Check:
          {{env.ENVIRONMENT == "production" && env.API_URL.startsWith("https://") ? "✅ Production uses HTTPS" : ""}}
          {{env.ENVIRONMENT != "production" || env.API_URL.startsWith("https://") ? "" : "⚠️ Production should use HTTPS"}}
```

**Usage with validation:**
```bash
# Validate configuration before running main workflow
probe validation-config.yml,base-config.yml,production.yml && \
probe base-workflow.yml,production.yml
```

## Best Practices

### 1. File Organization

```yaml
# Good: Logical file organization
# base.yml - Core workflow structure
# environment.yml - Environment-specific variables
# features.yml - Optional feature additions

# Avoid: Monolithic files that try to handle everything
```

### 2. Clear Merge Order

```bash
# Good: Logical merge order (general to specific)
probe base.yml,environment.yml,team-overrides.yml

# Avoid: Confusing merge order
probe overrides.yml,base.yml,environment.yml  # Overrides might be ignored
```

### 3. Environment Variable Strategy

```yaml
# Good: Use environment variables for dynamic values
env:
  API_URL: "{{env.EXTERNAL_API_URL}}"
  TIMEOUT: "{{env.REQUEST_TIMEOUT || '30s'}}"

# Good: Provide defaults in base configuration
env:
  DEFAULT_TIMEOUT: 30s
  DEFAULT_RETRY_COUNT: 3
```

### 4. Documentation

```yaml
# Document merge strategy in workflow
name: Multi-Environment API Test
description: |
  This workflow supports multiple environments through file merging.
  
  Usage:
  - Development: probe workflow.yml,development.yml
  - Staging: probe workflow.yml,staging.yml
  - Production: probe workflow.yml,production.yml
  
  The base workflow.yml provides the core structure, while environment
  files provide environment-specific URLs, timeouts, and credentials.
```

### 5. Merge Validation

```yaml
jobs:
  pre-execution-check:
    name: Pre-execution Validation
    steps:
      - name: Validate Merged Configuration
        echo: |
          Merged Configuration Summary:
          
          Environment: {{env.ENVIRONMENT || "Not specified"}}
          API URL: {{env.API_URL || "Not configured"}}
          Timeout: {{defaults.http.timeout || "Default"}}
          
          Required configurations: {{
            env.API_URL && env.ENVIRONMENT ? "✅ Present" : "❌ Missing"
          }}
```

## Common Anti-Patterns

### 1. Circular Dependencies

```yaml
# Avoid: Files that depend on each other
# base.yml includes references that only work with production.yml
# production.yml includes jobs that only work with base.yml
```

### 2. Deep Override Hierarchies

```bash
# Avoid: Too many override layers
probe base.yml,region.yml,environment.yml,team.yml,user.yml,local.yml
# Hard to understand what the final configuration will be
```

### 3. Inconsistent Merge Strategies

```yaml
# Avoid: Mixing different merge approaches
# Some files override completely, others merge additively
# Creates unpredictable results
```

## What's Next?

Now that you understand file merging, explore:

1. **[How-tos](../../how-tos/)** - See practical file merging patterns in action
2. **[Reference](../../reference/)** - Detailed syntax and configuration reference
3. **[Tutorials](../../tutorials/)** - Step-by-step guides for common scenarios

File merging is your key to building flexible, maintainable workflow configurations. Master these patterns to create reusable, environment-aware automation that scales with your needs.
