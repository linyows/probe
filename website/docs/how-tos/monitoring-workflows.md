---
title: Monitoring Workflows
description: Build comprehensive monitoring systems with Probe workflows
weight: 10
---

# Monitoring Workflows

This guide shows you how to build comprehensive monitoring systems using Probe. You'll learn to create workflows that check service health, monitor performance, and alert on issues.

## Basic Service Monitoring

### Simple Health Check

Start with a basic health check workflow:

```yaml
name: Basic Service Health Check
description: Monitor essential service endpoints

env:
  API_BASE_URL: https://api.yourcompany.com
  HEALTH_ENDPOINT: /health
  TIMEOUT: 30s

defaults:
  http:
    timeout: "{{env.TIMEOUT}}"
    headers:
      User-Agent: "Probe Monitor v1.0"

jobs:
  health-check:
    name: Service Health Check
    steps:
      - name: API Health Check
        action: http
        with:
          url: "{{env.API_BASE_URL}}{{env.HEALTH_ENDPOINT}}"
        test: res.status == 200
        outputs:
          api_healthy: res.status == 200
          response_time: res.time
          api_version: res.json.version

      - name: Health Status Report
        echo: |
          🏥 Health Check Results:
          
          API Status: {{outputs.api_healthy ? "✅ Healthy" : "❌ Down"}}
          Response Time: {{outputs.response_time}}ms
          API Version: {{outputs.api_version}}
          Timestamp: {{unixtime()}}
```

**Usage:**
```bash
probe health-check.yml
```

### Multi-Service Health Monitoring

Monitor multiple services in parallel:

```yaml
name: Multi-Service Health Monitor
description: Check health of all critical services

env:
  USER_SERVICE_URL: https://users.api.yourcompany.com
  ORDER_SERVICE_URL: https://orders.api.yourcompany.com
  PAYMENT_SERVICE_URL: https://payments.api.yourcompany.com
  NOTIFICATION_SERVICE_URL: https://notifications.api.yourcompany.com

jobs:
  user-service:
    name: User Service Health
    steps:
      - name: User Service Check
        action: http
        with:
          url: "{{env.USER_SERVICE_URL}}/health"
        test: res.status == 200
        outputs:
          healthy: res.status == 200
          response_time: res.time
          user_count: res.json.active_users

  order-service:
    name: Order Service Health
    steps:
      - name: Order Service Check
        action: http
        with:
          url: "{{env.ORDER_SERVICE_URL}}/health"
        test: res.status == 200
        outputs:
          healthy: res.status == 200
          response_time: res.time
          pending_orders: res.json.pending_orders

  payment-service:
    name: Payment Service Health
    steps:
      - name: Payment Service Check
        action: http
        with:
          url: "{{env.PAYMENT_SERVICE_URL}}/health"
        test: res.status == 200
        outputs:
          healthy: res.status == 200
          response_time: res.time
          transaction_queue: res.json.queue_length

  notification-service:
    name: Notification Service Health
    steps:
      - name: Notification Service Check
        action: http
        with:
          url: "{{env.NOTIFICATION_SERVICE_URL}}/health"
        test: res.status == 200
        outputs:
          healthy: res.status == 200
          response_time: res.time
          queue_size: res.json.notification_queue

  summary-report:
    name: Health Summary
    needs: [user-service, order-service, payment-service, notification-service]
    steps:
      - name: Generate Health Report
        echo: |
          🎯 Multi-Service Health Report
          ===============================
          
          User Service: {{outputs.user-service.healthy ? "✅" : "❌"}} ({{outputs.user-service.response_time}}ms)
            Active Users: {{outputs.user-service.user_count}}
          
          Order Service: {{outputs.order-service.healthy ? "✅" : "❌"}} ({{outputs.order-service.response_time}}ms)
            Pending Orders: {{outputs.order-service.pending_orders}}
          
          Payment Service: {{outputs.payment-service.healthy ? "✅" : "❌"}} ({{outputs.payment-service.response_time}}ms)
            Transaction Queue: {{outputs.payment-service.transaction_queue}}
          
          Notification Service: {{outputs.notification-service.healthy ? "✅" : "❌"}} ({{outputs.notification-service.response_time}}ms)
            Notification Queue: {{outputs.notification-service.queue_size}}
          
          Overall System Status: {{
            outputs.user-service.healthy && 
            outputs.order-service.healthy && 
            outputs.payment-service.healthy && 
            outputs.notification-service.healthy ? 
            "🟢 ALL SYSTEMS OPERATIONAL" : "🔴 ISSUES DETECTED"
          }}
          
          Timestamp: {{unixtime()}}
```

