# Testing and Assertions

Testing and assertions are the quality gates of Probe workflows. They validate that your actions produce expected results and ensure system reliability. This guide explores test expressions, assertion patterns, and strategies for building robust validation into your workflows.

## Testing Fundamentals

Every step in Probe can include a `test` condition that validates the action's result. Tests are boolean expressions that determine whether a step succeeded or failed.

### Basic Test Structure

```yaml
- name: API Health Check
  action: http
  with:
    url: "{{env.API_URL}}/health"
  test: res.status == 200
```

The test expression evaluates the response (`res`) and returns true for success or false for failure.

### Test Expression Context

Test expressions have access to comprehensive response data:

```yaml
# HTTP Response Testing Context
test: |
  res.status == 200 &&           # HTTP status code
  res.time < 1000 &&             # Response time in milliseconds
  res.body_size > 0 &&           # Response body size in bytes
  res.headers["content-type"] == "application/json" &&  # Response headers
  res.json.status == "healthy" && # Parsed JSON response
  res.text.contains("success")   # Response body as text
```

## HTTP Response Testing

### Status Code Validation

```yaml
# Exact status code
test: res.status == 200

# Status code ranges
test: res.status >= 200 && res.status < 300

# Multiple acceptable codes
test: res.status in [200, 201, 202]

# Client vs server errors
test: res.status < 400  # Success or redirect
test: res.status >= 400 && res.status < 500  # Client error
test: res.status >= 500  # Server error
```

### Response Time Testing

```yaml
# Performance validation
test: res.time < 1000                    # Must respond within 1 second
test: res.time >= 100 && res.time <= 500  # Response time range
test: res.time < {{env.MAX_RESPONSE_TIME || 2000}}  # Configurable threshold

# Performance categories
test: |
  res.status == 200 && (
    res.time < 200 ? "excellent" :
    res.time < 500 ? "good" :
    res.time < 1000 ? "acceptable" : "poor"
  ) != "poor"
```

### Response Size Validation

```yaml
# Content presence
test: res.body_size > 0                    # Has content
test: res.body_size > 100                  # Minimum content size
test: res.body_size < 1048576             # Maximum 1MB response

# Size-based validation
test: |
  res.status == 200 &&
  res.body_size > 50 &&                   # Not empty error message
  res.body_size < 100000                  # Not unexpectedly large
```

### Header Validation

```yaml
# Content type checking
test: res.headers["content-type"] == "application/json"
test: res.headers["content-type"].startsWith("text/")
test: res.headers["content-type"].contains("charset=utf-8")

# Security headers
test: |
  res.headers.has("x-frame-options") &&
  res.headers.has("x-content-type-options") &&
  res.headers["x-frame-options"] == "DENY"

# Cache control
test: res.headers["cache-control"].contains("no-cache")

# Rate limiting
test: res.headers["x-rate-limit-remaining"] > "10"

# Custom headers
test: |
  res.headers.has("x-request-id") &&
  res.headers["x-request-id"].length == 36  # UUID format
```

## JSON Response Testing

### Basic JSON Validation

```yaml
# JSON structure validation
test: |
  res.status == 200 &&
  res.json != null &&
  res.json.status == "success" &&
  res.json.data != null

# Required fields presence
test: |
  res.json.has("id") &&
  res.json.has("name") &&
  res.json.has("email") &&
  res.json.has("created_at")
```

### Data Type Validation

```yaml
# Type checking
test: |
  typeof(res.json.id) == "number" &&
  typeof(res.json.name) == "string" &&
  typeof(res.json.active) == "boolean" &&
  typeof(res.json.tags) == "array" &&
  typeof(res.json.metadata) == "object"

# Value constraints
test: |
  res.json.id > 0 &&
  res.json.name.length >= 2 &&
  res.json.score >= 0 && res.json.score <= 100
```

### Array and Collection Testing

