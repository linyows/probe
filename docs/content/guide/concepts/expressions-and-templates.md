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
  echo: "Hello {{vars.USERNAME}}!"

# Accessing nested data
- name: API Request
  uses: http
  with:
    method: GET
    url: "{{vars.API_BASE_URL}}/users/{{outputs.auth.user_id}}"
    headers:
      Authorization: "Bearer {{outputs.auth.access_token}}"

# Complex expressions
- name: Dynamic Configuration
  echo: "Environment: {{vars.NODE_ENV || 'development'}}, Users: {{outputs.api.user_count || 0}}"
```

### Template Expression Context

Template expressions have access to several data sources:

#### Environment Variables (`env`)
```yaml
variables:
  api_url: "{{vars.API_URL}}"                    # Environment variable
  port: "{{vars.PORT || '3000'}}"               # With default value
  debug_mode: "{{vars.DEBUG == 'true'}}"        # Boolean conversion
```

#### Step Outputs (`outputs`)
```yaml
steps:
  - name: Get User Info
    id: user-info
    uses: http
    with:
      method: GET
      url: "{{vars.API_URL}}/user/current"
    outputs:
      user_id: res.body.id
      user_name: res.body.name
      user_email: res.body.email

  - name: Send Welcome Email
    uses: smtp
    with:
      addr: "{{vars.smtp_addr}}"
      from: "probe@example.com"
      to: "{{outputs['user-info'].user_email}}"
      subject: "Welcome {{outputs['user-info'].user_name}}!"
      session: 1
      message: 1
      length: 500
    echo: "Your user ID is: {{outputs['user-info'].user_id}}"
```

#### Job Outputs (Cross-job references)
```yaml
jobs:
- id: setup
  name: setup
  steps:
    - name: Initialize
      id: setup
      outputs:
        session_id: "{{random_str(16)}}"

- id: main-test
  name: main-test
  needs: [setup]
  steps:
    - name: Use Session
      uses: http
      with:
        headers:
          X-Session-ID: "{{outputs.setup.session_id}}"
```

### Advanced Template Patterns

#### Conditional Values
```yaml
# Ternary operator
- name: Environment-specific URL
  echo: "URL: {{vars.NODE_ENV == 'production' ? 'https://api.prod.com' : 'https://api.dev.com'}}"

# Null coalescing
- name: Default Configuration
  echo: "Timeout: {{vars.TIMEOUT || '30s'}}"
```

#### String Manipulation
```yaml
# String concatenation
- name: Build File Path
  echo: "File: {{vars.BASE_PATH}}/{{vars.FILE_NAME}}.{{vars.FILE_EXT}}"

# String methods (limited support)
- name: Format Output
  echo: "User: {{upper(outputs.user.name)}} ({{lower(outputs.user.email)}})"
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

Test expressions are boolean conditions used in `test` and `skipif`.

### Basic Test Syntax

```yaml
# Simple status check
- name: Health Check
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/health"
  test: res.code == 200

# Complex conditions
- name: Comprehensive API Test
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/api/data"
  test: |
    res.code == 200 &&
    res.body.success == true &&
    res.body.data != null &&
    (rt.sec * 1000) < 1000
```

### HTTP Response Testing

The `res` object provides comprehensive response data:

```yaml
# Status code testing
test: res.code == 200
test: res.code >= 200 && res.code < 300
test: res.status in [200, 201, 202]

# Response time testing
test: (rt.sec * 1000) < 1000                           # Less than 1 second
test: (rt.sec * 1000) >= 100 && (rt.sec * 1000) <= 500      # Between 100-500ms

# Response size testing
test: res.body_size > 0                        # Has content
test: res.body_size < 1048576                  # Less than 1MB

# Header testing
test: res.headers["Content-Type"] == "application/json"
test: res.headers["X-Rate-Limit-Remaining"] > "10"

# JSON response testing
test: res.body.status == "success"
test: len(res.body.data.users) > 0
test: res.body.error == null

# Text response testing
test: res.body contains "Success"
test: res.body startsWith "<!DOCTYPE html>"
test: len(res.body) > 100
```

### Advanced Test Conditions

#### Regular Expressions
```yaml
# Pattern matching in response text
test: res.body matches "user-\\d+@example\\.com"

# JSON field pattern validation
test: res.body.user.email matches "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"
```

#### Array and Object Testing
```yaml
# Array testing
test: len(res.body.users) == 5
test: res.body.tags contains "production"
test: all(res.body.permissions, #.active == true)
test: any(res.body.items, #.price > 100)

# Object property testing
test: "id" in res.body.user && "email" in res.body.user
test: res.body.config.database.host != null
```