## Database and Infrastructure Monitoring

### Database Health Monitoring

```yaml
name: Database Health Monitor
description: Monitor database connectivity and performance

env:
  DB_HOST: db.yourcompany.com
  DB_PORT: 5432
  DB_NAME: production
  REDIS_HOST: redis.yourcompany.com
  REDIS_PORT: 6379

jobs:
  database-connectivity:
    name: Database Connectivity
    steps:
      - name: PostgreSQL Connection Test
        action: http
        with:
          url: "{{env.DB_API_URL}}/ping"
        test: res.status == 200
        outputs:
          db_connected: res.status == 200
          connection_time: res.time
          active_connections: res.json.active_connections
          max_connections: res.json.max_connections

      - name: Database Performance Check
        action: http
        with:
          url: "{{env.DB_API_URL}}/stats"
        test: res.status == 200 && res.json.query_performance.avg_ms < 100
        outputs:
          avg_query_time: res.json.query_performance.avg_ms
          slow_queries: res.json.slow_queries.count
          db_size_mb: res.json.database_size_mb

  cache-monitoring:
    name: Cache System Health
    steps:
      - name: Redis Connection Test
        action: http
        with:
          url: "{{env.CACHE_API_URL}}/ping"
        test: res.status == 200
        outputs:
          cache_connected: res.status == 200
          cache_response_time: res.time
          memory_usage_percent: res.json.memory.usage_percent
          keys_count: res.json.keys.total

      - name: Cache Performance Check
        action: http
        with:
          url: "{{env.CACHE_API_URL}}/stats"
        test: res.status == 200 && res.json.hit_rate > 0.8
        outputs:
          hit_rate: res.json.hit_rate
          miss_rate: res.json.miss_rate
          evicted_keys: res.json.evicted_keys

  infrastructure-report:
    name: Infrastructure Report
    needs: [database-connectivity, cache-monitoring]
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
          Connection: {{outputs.cache-monitoring.cache_connected ? "✅ Connected" : "❌ Failed"}}
          Response Time: {{outputs.cache-monitoring.cache_response_time}}ms
          Memory Usage: {{outputs.cache-monitoring.memory_usage_percent}}%
          Total Keys: {{outputs.cache-monitoring.keys_count}}
          Hit Rate: {{(outputs.cache-monitoring.hit_rate * 100)}}%
          Miss Rate: {{(outputs.cache-monitoring.miss_rate * 100)}}%
          
          Performance Alerts:
          {{outputs.database-connectivity.avg_query_time > 100 ? "⚠️ Database queries are slow (>" + outputs.database-connectivity.avg_query_time + "ms)" : ""}}
          {{outputs.database-connectivity.slow_queries > 10 ? "⚠️ High number of slow queries (" + outputs.database-connectivity.slow_queries + ")" : ""}}
          {{outputs.cache-monitoring.memory_usage_percent > 80 ? "⚠️ Cache memory usage high (" + outputs.cache-monitoring.memory_usage_percent + "%)" : ""}}
          {{outputs.cache-monitoring.hit_rate < 0.8 ? "⚠️ Cache hit rate low (" + (outputs.cache-monitoring.hit_rate * 100) + "%)" : ""}}
```

## Comprehensive System Monitoring

### Full-Stack Monitoring Workflow

