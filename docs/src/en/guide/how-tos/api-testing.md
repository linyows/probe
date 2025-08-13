# API Testing

This guide shows you how to build comprehensive API testing workflows with Probe. You'll learn to test REST APIs thoroughly, validate responses, handle authentication, and implement advanced testing patterns.

## Basic API Testing

### Simple GET Request Test

Start with a basic API endpoint test:

```yaml
name: Basic API Test
description: Test a simple REST API endpoint

vars:
  api_base_url: "{{API_BASE_URL ?? 'https://jsonplaceholder.typicode.com'}}"
  timeout: "{{TIMEOUT ?? '30s'}}"

defaults:
  http:
    timeout: "{{vars.timeout}}"
    headers:
      Accept: "application/json"
      User-Agent: "Probe API Tester v1.0"

jobs:
  basic-api-test:
    name: Basic API Test
    steps:
      - name: Get Posts
        action: http
        with:
          url: "{{vars.api_base_url}}/posts"
        test: |
          res.status == 200 &&
          res.headers["content-type"].contains("application/json") &&
          res.json != null &&
          res.json.length > 0
        outputs:
          post_count: res.json.length
          first_post_id: res.json[0].id
          response_time: res.time

      - name: Get Single Post
        action: http
        with:
          url: "{{vars.api_base_url}}/posts/{{outputs.first_post_id}}"
        test: |
          res.status == 200 &&
          res.json.id == outputs.first_post_id &&
          res.json.title != null &&
          res.json.body != null
        outputs:
          post_title: res.json.title
          post_body: res.json.body

      - name: Test Results Summary
        echo: |
          📊 API Test Results:
          
          Posts Retrieved: {{outputs.post_count}}
          First Post ID: {{outputs.first_post_id}}
          Post Title: "{{outputs.post_title}}"
          Response Time: {{outputs.response_time}}ms
          
          ✅ Basic API tests completed successfully
```

### CRUD Operations Testing

Test Create, Read, Update, Delete operations:

```yaml
name: CRUD API Testing
description: Test complete CRUD operations on a REST API

vars:
  api_base_url: "{{API_BASE_URL ?? 'https://jsonplaceholder.typicode.com'}}"
  test_user_id: "{{TEST_USER_ID ?? '1'}}"

jobs:
  crud-operations:
    name: CRUD Operations Test
    steps:
      # CREATE - POST Request
      - name: Create New Post
        id: create
        action: http
        with:
          url: "{{vars.api_base_url}}/posts"
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "title": "Test Post {{random_str(6)}}",
              "body": "This is a test post created by Probe at {{unixtime()}}",
              "userId": {{vars.test_user_id}}
            }
        test: |
          res.status == 201 &&
          res.json.id != null &&
          res.json.title != null &&
          res.json.userId == {{vars.test_user_id}}
        outputs:
          created_post_id: res.json.id
          created_title: res.json.title
          created_body: res.json.body

      # READ - GET Request
      - name: Read Created Post
        action: http
        with:
          url: "{{vars.api_base_url}}/posts/{{outputs.create.created_post_id}}"
        test: |
          res.status == 200 &&
          res.json.id == outputs.create.created_post_id &&
          res.json.title == "{{outputs.create.created_title}}"
        outputs:
          read_success: true

      # UPDATE - PUT Request
      - name: Update Post
        id: update
        action: http
        with:
          url: "{{vars.api_base_url}}/posts/{{outputs.create.created_post_id}}"
          method: PUT
          headers:
            Content-Type: "application/json"
          body: |
            {
              "id": {{outputs.create.created_post_id}},
              "title": "Updated: {{outputs.create.created_title}}",
              "body": "This post was updated by Probe at {{unixtime()}}",
              "userId": {{vars.test_user_id}}
            }
        test: |
          res.status == 200 &&
          res.json.id == outputs.create.created_post_id &&
          res.json.title.startsWith("Updated:")
        outputs:
          updated_title: res.json.title

      # PARTIAL UPDATE - PATCH Request
      - name: Partial Update Post
        action: http
        with:
          url: "{{vars.api_base_url}}/posts/{{outputs.create.created_post_id}}"
          method: PATCH
          headers:
            Content-Type: "application/json"
          body: |
            {
              "title": "Patched: {{outputs.update.updated_title}}"
            }
        test: |
          res.status == 200 &&
          res.json.title.startsWith("Patched:")
        outputs:
          patched_title: res.json.title

      # DELETE - DELETE Request
      - name: Delete Post
        action: http
        with:
          url: "{{vars.api_base_url}}/posts/{{outputs.create.created_post_id}}"
          method: DELETE
        test: res.status == 200
        outputs:
          deleted: true

      # VERIFY DELETION
      - name: Verify Deletion
        action: http
        with:
          url: "{{vars.api_base_url}}/posts/{{outputs.create.created_post_id}}"
        test: res.status == 404
        continue_on_error: true
        outputs:
          deletion_verified: res.status == 404

      - name: CRUD Test Summary
        echo: |
          🔄 CRUD Operations Test Summary:
          
          ✅ CREATE: Post ID {{outputs.create.created_post_id}} created
             Title: "{{outputs.create.created_title}}"
          
          ✅ READ: Successfully retrieved created post
          
          ✅ UPDATE: Title updated to "{{outputs.update.updated_title}}"
          
          ✅ PATCH: Title patched to "{{outputs.patched_title}}"
          
          ✅ DELETE: Post deletion {{outputs.deleted ? "successful" : "failed"}}
          
          ✅ VERIFY: Deletion {{outputs.deletion_verified ? "verified (404)" : "not verified"}}
          
          All CRUD operations completed successfully!
```

