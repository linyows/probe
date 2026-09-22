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
  uses: http
  with:
    url: https://api.example.com/users
    method: GET
  test: res.code == 200
```

#### Complete HTTP Action Reference

```yaml
- name: Comprehensive HTTP Request
  uses: http
  timeout: 30s                                  # Optional: Request timeout
  with:
    url: https://api.example.com/users/123        # Required: Target URL
    method: POST                                  # Optional: HTTP method (default: GET)
    headers:                                      # Optional: Request headers
      Content-Type: "application/json"
      Authorization: "Bearer {{vars.api_token}}"
      X-Request-ID: "{{random_str(16)}}"
    body: |                                       # Optional: Request body
      {
        "name": "John Doe",
        "email": "john@example.com",
        "active": true
      }
  test: res.code == 200 && res.body.success == true
  outputs:
    user_id: res.body.user.id
    created_at: res.body.user.created_at
    response_time: (rt.sec * 1000)
```

#### HTTP Response Object

The HTTP action provides a rich response object:

```yaml
# Available response properties:
test: |
  res.code == 200 &&                    # HTTP status code
  (rt.sec * 1000) < 1000 &&                      # Response time in milliseconds
  res.body_size < 10000 &&               # Response body size in bytes
  res.headers["Content-Type"] == "application/json" &&  # Response headers
  res.body.success == true &&            # Parsed JSON body (if applicable)
  res.body contains "success"           # Response body as text
```

#### Common HTTP Patterns

**API Authentication:**
```yaml
jobs:
- id: api-test
  name: api-test
  steps:
    - name: Authenticate
      id: auth
      uses: http
      with:
        url: "{{vars.api_base_url}}/auth/login"
        method: POST
        headers:
          Content-Type: "application/json"
        body: |
          {
            "username": "{{vars.api_username}}",
            "password": "{{vars.api_password}}"
          }
      test: res.code == 200
      outputs:
        access_token: res.body.access_token
        refresh_token: res.body.refresh_token

    - name: Make Authenticated Request
      uses: http
      with:
        url: "{{vars.api_base_url}}/protected/resource"
        method: GET
        headers:
          Authorization: "Bearer {{outputs.auth.access_token}}"
      test: res.code == 200
```

**File Upload:**
```yaml
- name: Upload File
  uses: http
  with:
    url: "{{vars.api_url}}/upload"
    method: POST
    headers:
      Content-Type: "multipart/form-data"
    body: |
      --boundary123
      Content-Disposition: form-data; name="file"; filename="test.txt"
      Content-Type: text/plain
      
      This is test file content
      --boundary123--
  test: res.code == 201
```

**GraphQL Queries:**
```yaml
- name: GraphQL Query
  uses: http
  with:
    url: "{{vars.graphql_endpoint}}"
    method: POST
    headers:
      Content-Type: "application/json"
      Authorization: "Bearer {{vars.graphql_token}}"
    body: |
      {
        "query": "query GetUser($id: ID!) { user(id: $id) { name email active } }",
        "variables": { "id": "{{vars.test_user_id}}" }
      }
  test: res.code == 200 && res.body.data.user != null
  outputs:
    user_name: res.body.data.user.name
    user_email: res.body.data.user.email
```

### Shell Action

The `shell` action enables secure execution of shell commands and scripts within workflows. It provides comprehensive output capture, timeout protection, and environment variable support.

#### Basic Usage

```yaml
- name: Build Application
  uses: shell
  with:
    cmd: "npm run build"
    workdir: "/app"
    timeout: "5m"
  test: res.code == 0
```

#### Complete Shell Action Reference

```yaml
- name: Deploy Application
  uses: shell
  with:
    cmd: "./deploy.sh production"              # Required: Command to execute
    shell: "/bin/bash"                        # Optional: Shell to use (default: /bin/sh)
    workdir: "/deploy"                        # Optional: Working directory (absolute path)
    timeout: "15m"                           # Optional: Execution timeout (default: 30s)
    env:                                     # Optional: Environment variables
      DEPLOY_ENV: "production"
      API_KEY: "{{vars.production_api_key}}"
      BUILD_VERSION: "{{vars.version}}"
  test: res.code == 0 && (res.stdout | contains("Deploy successful"))
  outputs:
    deploy_time: res.rt
    deploy_log: res.stdout
```

#### Shell Response Object

```yaml
# Available response properties:
test: |
  res.code == 0 &&                         # Exit code (0 = success)
  res.stdout contains "success" &&        # Standard output
  res.stderr == "" &&                      # Standard error (empty = no errors)
  req.cmd == "npm run build" &&           # Original command
  req.shell == "/bin/bash"                # Shell used for execution
```

#### Common Shell Patterns

**Build and Test Pipeline:**
```yaml
jobs:
- id: build-and-test
  name: build-and-test
  steps:
    - name: Install Dependencies
      uses: shell
      with:
        cmd: "npm ci"
        workdir: "/app"
        timeout: "5m"
      test: res.code == 0

    - name: Run Tests
      uses: shell
      with:
        cmd: "npm test"
        workdir: "/app"
        env:
          NODE_ENV: "test"
          CI: "true"
      test: res.code == 0

    - name: Build Application
      uses: shell
      with:
        cmd: "npm run build"
        workdir: "/app"
        env:
          NODE_ENV: "production"
      test: res.code == 0
