---
title: Workflows
description: Understanding workflow structure, design patterns, and best practices
weight: 10
---

# Workflows

A workflow is the top-level container in Probe that defines a complete automation or monitoring process. This guide explores workflow structure, design patterns, and best practices for creating maintainable and effective workflows.

## Workflow Anatomy

Every Probe workflow consists of several key components:

```yaml
name: Workflow Name                    # Required: Human-readable name
description: What this workflow does   # Optional: Detailed description
env:                                   # Optional: Environment variables
  API_BASE_URL: https://api.example.com
defaults:                             # Optional: Default settings
  http:
    timeout: 30s
    headers:
      User-Agent: "Probe Monitor"
jobs:                                 # Required: One or more jobs
  job-name:
    # Job definition...
```

### Required Components

**Name**: Every workflow must have a descriptive name that clearly identifies its purpose.

```yaml
# Good examples
name: Production API Health Check
name: E-commerce Checkout Flow Test
name: Database Migration Validation

# Avoid generic names
name: Test
name: Workflow
name: Check
```

**Jobs**: At least one job must be defined. Jobs contain the actual work to be performed.

### Optional Components

**Description**: Provides detailed context about the workflow's purpose, scope, and expected outcomes.

```yaml
description: |
  Comprehensive health check for the production API including:
  - Authentication endpoint validation
  - Core business logic verification
  - Database connectivity testing
  - Third-party service integration checks
```

**Environment Variables**: Define environment-specific or sensitive configuration.

```yaml
env:
  API_BASE_URL: https://api.production.example.com
  TIMEOUT_SECONDS: 30
  MAX_RETRY_COUNT: 3
```

**Defaults**: Set common configuration that applies to all jobs and steps.

```yaml
defaults:
  http:
    timeout: 30s
    headers:
      Accept: "application/json"
      User-Agent: "Probe Health Monitor v1.0"
  retry:
    count: 3
    delay: 5s
```

## Workflow Design Patterns

### 1. Linear Workflow

Steps execute sequentially, each depending on the previous one's success.

```yaml
name: Database Migration
description: Execute database schema changes in order

jobs:
  migration:
    name: Run Migration Steps
    steps:
      - name: Backup Current Schema
        action: http
        with:
          url: "{{env.DB_API}}/backup"
          method: POST
        test: res.status == 200

      - name: Apply Schema Changes
        action: http
        with:
          url: "{{env.DB_API}}/migrate"
          method: POST
        test: res.status == 200

      - name: Verify Migration
        action: http
        with:
          url: "{{env.DB_API}}/schema/version"
          method: GET
        test: res.json.version == "2.1.0"

      - name: Update Documentation
        echo: "Migration to v2.1.0 completed successfully"
```

**Use cases:**
- Database migrations
- Deployment pipelines
- Setup/teardown processes

### 2. Parallel Workflow

Multiple independent checks run simultaneously for efficiency.

```yaml
name: Multi-Service Health Check
description: Check health of all microservices in parallel

jobs:
  user-service:
    name: User Service Health
    steps:
      - name: Check User API
        action: http
        with:
          url: "{{env.USER_SERVICE_URL}}/health"
        test: res.status == 200

  payment-service:
    name: Payment Service Health
    steps:
      - name: Check Payment API
        action: http
        with:
          url: "{{env.PAYMENT_SERVICE_URL}}/health"
        test: res.status == 200

  notification-service:
    name: Notification Service Health
    steps:
      - name: Check Notification API
        action: http
        with:
          url: "{{env.NOTIFICATION_SERVICE_URL}}/health"
        test: res.status == 200
```

**Use cases:**
- Multi-service monitoring
- Independent feature testing
- Resource validation

### 3. Staged Workflow

Combines parallel and sequential execution with dependencies.