## Authentication Testing

### Bearer Token Authentication

```yaml
name: Bearer Token API Testing
description: Test APIs with Bearer token authentication

vars:
  api_base_url: "{{API_BASE_URL ?? 'https://api.yourapp.com'}}"
  auth_url: "{{AUTH_URL ?? 'https://auth.yourapp.com'}}"
  test_username: "{{TEST_USERNAME ?? 'test@example.com'}}"
  test_password: "{{TEST_PASSWORD ?? 'test_password_123'}}"
  client_id: "{{CLIENT_ID}}"
  client_secret: "{{CLIENT_SECRET}}"

jobs:
  authentication-flow:
    name: Authentication Flow Test
    steps:
      # Step 1: Obtain Bearer Token
      - name: Login and Get Token
        id: login
        action: http
        with:
          url: "{{vars.auth_url}}/oauth/token"
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "grant_type": "password",
              "username": "{{vars.test_username}}",
              "password": "{{vars.test_password}}",
              "client_id": "{{vars.client_id}}",
              "client_secret": "{{vars.client_secret}}"
            }
        test: |
          res.status == 200 &&
          res.json.access_token != null &&
          res.json.token_type == "Bearer" &&
          res.json.expires_in > 0
        outputs:
          access_token: res.json.access_token
          refresh_token: res.json.refresh_token
          expires_in: res.json.expires_in
          token_type: res.json.token_type

      # Step 2: Test Authenticated Endpoint
      - name: Get User Profile
        id: profile
        action: http
        with:
          url: "{{vars.api_base_url}}/user/profile"
          headers:
            Authorization: "{{outputs.login.token_type}} {{outputs.login.access_token}}"
            Accept: "application/json"
        test: |
          res.status == 200 &&
          res.json.id != null &&
          res.json.email == "{{vars.test_username}}"
        outputs:
          user_id: res.json.id
          user_email: res.json.email
          user_name: res.json.name

      # Step 3: Test Protected Resource
      - name: Access Protected Resource
        action: http
        with:
          url: "{{vars.api_base_url}}/user/{{outputs.profile.user_id}}/data"
          headers:
            Authorization: "{{outputs.login.token_type}} {{outputs.login.access_token}}"
        test: |
          res.status == 200 &&
          res.json.user_id == outputs.profile.user_id
        outputs:
          protected_data_accessible: true

      # Step 4: Test Token Refresh
      - name: Refresh Token
        id: refresh
        action: http
        with:
          url: "{{vars.auth_url}}/oauth/token"
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "grant_type": "refresh_token",
              "refresh_token": "{{outputs.login.refresh_token}}",
              "client_id": "{{vars.client_id}}",
              "client_secret": "{{vars.client_secret}}"
            }
        test: |
          res.status == 200 &&
          res.json.access_token != null &&
          res.json.access_token != "{{outputs.login.access_token}}"
        outputs:
          new_access_token: res.json.access_token

      # Step 5: Test with New Token
      - name: Test New Token
        action: http
        with:
          url: "{{vars.api_base_url}}/user/profile"
          headers:
            Authorization: "Bearer {{outputs.refresh.new_access_token}}"
        test: res.status == 200
        outputs:
          new_token_valid: true

  unauthorized-access-test:
    name: Unauthorized Access Test
    steps:
      # Test without token
      - name: Test No Authorization
        action: http
        with:
          url: "{{vars.api_base_url}}/user/profile"
        test: res.status == 401
        outputs:
          no_auth_rejected: res.status == 401

      # Test with invalid token
      - name: Test Invalid Token
        action: http
        with:
          url: "{{vars.api_base_url}}/user/profile"
          headers:
            Authorization: "Bearer invalid_token_123"
        test: res.status == 401
        outputs:
          invalid_token_rejected: res.status == 401

      # Test with expired token (if available)
      - name: Test Expired Token
        if: vars.expired_token
        action: http
        with:
          url: "{{vars.api_base_url}}/user/profile"
          headers:
            Authorization: "Bearer {{vars.expired_token}}"
        test: res.status == 401
        outputs:
          expired_token_rejected: res.status == 401

  security-test-summary:
    name: Security Test Summary
    needs: [authentication-flow, unauthorized-access-test]
    steps:
      - name: Authentication Summary
        echo: |
          🔐 Authentication & Authorization Test Results:
          
          AUTHENTICATION FLOW:
          ✅ Login: {{outputs.authentication-flow.access_token ? "Token obtained" : "Failed"}}
          ✅ Profile Access: {{outputs.authentication-flow.user_email ? "Success" : "Failed"}}
          ✅ Protected Resource: {{outputs.authentication-flow.protected_data_accessible ? "Accessible" : "Failed"}}
          ✅ Token Refresh: {{outputs.authentication-flow.new_access_token ? "Success" : "Failed"}}
          ✅ New Token Valid: {{outputs.authentication-flow.new_token_valid ? "Yes" : "No"}}
          
          SECURITY VALIDATION:
          ✅ No Auth Rejected: {{outputs.unauthorized-access-test.no_auth_rejected ? "Yes (401)" : "Security Issue!"}}
          ✅ Invalid Token Rejected: {{outputs.unauthorized-access-test.invalid_token_rejected ? "Yes (401)" : "Security Issue!"}}
          {{vars.expired_token ? "✅ Expired Token Rejected: " + (outputs.unauthorized-access-test.expired_token_rejected ? "Yes (401)" : "Security Issue!") : ""}}
          
          USER INFORMATION:
          User ID: {{outputs.authentication-flow.user_id}}
          Email: {{outputs.authentication-flow.user_email}}
          Name: {{outputs.authentication-flow.user_name}}
          Token Expires In: {{outputs.authentication-flow.expires_in}} seconds
```

