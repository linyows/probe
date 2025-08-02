---
title: Actions
description: Understanding the action system, built-in actions, and plugin architecture
weight: 30
---

# Actions

Actions are the core execution units in Probe that perform actual work. They are implemented as plugins, making Probe extensible and modular. This guide explores the action system, built-in actions, and how to work with the plugin architecture.

## Action System Overview

The action system in Probe is built on a plugin architecture that provides:

- **Modularity**: Each action is a separate plugin
- **Extensibility**: Custom actions can be added easily
- **Isolation**: Actions run in separate processes for stability
- **Standardization**: All actions follow the same interface

### Action Execution Flow

1. **Plugin Discovery**: Probe identifies available action plugins
2. **Plugin Initialization**: The action plugin is started in a separate process
3. **Communication**: Probe communicates with plugins via gRPC
4. **Execution**: The plugin executes the requested action
5. **Response**: Results are returned to Probe for processing
6. **Cleanup**: Plugin processes are terminated after use

## Built-in Actions

Probe comes with several built-in actions that cover common use cases.

### HTTP Action

The `http` action is the most versatile and commonly used action for making HTTP/HTTPS requests.

#### Basic Usage

```yaml
- name: Simple GET Request
  action: http
  with:
    url: https://api.example.com/users
    method: GET
  test: res.status == 200
```

#### Complete HTTP Action Reference

```yaml
- name: Comprehensive HTTP Request
  action: http
  with:
    url: https://api.example.com/users/123        # Required: Target URL
    method: POST                                  # Optional: HTTP method (default: GET)
    headers:                                      # Optional: Request headers
      Content-Type: "application/json"
      Authorization: "Bearer {{env.API_TOKEN}}"
      X-Request-ID: "{{random_str(16)}}"
    body: |                                       # Optional: Request body
      {
        "name": "John Doe",
        "email": "john@example.com",
        "active": true
      }
    timeout: 30s                                  # Optional: Request timeout
    follow_redirects: true                        # Optional: Follow HTTP redirects
    verify_ssl: true                             # Optional: Verify SSL certificates
    max_redirects: 5                             # Optional: Maximum redirect count
  test: res.status == 200 && res.json.success == true
  outputs:
    user_id: res.json.user.id
    created_at: res.json.user.created_at
    response_time: res.time
```

#### HTTP Response Object

The HTTP action provides a rich response object:

```yaml
# Available response properties:
test: |
  res.status == 200 &&                    # HTTP status code
  res.time < 1000 &&                      # Response time in milliseconds
  res.body_size < 10000 &&               # Response body size in bytes
  res.headers["content-type"] == "application/json" &&  # Response headers
  res.json.success == true &&            # Parsed JSON body (if applicable)
  res.text.contains("success")           # Response body as text
```

#### Common HTTP Patterns

**API Authentication:**
```yaml
jobs:
  api-test:
    steps:
      - name: Authenticate
        id: auth
        action: http
        with:
          url: "{{env.API_BASE_URL}}/auth/login"
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "username": "{{env.API_USERNAME}}",
              "password": "{{env.API_PASSWORD}}"
            }
        test: res.status == 200
        outputs:
          access_token: res.json.access_token
          refresh_token: res.json.refresh_token

      - name: Make Authenticated Request
        action: http
        with:
          url: "{{env.API_BASE_URL}}/protected/resource"
          method: GET
          headers:
            Authorization: "Bearer {{outputs.auth.access_token}}"
        test: res.status == 200
```

**File Upload:**
```yaml
- name: Upload File
  action: http
  with:
    url: "{{env.API_URL}}/upload"
    method: POST
    headers:
      Content-Type: "multipart/form-data"
    body: |
      --boundary123
      Content-Disposition: form-data; name="file"; filename="test.txt"
      Content-Type: text/plain
      
      This is test file content
      --boundary123--
  test: res.status == 201
```

**GraphQL Queries:**
```yaml
- name: GraphQL Query
  action: http
  with:
    url: "{{env.GRAPHQL_ENDPOINT}}"
    method: POST
    headers:
      Content-Type: "application/json"
      Authorization: "Bearer {{env.GRAPHQL_TOKEN}}"
    body: |
      {
        "query": "query GetUser($id: ID!) { user(id: $id) { name email active } }",
        "variables": { "id": "{{env.TEST_USER_ID}}" }
      }
  test: res.status == 200 && res.json.data.user != null
  outputs:
    user_name: res.json.data.user.name
    user_email: res.json.data.user.email
```