```yaml
name: Full-Stack System Monitor
description: Comprehensive monitoring of all system components

env:
  # Service URLs
  FRONTEND_URL: https://app.yourcompany.com
  API_GATEWAY_URL: https://api.yourcompany.com
  
  # Monitoring thresholds
  MAX_RESPONSE_TIME: 2000
  MIN_SUCCESS_RATE: 0.95
  MAX_ERROR_RATE: 0.05

jobs:
  # Tier 1: Infrastructure Layer
  infrastructure-health:
    name: Infrastructure Health Check
    steps:
      - name: Load Balancer Health
        action: http
        with:
          url: "{{env.LOAD_BALANCER_URL}}/health"
        test: res.status == 200
        outputs:
          lb_healthy: res.status == 200
          active_backends: res.json.active_backends
          total_backends: res.json.total_backends

      - name: CDN Performance
        action: http
        with:
          url: "{{env.CDN_URL}}/health"
        test: res.status == 200 && res.time < 500
        outputs:
          cdn_healthy: res.status == 200
          cdn_response_time: res.time
          cache_hit_ratio: res.json.cache_hit_ratio

  # Tier 2: Application Layer
  application-health:
    name: Application Health Check
    needs: [infrastructure-health]
    steps:
      - name: Frontend Health
        action: http
        with:
          url: "{{env.FRONTEND_URL}}/health"
        test: res.status == 200
        outputs:
          frontend_healthy: res.status == 200
          frontend_version: res.json.version
          frontend_build: res.json.build

      - name: API Gateway Health
        action: http
        with:
          url: "{{env.API_GATEWAY_URL}}/health"
        test: res.status == 200
        outputs:
          gateway_healthy: res.status == 200
          gateway_version: res.json.version
          registered_services: res.json.services.length

  # Tier 3: Business Logic Layer
  business-logic-health:
    name: Business Logic Health
    needs: [application-health]
    steps:
      - name: User Service Functional Test
        action: http
        with:
          url: "{{env.API_GATEWAY_URL}}/users/health-check"
        test: res.status == 200 && res.json.functional_test_passed == true
        outputs:
          user_service_functional: res.json.functional_test_passed
          active_sessions: res.json.active_sessions

      - name: Order Service Functional Test
        action: http
        with:
          url: "{{env.API_GATEWAY_URL}}/orders/health-check"
        test: res.status == 200 && res.json.functional_test_passed == true
        outputs:
          order_service_functional: res.json.functional_test_passed
          processing_queue_length: res.json.queue_length

  # Tier 4: Performance Validation
  performance-validation:
    name: Performance Validation
    needs: [business-logic-health]
    steps:
      - name: End-to-End Performance Test
        action: http
        with:
          url: "{{env.API_GATEWAY_URL}}/performance/e2e-test"
          method: POST
          body: |
            {
              "test_type": "quick_validation",
              "max_duration_seconds": 30
            }
        test: |
          res.status == 200 && 
          res.json.success_rate >= {{env.MIN_SUCCESS_RATE}} &&
          res.json.avg_response_time <= {{env.MAX_RESPONSE_TIME}}
        outputs:
          success_rate: res.json.success_rate
          avg_response_time: res.json.avg_response_time
          p95_response_time: res.json.p95_response_time
          error_rate: res.json.error_rate

  # Tier 5: Security and Compliance
  security-checks:
    name: Security Health Checks
    needs: [performance-validation]
    steps:
      - name: SSL Certificate Check
        action: http
        with:
          url: "{{env.SECURITY_API_URL}}/ssl-check"
          method: POST
          body: |
            {
              "domains": [
                "{{env.FRONTEND_URL}}",
                "{{env.API_GATEWAY_URL}}"
              ]
            }
        test: res.status == 200 && res.json.all_certificates_valid == true
        outputs:
          ssl_valid: res.json.all_certificates_valid
          cert_expiry_days: res.json.min_days_to_expiry

      - name: Security Headers Check
        action: http
        with:
          url: "{{env.SECURITY_API_URL}}/headers-check"
          method: POST
          body: |
            {
              "url": "{{env.FRONTEND_URL}}"
            }
        test: res.status == 200 && res.json.security_score >= 0.8
        outputs:
          security_score: res.json.security_score
          missing_headers: res.json.missing_headers

  # Final Report
  system-health-report:
    name: System Health Report
    needs: [infrastructure-health, application-health, business-logic-health, performance-validation, security-checks]
    steps:
      - name: Generate Comprehensive Report
        echo: |
          🌐 Full-Stack System Health Report
          ===================================
          Generated: {{unixtime()}}
          
          📊 INFRASTRUCTURE LAYER
          Load Balancer: {{outputs.infrastructure-health.lb_healthy ? "✅ Healthy" : "❌ Issues"}}
            Backends: {{outputs.infrastructure-health.active_backends}}/{{outputs.infrastructure-health.total_backends}} active
          CDN: {{outputs.infrastructure-health.cdn_healthy ? "✅ Healthy" : "❌ Issues"}} ({{outputs.infrastructure-health.cdn_response_time}}ms)
            Cache Hit Ratio: {{(outputs.infrastructure-health.cache_hit_ratio * 100)}}%
          
          🖥️ APPLICATION LAYER
          Frontend: {{outputs.application-health.frontend_healthy ? "✅ Healthy" : "❌ Issues"}}
            Version: {{outputs.application-health.frontend_version}} (Build: {{outputs.application-health.frontend_build}})
          API Gateway: {{outputs.application-health.gateway_healthy ? "✅ Healthy" : "❌ Issues"}}
            Version: {{outputs.application-health.gateway_version}}
            Services: {{outputs.application-health.registered_services}} registered
          
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
          SSL Certificates: {{outputs.security-checks.ssl_valid ? "✅ Valid" : "❌ Issues"}}
            Expiry: {{outputs.security-checks.cert_expiry_days}} days minimum
          Security Headers: Score {{(outputs.security-checks.security_score * 100)}}%
          {{outputs.security-checks.missing_headers ? "Missing Headers: " + outputs.security-checks.missing_headers : ""}}
          
          🎯 OVERALL SYSTEM STATUS
          {{
            outputs.infrastructure-health.lb_healthy &&
            outputs.infrastructure-health.cdn_healthy &&
            outputs.application-health.frontend_healthy &&
            outputs.application-health.gateway_healthy &&
            outputs.business-logic-health.user_service_functional &&
            outputs.business-logic-health.order_service_functional &&
            outputs.performance-validation.success_rate >= env.MIN_SUCCESS_RATE &&
            outputs.performance-validation.avg_response_time <= env.MAX_RESPONSE_TIME &&
            outputs.security-checks.ssl_valid &&
            outputs.security-checks.security_score >= 0.8
            ? "🟢 ALL SYSTEMS OPERATIONAL" 
            : "🔴 ISSUES REQUIRE ATTENTION"
          }}
          
          ⚠️ ALERTS
          {{outputs.infrastructure-health.active_backends != outputs.infrastructure-health.total_backends ? "• Load balancer has inactive backends" : ""}}
          {{outputs.infrastructure-health.cdn_response_time > 1000 ? "• CDN response time is high" : ""}}
          {{outputs.performance-validation.success_rate < env.MIN_SUCCESS_RATE ? "• Success rate below threshold" : ""}}
          {{outputs.performance-validation.avg_response_time > env.MAX_RESPONSE_TIME ? "• Average response time exceeds threshold" : ""}}
          {{outputs.security-checks.cert_expiry_days < 30 ? "• SSL certificates expiring soon" : ""}}
          {{outputs.security-checks.security_score < 0.8 ? "• Security headers need improvement" : ""}}
```