### API Key Authentication

```yaml
name: API Key Authentication Testing
description: Test APIs using API key authentication

vars:
  api_base_url: "{{API_BASE_URL ?? 'https://api.yourservice.com'}}"
  api_key: "{{API_KEY}}"
  rate_limit_tier: "{{RATE_LIMIT_TIER ?? 'premium'}}"

jobs:
  api-key-tests:
    name: API Key Authentication Tests
    steps:
      # Header-based API Key
      - name: Test API Key in Header
        action: http
        with:
          url: "{{vars.api_base_url}}/data"
          headers:
            X-API-Key: "{{vars.api_key}}"
            Accept: "application/json"
        test: |
          res.status == 200 &&
          res.json.authenticated == true &&
          res.json.rate_limit != null
        outputs:
          header_auth_success: true
          rate_limit_remaining: res.json.rate_limit.remaining
          rate_limit_reset: res.json.rate_limit.reset_time

      # Query Parameter API Key
      - name: Test API Key in Query
        action: http
        with:
          url: "{{vars.api_base_url}}/data?api_key={{vars.api_key}}"
        test: res.status == 200
        outputs:
          query_auth_success: true

      # Test Rate Limiting
      - name: Test Rate Limit Info
        action: http
        with:
          url: "{{vars.api_base_url}}/rate-limit-status"
          headers:
            X-API-Key: "{{vars.api_key}}"
        test: |
          res.status == 200 &&
          res.json.tier == "{{vars.rate_limit_tier}}" &&
          res.json.requests_remaining > 0
        outputs:
          requests_remaining: res.json.requests_remaining
          requests_per_hour: res.json.limits.per_hour
          current_usage: res.json.current_usage

      # Test Invalid API Key
      - name: Test Invalid API Key
        action: http
        with:
          url: "{{vars.api_base_url}}/data"
          headers:
            X-API-Key: "invalid_key_123"
        test: res.status == 401 || res.status == 403
        outputs:
          invalid_key_rejected: res.status == 401 || res.status == 403

      - name: API Key Test Summary
        echo: |
          🔑 API Key Authentication Results:
          
          Header Authentication: {{outputs.header_auth_success ? "✅ Success" : "❌ Failed"}}
          Query Authentication: {{outputs.query_auth_success ? "✅ Success" : "❌ Failed"}}
          Invalid Key Rejected: {{outputs.invalid_key_rejected ? "✅ Yes" : "❌ Security Issue"}}
          
          Rate Limiting:
          Tier: {{vars.rate_limit_tier}}
          Requests Remaining: {{outputs.requests_remaining}}/{{outputs.requests_per_hour}}
          Current Usage: {{outputs.current_usage}}
          Reset Time: {{outputs.rate_limit_reset}}
```