```yaml
name: Application Deployment Validation
description: Validate deployment across multiple stages

jobs:
  # Stage 1: Infrastructure checks (parallel)
  database-check:
    name: Database Connectivity
    steps:
      - name: Test Database Connection
        action: http
        with:
          url: "{{env.DB_HEALTH_URL}}"
        test: res.status == 200

  cache-check:
    name: Cache Service Check
    steps:
      - name: Test Redis Connection
        action: http
        with:
          url: "{{env.REDIS_HEALTH_URL}}"
        test: res.status == 200

  # Stage 2: Application checks (depends on infrastructure)
  api-validation:
    name: API Service Validation
    needs: [database-check, cache-check]
    steps:
      - name: Test Core API Endpoints
        action: http
        with:
          url: "{{env.API_URL}}/health"
        test: res.status == 200

  # Stage 3: End-to-end testing (depends on API)
  e2e-tests:
    name: End-to-End Tests
    needs: [api-validation]
    steps:
      - name: Test User Registration Flow
        action: http
        with:
          url: "{{env.API_URL}}/auth/register"
          method: POST
          body: |
            {
              "email": "test@example.com",
              "password": "testpass123"
            }
        test: res.status == 201
```

**Use cases:**
- Deployment validation
- Complex system testing
- Multi-tier application monitoring

### 4. Fan-out/Fan-in Workflow

Parallel execution followed by aggregation.

```yaml
name: Regional Service Check
description: Check services across multiple regions and aggregate results

jobs:
  # Fan-out: Check each region in parallel
  us-east-check:
    name: US East Region Check
    steps:
      - name: Check US East API
        action: http
        with:
          url: https://us-east.api.example.com/health
        test: res.status == 200
        outputs:
          region: "us-east"
          status: res.json.status
          response_time: res.time

  us-west-check:
    name: US West Region Check
    steps:
      - name: Check US West API
        action: http
        with:
          url: https://us-west.api.example.com/health
        test: res.status == 200
        outputs:
          region: "us-west"
          status: res.json.status
          response_time: res.time

  eu-check:
    name: Europe Region Check
    steps:
      - name: Check EU API
        action: http
        with:
          url: https://eu.api.example.com/health
        test: res.status == 200
        outputs:
          region: "eu"
          status: res.json.status
          response_time: res.time

  # Fan-in: Aggregate results
  summary:
    name: Regional Summary
    needs: [us-east-check, us-west-check, eu-check]
    steps:
      - name: Generate Report
        echo: |
          Regional Health Check Results:
          
          US East: {{outputs.us-east-check.status}} ({{outputs.us-east-check.response_time}}ms)
          US West: {{outputs.us-west-check.status}} ({{outputs.us-west-check.response_time}}ms)
          Europe: {{outputs.eu-check.status}} ({{outputs.eu-check.response_time}}ms)
          
          Total regions healthy: {{
            (outputs.us-east-check.status == "healthy" ? 1 : 0) +
            (outputs.us-west-check.status == "healthy" ? 1 : 0) +
            (outputs.eu-check.status == "healthy" ? 1 : 0)
          }}/3
```

**Use cases:**
- Multi-region monitoring
- Load testing across environments
- Distributed system validation

## Workflow Organization Strategies

### 1. Single-Purpose Workflows

Keep workflows focused on a single, well-defined purpose.

```yaml
# Good: Focused on API health checking
name: API Health Check
description: Monitor the health of our REST API endpoints

# Good: Focused on database operations
name: Database Maintenance
description: Perform routine database maintenance tasks

# Avoid: Mixed responsibilities
name: API and Database and Email Check
description: Check everything
```

### 2. Layered Workflows

Organize workflows by architectural layers.

```yaml
# infrastructure-health.yml
name: Infrastructure Health Check
description: Check foundational infrastructure components

# application-health.yml  
name: Application Health Check
description: Check application-level services

# business-logic-tests.yml
name: Business Logic Validation
description: Test core business functionality
```

### 3. Environment-Aware Workflows

Design workflows that work across different environments using configuration merging.

