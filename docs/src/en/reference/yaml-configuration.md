---
title: YAML Configuration Reference
description: Complete YAML configuration syntax and options reference
weight: 20
---

# YAML Configuration Reference

This page provides complete documentation for Probe's YAML configuration syntax, including all available options, data types, and validation rules.

## Workflow Structure

The basic structure of a Probe workflow:

```yaml
name: string                    # Required: Workflow name
description: string             # Optional: Workflow description
vars:                          # Optional: Variables (including environment variables)
  KEY: "{{ENV_VAR ?? 'default'}}"
defaults:                      # Optional: Default settings
  http:
    timeout: duration
    headers:
      KEY: value
jobs:                          # Required: Job definitions
  job-id:
    # Job configuration
```

## Top-Level Properties

### `name`

**Type:** String (required)  
**Description:** Human-readable name for the workflow  
**Constraints:** Must be non-empty

```yaml
name: "API Health Check"
name: "Production Monitoring Workflow"
```

### `description`

**Type:** String (optional)  
**Description:** Detailed description of the workflow's purpose  
**Supports:** Multi-line strings using YAML literal block syntax

```yaml
description: "Monitors the health of production APIs"

# Multi-line description
description: |
  This workflow performs comprehensive health checks including:
  - API endpoint validation
  - Database connectivity testing
  - Performance monitoring
```

### `env`

**Type:** Object (optional)  
**Description:** Environment variables available to all jobs and steps  
**Key Format:** Valid environment variable names (alphanumeric + underscore)  
**Value Types:** String, number, boolean

```yaml
env:
  API_BASE_URL: "https://api.example.com"
  TIMEOUT_SECONDS: 30
  DEBUG_MODE: true
  USER_AGENT: "Probe Monitor v1.0"
```

**Environment Variable Resolution:**
```yaml
vars:
  # Static values
  api_url: "https://api.example.com"
  
  # Reference external environment variables
  db_password: "{{DATABASE_PASSWORD}}"
  
  # Default values
  timeout: "{{REQUEST_TIMEOUT ?? '30s'}}"
  
  # Computed values
  build_info: "Build {{BUILD_NUMBER ?? 'unknown'}} at {{unixtime()}}"
```

### `defaults`

**Type:** Object (optional)  
**Description:** Default settings that apply to all actions unless overridden

#### `defaults.http`

HTTP-specific default settings:

```yaml
defaults:
  http:
    timeout: "30s"                    # Default timeout for HTTP actions
    follow_redirects: true            # Follow HTTP redirects
    verify_ssl: true                  # Verify SSL certificates
    max_redirects: 5                  # Maximum redirect count
    headers:                          # Default headers for all HTTP requests
      User-Agent: "Probe Monitor"
      Accept: "application/json"
      Authorization: "Bearer {{vars.api_token}}"
```

**Supported HTTP Defaults:**

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `timeout` | Duration | `30s` | Request timeout |
| `follow_redirects` | Boolean | `true` | Follow HTTP redirects |
| `verify_ssl` | Boolean | `true` | Verify SSL certificates |
| `max_redirects` | Integer | `10` | Maximum redirects to follow |
| `headers` | Object | `{}` | Default headers |

## Jobs Configuration

### Job Structure

```yaml
jobs:
  job-id:                           # Unique job identifier
    name: string                    # Optional: Human-readable job name
    needs: [job-id, ...]           # Optional: Job dependencies
    if: expression                  # Optional: Conditional execution
    continue_on_error: boolean      # Optional: Continue workflow on job failure
    timeout: duration               # Optional: Job timeout
    steps:                         # Required: Array of steps
      - # Step configuration
```

### Job Properties

#### `name`

**Type:** String (optional)  
**Description:** Human-readable name for the job  
**Default:** Uses job ID if not specified

```yaml
jobs:
  api-test:
    name: "API Health Check"
```

#### `needs`

**Type:** Array of strings (optional)  
**Description:** List of job IDs that must complete before this job runs  
**Constraints:** Referenced jobs must exist

```yaml
jobs:
  setup:
    # Setup job
  
  test:
    needs: [setup]              # Single dependency
  
  cleanup:
    needs: [setup, test]        # Multiple dependencies
```

#### `if`

**Type:** Expression string (optional)  
**Description:** Conditional expression determining if job should execute  
**Context:** Access to environment variables and other job results