## Data Validation and Response Testing

### JSON Schema Validation

```yaml
name: JSON Response Validation
description: Validate API responses against expected schemas

env:
  API_BASE_URL: https://api.yourservice.com

jobs:
  response-validation:
    name: Response Schema Validation
    steps:
      - name: Get User List
        id: users
        action: http
        with:
          url: "{{vars.api_base_url}}/users"
          headers:
            Authorization: "Bearer {{vars.api_token}}"
        test: |
          res.status == 200 &&
          res.json != null &&
          res.json.data != null &&
          res.json.data.length > 0 &&
          
          # Validate response structure
          res.json.meta != null &&
          res.json.meta.total != null &&
          res.json.meta.page != null &&
          res.json.meta.per_page != null &&
          
          # Validate first user object
          res.json.data[0].id != null &&
          res.json.data[0].email != null &&
          res.json.data[0].name != null &&
          res.json.data[0].created_at != null &&
          
          # Validate data types
          typeof(res.json.data[0].id) == "number" &&
          typeof(res.json.data[0].email) == "string" &&
          typeof(res.json.data[0].active) == "boolean"
        outputs:
          total_users: res.json.meta.total
          users_per_page: res.json.meta.per_page
          first_user_id: res.json.data[0].id
          first_user_email: res.json.data[0].email

      - name: Get Single User Details
        action: http
        with:
          url: "{{vars.api_base_url}}/users/{{outputs.users.first_user_id}}"
          headers:
            Authorization: "Bearer {{vars.api_token}}"
        test: |
          res.status == 200 &&
          res.json.user != null &&
          
          # Validate required fields
          res.json.user.id == outputs.users.first_user_id &&
          res.json.user.email == "{{outputs.users.first_user_email}}" &&
          res.json.user.profile != null &&
          
          # Validate nested objects
          res.json.user.profile.first_name != null &&
          res.json.user.profile.last_name != null &&
          res.json.user.preferences != null &&
          
          # Validate arrays
          res.json.user.roles != null &&
          res.json.user.roles.length > 0 &&
          res.json.user.permissions != null
        outputs:
          user_roles: res.json.user.roles
          user_permissions_count: res.json.user.permissions.length
          profile_complete: res.json.user.profile.first_name != null && res.json.user.profile.last_name != null

      - name: Test Data Consistency
        action: http
        with:
          url: "{{vars.api_base_url}}/users/{{outputs.users.first_user_id}}/orders"
          headers:
            Authorization: "Bearer {{vars.api_token}}"
        test: |
          res.status == 200 &&
          res.json.orders != null &&
          
          # Validate all orders belong to the user
          res.json.orders.all(order -> order.user_id == outputs.users.first_user_id) &&
          
          # Validate order structure
          res.json.orders.all(order -> 
            order.id != null &&
            order.total != null &&
            order.status != null &&
            order.created_at != null
          ) &&
          
          # Validate business logic
          res.json.orders.all(order -> order.total >= 0) &&
          res.json.orders.filter(order -> order.status == "completed").all(order -> order.completed_at != null)
        outputs:
          order_count: res.json.orders.length
          completed_orders: res.json.orders.filter(order -> order.status == "completed").length
          total_spent: res.json.orders.filter(order -> order.status == "completed").map(order -> order.total).sum()

      - name: Validation Summary
        echo: |
          📋 Data Validation Results:
          
          USER LIST VALIDATION:
          ✅ Response Structure: Valid pagination metadata
          ✅ User Objects: All required fields present
          ✅ Data Types: Correct types for all fields
          Total Users: {{outputs.users.total_users}}
          Users Per Page: {{outputs.users.users_per_page}}
          
          USER DETAILS VALIDATION:
          ✅ User Profile: {{outputs.profile_complete ? "Complete" : "Incomplete"}}
          ✅ Security: {{outputs.user_roles.length}} roles, {{outputs.user_permissions_count}} permissions
          
          DATA CONSISTENCY:
          ✅ Orders: {{outputs.order_count}} total orders
          ✅ Completed: {{outputs.completed_orders}} completed orders
          ✅ Business Logic: All orders have valid totals and timestamps
          Total Customer Value: ${{outputs.total_spent}}
```