### Hello Action

The `hello` action is primarily used for testing and demonstrations. It provides a simple way to verify plugin functionality.

```yaml
- name: Test Hello Action
  action: hello
  with:
    message: "Test message"           # Optional: Custom message
    delay: 1s                        # Optional: Artificial delay
  test: res.status == "success"
  outputs:
    greeting: res.message
    timestamp: res.timestamp
```

**Hello Action Response:**
```yaml
# Available response properties:
test: |
  res.status == "success" &&         # Always "success"
  res.message != null &&             # Greeting message
  res.timestamp != null              # Execution timestamp
```

### SMTP Action

The `smtp` action enables email sending capabilities for notifications and alerts.

```yaml
- name: Send Email Notification
  action: smtp
  with:
    host: smtp.gmail.com              # SMTP server host
    port: 587                         # SMTP server port
    username: "{{env.SMTP_USERNAME}}" # SMTP authentication username
    password: "{{env.SMTP_PASSWORD}}" # SMTP authentication password
    from: alerts@mycompany.com        # Sender email address
    to: ["admin@mycompany.com", "team@mycompany.com"]  # Recipients
    cc: ["manager@mycompany.com"]     # CC recipients (optional)
    bcc: ["audit@mycompany.com"]      # BCC recipients (optional)
    subject: "System Alert: {{env.ALERT_TYPE}}"       # Email subject
    body: |                           # Email body (plain text or HTML)
      System Alert Notification
      
      Alert Type: {{env.ALERT_TYPE}}
      Time: {{unixtime()}}
      Service: {{env.SERVICE_NAME}}
      
      Please investigate immediately.
    html: true                        # Optional: Send as HTML
    tls: true                        # Optional: Use TLS encryption
  test: res.status == "sent"
  outputs:
    message_id: res.message_id
    recipients_count: res.recipients_count
```

**SMTP Configuration Examples:**

**Gmail:**
```yaml
with:
  host: smtp.gmail.com
  port: 587
  username: "your-email@gmail.com"
  password: "your-app-password"
  tls: true
```

**AWS SES:**
```yaml
with:
  host: email-smtp.us-east-1.amazonaws.com
  port: 587
  username: "{{env.AWS_SES_USERNAME}}"
  password: "{{env.AWS_SES_PASSWORD}}"
  tls: true
```

**Office 365:**
```yaml
with:
  host: smtp.office365.com
  port: 587
  username: "your-email@company.com"
  password: "{{env.O365_PASSWORD}}"
  tls: true
```

## Advanced Action Usage

### Error Handling in Actions

Implement robust error handling for action failures:

```yaml
jobs:
  resilient-http-check:
    steps:
      - name: Primary Endpoint Check
        id: primary
        action: http
        with:
          url: "{{env.PRIMARY_URL}}/health"
          method: GET
          timeout: 10s
        test: res.status == 200
        continue_on_error: true
        outputs:
          primary_healthy: res.status == 200
          primary_response_time: res.time

      - name: Secondary Endpoint Check
        if: "!outputs.primary.primary_healthy"
        id: secondary
        action: http
        with:
          url: "{{env.SECONDARY_URL}}/health"
          method: GET
          timeout: 15s
        test: res.status == 200
        continue_on_error: true
        outputs:
          secondary_healthy: res.status == 200
          secondary_response_time: res.time

      - name: Alert on Total Failure
        if: "!outputs.primary.primary_healthy && !outputs.secondary.secondary_healthy"
        action: smtp
        with:
          host: "{{env.SMTP_HOST}}"
          port: 587
          username: "{{env.SMTP_USER}}"
          password: "{{env.SMTP_PASS}}"
          from: "alerts@company.com"
          to: ["ops-team@company.com"]
          subject: "CRITICAL: All endpoints down"
          body: |
            CRITICAL ALERT: All monitored endpoints are down
            
            Primary Endpoint: FAILED
            Secondary Endpoint: FAILED
            
            Time: {{unixtime()}}
            
            Immediate investigation required!
```

### Action Composition Patterns

Combine actions to create complex workflows:

```yaml
jobs:
  comprehensive-api-test:
    name: Comprehensive API Testing
    steps:
      # 1. Health check
      - name: Verify API Health
        id: health
        action: http
        with:
          url: "{{env.API_URL}}/health"
        test: res.status == 200
        outputs:
          api_version: res.json.version
          database_connected: res.json.database.connected

      # 2. Authentication test
      - name: Test Authentication
        id: auth
        action: http
        with:
          url: "{{env.API_URL}}/auth/token"
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "client_id": "{{env.CLIENT_ID}}",
              "client_secret": "{{env.CLIENT_SECRET}}",
              "grant_type": "client_credentials"
            }
        test: res.status == 200
        outputs:
          access_token: res.json.access_token
          token_expires: res.json.expires_in

      # 3. Functional test
      - name: Test Core Functionality
        id: functional
        action: http
        with:
          url: "{{env.API_URL}}/api/test"
          method: GET
          headers:
            Authorization: "Bearer {{outputs.auth.access_token}}"
        test: res.status == 200 && res.json.test_passed == true
        outputs:
          test_duration: res.time
          test_results: res.json.results

      # 4. Performance validation
      - name: Validate Performance
        if: outputs.functional.test_duration > 2000
        action: smtp
        with:
          host: "{{env.SMTP_HOST}}"
          port: 587
          username: "{{env.SMTP_USER}}"
          password: "{{env.SMTP_PASS}}"
          from: "performance@company.com"
          to: ["dev-team@company.com"]
          subject: "Performance Alert: Slow API Response"
          body: |
            Performance Alert
            
            API Version: {{outputs.health.api_version}}
            Response Time: {{outputs.functional.test_duration}}ms
            Expected: < 2000ms
            
            Please investigate performance degradation.

      # 5. Success notification
      - name: Success Report
        if: outputs.functional.test_duration <= 2000
        echo: |
          ✅ API Test Suite Completed Successfully
          
          Health Check: ✅ (v{{outputs.health.api_version}})
          Authentication: ✅ (expires in {{outputs.auth.token_expires}}s)
          Functionality: ✅ ({{outputs.functional.test_duration}}ms)
          Performance: ✅ (within acceptable limits)
```

### Dynamic Action Configuration

Configure actions dynamically based on runtime conditions:

```yaml
jobs:
  adaptive-monitoring:
    steps:
      - name: Determine Environment
        id: env
        action: http
        with:
          url: "{{env.CONFIG_SERVICE_URL}}/environment"
        test: res.status == 200
        outputs:
          environment: res.json.environment
          notification_level: res.json.notifications.level
          smtp_config: res.json.smtp

      - name: Environment-Specific Health Check
        id: health
        action: http
        with:
          url: "{{env.SERVICE_URL}}/health"
          timeout: "{{outputs.env.environment == 'production' ? '5s' : '30s'}}"
        test: res.status == 200
        outputs:
          service_status: res.json.status
          error_count: res.json.errors

      - name: Conditional Alert
        if: |
          outputs.health.error_count > 0 && 
          (outputs.env.environment == "production" || outputs.env.notification_level == "verbose")
        action: smtp
        with:
          host: "{{outputs.env.smtp_config.host}}"
          port: "{{outputs.env.smtp_config.port}}"
          username: "{{outputs.env.smtp_config.username}}"
          password: "{{env.SMTP_PASSWORD}}"
          from: "monitoring@company.com"
          to: "{{outputs.env.environment == 'production' ? ['ops@company.com', 'management@company.com'] : ['dev@company.com']}}"
          subject: "{{outputs.env.environment == 'production' ? 'PRODUCTION' : 'NON-PROD'}} Alert: Service Errors Detected"
          body: |
            Service Error Alert
            
            Environment: {{outputs.env.environment}}
            Service Status: {{outputs.health.service_status}}
            Error Count: {{outputs.health.error_count}}
            
            {{outputs.env.environment == "production" ? "IMMEDIATE ACTION REQUIRED" : "Please investigate when convenient"}}
```

## Plugin Architecture Deep Dive

### Plugin Communication

Probe uses gRPC for plugin communication, providing:

- **Type Safety**: Strong typing with Protocol Buffers
- **Performance**: Efficient binary serialization
- **Cross-Language**: Plugins can be written in any language supporting gRPC
- **Reliability**: Built-in error handling and timeouts

### Plugin Lifecycle