```yaml
vars:
  environment: "{{ENVIRONMENT}}"

jobs:
  production-only:
    if: vars.environment == "production"
  
  cleanup:
    if: jobs.test.failed
  
  notification:
    if: jobs.test.success || jobs.fallback.success
```

#### `continue_on_error`

**Type:** Boolean (optional)  
**Default:** `false`  
**Description:** Whether workflow should continue if this job fails

```yaml
jobs:
  critical-test:
    continue_on_error: false    # Stop workflow on failure (default)
  
  optional-check:
    continue_on_error: true     # Continue workflow even if this fails
```

#### `timeout`

**Type:** Duration (optional)  
**Description:** Maximum time this job can run  
**Format:** Duration string (e.g., `30s`, `5m`, `1h`)

```yaml
jobs:
  quick-check:
    timeout: "30s"
  
  comprehensive-test:
    timeout: "10m"
```

## Steps Configuration

### Step Structure

```yaml
steps:
  - name: string                    # Required: Step name
    id: string                      # Optional: Step identifier for referencing
    action: string                  # Optional: Action to execute
    with:                          # Optional: Action parameters
      parameter: value
    test: expression               # Optional: Test condition
    outputs:                       # Optional: Output definitions
      key: expression
    echo: string                   # Optional: Message to display
    if: expression                 # Optional: Conditional execution
    continue_on_error: boolean     # Optional: Continue on step failure
    timeout: duration              # Optional: Step timeout
```

### Step Properties

#### `name`

**Type:** String (required)  
**Description:** Human-readable name for the step

```yaml
steps:
  - name: "Check API Health"
  - name: "Validate User Authentication"
```

#### `id`

**Type:** String (optional)  
**Description:** Unique identifier for referencing step outputs  
**Constraints:** Must be unique within the job, alphanumeric + hyphens/underscores

```yaml
steps:
  - name: "Get Auth Token"
    id: auth
    # ... step configuration
  
  - name: "Use Auth Token"
    action: http
    with:
      headers:
        Authorization: "Bearer {{outputs.auth.token}}"
```

#### `action`

**Type:** String (optional)  
**Description:** Action plugin to execute  
**Built-in Actions:** `http`, `hello`, `smtp`

```yaml
steps:
  - name: "HTTP Request"
    action: http
  
  - name: "Send Email"
    action: smtp
  
  - name: "Test Plugin"
    action: hello
```

#### `with`

**Type:** Object (optional)  
**Description:** Parameters passed to the action  
**Structure:** Varies by action type

**HTTP Action Parameters:**
```yaml
steps:
  - name: "API Request"
    action: http
    with:
      url: "https://api.example.com/users"     # Required
      method: "GET"                            # Optional, default: GET
      headers:                                 # Optional
        Authorization: "Bearer {{env.TOKEN}}"
        Content-Type: "application/json"
      body: |                                  # Optional
        {
          "name": "Test User"
        }
      timeout: "30s"                          # Optional
      follow_redirects: true                   # Optional
      verify_ssl: true                        # Optional
      max_redirects: 5                        # Optional
```

**SMTP Action Parameters:**
```yaml
vars:
  smtp_user: "{{SMTP_USER}}"
  smtp_pass: "{{SMTP_PASS}}"
  service_name: "{{SERVICE_NAME}}"

steps:
  - name: "Send Notification"
    uses: smtp
    with:
      host: "smtp.gmail.com"                  # Required
      port: 587                               # Optional, default: 587
      username: "{{vars.smtp_user}}"          # Required
      password: "{{vars.smtp_pass}}"          # Required
      from: "alerts@example.com"              # Required
      to: ["admin@example.com"]               # Required
      cc: ["team@example.com"]                # Optional
      bcc: ["audit@example.com"]              # Optional
      subject: "Alert: {{vars.service_name}}" # Required
      body: "Service alert message"           # Required
      html: false                             # Optional, default: false
      tls: true                              # Optional, default: true
```

**Hello Action Parameters:**
```yaml
steps:
  - name: "Test Hello"
    action: hello
    with:
      message: "Test message"                 # Optional
      delay: "1s"                            # Optional
```

#### `test`

**Type:** Expression string (optional)  
**Description:** Boolean expression that determines step success/failure  
**Context:** Access to action response via `res` object

