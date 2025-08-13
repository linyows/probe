# Expressions and Templates

Expressions and templates are the dynamic heart of Probe workflows. They enable conditional logic, data transformation, and dynamic configuration. This guide explores the expression system, template syntax, and advanced usage patterns.

## Expression System Overview

Probe uses two types of expressions:

1. **Template Expressions** (`{{expression}}`) - For dynamic value insertion
2. **Test Expressions** (`expression`) - For boolean conditions and validation

Both use the same underlying expression engine based on [expr](https://github.com/antonmedv/expr) with security enhancements and custom functions.

## Template Expressions

Template expressions use `{{}}` syntax to insert dynamic values into strings.

### Basic Template Syntax

```yaml
# Simple variable substitution
- name: Greet User
  echo: "Hello {{env.USERNAME}}!"

# Accessing nested data
- name: API Request
  action: http
  with:
    url: "{{env.API_BASE_URL}}/users/{{outputs.auth.user_id}}"
    headers:
      Authorization: "Bearer {{outputs.auth.access_token}}"

# Complex expressions
- name: Dynamic Configuration
  echo: "Environment: {{env.NODE_ENV || 'development'}}, Users: {{outputs.api.user_count || 0}}"
```

### Template Expression Context

Template expressions have access to several data sources:

#### Environment Variables (`env`)
```yaml
variables:
  api_url: "{{env.API_URL}}"                    # Environment variable
  port: "{{env.PORT || '3000'}}"               # With default value
  debug_mode: "{{env.DEBUG == 'true'}}"        # Boolean conversion
```

#### Step Outputs (`outputs`)
```yaml
steps:
  - name: Get User Info
    id: user-info
    action: http
    with:
      url: "{{env.API_URL}}/user/current"
    outputs:
      user_id: res.json.id
      user_name: res.json.name
      user_email: res.json.email

  - name: Send Welcome Email
    action: smtp
    with:
      to: ["{{outputs.user-info.user_email}}"]
      subject: "Welcome {{outputs.user-info.user_name}}!"
      body: "Your user ID is: {{outputs.user-info.user_id}}"
```

#### Job Outputs (Cross-job references)
```yaml
jobs:
  setup:
    steps:
      - name: Initialize
        outputs:
          session_id: "{{random_str(16)}}"

  main-test:
    needs: [setup]
    steps:
      - name: Use Session
        action: http
        with:
          headers:
            X-Session-ID: "{{outputs.setup.session_id}}"
```

### Advanced Template Patterns

#### Conditional Values
```yaml
# Ternary operator
- name: Environment-specific URL
  echo: "URL: {{env.NODE_ENV == 'production' ? 'https://api.prod.com' : 'https://api.dev.com'}}"

# Null coalescing
- name: Default Configuration
  echo: "Timeout: {{env.TIMEOUT || '30s'}}"
```

#### String Manipulation
```yaml
# String concatenation
- name: Build File Path
  echo: "File: {{env.BASE_PATH}}/{{env.FILE_NAME}}.{{env.FILE_EXT}}"

# String methods (limited support)
- name: Format Output
  echo: "User: {{outputs.user.name.upper()}} ({{outputs.user.email.lower()}})"
```

#### Arithmetic Operations
```yaml
# Mathematical operations
- name: Calculate Metrics
  echo: |
    Performance Metrics:
    Average Response Time: {{(outputs.test1.time + outputs.test2.time + outputs.test3.time) / 3}}ms
    Total Requests: {{outputs.test1.requests + outputs.test2.requests + outputs.test3.requests}}
    Success Rate: {{(outputs.successful.count / outputs.total.count) * 100}}%
```

#### Complex Data Access
```yaml
# Array access
- name: Process User List
  echo: "First user: {{outputs.users.list[0].name}}"

# Object property access
- name: Nested Data Access
  echo: "Database: {{outputs.config.database.host}}:{{outputs.config.database.port}}"
```

## Test Expressions

Test expressions are boolean conditions used in `test` and `if` statements.

### Basic Test Syntax

```yaml
# Simple status check
- name: Health Check
  action: http
  with:
    url: "{{env.API_URL}}/health"
  test: res.status == 200

# Complex conditions
- name: Comprehensive API Test
  action: http
  with:
    url: "{{env.API_URL}}/api/data"
  test: |
    res.status == 200 &&
    res.json.success == true &&
    res.json.data != null &&
    res.time < 1000
```

### HTTP Response Testing

The `res` object provides comprehensive response data:

```yaml
# Status code testing
test: res.status == 200
test: res.status >= 200 && res.status < 300
test: res.status in [200, 201, 202]

# Response time testing
test: res.time < 1000                           # Less than 1 second
test: res.time >= 100 && res.time <= 500      # Between 100-500ms

# Response size testing
test: res.body_size > 0                        # Has content
test: res.body_size < 1048576                  # Less than 1MB

# Header testing
test: res.headers["content-type"] == "application/json"
test: res.headers["x-rate-limit-remaining"] > "10"

# JSON response testing
test: res.json.status == "success"
test: res.json.data.users.length > 0
test: res.json.error == null

# Text response testing
test: res.text.contains("Success")
test: res.text.startsWith("<!DOCTYPE html>")
test: res.text.length > 100
```

### Advanced Test Conditions

#### Regular Expressions
```yaml
# Pattern matching in response text
test: res.text.matches("user-\\d+@example\\.com")

# JSON field pattern validation
test: res.json.user.email.matches("[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}")
```

#### Array and Object Testing
```yaml
# Array testing
test: res.json.users.length == 5
test: res.json.tags.contains("production")
test: res.json.permissions.all(p -> p.active == true)
test: res.json.items.any(item -> item.price > 100)

# Object property testing
test: res.json.user.has("id") && res.json.user.has("email")
test: res.json.config.database.host != null
```

#### Complex Logical Conditions
```yaml
# Multi-condition validation
test: |
  (res.status == 200 && res.json.success == true) ||
  (res.status == 202 && res.json.processing == true)

# Nested condition validation
test: |
  res.status == 200 &&
  res.json.data != null &&
  (
    (res.json.data.type == "user" && res.json.data.user.active == true) ||
    (res.json.data.type == "system" && res.json.data.system.healthy == true)
  )
```

## Built-in Functions

Probe provides several built-in functions for common operations.

### Random Functions

#### `random_int(max)`
Generate random integers:

```yaml
# Generate random user ID
- name: Create Test User
  action: http
  with:
    url: "{{env.API_URL}}/users"
    method: POST
    body: |
      {
        "id": {{random_int(999999)}},
        "name": "TestUser{{random_int(1000)}}",
        "group": {{random_int(10)}}
      }
```

#### `random_str(length)`
Generate random strings:

```yaml
# Generate unique identifiers
- name: Create Session
  outputs:
    session_id: "session_{{random_str(16)}}"
    transaction_id: "txn_{{random_str(12)}}"
    correlation_id: "{{random_str(32)}}"

# Generate test data
- name: Create Test Record
  action: http
  with:
    body: |
      {
        "username": "user_{{random_str(8)}}",
        "email": "test_{{random_str(6)}}@example.com",
        "api_key": "{{random_str(40)}}"
      }
```

### Time Functions

#### `unixtime()`
Get current Unix timestamp:

```yaml
# Add timestamps to requests
- name: Timestamped Request
  action: http
  with:
    url: "{{env.API_URL}}/events"
    method: POST
    body: |
      {
        "event": "test_execution",
        "timestamp": {{unixtime()}},
        "execution_id": "exec_{{unixtime()}}_{{random_str(8)}}"
      }

# Time-based testing
- name: Check Timestamp
  action: http
  with:
    url: "{{env.API_URL}}/status"
  test: res.json.server_time >= {{unixtime() - 300}}  # Within last 5 minutes
```

### Custom Function Usage Patterns

#### Unique Test Data Generation
```yaml
jobs:
  user-lifecycle-test:
    steps:
      - name: Create Unique User
        id: create-user
        action: http
        with:
          url: "{{env.API_URL}}/users"
          method: POST
          body: |
            {
              "username": "testuser_{{unixtime()}}_{{random_str(6)}}",
              "email": "test_{{random_str(8)}}@example.com",
              "password": "{{random_str(16)}}",
              "user_id": {{random_int(1000000)}}
            }
        test: res.status == 201
        outputs:
          user_id: res.json.user.id
          username: res.json.user.username

      - name: Verify User Creation
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.create-user.user_id}}"
        test: |
          res.status == 200 &&
          res.json.user.username == "{{outputs.create-user.username}}"

      - name: Clean Up User
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.create-user.user_id}}"
          method: DELETE
        test: res.status == 204
```

#### Session and Correlation IDs
```yaml
jobs:
  distributed-trace-test:
    steps:
      - name: Initialize Trace
        id: trace
        echo: "Starting distributed trace"
        outputs:
          trace_id: "trace_{{unixtime()}}_{{random_str(16)}}"
          correlation_id: "corr_{{random_str(32)}}"

      - name: Service A Call
        action: http
        with:
          url: "{{env.SERVICE_A_URL}}/process"
          headers:
            X-Trace-ID: "{{outputs.trace.trace_id}}"
            X-Correlation-ID: "{{outputs.trace.correlation_id}}"
        test: res.status == 200

      - name: Service B Call
        action: http
        with:
          url: "{{env.SERVICE_B_URL}}/process"
          headers:
            X-Trace-ID: "{{outputs.trace.trace_id}}"
            X-Correlation-ID: "{{outputs.trace.correlation_id}}"
        test: res.status == 200

      - name: Verify Trace Correlation
        action: http
        with:
          url: "{{env.TRACING_URL}}/traces/{{outputs.trace.trace_id}}"
        test: |
          res.status == 200 &&
          res.json.spans.length >= 2 &&
          res.json.correlation_id == "{{outputs.trace.correlation_id}}"
```

## Conditional Logic Patterns

### Step-level Conditions

```yaml
steps:
  - name: Check Primary Service
    id: primary
    action: http
    with:
      url: "{{env.PRIMARY_URL}}/health"
    test: res.status == 200
    continue_on_error: true
    outputs:
      primary_healthy: res.status == 200

  - name: Check Secondary Service
    if: "!outputs.primary.primary_healthy"
    id: secondary
    action: http
    with:
      url: "{{env.SECONDARY_URL}}/health"
    test: res.status == 200
    outputs:
      secondary_healthy: res.status == 200

  - name: Success Path
    if: outputs.primary.primary_healthy || outputs.secondary.secondary_healthy
    echo: "At least one service is healthy"

  - name: Failure Path
    if: "!outputs.primary.primary_healthy && (!outputs.secondary || !outputs.secondary.secondary_healthy)"
    echo: "All services are down!"
```

### Job-level Conditions

```yaml
jobs:
  health-check:
    steps:
      - name: Basic Health Check
        outputs:
          healthy: res.status == 200

  detailed-analysis:
    if: jobs.health-check.failed
    steps:
      - name: Deep Diagnostic
        action: http
        with:
          url: "{{env.API_URL}}/diagnostics"

  performance-test:
    if: jobs.health-check.success
    steps:
      - name: Load Test
        action: http
        with:
          url: "{{env.API_URL}}/load-test"
```

### Environment-based Conditions

```yaml
steps:
  - name: Development Setup
    if: env.NODE_ENV == "development"
    echo: "Running in development mode"

  - name: Production Validation
    if: env.NODE_ENV == "production"
    action: http
    with:
      url: "{{env.API_URL}}/production-check"
    test: res.status == 200

  - name: Feature Flag Check
    if: env.FEATURE_FLAGS.contains("new-api")
    action: http
    with:
      url: "{{env.API_URL}}/v2/endpoint"
```

## Security Considerations

### Expression Security Features

Probe implements several security measures:

1. **Expression Length Limits**: Prevents resource exhaustion
2. **Dangerous Function Blocking**: Blocks access to system functions
3. **Environment Variable Filtering**: Limits access to sensitive variables
4. **Timeout Protection**: Prevents infinite loops in expressions

### Safe Expression Patterns

```yaml
# Good: Safe environment variable access
- name: Safe Config
  echo: "API URL: {{env.API_URL}}"

# Good: Bounded data access
- name: Safe Data Access
  test: res.json.users.length <= 1000

# Avoid: Unbounded operations
# test: res.json.data.some_huge_array.all(item -> expensive_operation(item))

# Good: Simple conditions
- name: Simple Validation
  test: res.status == 200 && res.json.success == true

# Avoid: Complex nested expressions
# test: deeply.nested.complex.expression.with.many.operations()
```

### Sensitive Data Handling

```yaml
# Good: Use environment variables for secrets
- name: Authenticated Request
  action: http
  with:
    headers:
      Authorization: "Bearer {{env.API_TOKEN}}"

# Good: Avoid logging sensitive data
- name: Login Test
  action: http
  with:
    body: |
      {
        "username": "{{env.TEST_USERNAME}}",
        "password": "{{env.TEST_PASSWORD}}"
      }
  # Don't output sensitive response data
  outputs:
    login_successful: res.status == 200
    # NOT: auth_token: res.json.token (would expose in logs)
```

## Performance Optimization

### Efficient Expression Writing

```yaml
# Good: Simple, direct expressions
test: res.status == 200

# Good: Early termination with &&
test: res.status == 200 && res.json.success == true

# Avoid: Complex computations in expressions
# test: expensive_calculation(res.json.large_dataset) == expected_value

# Good: Pre-compute complex values
outputs:
  user_count: res.json.users.length
  active_users: res.json.users.filter(u -> u.active == true).length
```

### Template Optimization

```yaml
# Good: Simple template substitution
echo: "User {{outputs.user.name}} logged in"

# Good: Minimal string operations
url: "{{env.BASE_URL}}/users/{{outputs.user.id}}"

# Avoid: Complex template expressions
# echo: "{{complex_calculation(outputs.data) + another_operation(env.CONFIG)}}"
```

## Debugging Expressions

### Common Issues and Solutions

#### Template Expression Errors
```yaml
# Error: Missing quotes in JSON
body: |
  {
    "name": {{outputs.user.name}}      # ERROR: Missing quotes
  }

# Solution: Proper JSON quoting
body: |
  {
    "name": "{{outputs.user.name}}"    # CORRECT: Quoted string
  }
```

#### Test Expression Debugging
```yaml
# Debug with verbose mode
probe -v workflow.yml

# Add debug outputs
- name: Debug Values
  echo: |
    Debug Information:
    Status: {{res.status}}
    Response Time: {{res.time}}
    JSON Success: {{res.json.success}}
    Headers: {{res.headers}}
```

#### Null Value Handling
```yaml
# Good: Handle potential null values
test: res.json.user != null && res.json.user.active == true

# Good: Use default values
echo: "User count: {{outputs.api.user_count || 0}}"

# Good: Check existence before access
test: res.json.has("data") && res.json.data.has("users")
```

## Best Practices

### 1. Keep Expressions Simple
```yaml
# Good: Simple, readable expressions
test: res.status == 200 && res.time < 1000

# Avoid: Overly complex expressions
# test: (res.status >= 200 && res.status < 300) && (res.time < (env.MAX_TIME || 1000)) && (res.json.data.items.filter(i -> i.active && i.validated).length > 0)
```

### 2. Use Meaningful Variable Names
```yaml
# Good: Descriptive output names
outputs:
  user_id: res.json.user.id
  auth_token: res.json.access_token
  expires_at: res.json.expires_in

# Avoid: Generic names
outputs:
  data1: res.json.user.id
  value: res.json.access_token
```

### 3. Handle Edge Cases
```yaml
# Good: Defensive programming
test: |
  res.status == 200 &&
  res.json != null &&
  res.json.users != null &&
  res.json.users.length > 0

# Good: Provide defaults
echo: "Processing {{outputs.api.item_count || 0}} items"
```

### 4. Document Complex Expressions
```yaml
- name: Complex Business Logic Validation
  action: http
  with:
    url: "{{env.API_URL}}/business-data"
  # Test validates that:
  # 1. Response is successful (200)
  # 2. Processing time is acceptable (< 2s)
  # 3. Data integrity is maintained (required fields present)
  # 4. Business rules are satisfied (active users > 0, revenue > threshold)
  test: |
    res.status == 200 &&
    res.time < 2000 &&
    res.json.users != null &&
    res.json.revenue != null &&
    res.json.users.filter(u -> u.active == true).length > 0 &&
    res.json.revenue > 1000
```

## What's Next?

Now that you understand expressions and templates, explore:

1. **[Data Flow](../data-flow/)** - Learn how data moves through workflows
2. **[Testing and Assertions](../testing-and-assertions/)** - Master validation techniques
3. **[How-tos](../../how-tos/)** - See practical expression usage patterns

Expressions and templates are the dynamic engine of Probe. Master these concepts to build flexible, data-driven automation workflows.