## Alerting and Notification Integration

### Monitoring with Email Alerts

```yaml
name: Monitoring with Email Alerts
description: Health monitoring with automated email notifications

env:
  # SMTP Configuration
  SMTP_HOST: smtp.gmail.com
  SMTP_PORT: 587
  SMTP_USERNAME: alerts@yourcompany.com
  ALERT_RECIPIENTS: ["ops@yourcompany.com", "dev-team@yourcompany.com"]
  
  # Monitoring Configuration
  CRITICAL_SERVICES: ["user-service", "payment-service", "order-service"]

jobs:
  health-monitoring:
    name: Health Monitoring
    steps:
      - name: User Service Check
        id: user-service
        action: http
        with:
          url: "{{env.USER_SERVICE_URL}}/health"
        test: res.status == 200
        continue_on_error: true
        outputs:
          healthy: res.status == 200
          status_code: res.status
          response_time: res.time

      - name: Payment Service Check
        id: payment-service
        action: http
        with:
          url: "{{env.PAYMENT_SERVICE_URL}}/health"
        test: res.status == 200
        continue_on_error: true
        outputs:
          healthy: res.status == 200
          status_code: res.status
          response_time: res.time

      - name: Order Service Check
        id: order-service
        action: http
        with:
          url: "{{env.ORDER_SERVICE_URL}}/health"
        test: res.status == 200
        continue_on_error: true
        outputs:
          healthy: res.status == 200
          status_code: res.status
          response_time: res.time

  alert-processing:
    name: Alert Processing
    needs: [health-monitoring]
    steps:
      - name: Critical Service Alert
        if: "!outputs.user-service.healthy || !outputs.payment-service.healthy || !outputs.order-service.healthy"
        action: smtp
        with:
          host: "{{env.SMTP_HOST}}"
          port: "{{env.SMTP_PORT}}"
          username: "{{env.SMTP_USERNAME}}"
          password: "{{env.SMTP_PASSWORD}}"
          from: "{{env.SMTP_USERNAME}}"
          to: "{{env.ALERT_RECIPIENTS}}"
          subject: "🚨 CRITICAL: Service Health Alert - {{unixtime()}}"
          body: |
            CRITICAL SERVICE HEALTH ALERT
            =============================
            
            Time: {{unixtime()}}
            Environment: {{env.ENVIRONMENT || "Production"}}
            
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
            
            Monitoring Dashboard: {{env.DASHBOARD_URL}}
            Incident Management: {{env.INCIDENT_URL}}

      - name: Performance Warning Alert
        if: |
          (outputs.user-service.healthy && outputs.user-service.response_time > 2000) ||
          (outputs.payment-service.healthy && outputs.payment-service.response_time > 2000) ||
          (outputs.order-service.healthy && outputs.order-service.response_time > 2000)
        action: smtp
        with:
          host: "{{env.SMTP_HOST}}"
          port: "{{env.SMTP_PORT}}"
          username: "{{env.SMTP_USERNAME}}"
          password: "{{env.SMTP_PASSWORD}}"
          from: "{{env.SMTP_USERNAME}}"
          to: "{{env.ALERT_RECIPIENTS}}"
          subject: "⚠️ WARNING: Performance Degradation Detected"
          body: |
            PERFORMANCE WARNING
            ===================
            
            Time: {{unixtime()}}
            Environment: {{env.ENVIRONMENT || "Production"}}
            
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

## Environment-Specific Monitoring

### Multi-Environment Configuration

**base-monitoring.yml:**
```yaml
name: Service Health Monitor
description: Base monitoring workflow for all environments