```

**System Health Monitoring:**
```yaml
- name: Check System Health
  uses: shell
  with:
    cmd: |
      echo "=== System Health Report ===" &&
      echo "CPU Usage: $(top -bn1 | grep Cpu | cut -d' ' -f2)" &&
      echo "Memory: $(free -h | grep Mem)" &&
      echo "Disk: $(df -h /)"
  test: res.code == 0
  outputs:
    health_report: res.stdout
```

#### Security Features

The shell action implements multiple security layers:

- **Shell Restriction**: Only allows approved shell executables
- **Path Validation**: Working directories must be absolute paths
- **Timeout Protection**: Prevents runaway processes
- **Environment Isolation**: Safe environment variable handling
- **Output Sanitization**: Secure capture of command output

### Hello Action

The `hello` action is primarily used for testing and demonstrations. It provides a simple way to verify plugin functionality.

```yaml
- name: Test Hello Action
  id: hello
  uses: hello
  with:
    message: "Test message"           # Echoed back on res
  echo: "{{res.message}}"
  outputs:
    greeting: res.message
```

**Hello Action Response:** the action takes no parameters of its own. Every key given in `with` comes back on `res`, and `status` is always `0`.

```yaml
test: status == 0 && res.message != null
```

### SMTP Action

The `smtp` action enables email sending capabilities for notifications and alerts.

```yaml
- name: Send Email Notification
  uses: smtp
  with:
    addr: "smtp.gmail.com              # SMTP server host:587                         # SMTP server port"
    from: alerts@mycompany.com        # Sender email address
    to: ["admin@mycompany.com", "team@mycompany.com"]  # Recipients
    subject: "System Alert: {{vars.alert_type}}"       # Email subject
    session: 1
    message: 1
    length: 500
  echo: |                           # Email body (plain text or HTML)
  test: res.code == "sent"
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
  username: "{{vars.aws_ses_username}}"
  password: "{{vars.aws_ses_password}}"
  tls: true
```

**Office 365:**
```yaml
with:
  host: smtp.office365.com
  port: 587
  username: "your-email@company.com"
  password: "{{vars.o365_password}}"
  tls: true
```

## Advanced Action Usage

### Error Handling in Actions

Implement robust error handling for action failures:

```yaml
jobs:
- id: resilient-http-check
  name: resilient-http-check
  steps:
    - name: Primary Endpoint Check
      id: primary
      uses: http
      timeout: 10s
      with:
        url: "{{vars.primary_url}}/health"
        method: GET
      test: res.code == 200
      outputs:
        primary_healthy: res.code == 200
        primary_response_time: (rt.sec * 1000)

    - name: Secondary Endpoint Check
      id: secondary
      uses: http
      timeout: 15s
      with:
        url: "{{vars.secondary_url}}/health"
        method: GET
      test: res.code == 200
      outputs:
        secondary_healthy: res.code == 200
        secondary_response_time: (rt.sec * 1000)

    - name: Alert on Total Failure
      uses: smtp
      with:
        addr: "{{vars.smtp_host}}:587"
        from: "alerts@company.com"
        to: "ops-team@company.com"
        subject: "CRITICAL: All endpoints down"
        session: 1
        message: 1
        length: 500
      echo: |
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
- id: comprehensive-api-test
  name: Comprehensive API Testing
  steps:
    # 1. Health check
    - name: Verify API Health
      id: health
      uses: http
      with:
        method: GET
        url: "{{vars.api_url}}/health"
      test: res.code == 200
      outputs:
        api_version: res.body.version
        database_connected: res.body.database.connected

    # 2. Authentication test
    - name: Test Authentication
      id: auth
      uses: http
      with:
        url: "{{vars.api_url}}/auth/token"
        method: POST
        headers:
          Content-Type: "application/json"
        body: |
          {
            "client_id": "{{vars.client_id}}",
            "client_secret": "{{vars.client_secret}}",
            "grant_type": "client_credentials"
          }
      test: res.code == 200
      outputs:
        access_token: res.body.access_token
        token_expires: res.body.expires_in

    # 3. Functional test
    - name: Test Core Functionality
      id: functional
      uses: http
      with:
        url: "{{vars.api_url}}/api/test"
        method: GET
        headers:
          Authorization: "Bearer {{outputs.auth.access_token}}"
      test: res.code == 200 && res.body.test_passed == true
      outputs:
        test_duration: (rt.sec * 1000)
        test_results: res.body.results

    # 4. Performance validation
    - name: Validate Performance
      uses: smtp
      with:
        addr: "{{vars.smtp_host}}:587"
        from: "performance@company.com"
        to: "dev-team@company.com"
        subject: "Performance Alert: Slow API Response"
        session: 1
        message: 1
        length: 500
      echo: |
        Performance Alert
          
        API Version: {{outputs.health.api_version}}
        Response Time: {{outputs.functional.test_duration}}ms
        Expected: < 2000ms
          
        Please investigate performance degradation.

        uccess notification
    - name: Success Report
      uses: hello
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
- id: adaptive-monitoring
  name: adaptive-monitoring
  steps:
    - name: Determine Environment
      id: env
      uses: http
      with:
        method: GET
        url: "{{vars.config_service_url}}/environment"
      test: res.code == 200
      outputs:
        environment: res.body.environment
        notification_level: res.body.notifications.level
        smtp_config: res.body.smtp

    - name: Environment-Specific Health Check
      id: health
      uses: http
      timeout: "{{outputs.vars.environment == 'production' ? '5s' : '30s'}}"
      with:
        method: GET
        url: "{{vars.service_url}}/health"
      test: res.code == 200
      outputs:
        service_status: res.body.status
        error_count: res.body.errors

    - name: Conditional Alert
      uses: smtp
      with:
        addr: "{{outputs.vars.smtp_config.host}}:{{outputs.vars.smtp_config.port}}"
        from: "monitoring@company.com"
        to: "{{outputs.vars.environment == 'production' ? ['ops@company.com', 'management@company.com'] : ['dev@company.com']}}"
        subject: "{{outputs.vars.environment == 'production' ? 'PRODUCTION' : 'NON-PROD'}} Alert: Service Errors Detected"
        session: 1
        message: 1
        length: 500
      echo: |
        Service Error Alert
          
        Environment: {{outputs.vars.environment}}
        Service Status: {{outputs.health.service_status}}
        Error Count: {{outputs.health.error_count}}
          
        {{outputs.vars.environment == "production" ? "IMMEDIATE ACTION REQUIRED" : "Please investigate when convenient"}}
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
  uses: http
  timeout: 5s              # Quick ping should respond fast
  with:
    method: GET
    url: "{{vars.api_url}}/ping"