1. **Discovery**: Probe discovers available plugins at startup
2. **On-Demand Loading**: Plugins are loaded only when needed
3. **Process Isolation**: Each plugin runs in its own process
4. **Resource Management**: Plugin processes are cleaned up after use
5. **Error Isolation**: Plugin failures don't crash Probe

### Built-in Plugin Management

Probe manages built-in plugins automatically:

```bash
# Built-in plugins are embedded in the Probe binary
probe workflow.yml  # Automatically loads required plugins

# No separate installation needed for built-in actions:
# - http
# - hello  
# - smtp
```

## Action Best Practices

### 1. Timeout Configuration

Always set appropriate timeouts:

```yaml
# Good: Specific timeouts based on expected response time
- name: Quick Health Check
  action: http
  with:
    url: "{{env.API_URL}}/ping"
    timeout: 5s              # Quick ping should respond fast

- name: Complex Query
  action: http
  with:
    url: "{{env.API_URL}}/complex-report"
    timeout: 60s             # Complex operations need more time
```

### 2. Error Handling Strategy

Implement appropriate error handling:

```yaml
# Critical actions - fail fast
- name: Database Connectivity Check
  action: http
  with:
    url: "{{env.DB_URL}}/ping"
  test: res.status == 200
  continue_on_error: false   # Default: stop on failure

# Non-critical actions - continue on error
- name: Optional Analytics Update
  action: http
  with:
    url: "{{env.ANALYTICS_URL}}/update"
  test: res.status == 200
  continue_on_error: true    # Continue even if this fails
```

### 3. Secure Configuration

Handle sensitive data properly:

```yaml
# Good: Use environment variables for secrets
- name: Authenticated Request
  action: http
  with:
    url: "{{env.API_URL}}/secure"
    headers:
      Authorization: "Bearer {{env.API_TOKEN}}"  # From environment

# Good: Use secure SMTP configuration
- name: Send Alert
  action: smtp
  with:
    host: "{{env.SMTP_HOST}}"
    username: "{{env.SMTP_USER}}"
    password: "{{env.SMTP_PASS}}"     # Never hardcode passwords

# Avoid: Hardcoded secrets
- name: Bad Example
  action: http
  with:
    headers:
      Authorization: "Bearer secret-token-123"  # Never do this!
```

### 4. Response Validation

Validate action responses thoroughly:

```yaml
- name: Comprehensive API Test
  action: http
  with:
    url: "{{env.API_URL}}/users"
  test: |
    res.status == 200 &&
    res.headers["content-type"].contains("application/json") &&
    res.json.users != null &&
    res.json.users.length > 0 &&
    res.time < 1000
  outputs:
    user_count: res.json.users.length
    response_time: res.time
```

### 5. Meaningful Outputs

Define useful outputs for other steps:

```yaml
- name: User Creation Test
  action: http
  with:
    url: "{{env.API_URL}}/users"
    method: POST
    body: '{"name": "Test User", "email": "test@example.com"}'
  test: res.status == 201
  outputs:
    created_user_id: res.json.user.id
    created_user_email: res.json.user.email
    creation_timestamp: res.json.user.created_at
    response_time: res.time
```

## Custom Actions (Advanced)

While Probe comes with powerful built-in actions, you can extend it with custom actions for specialized needs.

### Custom Action Interface

Custom actions must implement the Actions interface:

```go
type Actions interface {
    Run(args []string, with map[string]string) (map[string]string, error)
}
```

### Action Plugin Structure

```go
// Example custom action plugin
package main

import (
    "github.com/linyows/probe"
    "github.com/hashicorp/go-plugin"
)

type CustomAction struct{}

func (c *CustomAction) Run(args []string, with map[string]string) (map[string]string, error) {
    // Custom action logic here
    return map[string]string{
        "status": "success",
        "result": "custom action completed",
    }, nil
}

func main() {
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: probe.Handshake,
        Plugins: map[string]plugin.Plugin{
            "actions": &probe.ActionsPlugin{Impl: &CustomAction{}},
        },
        GRPCServer: plugin.DefaultGRPCServer,
    })
}
```

## What's Next?

Now that you understand the action system, explore:

1. **[Expressions and Templates](../expressions-and-templates/)** - Learn dynamic configuration and testing
2. **[Data Flow](../data-flow/)** - Understand how data moves between actions
3. **[How-tos](../../how-tos/)** - See practical action usage patterns

Actions are the workhorses of Probe. Master the built-in actions and understand the plugin architecture to build powerful, extensible automation workflows.