defaults:
  http:
    headers:
      User-Agent: "Probe Monitor"
      Accept: "application/json"

jobs:
  service-health:
    name: Service Health Check
    steps:
      - name: API Health
        action: http
        with:
          url: "{{env.API_URL}}/health"
        test: res.status == 200
        outputs:
          api_healthy: res.status == 200
          response_time: res.time

      - name: Database Health
        action: http
        with:
          url: "{{env.DB_API_URL}}/ping"
        test: res.status == 200
        outputs:
          db_healthy: res.status == 200
          db_response_time: res.time

  monitoring-report:
    name: Monitoring Report
    needs: [service-health]
    steps:
      - name: Status Report
        echo: |
          Environment: {{env.ENVIRONMENT}}
          API: {{outputs.service-health.api_healthy ? "✅" : "❌"}} ({{outputs.service-health.response_time}}ms)
          Database: {{outputs.service-health.db_healthy ? "✅" : "❌"}} ({{outputs.service-health.db_response_time}}ms)
```

**development.yml:**
```yaml
env:
  ENVIRONMENT: development
  API_URL: http://localhost:3000
  DB_API_URL: http://localhost:5432

defaults:
  http:
    timeout: 60s  # More lenient for development
```

**production.yml:**
```yaml
env:
  ENVIRONMENT: production
  API_URL: https://api.yourcompany.com
  DB_API_URL: https://db-api.yourcompany.com