```yaml
# Array validation
test: |
  res.json.users != null &&
  res.json.users.length > 0 &&
  res.json.users.length <= 100

# Array content validation
test: |
  res.json.users.all(user -> 
    user.id != null && 
    user.email != null
  )

# Specific element checks
test: |
  res.json.users.any(user -> user.role == "admin") &&
  res.json.users.filter(user -> user.active == true).length > 0

# Array uniqueness
test: |
  res.json.user_ids.length == res.json.user_ids.unique().length
```

### Nested Data Validation

```yaml
# Deep object validation
test: |
  res.json.user != null &&
  res.json.user.profile != null &&
  res.json.user.profile.preferences != null &&
  res.json.user.profile.preferences.notifications == true

# Complex nested structures
test: |
  res.json.data.orders.all(order ->
    order.id != null &&
    order.items.length > 0 &&
    order.items.all(item -> 
      item.product_id != null && 
      item.quantity > 0 && 
      item.price > 0
    ) &&
    order.total == order.items.map(item -> item.quantity * item.price).sum()
  )
```

## Text Response Testing

### Pattern Matching

```yaml
# Simple text matching
test: res.text.contains("success")
test: res.text.startsWith("<!DOCTYPE html>")
test: res.text.endsWith("</html>")

# Case-insensitive matching
test: res.text.lower().contains("error")

# Multiple patterns
test: |
  res.text.contains("status") &&
  res.text.contains("healthy") &&
  !res.text.contains("error")
```

### Regular Expression Testing

```yaml
# Email validation in response
test: res.text.matches("user-\\d+@example\\.com")

# URL pattern validation
test: res.text.matches("https://[a-zA-Z0-9.-]+/api/v\\d+/")

# Data format validation
test: |
  res.text.matches("\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z")  # ISO timestamp

# Extract and validate data
test: |
  res.text.matches("Version: v\\d+\\.\\d+\\.\\d+") &&
  res.text.extract("v(\\d+)\\.(\\d+)\\.(\\d+)")[1] >= "2"  # Major version >= 2
```

### Content Length and Quality

```yaml
# Content length validation
test: |
  res.text.length > 100 &&
  res.text.length < 10000

# Content quality checks
test: |
  res.text.split("\\n").length > 5 &&           # Multi-line content
  !res.text.contains("Lorem ipsum") &&          # Not placeholder text
  res.text.split(" ").length > 20               # Substantial content
```

## Advanced Testing Patterns

### Conditional Testing

```yaml
# Environment-specific tests
test: |
  res.status == 200 &&
  (env.NODE_ENV == "development" ? 
    res.time < 5000 :           # More lenient for dev
    res.time < 1000             # Strict for production
  )

# Feature flag testing
test: |
  res.status == 200 &&
  (res.json.features.beta_enabled == true ?
    res.json.beta_data != null :    # Beta features should have data
    res.json.beta_data == null      # Beta features should be absent
  )
```

### Cross-Step Validation

```yaml
jobs:
  data-consistency-test:
    steps:
      - name: Get User Count
        id: user-count
        action: http
        with:
          url: "{{env.API_URL}}/users/count"
        test: res.status == 200
        outputs:
          total_users: res.json.count

      - name: Get User List
        id: user-list
        action: http
        with:
          url: "{{env.API_URL}}/users"
        test: |
          res.status == 200 &&
          res.json.users.length == outputs.user-count.total_users  # Consistency check
        outputs:
          user_list: res.json.users

      - name: Validate User Data Integrity
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.user-list.user_list[0].id}}"
        test: |
          res.status == 200 &&
          res.json.user.id == outputs.user-list.user_list[0].id &&
          res.json.user.email == outputs.user-list.user_list[0].email
```

### Business Logic Testing