### Error Response Validation

```yaml
name: Error Response Validation
description: Test error handling and response formats

env:
  API_BASE_URL: https://api.yourservice.com

jobs:
  error-response-tests:
    name: Error Response Tests
    steps:
      # Test 400 - Bad Request
      - name: Test Bad Request
        action: http
        with:
          url: "{{vars.api_base_url}}/users"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{vars.api_token}}"
          body: |
            {
              "email": "invalid-email",
              "name": ""
            }
        test: |
          res.status == 400 &&
          res.json.error != null &&
          res.json.error.code == "validation_error" &&
          res.json.error.message != null &&
          res.json.error.details != null &&
          res.json.error.details.length > 0
        outputs:
          validation_errors: res.json.error.details
          error_code: res.json.error.code

      # Test 401 - Unauthorized
      - name: Test Unauthorized Access
        action: http
        with:
          url: "{{vars.api_base_url}}/admin/users"
        test: |
          res.status == 401 &&
          res.json.error != null &&
          res.json.error.code == "unauthorized" &&
          res.json.error.message.contains("authentication")
        outputs:
          auth_error_proper: true

      # Test 403 - Forbidden
      - name: Test Forbidden Access
        action: http
        with:
          url: "{{vars.api_base_url}}/admin/users"
          headers:
            Authorization: "Bearer {{env.USER_TOKEN}}"  # Non-admin token
        test: |
          res.status == 403 &&
          res.json.error != null &&
          res.json.error.code == "forbidden" &&
          res.json.error.message.contains("permission")
        outputs:
          permission_error_proper: true

      # Test 404 - Not Found
      - name: Test Not Found
        action: http
        with:
          url: "{{vars.api_base_url}}/users/99999999"
          headers:
            Authorization: "Bearer {{vars.api_token}}"
        test: |
          res.status == 404 &&
          res.json.error != null &&
          res.json.error.code == "not_found" &&
          res.json.error.resource == "user"
        outputs:
          not_found_error_proper: true

      # Test 422 - Unprocessable Entity
      - name: Test Unprocessable Entity
        action: http
        with:
          url: "{{vars.api_base_url}}/users"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{vars.api_token}}"
          body: |
            {
              "email": "existing@example.com",
              "name": "Test User"
            }
        test: |
          res.status == 422 &&
          res.json.error != null &&
          res.json.error.code == "unprocessable_entity" &&
          res.json.error.details != null
        outputs:
          duplicate_email_handled: true

      # Test Rate Limiting - 429
      - name: Test Rate Limiting
        action: http
        with:
          url: "{{vars.api_base_url}}/high-rate-endpoint"
          headers:
            Authorization: "Bearer {{env.LIMITED_TOKEN}}"
        test: |
          res.status in [200, 429] &&
          (res.status == 429 ? 
            res.json.error.code == "rate_limit_exceeded" &&
            res.headers["retry-after"] != null :
            true
          )
        continue_on_error: true
        outputs:
          rate_limiting_works: res.status == 429

      - name: Error Handling Summary
        echo: |
          🚨 Error Response Validation Results:
          
          400 Bad Request: ✅ Proper validation error format
            Error Code: {{outputs.error_code}}
            Validation Issues: {{outputs.validation_errors.length}}
          
          401 Unauthorized: {{outputs.auth_error_proper ? "✅ Proper format" : "❌ Issues detected"}}
          
          403 Forbidden: {{outputs.permission_error_proper ? "✅ Proper format" : "❌ Issues detected"}}
          
          404 Not Found: {{outputs.not_found_error_proper ? "✅ Proper format" : "❌ Issues detected"}}
          
          422 Unprocessable: {{outputs.duplicate_email_handled ? "✅ Business logic validated" : "❌ Issues detected"}}
          
          429 Rate Limited: {{outputs.rate_limiting_works ? "✅ Rate limiting active" : "ℹ️ No rate limit hit"}}
          
          All error responses follow consistent format and provide helpful information.
```