**base-monitoring.yml:**
```yaml
name: Service Monitoring
description: Monitor critical services

jobs:
  api-check:
    steps:
      - name: Check API Health
        action: http
        with:
          url: "{{env.API_BASE_URL}}/health"
        test: res.status == 200
```

**production.yml:**
```yaml
env:
  API_BASE_URL: https://api.production.example.com
defaults:
  http:
    timeout: 10s
```

**staging.yml:**
```yaml
env:
  API_BASE_URL: https://api.staging.example.com
defaults:
  http:
    timeout: 30s
```

Usage:
```bash
# Production monitoring
probe base-monitoring.yml,production.yml

# Staging monitoring  
probe base-monitoring.yml,staging.yml
```

## Advanced Workflow Techniques

### 1. Conditional Job Execution

Execute jobs only when certain conditions are met.

```yaml
jobs:
  health-check:
    name: Basic Health Check
    steps:
      - name: Check Service
        id: service-check
        action: http
        with:
          url: "{{env.SERVICE_URL}}/health"
        test: res.status == 200
        outputs:
          service_healthy: res.status == 200

  deep-diagnostic:
    name: Deep Diagnostic
    if: jobs.health-check.failed
    steps:
      - name: Run Diagnostics
        action: http
        with:
          url: "{{env.SERVICE_URL}}/diagnostics"
        test: res.status == 200

  alert:
    name: Send Alert
    if: jobs.deep-diagnostic.executed && jobs.deep-diagnostic.failed
    steps:
      - name: Critical Alert
        echo: "CRITICAL: Service is down and diagnostics failed"
```

### 2. Dynamic Configuration

Use expressions to make workflows adapt to runtime conditions.

```yaml
jobs:
  load-test:
    name: Load Testing
    steps:
      - name: Determine Load Parameters
        id: params
        echo: "Load test configuration determined"
        outputs:
          concurrent_users: "{{env.LOAD_TEST_USERS || 10}}"
          test_duration: "{{env.LOAD_TEST_DURATION || 60}}"

      - name: Execute Load Test
        action: http
        with:
          url: "{{env.LOAD_TEST_URL}}"
          method: POST
          body: |
            {
              "concurrent_users": {{outputs.params.concurrent_users}},
              "duration_seconds": {{outputs.params.test_duration}}
            }
        test: res.status == 200
```

### 3. Workflow Composition

Break complex workflows into reusable components.

**common-setup.yml:**
```yaml
jobs:
  setup:
    name: Common Setup
    steps:
      - name: Initialize Environment
        echo: "Environment initialized"
        outputs:
          timestamp: "{{unixtime()}}"
          session_id: "{{random_str(8)}}"
```

**main-workflow.yml:**
```yaml
name: Complete System Test
description: Full system validation with common setup

# This will be merged with common-setup.yml
jobs:
  api-tests:
    name: API Tests
    needs: [setup]
    steps:
      - name: Test API with Session
        action: http
        with:
          url: "{{env.API_URL}}/test"
          headers:
            X-Session-ID: "{{outputs.setup.session_id}}"
        test: res.status == 200
```

Usage:
```bash
probe common-setup.yml,main-workflow.yml
```

## Best Practices

### 1. Naming Conventions

Use consistent, descriptive naming:

```yaml
# Workflow names: Use Title Case
name: Production API Health Check

# Job names: Descriptive and specific
jobs:
  user-authentication-test:
    name: User Authentication Test
  
  database-connectivity-check:
    name: Database Connectivity Check

# Step names: Action-oriented
steps:
  - name: Verify User Login Endpoint
  - name: Test Database Connection Pool
  - name: Validate Cache Expiration
```

### 2. Documentation

Include comprehensive documentation:

```yaml
name: E-commerce Checkout Flow Test
description: |
  Validates the complete e-commerce checkout process including:
  
  1. Product catalog browsing
  2. Shopping cart management  
  3. User authentication
  4. Payment processing
  5. Order confirmation
  6. Email notification delivery
  
  This workflow simulates a real user journey from product selection
  to order completion, ensuring all critical business logic functions
  correctly.
  
  Prerequisites:
  - Test user account with valid payment method
  - Product catalog populated with test data
  - Email service configured for notifications
  
  Expected duration: 2-3 minutes
  
  Failure scenarios tested:
  - Invalid payment information
  - Out of stock products
  - Email delivery failures
```

### 3. Error Handling Strategy

Plan for failure scenarios:

```yaml
jobs:
  primary-check:
    name: Primary Service Check
    steps:
      - name: Check Primary Service
        id: primary
        action: http
        with:
          url: "{{env.PRIMARY_SERVICE_URL}}"
        test: res.status == 200
        continue_on_error: true

  fallback-check:
    name: Fallback Service Check
    if: jobs.primary-check.failed
    steps:
      - name: Check Fallback Service
        action: http
        with:
          url: "{{env.FALLBACK_SERVICE_URL}}"
        test: res.status == 200

  notification:
    name: Send Notifications
    needs: [primary-check, fallback-check]
    steps:
      - name: Success Notification
        if: jobs.primary-check.success
        echo: "✅ Primary service is healthy"
        
      - name: Fallback Notification
        if: jobs.primary-check.failed && jobs.fallback-check.success
        echo: "⚠️ Primary service down, fallback operational"
        
      - name: Critical Alert
        if: jobs.primary-check.failed && jobs.fallback-check.failed
        echo: "🚨 CRITICAL: Both primary and fallback services are down"
```

### 4. Performance Considerations

Design workflows for optimal performance:

```yaml
# Good: Parallel execution for independent operations
jobs:
  frontend-check:    # These run in parallel
  backend-check:     # for better performance
  database-check:

# Good: Efficient job dependencies
jobs:
  infrastructure:    # Foundation checks first
  application:       # Then application checks
    needs: [infrastructure]
  integration:       # Finally integration tests
    needs: [application]

# Avoid: Unnecessary sequential dependencies
jobs:
  check-a:
  check-b:
    needs: [check-a]  # Only if B actually depends on A
```

## Common Anti-Patterns

### 1. Monolithic Workflows

**Avoid:**
```yaml
name: Everything Check
jobs:
  massive-job:
    steps:
      - name: Check API
      - name: Check Database  
      - name: Check Cache
      - name: Check Email
      - name: Check Files
      - name: Check Logs
      # ... 50 more steps
```

**Instead:**
```yaml
# Split into focused workflows
name: API Health Check
name: Database Health Check  
name: Infrastructure Health Check
```

### 2. Tight Coupling

**Avoid:**
```yaml
# Hard-coded values throughout
- name: Check Production API
  action: http
  with:
    url: https://prod-api.company.com/health
```

**Instead:**
```yaml
# Use configuration and environment variables
- name: Check API
  action: http
  with:
    url: "{{env.API_BASE_URL}}/health"
```

### 3. Missing Error Handling

**Avoid:**
```yaml
steps:
  - name: Critical Operation
    action: http
    with:
      url: "{{env.CRITICAL_SERVICE}}"
    # No test condition or error handling
```

**Instead:**
```yaml
steps:
  - name: Critical Operation
    action: http
    with:
      url: "{{env.CRITICAL_SERVICE}}"
    test: res.status == 200
    continue_on_error: false
```

## What's Next?

Now that you understand workflow design and patterns, explore:

1. **[Jobs and Steps](../jobs-and-steps/)** - Deep dive into job and step mechanics
2. **[Actions](../actions/)** - Learn about the action system and plugins
3. **[Expressions and Templates](../expressions-and-templates/)** - Master dynamic configuration

Workflows are the foundation of Probe automation. With solid workflow design skills, you can build maintainable, efficient, and reliable automation processes.