```yaml
- name: E-commerce Business Logic Test
  action: http
  with:
    url: "{{env.API_URL}}/orders/{{env.TEST_ORDER_ID}}"
  test: |
    res.status == 200 &&
    res.json.order != null &&
    
    # Order total equals sum of line items
    res.json.order.total == 
      res.json.order.line_items.map(item -> item.quantity * item.price).sum() &&
    
    # Tax calculation is correct (assuming 8% tax rate)
    res.json.order.tax_amount == 
      Math.round(res.json.order.subtotal * 0.08 * 100) / 100 &&
    
    # Shipping is applied correctly
    (res.json.order.subtotal >= 100 ? 
      res.json.order.shipping_cost == 0 :     # Free shipping over $100
      res.json.order.shipping_cost == 9.99    # Standard shipping
    ) &&
    
    # Final total calculation
    res.json.order.total == 
      res.json.order.subtotal + res.json.order.tax_amount + res.json.order.shipping_cost
```

## Error Testing and Negative Cases

### Expected Error Scenarios

```yaml
- name: Test Invalid Authentication
  action: http
  with:
    url: "{{env.API_URL}}/protected"
    headers:
      Authorization: "Bearer invalid-token"
  test: |
    res.status == 401 &&
    res.json.error == "invalid_token" &&
    res.json.message.contains("authentication")

- name: Test Rate Limiting
  action: http
  with:
    url: "{{env.API_URL}}/rate-limited-endpoint"
  test: |
    res.status in [200, 429] &&  # Either success or rate limited
    (res.status == 429 ? 
      res.headers.has("retry-after") && 
      res.json.error == "rate_limit_exceeded" :
      res.json.status == "success"
    )

- name: Test Malformed Request
  action: http
  with:
    url: "{{env.API_URL}}/users"
    method: POST
    body: '{"invalid": json}'  # Intentionally malformed
  test: |
    res.status == 400 &&
    res.json.error.contains("json") &&
    res.json.details != null
```

### Boundary Testing

```yaml
- name: Test Input Boundaries
  action: http
  with:
    url: "{{env.API_URL}}/users"
    method: POST
    body: |
      {
        "name": "{{random_str(255)}}",  # Maximum length
        "age": 150,                     # Upper boundary
        "score": 0                      # Lower boundary
      }
  test: |
    res.status in [201, 400] &&  # Either created or validation error
    (res.status == 400 ? 
      res.json.validation_errors != null :
      res.json.user.id != null
    )
```

## Test Organization Patterns

### Layered Testing Strategy

```yaml
jobs:
  smoke-tests:
    name: Smoke Tests
    steps:
      - name: Basic Connectivity
        action: http
        with:
          url: "{{env.API_URL}}/ping"
        test: res.status == 200

  functional-tests:
    name: Functional Tests
    needs: [smoke-tests]
    steps:
      - name: User Management
        action: http
        with:
          url: "{{env.API_URL}}/users"
        test: |
          res.status == 200 &&
          res.json.users != null &&
          res.json.pagination != null

  integration-tests:
    name: Integration Tests
    needs: [functional-tests]
    steps:
      - name: Cross-Service Integration
        action: http
        with:
          url: "{{env.API_URL}}/integration/full-flow"
        test: |
          res.status == 200 &&
          res.json.all_services_connected == true &&
          res.json.data_consistency_check == true
```

### Comprehensive Test Suites