## Advanced API Testing Patterns

### Workflow Testing

```yaml
name: E-commerce Workflow Testing
description: Test complete e-commerce user workflow

env:
  API_BASE_URL: https://api.ecommerce.com
  TEST_PRODUCT_ID: 123
  TEST_USER_EMAIL: test@example.com

jobs:
  user-registration-flow:
    name: User Registration Workflow
    steps:
      - name: Register New User
        id: register
        action: http
        with:
          url: "{{vars.api_base_url}}/auth/register"
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "email": "test{{random_str(8)}}@example.com",
              "password": "TestPassword123!",
              "first_name": "Test",
              "last_name": "User",
              "phone": "+1234567890"
            }
        test: |
          res.status == 201 &&
          res.json.user.id != null &&
          res.json.user.email != null &&
          res.json.access_token != null
        outputs:
          user_id: res.json.user.id
          user_email: res.json.user.email
          access_token: res.json.access_token

      - name: Email Verification Check
        action: http
        with:
          url: "{{vars.api_base_url}}/user/profile"
          headers:
            Authorization: "Bearer {{outputs.register.access_token}}"
        test: |
          res.status == 200 &&
          res.json.email_verified == false &&
          res.json.verification_email_sent == true
        outputs:
          verification_pending: true

  shopping-flow:
    name: Shopping Workflow
    needs: [user-registration-flow]
    steps:
      - name: Browse Products
        id: browse
        action: http
        with:
          url: "{{vars.api_base_url}}/products?category=electronics&limit=10"
        test: |
          res.status == 200 &&
          res.json.products.length > 0 &&
          res.json.products[0].id != null &&
          res.json.products[0].price > 0
        outputs:
          available_products: res.json.products.length
          first_product_id: res.json.products[0].id
          first_product_price: res.json.products[0].price

      - name: Add to Cart
        id: add-cart
        action: http
        with:
          url: "{{vars.api_base_url}}/cart/items"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{outputs.user-registration-flow.access_token}}"
          body: |
            {
              "product_id": {{outputs.browse.first_product_id}},
              "quantity": 2
            }
        test: |
          res.status == 201 &&
          res.json.cart_item.id != null &&
          res.json.cart_item.quantity == 2 &&
          res.json.cart_total > 0
        outputs:
          cart_item_id: res.json.cart_item.id
          cart_total: res.json.cart_total

      - name: View Cart
        action: http
        with:
          url: "{{vars.api_base_url}}/cart"
          headers:
            Authorization: "Bearer {{outputs.user-registration-flow.access_token}}"
        test: |
          res.status == 200 &&
          res.json.items.length == 1 &&
          res.json.items[0].product_id == outputs.browse.first_product_id &&
          res.json.total == outputs.add-cart.cart_total
        outputs:
          cart_verified: true

  checkout-flow:
    name: Checkout Workflow
    needs: [shopping-flow]
    steps:
      - name: Apply Discount Code
        id: discount
        action: http
        with:
          url: "{{vars.api_base_url}}/cart/discount"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{outputs.user-registration-flow.access_token}}"
          body: |
            {
              "code": "TESTDISCOUNT10"
            }
        test: |
          res.status == 200 &&
          res.json.discount_applied == true &&
          res.json.discount_amount > 0 &&
          res.json.new_total < outputs.shopping-flow.cart_total
        continue_on_error: true
        outputs:
          discount_applied: res.status == 200
          discount_amount: res.json.discount_amount
          final_total: res.json.new_total

      - name: Add Payment Method
        id: payment
        action: http
        with:
          url: "{{vars.api_base_url}}/payment-methods"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{outputs.user-registration-flow.access_token}}"
          body: |
            {
              "type": "credit_card",
              "card_number": "4111111111111111",
              "expiry_month": 12,
              "expiry_year": 2025,
              "cvv": "123",
              "name": "Test User"
            }
        test: |
          res.status == 201 &&
          res.json.payment_method.id != null &&
          res.json.payment_method.last_four == "1111"
        outputs:
          payment_method_id: res.json.payment_method.id

      - name: Create Order
        id: order
        action: http
        with:
          url: "{{vars.api_base_url}}/orders"
          method: POST
          headers:
            Content-Type: "application/json"
            Authorization: "Bearer {{outputs.user-registration-flow.access_token}}"
          body: |
            {
              "payment_method_id": {{outputs.payment.payment_method_id}},
              "shipping_address": {
                "street": "123 Test St",
                "city": "Test City",
                "state": "TS",
                "zip": "12345",
                "country": "US"
              }
            }
        test: |
          res.status == 201 &&
          res.json.order.id != null &&
          res.json.order.status == "processing" &&
          res.json.order.total > 0
        outputs:
          order_id: res.json.order.id
          order_status: res.json.order.status
          order_total: res.json.order.total

      - name: Verify Order Processing
        action: http
        with:
          url: "{{vars.api_base_url}}/orders/{{outputs.order.order_id}}"
          headers:
            Authorization: "Bearer {{outputs.user-registration-flow.access_token}}"
        test: |
          res.status == 200 &&
          res.json.order.id == outputs.order.order_id &&
          res.json.order.user_id == outputs.user-registration-flow.user_id &&
          res.json.order.items.length > 0
        outputs:
          order_verified: true

  workflow-summary:
    name: Workflow Test Summary
    needs: [user-registration-flow, shopping-flow, checkout-flow]
    steps:
      - name: Complete Workflow Results
        echo: |
          🛒 E-commerce Workflow Test Results:
          =====================================
          
          USER REGISTRATION:
          ✅ User Created: {{outputs.user-registration-flow.user_email}}
          ✅ Authentication: Token obtained
          ✅ Email Verification: {{outputs.user-registration-flow.verification_pending ? "Pending (as expected)" : "Issue detected"}}
          
          SHOPPING EXPERIENCE:
          ✅ Product Browse: {{outputs.shopping-flow.available_products}} products found
          ✅ Add to Cart: Product ID {{outputs.shopping-flow.first_product_id}} added
          ✅ Cart Verification: {{outputs.shopping-flow.cart_verified ? "Confirmed" : "Failed"}}
          Cart Total: ${{outputs.shopping-flow.cart_total}}
          
          CHECKOUT PROCESS:
          {{outputs.checkout-flow.discount_applied ? "✅ Discount Applied: $" + outputs.checkout-flow.discount_amount + " off" : "ℹ️ No discount applied"}}
          ✅ Payment Method: Added (ending in 1111)
          ✅ Order Created: Order ID {{outputs.checkout-flow.order_id}}
          ✅ Order Status: {{outputs.checkout-flow.order_status}}
          ✅ Order Verified: {{outputs.checkout-flow.order_verified ? "Confirmed" : "Failed"}}
          Final Total: ${{outputs.checkout-flow.order_total}}
          
          🎉 Complete e-commerce workflow tested successfully!
          User can register, browse, shop, and checkout without issues.
```