```yaml
steps:
  - name: "API Health Check"
    action: http
    with:
      url: "{{env.API_URL}}/health"
    test: res.status == 200
  
  - name: "Complex Validation"
    action: http
    with:
      url: "{{env.API_URL}}/data"
    test: |
      res.status == 200 &&
      res.json.success == true &&
      res.json.data.length > 0 &&
      res.time < 1000
```

**Response Object Properties:**

For HTTP actions, the `res` object contains:

| Property | Type | Description |
|----------|------|-------------|
| `status` | Integer | HTTP status code |
| `time` | Integer | Response time in milliseconds |
| `body_size` | Integer | Response body size in bytes |
| `headers` | Object | Response headers |
| `json` | Object | Parsed JSON response (if applicable) |
| `text` | String | Response body as text |

#### `outputs`

**Type:** Object (optional)  
**Description:** Named values extracted from the step for use in later steps  
**Key Format:** Valid identifier names  
**Value Type:** Expression strings

```yaml
steps:
  - name: "Get User Data"
    id: user-data
    action: http
    with:
      url: "{{env.API_URL}}/users/1"
    test: res.status == 200
    outputs:
      user_id: res.json.id
      user_name: res.json.name
      user_email: res.json.email
      response_time: res.time
      is_active: res.json.active == true
      full_name: "{{res.json.first_name}} {{res.json.last_name}}"
```

#### `echo`

**Type:** String (optional)  
**Description:** Message to display during step execution  
**Supports:** Template expressions and multi-line strings

```yaml
steps:
  - name: "Display Status"
    echo: "Current time: {{unixtime()}}"
  
  - name: "Multi-line Report"
    echo: |
      Test Results:
      API Status: {{outputs.api-test.success ? "✅ Healthy" : "❌ Failed"}}
      Response Time: {{outputs.api-test.response_time}}ms
      User Count: {{outputs.api-test.user_count}}
```

#### `if`

**Type:** Expression string (optional)  
**Description:** Conditional expression determining if step should execute

```yaml
steps:
  - name: "Production Only Step"
    if: env.ENVIRONMENT == "production"
    action: http
    with:
      url: "{{env.PROD_API_URL}}/check"
  
  - name: "Retry on Failure"
    if: steps.previous-step.failed
    action: http
    with:
      url: "{{env.FALLBACK_URL}}/retry"
```

#### `continue_on_error`

**Type:** Boolean (optional)  
**Default:** `false`  
**Description:** Whether job should continue if this step fails

```yaml
steps:
  - name: "Critical Step"
    action: http
    with:
      url: "{{env.CRITICAL_URL}}/check"
    continue_on_error: false      # Stop job on failure (default)
  
  - name: "Optional Step"
    action: http
    with:
      url: "{{env.OPTIONAL_URL}}/info"
    continue_on_error: true       # Continue job even if this fails
```

#### `timeout`

**Type:** Duration (optional)  
**Description:** Maximum time this step can run

```yaml
steps:
  - name: "Quick Check"
    timeout: "5s"
    action: http
    with:
      url: "{{env.API_URL}}/ping"
  
  - name: "Long Running Process"
    timeout: "5m"
    action: http
    with:
      url: "{{env.API_URL}}/long-process"
```

## Data Types

### Duration

Duration strings specify time periods:

**Format:** `<number><unit>`  
**Units:** `ns`, `us`, `ms`, `s`, `m`, `h`

```yaml
# Examples
timeout: "30s"          # 30 seconds
timeout: "5m"           # 5 minutes
timeout: "1h30m"        # 1 hour 30 minutes
timeout: "500ms"        # 500 milliseconds
```

### Expression Strings

Expression strings use Go template syntax with custom functions:

**Template Expressions:** `{{expression}}`  
**Test Expressions:** Plain boolean expressions

```yaml
# Template expressions (for values)
url: "{{env.BASE_URL}}/api/{{env.VERSION}}"
message: "Hello {{outputs.user.name}}"

# Test expressions (for conditions)
test: res.status == 200 && res.time < 1000
if: env.ENVIRONMENT == "production"
```

### Environment Variable References

Reference environment variables in expressions:

```yaml
env:
  API_URL: "{{env.EXTERNAL_API_URL}}"           # Reference external env var
  TIMEOUT: "{{env.REQUEST_TIMEOUT || '30s'}}"   # With default value
  DEBUG: "{{env.DEBUG_MODE == 'true'}}"         # Boolean conversion
```