```yaml
jobs:
  api-test-suite:
    name: Comprehensive API Test Suite
    steps:
      # Authentication Tests
      - name: Valid Login
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
        test: |
          res.status == 200 &&
          res.json.access_token != null &&
          res.json.refresh_token != null &&
          res.json.expires_in > 0
        outputs:
          access_token: res.json.access_token

      - name: Invalid Login
        action: http
        with:
          url: "{{env.API_URL}}/auth/login"
          method: POST
          body: |
            {
              "username": "invalid",
              "password": "wrong"
            }
        test: |
          res.status == 401 &&
          res.json.error == "invalid_credentials"

      # CRUD Operations Tests
      - name: Create User
        id: create-user
        action: http
        with:
          url: "{{env.API_URL}}/users"
          method: POST
          headers:
            Authorization: "Bearer {{outputs.login.access_token}}"
          body: |
            {
              "name": "Test User {{random_str(6)}}",
              "email": "test{{random_str(8)}}@example.com",
              "role": "user"
            }
        test: |
          res.status == 201 &&
          res.json.user.id != null &&
          res.json.user.name != null &&
          res.json.user.email != null
        outputs:
          user_id: res.json.user.id
          user_email: res.json.user.email

      - name: Read User
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.create-user.user_id}}"
          headers:
            Authorization: "Bearer {{outputs.login.access_token}}"
        test: |
          res.status == 200 &&
          res.json.user.id == outputs.create-user.user_id &&
          res.json.user.email == "{{outputs.create-user.user_email}}"

      - name: Update User
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.create-user.user_id}}"
          method: PUT
          headers:
            Authorization: "Bearer {{outputs.login.access_token}}"
          body: |
            {
              "name": "Updated Test User"
            }
        test: |
          res.status == 200 &&
          res.json.user.name == "Updated Test User"

      - name: Delete User
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.create-user.user_id}}"
          method: DELETE
          headers:
            Authorization: "Bearer {{outputs.login.access_token}}"
        test: res.status == 204

      - name: Verify Deletion
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.create-user.user_id}}"
          headers:
            Authorization: "Bearer {{outputs.login.access_token}}"
        test: res.status == 404
```

## Performance Testing

### Response Time Benchmarks

```yaml
- name: Performance Benchmark Test
  action: http
  with:
    url: "{{env.API_URL}}/performance-test"
  test: |
    res.status == 200 &&
    
    # Tiered performance expectations
    (env.NODE_ENV == "production" ? 
      res.time < 500 :              # Production: < 500ms
      res.time < 2000               # Non-production: < 2s
    ) &&
    
    # Additional performance metrics
    res.json.query_time < 100 &&    # Database query time
    res.json.render_time < 50       # Template render time
  outputs:
    response_time: res.time
    query_time: res.json.query_time
    render_time: res.json.render_time
```

### Load Testing Validation

```yaml
- name: Load Test Results Validation
  action: http
  with:
    url: "{{env.LOAD_TEST_URL}}/results"
  test: |
    res.status == 200 &&
    res.json.test_completed == true &&
    
    # Success rate requirements
    res.json.success_rate >= 0.95 &&
    
    # Performance percentiles
    res.json.percentiles.p50 < 1000 &&
    res.json.percentiles.p95 < 2000 &&
    res.json.percentiles.p99 < 5000 &&
    
    # Error rate limits
    res.json.error_rate < 0.05 &&
    
    # No critical errors
    res.json.critical_errors == 0
```

## Security Testing

### Authentication and Authorization

```yaml
- name: Test Unauthorized Access
  action: http
  with:
    url: "{{env.API_URL}}/admin/users"
  test: |
    res.status == 401 &&
    res.json.error == "authentication_required"

- name: Test Insufficient Permissions
  action: http
  with:
    url: "{{env.API_URL}}/admin/users"
    headers:
      Authorization: "Bearer {{env.USER_TOKEN}}"  # Regular user token
  test: |
    res.status == 403 &&
    res.json.error == "insufficient_permissions"

- name: Test Token Expiration
  action: http
  with:
    url: "{{env.API_URL}}/protected"
    headers:
      Authorization: "Bearer {{env.EXPIRED_TOKEN}}"
  test: |
    res.status == 401 &&
    res.json.error == "token_expired"
```

### Input Validation Security

```yaml
- name: Test SQL Injection Protection
  action: http
  with:
    url: "{{env.API_URL}}/users?search='; DROP TABLE users; --"
  test: |
    res.status in [200, 400] &&  # Either filtered or rejected
    !res.text.contains("sql") && # No SQL error messages
    !res.text.contains("syntax") &&
    res.json.error != "internal_server_error"  # Should not cause server error

- name: Test XSS Protection
  action: http
  with:
    url: "{{env.API_URL}}/comments"
    method: POST
    body: |
      {
        "content": "<script>alert('xss')</script>"
      }
  test: |
    res.status in [201, 400] &&
    (res.status == 201 ? 
      !res.json.comment.content.contains("<script>") :  # Should be sanitized
      res.json.validation_errors != null               # Or rejected
    )
```