## Performance and Load Testing

### Response Time Testing

```yaml
name: API Performance Testing
description: Test API response times and performance characteristics

env:
  API_BASE_URL: https://api.yourservice.com
  PERFORMANCE_THRESHOLD_MS: 1000
  ACCEPTABLE_THRESHOLD_MS: 2000

jobs:
  response-time-tests:
    name: Response Time Performance Tests
    steps:
      - name: Lightweight Endpoint Test
        id: ping
        action: http
        with:
          url: "{{vars.api_base_url}}/ping"
        test: |
          res.status == 200 &&
          res.time < 500
        outputs:
          ping_time: res.time
          ping_fast: res.time < 200

      - name: Database Query Test
        id: db-query
        action: http
        with:
          url: "{{vars.api_base_url}}/users?limit=100"
          headers:
            Authorization: "Bearer {{vars.api_token}}"
        test: |
          res.status == 200 &&
          res.time < {{env.PERFORMANCE_THRESHOLD_MS}}
        outputs:
          query_time: res.time
          query_performance: |
            {{res.time < 500 ? "excellent" : 
              res.time < 1000 ? "good" : 
              res.time < 2000 ? "acceptable" : "poor"}}

      - name: Complex Aggregation Test
        id: aggregation
        action: http
        with:
          url: "{{vars.api_base_url}}/analytics/summary"
          headers:
            Authorization: "Bearer {{vars.api_token}}"
        test: |
          res.status == 200 &&
          res.time < {{env.ACCEPTABLE_THRESHOLD_MS}}
        outputs:
          aggregation_time: res.time
          aggregation_acceptable: res.time < {{env.ACCEPTABLE_THRESHOLD_MS}}

      - name: File Upload Test
        id: upload
        action: http
        with:
          url: "{{vars.api_base_url}}/files/upload"
          method: POST
          headers:
            Authorization: "Bearer {{vars.api_token}}"
            Content-Type: "multipart/form-data"
          body: |
            --boundary123
            Content-Disposition: form-data; name="file"; filename="test.txt"
            Content-Type: text/plain
            
            This is a test file for upload performance testing.
            It contains multiple lines of text to simulate a real file.
            --boundary123--
        test: |
          res.status == 201 &&
          res.time < 5000
        outputs:
          upload_time: res.time
          upload_acceptable: res.time < 3000

      - name: Performance Summary
        echo: |
          ⚡ API Performance Test Results:
          
          ENDPOINT PERFORMANCE:
          Ping: {{outputs.ping_time}}ms {{outputs.ping_fast ? "(🚀 Fast)" : "(⚡ OK)"}}
          Database Query: {{outputs.query_time}}ms ({{outputs.query_performance}})
          Complex Aggregation: {{outputs.aggregation_time}}ms {{outputs.aggregation_acceptable ? "(✅ Acceptable)" : "(⚠️ Slow)"}}
          File Upload: {{outputs.upload_time}}ms {{outputs.upload_acceptable ? "(✅ Acceptable)" : "(⚠️ Slow)"}}
          
          PERFORMANCE CLASSIFICATION:
          {{outputs.ping_time < 200 && outputs.query_time < 500 && outputs.aggregation_time < 1000 ? "🟢 EXCELLENT - All endpoints performing optimally" : ""}}
          {{outputs.ping_time < 500 && outputs.query_time < 1000 && outputs.aggregation_time < 2000 ? "🟡 GOOD - Performance within acceptable ranges" : ""}}
          {{outputs.aggregation_time > 2000 || outputs.upload_time > 5000 ? "🔴 NEEDS ATTENTION - Some endpoints are slow" : ""}}
          
          RECOMMENDATIONS:
          {{outputs.query_time > 800 ? "• Consider database query optimization" : ""}}
          {{outputs.aggregation_time > 1500 ? "• Review aggregation query efficiency" : ""}}
          {{outputs.upload_time > 3000 ? "• Optimize file upload handling" : ""}}
```