#### Complex Logical Conditions
```yaml
# Multi-condition validation
test: |
  (res.code == 200 && res.body.success == true) ||
  (res.code == 202 && res.body.processing == true)

# Nested condition validation
test: |
  res.code == 200 &&
  res.body.data != null &&
  (
    (res.body.data.type == "user" && res.body.data.user.active == true) ||
    (res.body.data.type == "system" && res.body.data.system.healthy == true)
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
  uses: http
  with:
    url: "{{vars.API_URL}}/users"
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
  uses: http
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
  uses: http
  with:
    url: "{{vars.API_URL}}/events"
    method: POST
    body: |
      {
        "event": "test_execution",
        "timestamp": {{unixtime()}},
        "execution_id": "exec_{{unixtime()}}_{{random_str(8)}}"
      }

# Time-based testing
- name: Check Timestamp
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/status"
  test: res.body.server_time >= {{unixtime() - 300}}  # Within last 5 minutes
```

### Custom Function Usage Patterns

#### Unique Test Data Generation
```yaml
jobs:
- id: user-lifecycle-test
  name: user-lifecycle-test
  steps:
    - name: Create Unique User
      id: create-user
      uses: http
      with:
        url: "{{vars.API_URL}}/users"
        method: POST
        body: |
          {
            "username": "testuser_{{unixtime()}}_{{random_str(6)}}",
            "email": "test_{{random_str(8)}}@example.com",
            "password": "{{random_str(16)}}",
            "user_id": {{random_int(1000000)}}
          }
      test: res.code == 201
      outputs:
        user_id: res.body.user.id
        username: res.body.user.username

    - name: Verify User Creation
      uses: http
      with:
        method: GET
        url: "{{vars.API_URL}}/users/{{outputs['create-user'].user_id}}"
      test: |
        res.code == 200 &&
        res.body.user.username == "{{outputs['create-user'].username}}"

    - name: Clean Up User
      uses: http
      with:
        url: "{{vars.API_URL}}/users/{{outputs['create-user'].user_id}}"
        method: DELETE
      test: res.code == 204
```

#### Session and Correlation IDs
```yaml
jobs:
- id: distributed-trace-test
  name: distributed-trace-test
  steps:
    - name: Initialize Trace
      uses: hello
      id: trace
      echo: "Starting distributed trace"
      outputs:
        trace_id: "trace_{{unixtime()}}_{{random_str(16)}}"
        correlation_id: "corr_{{random_str(32)}}"

    - name: Service A Call
      uses: http
      with:
        method: GET
        url: "{{vars.SERVICE_A_URL}}/process"
        headers:
          X-Trace-ID: "{{outputs.trace.trace_id}}"
          X-Correlation-ID: "{{outputs.trace.correlation_id}}"
      test: res.code == 200

    - name: Service B Call
      uses: http
      with:
        method: GET
        url: "{{vars.SERVICE_B_URL}}/process"
        headers:
          X-Trace-ID: "{{outputs.trace.trace_id}}"
          X-Correlation-ID: "{{outputs.trace.correlation_id}}"
      test: res.code == 200

    - name: Verify Trace Correlation
      uses: http
      with:
        method: GET
        url: "{{vars.TRACING_URL}}/traces/{{outputs.trace.trace_id}}"
      test: |
        res.code == 200 &&
        len(res.body.spans) >= 2 &&
        res.body.correlation_id == "{{outputs.trace.correlation_id}}"
```

## Conditional Logic Patterns

### Step-level Conditions

A step is skipped when `skipif` is true. Publish what the decision depends on as an output of an earlier step.

```yaml
steps:
  - name: Check Primary Service
    id: primary
    uses: http
    with:
      method: GET
      url: "{{vars.primary_url}}/health"
    outputs:
      primary_healthy: res.code == 200

  - name: Check Secondary Service
    id: secondary
    uses: http
    skipif: outputs.primary.primary_healthy
    with:
      method: GET
      url: "{{vars.secondary_url}}/health"
    outputs:
      secondary_healthy: res.code == 200

  - name: Success Path
    uses: hello
    skipif: "!(outputs.primary_healthy || (outputs.secondary_healthy ?? false))"
    echo: "At least one service is healthy"

  - name: Failure Path
    uses: hello
    skipif: outputs.primary_healthy || (outputs.secondary_healthy ?? false)
    echo: "All services are down!"
```

### Job-level Conditions

A job's `skipif` reads `vars` and the outputs of the jobs it depends on.

```yaml
jobs:
- id: health-check
  name: Health Check
  steps:
    - name: Basic Health Check
      id: health
      uses: http
      with:
        method: GET
        url: "{{vars.api_url}}/health"
      outputs:
        healthy: res.code == 200

- name: Detailed Analysis
  needs: [health-check]
  skipif: outputs.health.healthy
  steps:
    - name: Deep Diagnostic
      uses: http
      with:
        method: GET
        url: "{{vars.api_url}}/diagnostics"
      test: res.code == 200

- name: Performance Test
  needs: [health-check]
  skipif: "!outputs.health.healthy"
  steps:
    - name: Load Test
      uses: http
      with:
        method: GET
        url: "{{vars.api_url}}/load-test"
      test: res.code == 200
```