## Test Documentation and Reporting

### Self-Documenting Tests

```yaml
- name: User Registration Flow Test
  action: http
  with:
    url: "{{env.API_URL}}/auth/register"
    method: POST
    body: |
      {
        "email": "{{random_str(8)}}@example.com",
        "password": "TestPass123!",
        "confirm_password": "TestPass123!"
      }
  # Comprehensive test with clear validation points
  test: |
    res.status == 201 &&                                    # 1. Successful creation
    res.json.user.id != null &&                            # 2. User ID assigned
    res.json.user.email != null &&                         # 3. Email stored
    res.json.user.password == null &&                      # 4. Password not returned
    res.json.user.created_at != null &&                    # 5. Timestamp recorded
    res.json.user.email_verified == false &&               # 6. Email unverified initially
    res.json.verification_email_sent == true &&            # 7. Verification triggered
    res.headers.has("location") &&                         # 8. Location header present
    res.headers["location"].contains("/users/")             # 9. Correct redirect path
  outputs:
    user_id: res.json.user.id
    user_email: res.json.user.email
    test_summary: |
      Registration test completed:
      - User ID: {{res.json.user.id}}
      - Email: {{res.json.user.email}}
      - Verification: {{res.json.verification_email_sent ? "Sent" : "Failed"}}
      - Response time: {{res.time}}ms
```

### Test Result Aggregation

```yaml
jobs:
  test-summary:
    name: Test Results Summary
    needs: [smoke-tests, functional-tests, security-tests]
    steps:
      - name: Generate Test Report
        echo: |
          Test Execution Summary
          =====================
          
          Smoke Tests: {{jobs.smoke-tests.success ? "✅ PASSED" : "❌ FAILED"}}
          Functional Tests: {{jobs.functional-tests.success ? "✅ PASSED" : "❌ FAILED"}}
          Security Tests: {{jobs.security-tests.success ? "✅ PASSED" : "❌ FAILED"}}
          
          Overall Result: {{
            jobs.smoke-tests.success && 
            jobs.functional-tests.success && 
            jobs.security-tests.success ? "✅ ALL TESTS PASSED" : "❌ SOME TESTS FAILED"
          }}
          
          Execution Time: {{unixtime()}}
          Test Environment: {{env.NODE_ENV || "development"}}
```

## Best Practices

### 1. Clear Test Intentions

```yaml
# Good: Specific, testable conditions
test: |
  res.status == 200 &&
  res.json.users.length >= 1 &&
  res.json.users[0].id != null

# Avoid: Vague or incomplete tests
test: res.status == 200  # What about response content?
```

### 2. Comprehensive Error Coverage

```yaml
# Good: Test both success and failure paths
- name: Valid Request Test
  test: res.status == 200 && res.json.success == true

- name: Invalid Request Test  
  test: res.status == 400 && res.json.error != null
```

### 3. Performance-Aware Testing

```yaml
# Good: Include performance validation
test: |
  res.status == 200 &&
  res.time < 1000 &&
  res.json.data != null

# Good: Environment-specific performance thresholds
test: |
  res.status == 200 &&
  res.time < {{env.MAX_RESPONSE_TIME || 2000}}
```

### 4. Maintainable Test Expressions

```yaml
# Good: Readable, well-structured tests
test: |
  res.status == 200 &&
  res.json.user != null &&
  res.json.user.id > 0 &&
  res.json.user.email.contains("@")

# Avoid: Complex, hard-to-read tests
test: res.status == 200 && res.json.user != null && res.json.user.id > 0 && res.json.user.email.contains("@") && res.json.user.active == true && res.time < 1000
```

## What's Next?

Now that you understand testing and assertions, explore:

1. **[Error Handling](../error-handling/)** - Learn to handle failures gracefully
2. **[Execution Model](../execution-model/)** - Understand workflow execution flow
3. **[How-tos](../../how-tos/)** - See practical testing patterns in action

Testing and assertions are your quality gates. Master these concepts to build reliable, robust automation that catches issues before they impact your systems.