defaults:
  http:
    timeout: 10s  # Strict timeouts for production

jobs:
  # Add production-specific security monitoring
  security-monitoring:
    name: Security Monitoring
    needs: [service-health]
    steps:
      - name: SSL Certificate Check
        action: http
        with:
          url: "{{env.SECURITY_API_URL}}/ssl-status"
        test: res.status == 200 && res.json.all_valid == true
        outputs:
          ssl_valid: res.json.all_valid
          days_to_expiry: res.json.min_days_to_expiry
```

**Usage:**
```bash
# Development monitoring
probe base-monitoring.yml,development.yml

# Production monitoring (includes security checks)
probe base-monitoring.yml,production.yml
```

## Best Practices

### 1. Monitoring Strategy

- **Layer your monitoring**: Infrastructure → Application → Business Logic
- **Set appropriate timeouts**: Strict for production, lenient for development
- **Use continue_on_error**: For non-critical checks
- **Implement gradual alerting**: Info → Warning → Critical

### 2. Alert Fatigue Prevention

```yaml
# Good: Conditional alerting
- name: Smart Alerting
  if: errors.count > 5 && duration > 300  # Only alert on sustained issues
  action: smtp
  # ...

# Avoid: Alert on every issue
- name: Noisy Alerting
  if: any_error_detected
  action: smtp
  # Creates alert fatigue
```

### 3. Performance Considerations

```yaml
# Good: Parallel independent checks
jobs:
  service-a-check:    # Runs in parallel
  service-b-check:    # Runs in parallel
  service-c-check:    # Runs in parallel

# Good: Efficient outputs
outputs:
  service_healthy: res.status == 200  # Boolean flag
  response_time: res.time            # Specific metric
  # Avoid storing entire response: full_response: res.json
```

### 4. Documentation and Maintenance

```yaml
name: Well-Documented Monitor
description: |
  Monitoring workflow for the e-commerce platform.
  
  Checks:
  - User service health and performance
  - Order processing service
  - Payment gateway connectivity
  - Database performance
  
  Alerting:
  - Critical: Service completely down
  - Warning: Performance degradation
  - Info: All systems normal
  
  Expected execution time: 30-60 seconds
  
  Maintenance:
  - Review thresholds monthly
  - Update service URLs when services move
  - Test alert channels quarterly
```

## Troubleshooting Common Issues

### 1. Service Discovery Problems

```yaml
- name: Service Discovery Check
  action: http
  with:
    url: "{{env.SERVICE_REGISTRY_URL}}/services"
  test: res.status == 200 && res.json.services.length > 0
  outputs:
    available_services: res.json.services.map(s -> s.name)
    service_count: res.json.services.length
```

### 2. Network Connectivity Issues

```yaml
- name: Network Connectivity Test
  action: http
  with:
    url: "{{env.EXTERNAL_HEALTH_CHECK_URL}}"
    timeout: 5s
  test: res.status == 200
  continue_on_error: true
  outputs:
    external_connectivity: res.status == 200
```

### 3. Authentication Problems

```yaml
- name: Authentication Health Check
  action: http
  with:
    url: "{{env.AUTH_SERVICE_URL}}/health"
    headers:
      Authorization: "Bearer {{env.HEALTH_CHECK_TOKEN}}"
  test: res.status == 200
  continue_on_error: true
  outputs:
    auth_service_healthy: res.status == 200
```

## What's Next?

Now that you can build monitoring workflows, explore:

- **[API Testing](../api-testing/)** - Comprehensive API testing strategies
- **[Error Handling Strategies](../error-handling-strategies/)** - Robust error handling patterns
- **[Performance Testing](../performance-testing/)** - Load testing and performance validation

Monitoring is the foundation of reliable systems. Use these patterns to build comprehensive monitoring that catches issues before they impact users.