- name: Complex Query
  uses: http
  timeout: 60s             # Complex operations need more time
  with:
    method: GET
    url: "{{vars.api_url}}/complex-report"
```

### 2. Error Handling Strategy

Implement appropriate error handling:

```yaml
# Critical actions - fail fast
- name: Database Connectivity Check
  uses: http
  with:
    method: GET
    url: "{{vars.db_url}}/ping"
  test: res.code == 200

# Non-critical actions - continue on error
- name: Optional Analytics Update
  uses: http
  with:
    method: GET
    url: "{{vars.analytics_url}}/update"
  test: res.code == 200
```

### 3. Secure Configuration

Handle sensitive data properly:

```yaml
# Good: Use environment variables for secrets
- name: Authenticated Request
  uses: http
  with:
    method: GET
    url: "{{vars.api_url}}/secure"
    headers:
      Authorization: "Bearer {{vars.api_token}}"  # From vars

# Good: Use secure SMTP configuration
- name: Send Alert
  uses: smtp
  with:
    addr: "{{vars.smtp_host}}:25"
    from: "probe@example.com"
    to: "ops@example.com"
    session: 1
    message: 1
    length: 500
- name: Bad Example
  uses: http
  with:
    headers:
      Authorization: "Bearer secret-token-123"  # Never do this!
```

### 4. Response Validation

Validate action responses thoroughly:

```yaml
- name: Comprehensive API Test
  uses: http
  with:
    method: GET
    url: "{{vars.api_url}}/users"
  test: |
    res.code == 200 &&
    res.headers["Content-Type"] contains "application/json" &&
    res.body.users != null &&
    len(res.body.users) > 0 &&
    (rt.sec * 1000) < 1000
  outputs:
    user_count: len(res.body.users)
    response_time: (rt.sec * 1000)
```

### 5. Meaningful Outputs

Define useful outputs for other steps:

```yaml
- name: User Creation Test
  uses: http
  with:
    url: "{{vars.api_url}}/users"
    method: POST
    body: '{"name": "Test User", "email": "test@example.com"}'
  test: res.code == 201
  outputs:
    created_user_id: res.body.user.id
    created_user_email: res.body.user.email
    creation_timestamp: res.body.user.created_at
    response_time: (rt.sec * 1000)
```

## Custom Actions (Advanced)

While Probe comes with powerful built-in actions, you can extend it with custom actions for specialized needs.

### Custom Action Interface

Custom actions must implement the Actions interface:

```go
type Actions interface {
    Run(with map[string]any) (map[string]any, error)
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

func (c *CustomAction) Run(with map[string]any) (map[string]any, error) {
    // Custom action logic here
    return map[string]any{
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

1. **[Expressions and Templates](/guide/concepts/expressions-and-templates)** - Learn dynamic configuration and testing
2. **[Data Flow](/guide/concepts/data-flow)** - Understand how data moves between actions
3. **[How-tos](/guide/how-tos/api-testing)** - See practical action usage patterns

Actions are the workhorses of Probe. Master the built-in actions and understand the plugin architecture to build powerful, extensible automation workflows.