## Validation Rules

### Workflow Validation

- `name` is required and non-empty
- `jobs` is required and contains at least one job
- Job IDs must be unique
- Job IDs in `needs` must reference existing jobs
- No circular dependencies in job `needs`

### Job Validation

- Each job must have a `steps` array
- Step names are required and should be descriptive
- Step IDs must be unique within the job
- Action names must be valid (built-in or available plugins)

### Expression Validation

- Template expressions must use valid Go template syntax
- Test expressions must evaluate to boolean values
- Referenced variables and outputs must exist
- Function calls must use valid built-in functions

## Common Patterns

### Environment-Specific Configuration

```yaml
env:
  ENVIRONMENT: "{{env.NODE_ENV || 'development'}}"
  API_URL: |
    {{env.NODE_ENV == "production" ? 
      "https://api.prod.com" : 
      "https://api.dev.com"}}
  TIMEOUT: |
    {{env.NODE_ENV == "production" ? "10s" : "30s"}}
```

### Conditional Job Execution

```yaml
jobs:
  setup:
    # Always runs
  
  development-tests:
    if: env.ENVIRONMENT == "development"
    needs: [setup]
  
  production-checks:
    if: env.ENVIRONMENT == "production"  
    needs: [setup]
  
  cleanup:
    needs: [development-tests, production-checks]
    if: |
      jobs.development-tests.executed || 
      jobs.production-checks.executed
```

### Error Handling and Recovery

```yaml
jobs:
  primary-test:
    continue_on_error: true
    steps:
      - name: "Primary Service Test"
        action: http
        with:
          url: "{{env.PRIMARY_URL}}/test"
        continue_on_error: true
  
  fallback-test:
    if: jobs.primary-test.failed
    steps:
      - name: "Fallback Service Test"
        action: http
        with:
          url: "{{env.FALLBACK_URL}}/test"
```

### Data Flow Between Steps

```yaml
jobs:
  data-processing:
    steps:
      - name: "Fetch Data"
        id: fetch
        action: http
        with:
          url: "{{env.API_URL}}/data"
        outputs:
          data_count: res.json.items.length
          first_item_id: res.json.items[0].id
      
      - name: "Process Data"
        action: http
        with:
          url: "{{env.API_URL}}/process/{{outputs.fetch.first_item_id}}"
        test: res.status == 200
      
      - name: "Summary"
        echo: "Processed {{outputs.fetch.data_count}} items"
```

## Best Practices

### YAML Style

```yaml
# Good: Consistent indentation (2 spaces)
jobs:
  test:
    name: "API Test"
    steps:
      - name: "Health Check"
        action: http

# Good: Quoted strings with special characters
env:
  MESSAGE: "Hello, World!"
  PATTERN: "user-\\d+"

# Good: Multi-line strings for readability
description: |
  This workflow performs comprehensive testing including:
  - API endpoint validation
  - Database connectivity
  - Performance benchmarks
```

### Naming Conventions

```yaml
# Good: Descriptive names
name: "Production API Health Check"

jobs:
  user-authentication-test:
    name: "User Authentication Test"
    
  database-connectivity-check:
    name: "Database Connectivity Check"

steps:
  - name: "Verify SSL Certificate Validity"
  - name: "Test User Login Endpoint"
  - name: "Validate Database Connection Pool"
```

### Configuration Organization

```yaml
# Good: Logical grouping
env:
  # API Configuration
  API_BASE_URL: "https://api.example.com"
  API_VERSION: "v1"
  API_TIMEOUT: "30s"
  
  # Database Configuration  
  DB_HOST: "localhost"
  DB_PORT: 5432
  
  # Feature Flags
  ENABLE_CACHING: true
  ENABLE_METRICS: false

defaults:
  http:
    timeout: "{{env.API_TIMEOUT}}"
    headers:
      User-Agent: "Probe Monitor"
      Accept: "application/json"
```

## See Also

- **[CLI Reference](../cli-reference/)** - Command-line options and usage
- **[Actions Reference](../actions-reference/)** - Built-in actions and parameters
- **[Built-in Functions](../built-in-functions/)** - Expression functions
- **[Concepts: Workflows](../../concepts/workflows/)** - Workflow design patterns
- **[Concepts: Expressions and Templates](../../concepts/expressions-and-templates/)** - Expression language guide