### Environment-based Conditions

```yaml
steps:
  - name: Development Setup
    uses: hello
    skipif: vars.node_env != "development"
    echo: "Running in development mode"

  - name: Production Validation
    uses: http
    skipif: vars.node_env != "production"
    with:
      method: GET
      url: "{{vars.api_url}}/production-check"
    test: res.code == 200

  - name: Feature Flag Check
    uses: http
    skipif: "!(vars.feature_flags contains \"new-api\")"
    with:
      method: GET
      url: "{{vars.api_url}}/v2/endpoint"
    test: res.code == 200
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
  echo: "API URL: {{vars.API_URL}}"

# Good: Bounded data access
- name: Safe Data Access
  test: len(res.body.users) <= 1000

# Avoid: Unbounded operations
# test: all(res.body.data.some_huge_array, expensive_operation(#))

# Good: Simple conditions
- name: Simple Validation
  test: res.code == 200 && res.body.success == true

# Avoid: Complex nested expressions
# test: deeply.nested.complex.expression.with.many.operations()
```

### Sensitive Data Handling

```yaml
# Good: Use environment variables for secrets
- name: Authenticated Request
  uses: http
  with:
    headers:
      Authorization: "Bearer {{vars.API_TOKEN}}"

# Good: Avoid logging sensitive data
- name: Login Test
  uses: http
  with:
    body: |
      {
        "username": "{{vars.TEST_USERNAME}}",
        "password": "{{vars.TEST_PASSWORD}}"
      }
  # Don't output sensitive response data
  outputs:
    login_successful: res.code == 200
    # NOT: auth_token: res.body.token (would expose in logs)
```

## Performance Optimization

### Efficient Expression Writing

```yaml
# Good: Simple, direct expressions
test: res.code == 200

# Good: Early termination with &&
test: res.code == 200 && res.body.success == true

# Avoid: Complex computations in expressions
# test: expensive_calculation(res.body.large_dataset) == expected_value

# Good: Pre-compute complex values
outputs:
  user_count: len(res.body.users)
  active_users: len(filter(res.body.users, #.active == true))
```

### Template Optimization

```yaml
# Good: Simple template substitution
echo: "User {{outputs.user.name}} logged in"

# Good: Minimal string operations
url: "{{vars.BASE_URL}}/users/{{outputs.user.id}}"

# Avoid: Complex template expressions
# echo: "{{complex_calculation(outputs.data) + another_operation(vars.CONFIG)}}"
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
    Response Time: {{rt.duration}}
    JSON Success: {{res.body.success}}
    Headers: {{res.headers}}
```

#### Null Value Handling
```yaml
# Good: Handle potential null values
test: res.body.user != null && res.body.user.active == true

# Good: Use default values
echo: "User count: {{outputs.api.user_count || 0}}"

# Good: Check existence before access
test: "data" in res.body && "users" in res.body.data
```

## Best Practices

### 1. Keep Expressions Simple
```yaml
# Good: Simple, readable expressions
test: res.code == 200 && (rt.sec * 1000) < 1000

# Avoid: Overly complex expressions
# test: (res.code >= 200 && res.code < 300) && ((rt.sec * 1000) < (vars.MAX_TIME || 1000)) && (len(filter(res.body.data.items, #.active && #.validated)) > 0)
```

### 2. Use Meaningful Variable Names
```yaml
# Good: Descriptive output names
outputs:
  user_id: res.body.user.id
  auth_token: res.body.access_token
  expires_at: res.body.expires_in

# Avoid: Generic names
outputs:
  data1: res.body.user.id
  value: res.body.access_token
```

### 3. Handle Edge Cases
```yaml
# Good: Defensive programming
test: |
  res.code == 200 &&
  res.body != null &&
  res.body.users != null &&
  len(res.body.users) > 0

# Good: Provide defaults
echo: "Processing {{outputs.api.item_count || 0}} items"
```

### 4. Document Complex Expressions
```yaml
- name: Complex Business Logic Validation
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/business-data"
  # Test validates that:
  # 1. Response is successful (200)
  # 2. Processing time is acceptable (< 2s)
  # 3. Data integrity is maintained (required fields present)
  # 4. Business rules are satisfied (active users > 0, revenue > threshold)
  test: |
    res.code == 200 &&
    (rt.sec * 1000) < 2000 &&
    res.body.users != null &&
    res.body.revenue != null &&
    len(filter(res.body.users, #.active == true)) > 0 &&
    res.body.revenue > 1000
```

## What's Next?

Now that you understand expressions and templates, explore:

1. **[Data Flow](/guide/concepts/data-flow)** - Learn how data moves through workflows
2. **[Testing and Assertions](/guide/concepts/testing-and-assertions)** - Master validation techniques
3. **[How-tos](/guide/how-tos/api-testing)** - See practical expression usage patterns

Expressions and templates are the dynamic engine of Probe. Master these concepts to build flexible, data-driven automation workflows.