## Best Practices

### 1. Test Structure

```yaml
# Good: Organized test structure
jobs:
  authentication:     # Group related tests
  crud-operations:    # Clear test categories
  error-handling:     # Logical organization
  performance:        # Separate concerns
```

### 2. Data Management

```yaml
# Good: Use random data for isolation
body: |
  {
    "email": "test{{random_str(8)}}@example.com",
    "username": "user_{{unixtime()}}_{{random_str(4)}}"
  }

# Good: Clean up test data
- name: Cleanup Test User
  action: http
  with:
    url: "{{vars.api_base_url}}/users/{{outputs.create.user_id}}"
    method: DELETE
```

### 3. Comprehensive Validation

```yaml
# Good: Validate multiple aspects
test: |
  res.status == 200 &&                    # HTTP status
  res.headers["content-type"].contains("json") &&  # Content type
  res.json.id != null &&                  # Required fields
  typeof(res.json.id) == "number" &&      # Data types
  res.json.email.contains("@") &&         # Data format
  res.time < 1000                         # Performance
```

### 4. Error Handling

```yaml
# Good: Test error scenarios
- name: Test Invalid Input
  action: http
  with:
    body: '{"invalid": "data"}'
  test: res.status == 400
  continue_on_error: true

# Good: Validate error responses
test: |
  res.status == 400 &&
  res.json.error.code == "validation_error" &&
  res.json.error.details.length > 0
```

## What's Next?

Now that you can test APIs comprehensively, explore:

- **[Error Handling Strategies](../error-handling-strategies/)** - Robust error handling patterns
- **[Performance Testing](../performance-testing/)** - Advanced load testing techniques
- **[Environment Management](../environment-management/)** - Managing test configurations

API testing is crucial for reliable services. Use these patterns to build comprehensive test suites that catch issues before they reach production.
