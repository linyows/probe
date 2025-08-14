# APIテスト

このガイドでは、Probeを使用して包括的なAPIテストワークフローを構築する方法を説明します。REST APIの徹底的なテスト、レスポンスの検証、認証の処理、および高度なテストパターンの実装について学習します。

## 基本的なAPIテスト

### シンプルなGETリクエストテスト

基本的なAPIエンドポイントテストから始めましょう：

```yaml
name: Basic API Test
description: Test a simple REST API endpoint

vars:
  api_base_url: "{{API_BASE_URL ?? 'https://jsonplaceholder.typicode.com'}}"
  timeout: "{{TIMEOUT ?? '30s'}}"

jobs:
- name: Basic API Test
  defaults:
    http:
      timeout: "{{vars.timeout}}"
      headers:
        Accept: "application/json"
        User-Agent: "Probe API Tester v1.0"
  steps:
    - name: Get Posts
      uses: http
      with:
        url: "{{vars.api_base_url}}/posts"
      test: |
        res.code == 200 &&
        res.headers["content-type"].contains("application/json") &&
        res.body.json != null &&
        res.body.json.length > 0
      outputs:
        post_count: res.body.json.length
        first_post_id: res.body.json[0].id
        response_time: res.time

    - name: Get Single Post
      uses: http
      with:
        url: "{{vars.api_base_url}}/posts/{{outputs.first_post_id}}"
      test: |
        res.code == 200 &&
        res.body.json.id == outputs.first_post_id &&
        res.body.json.title != null &&
        res.body.json.body != null
      outputs:
        post_title: res.body.json.title
        post_body: res.body.json.body

    - name: Test Results Summary
      echo: |
        📊 API Test Results:
        
        Posts Retrieved: {{outputs.post_count}}
        First Post ID: {{outputs.first_post_id}}
        Post Title: "{{outputs.post_title}}"
        Response Time: {{outputs.response_time}}ms
        
        ✅ Basic API tests completed successfully
```

### CRUD操作のテスト

Create、Read、Update、Delete操作をテストします：

```yaml
name: CRUD API Testing
description: Test complete CRUD operations on a REST API

vars:
  api_base_url: "{{API_BASE_URL ?? 'https://jsonplaceholder.typicode.com'}}"
  test_user_id: "{{TEST_USER_ID ?? '1'}}"

jobs:
- name: CRUD Operations Test
  steps:
    # CREATE - POST Request
    - name: Create New Post
      id: create
      uses: http
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
        res.code == 201 &&
        res.body.json.id != null &&
        res.body.json.title != null &&
        res.body.json.userId == {{vars.test_user_id}}
      outputs:
        created_post_id: res.body.json.id
        created_title: res.body.json.title
        created_body: res.body.json.body

    # READ - GET Request
    - name: Read Created Post
      uses: http
      with:
        url: "{{vars.api_base_url}}/posts/{{outputs.create.created_post_id}}"
      test: |
        res.code == 200 &&
        res.body.json.id == outputs.create.created_post_id &&
        res.body.json.title == "{{outputs.create.created_title}}"
      outputs:
        read_success: true

    # UPDATE - PUT Request
    - name: Update Post
      id: update
      uses: http
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
        res.code == 200 &&
        res.body.json.id == outputs.create.created_post_id &&
        res.body.json.title.startsWith("Updated:")
      outputs:
        updated_title: res.body.json.title

    # PARTIAL UPDATE - PATCH Request
    - name: Partial Update Post
      uses: http
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
        res.code == 200 &&
        res.body.json.title.startsWith("Patched:")
      outputs:
        patched_title: res.body.json.title

    # DELETE - DELETE Request
    - name: Delete Post
      uses: http
      with:
        url: "{{vars.api_base_url}}/posts/{{outputs.create.created_post_id}}"
        method: DELETE
      test: res.code == 200
      outputs:
        deleted: true

    # VERIFY DELETION
    - name: Verify Deletion
      uses: http
      with:
        url: "{{vars.api_base_url}}/posts/{{outputs.create.created_post_id}}"
      test: res.code == 404
      continue_on_error: true
      outputs:
        deletion_verified: res.code == 404

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

## 認証テスト

### Bearer Token認証

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
- name: Authentication Flow Test
  steps:
    # Step 1: Obtain Bearer Token
    - name: Login and Get Token
      id: login
      uses: http
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
        res.code == 200 &&
        res.body.json.access_token != null &&
        res.body.json.token_type == "Bearer" &&
        res.body.json.expires_in > 0
      outputs:
        access_token: res.body.json.access_token
        refresh_token: res.body.json.refresh_token
        expires_in: res.body.json.expires_in
        token_type: res.body.json.token_type

    # Step 2: Test Authenticated Endpoint
    - name: Get User Profile
      id: profile
      uses: http
      with:
        url: "{{vars.api_base_url}}/user/profile"
        headers:
          Authorization: "{{outputs.login.token_type}} {{outputs.login.access_token}}"
          Accept: "application/json"
      test: |
        res.code == 200 &&
        res.body.json.id != null &&
        res.body.json.email == "{{vars.test_username}}"
      outputs:
        user_id: res.body.json.id
        user_email: res.body.json.email
        user_name: res.body.json.name

    # Step 3: Test Protected Resource
    - name: Access Protected Resource
      uses: http
      with:
        url: "{{vars.api_base_url}}/user/{{outputs.profile.user_id}}/data"
        headers:
          Authorization: "{{outputs.login.token_type}} {{outputs.login.access_token}}"
      test: |
        res.code == 200 &&
        res.body.json.user_id == outputs.profile.user_id
      outputs:
        protected_data_accessible: true

    # Step 4: Test Token Refresh
    - name: Refresh Token
      id: refresh
      uses: http
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
        res.code == 200 &&
        res.body.json.access_token != null &&
        res.body.json.access_token != "{{outputs.login.access_token}}"
      outputs:
        new_access_token: res.body.json.access_token

    # Step 5: Test with New Token
    - name: Test New Token
      uses: http
      with:
        url: "{{vars.api_base_url}}/user/profile"
        headers:
          Authorization: "Bearer {{outputs.refresh.new_access_token}}"
      test: res.code == 200
      outputs:
        new_token_valid: true

- name: Unauthorized Access Test
  steps:
    # Test without token
    - name: Test No Authorization
      uses: http
      with:
        url: "{{vars.api_base_url}}/user/profile"
      test: res.code == 401
      outputs:
        no_auth_rejected: res.code == 401

    # Test with invalid token
    - name: Test Invalid Token
      uses: http
      with:
        url: "{{vars.api_base_url}}/user/profile"
        headers:
          Authorization: "Bearer invalid_token_123"
      test: res.code == 401
      outputs:
        invalid_token_rejected: res.code == 401

    # Test with expired token (if available)
    - name: Test Expired Token
      if: vars.expired_token
      uses: http
      with:
        url: "{{vars.api_base_url}}/user/profile"
        headers:
          Authorization: "Bearer {{vars.expired_token}}"
      test: res.code == 401
      outputs:
        expired_token_rejected: res.code == 401

- name: Security Test Summary
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

### APIキー認証

```yaml
name: API Key Authentication Testing
description: Test APIs using API key authentication

vars:
  api_base_url: "{{API_BASE_URL ?? 'https://api.yourservice.com'}}"
  api_key: "{{API_KEY}}"
  rate_limit_tier: "{{RATE_LIMIT_TIER ?? 'premium'}}"

jobs:
- name: API Key Authentication Tests
  steps:
    # Header-based API Key
    - name: Test API Key in Header
      uses: http
      with:
        url: "{{vars.api_base_url}}/data"
        headers:
          X-API-Key: "{{vars.api_key}}"
          Accept: "application/json"
      test: |
        res.code == 200 &&
        res.body.json.authenticated == true &&
        res.body.json.rate_limit != null
      outputs:
        header_auth_success: true
        rate_limit_remaining: res.body.json.rate_limit.remaining
        rate_limit_reset: res.body.json.rate_limit.reset_time

    # Query Parameter API Key
    - name: Test API Key in Query
      uses: http
      with:
        url: "{{vars.api_base_url}}/data?api_key={{vars.api_key}}"
      test: res.code == 200
      outputs:
        query_auth_success: true

    # Test Rate Limiting
    - name: Test Rate Limit Info
      uses: http
      with:
        url: "{{vars.api_base_url}}/rate-limit-status"
        headers:
          X-API-Key: "{{vars.api_key}}"
      test: |
        res.code == 200 &&
        res.body.json.tier == "{{vars.rate_limit_tier}}" &&
        res.body.json.requests_remaining > 0
      outputs:
        requests_remaining: res.body.json.requests_remaining
        requests_per_hour: res.body.json.limits.per_hour
        current_usage: res.body.json.current_usage

    # Test Invalid API Key
    - name: Test Invalid API Key
      uses: http
      with:
        url: "{{vars.api_base_url}}/data"
        headers:
          X-API-Key: "invalid_key_123"
      test: res.code == 401 || res.code == 403
      outputs:
        invalid_key_rejected: res.code == 401 || res.code == 403

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

## データ検証とレスポンステスト

### JSONスキーマ検証

```yaml
name: JSON Response Validation
description: Validate API responses against expected schemas

vars:
  api_base_url: https://api.yourservice.com

jobs:
- name: Response Schema Validation
  steps:
    - name: Get User List
      id: users
      uses: http
      with:
        url: "{{vars.api_base_url}}/users"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: |
        res.code == 200 &&
        res.body.json != null &&
        res.body.json.data != null &&
        res.body.json.data.length > 0 &&
        
        # Validate response structure
        res.body.json.meta != null &&
        res.body.json.meta.total != null &&
        res.body.json.meta.page != null &&
        res.body.json.meta.per_page != null &&
        
        # Validate first user object
        res.body.json.data[0].id != null &&
        res.body.json.data[0].email != null &&
        res.body.json.data[0].name != null &&
        res.body.json.data[0].created_at != null &&
        
        # Validate data types
        typeof(res.body.json.data[0].id) == "number" &&
        typeof(res.body.json.data[0].email) == "string" &&
        typeof(res.body.json.data[0].active) == "boolean"
      outputs:
        total_users: res.body.json.meta.total
        users_per_page: res.body.json.meta.per_page
        first_user_id: res.body.json.data[0].id
        first_user_email: res.body.json.data[0].email

    - name: Get Single User Details
      uses: http
      with:
        url: "{{vars.api_base_url}}/users/{{outputs.users.first_user_id}}"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: |
        res.code == 200 &&
        res.body.json.user != null &&
        
        # Validate required fields
        res.body.json.user.id == outputs.users.first_user_id &&
        res.body.json.user.email == "{{outputs.users.first_user_email}}" &&
        res.body.json.user.profile != null &&
        
        # Validate nested objects
        res.body.json.user.profile.first_name != null &&
        res.body.json.user.profile.last_name != null &&
        res.body.json.user.preferences != null &&
        
        # Validate arrays
        res.body.json.user.roles != null &&
        res.body.json.user.roles.length > 0 &&
        res.body.json.user.permissions != null
      outputs:
        user_roles: res.body.json.user.roles
        user_permissions_count: res.body.json.user.permissions.length
        profile_complete: res.body.json.user.profile.first_name != null && res.body.json.user.profile.last_name != null

    - name: Test Data Consistency
      uses: http
      with:
        url: "{{vars.api_base_url}}/users/{{outputs.users.first_user_id}}/orders"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: |
        res.code == 200 &&
        res.body.json.orders != null &&
        
        # Validate all orders belong to the user
        res.body.json.orders.all(order -> order.user_id == outputs.users.first_user_id) &&
        
        # Validate order structure
        res.body.json.orders.all(order -> 
          order.id != null &&
          order.total != null &&
          order.status != null &&
          order.created_at != null
        ) &&
        
        # Validate business logic
        res.body.json.orders.all(order -> order.total >= 0) &&
        res.body.json.orders.filter(order -> order.status == "completed").all(order -> order.completed_at != null)
      outputs:
        order_count: res.body.json.orders.length
        completed_orders: res.body.json.orders.filter(order -> order.status == "completed").length
        total_spent: res.body.json.orders.filter(order -> order.status == "completed").map(order -> order.total).sum()

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

### エラーレスポンス検証

```yaml
name: Error Response Validation
description: Test error handling and response formats

vars:
  api_base_url: https://api.yourservice.com

jobs:
- name: Error Response Tests
  steps:
    # Test 400 - Bad Request
    - name: Test Bad Request
      uses: http
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
        res.code == 400 &&
        res.body.json.error != null &&
        res.body.json.error.code == "validation_error" &&
        res.body.json.error.message != null &&
        res.body.json.error.details != null &&
        res.body.json.error.details.length > 0
      outputs:
        validation_errors: res.body.json.error.details
        error_code: res.body.json.error.code

    # Test 401 - Unauthorized
    - name: Test Unauthorized Access
      uses: http
      with:
        url: "{{vars.api_base_url}}/admin/users"
      test: |
        res.code == 401 &&
        res.body.json.error != null &&
        res.body.json.error.code == "unauthorized" &&
        res.body.json.error.message.contains("authentication")
      outputs:
        auth_error_proper: true

    # Test 403 - Forbidden
    - name: Test Forbidden Access
      uses: http
      with:
        url: "{{vars.api_base_url}}/admin/users"
        headers:
          Authorization: "Bearer {{vars.USER_TOKEN}}"  # Non-admin token
      test: |
        res.code == 403 &&
        res.body.json.error != null &&
        res.body.json.error.code == "forbidden" &&
        res.body.json.error.message.contains("permission")
      outputs:
        permission_error_proper: true

    # Test 404 - Not Found
    - name: Test Not Found
      uses: http
      with:
        url: "{{vars.api_base_url}}/users/99999999"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: |
        res.code == 404 &&
        res.body.json.error != null &&
        res.body.json.error.code == "not_found" &&
        res.body.json.error.resource == "user"
      outputs:
        not_found_error_proper: true

    # Test 422 - Unprocessable Entity
    - name: Test Unprocessable Entity
      uses: http
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
        res.code == 422 &&
        res.body.json.error != null &&
        res.body.json.error.code == "unprocessable_entity" &&
        res.body.json.error.details != null
      outputs:
        duplicate_email_handled: true

    # Test Rate Limiting - 429
    - name: Test Rate Limiting
      uses: http
      with:
        url: "{{vars.api_base_url}}/high-rate-endpoint"
        headers:
          Authorization: "Bearer {{vars.LIMITED_TOKEN}}"
      test: |
        res.code in [200, 429] &&
        (res.code == 429 ? 
          res.body.json.error.code == "rate_limit_exceeded" &&
          res.headers["retry-after"] != null :
          true
        )
      continue_on_error: true
      outputs:
        rate_limiting_works: res.code == 429

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

## 高度なAPIテストパターン

### ワークフローテスト

```yaml
name: E-commerce Workflow Testing
description: Test complete e-commerce user workflow

vars:
  api_base_url: https://api.ecommerce.com
  test_product_id: 123
  test_user_email: test@example.com

jobs:
- name: User Registration Workflow
  steps:
    - name: Register New User
      id: register
      uses: http
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
        res.code == 201 &&
        res.body.json.user.id != null &&
        res.body.json.user.email != null &&
        res.body.json.access_token != null
      outputs:
        user_id: res.body.json.user.id
        user_email: res.body.json.user.email
        access_token: res.body.json.access_token

    - name: Email Verification Check
      uses: http
      with:
        url: "{{vars.api_base_url}}/user/profile"
        headers:
          Authorization: "Bearer {{outputs.register.access_token}}"
      test: |
        res.code == 200 &&
        res.body.json.email_verified == false &&
        res.body.json.verification_email_sent == true
      outputs:
        verification_pending: true

- name: Shopping Workflow
  needs: [user-registration-workflow]
  steps:
    - name: Browse Products
      id: browse
      uses: http
      with:
        url: "{{vars.api_base_url}}/products?category=electronics&limit=10"
      test: |
        res.code == 200 &&
        res.body.json.products.length > 0 &&
        res.body.json.products[0].id != null &&
        res.body.json.products[0].price > 0
      outputs:
        available_products: res.body.json.products.length
        first_product_id: res.body.json.products[0].id
        first_product_price: res.body.json.products[0].price

    - name: Add to Cart
      id: add-cart
      uses: http
      with:
        url: "{{vars.api_base_url}}/cart/items"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{outputs.user-registration-workflow.access_token}}"
        body: |
          {
            "product_id": {{outputs.browse.first_product_id}},
            "quantity": 2
          }
      test: |
        res.code == 201 &&
        res.body.json.cart_item.id != null &&
        res.body.json.cart_item.quantity == 2 &&
        res.body.json.cart_total > 0
      outputs:
        cart_item_id: res.body.json.cart_item.id
        cart_total: res.body.json.cart_total

    - name: View Cart
      uses: http
      with:
        url: "{{vars.api_base_url}}/cart"
        headers:
          Authorization: "Bearer {{outputs.user-registration-workflow.access_token}}"
      test: |
        res.code == 200 &&
        res.body.json.items.length == 1 &&
        res.body.json.items[0].product_id == outputs.browse.first_product_id &&
        res.body.json.total == outputs.add-cart.cart_total
      outputs:
        cart_verified: true

- name: Checkout Workflow
  needs: [shopping-workflow]
  steps:
    - name: Apply Discount Code
      id: discount
      uses: http
      with:
        url: "{{vars.api_base_url}}/cart/discount"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{outputs.user-registration-workflow.access_token}}"
        body: |
          {
            "code": "TESTDISCOUNT10"
          }
      test: |
        res.code == 200 &&
        res.body.json.discount_applied == true &&
        res.body.json.discount_amount > 0 &&
        res.body.json.new_total < outputs.shopping-workflow.cart_total
      continue_on_error: true
      outputs:
        discount_applied: res.code == 200
        discount_amount: res.body.json.discount_amount
        final_total: res.body.json.new_total

    - name: Add Payment Method
      id: payment
      uses: http
      with:
        url: "{{vars.api_base_url}}/payment-methods"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{outputs.user-registration-workflow.access_token}}"
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
        res.code == 201 &&
        res.body.json.payment_method.id != null &&
        res.body.json.payment_method.last_four == "1111"
      outputs:
        payment_method_id: res.body.json.payment_method.id

    - name: Create Order
      id: order
      uses: http
      with:
        url: "{{vars.api_base_url}}/orders"
        method: POST
        headers:
          Content-Type: "application/json"
          Authorization: "Bearer {{outputs.user-registration-workflow.access_token}}"
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
        res.code == 201 &&
        res.body.json.order.id != null &&
        res.body.json.order.status == "processing" &&
        res.body.json.order.total > 0
      outputs:
        order_id: res.body.json.order.id
        order_status: res.body.json.order.status
        order_total: res.body.json.order.total

    - name: Verify Order Processing
      uses: http
      with:
        url: "{{vars.api_base_url}}/orders/{{outputs.order.order_id}}"
        headers:
          Authorization: "Bearer {{outputs.user-registration-workflow.access_token}}"
      test: |
        res.code == 200 &&
        res.body.json.order.id == outputs.order.order_id &&
        res.body.json.order.user_id == outputs.user-registration-workflow.user_id &&
        res.body.json.order.items.length > 0
      outputs:
        order_verified: true

- name: Workflow Test Summary
  needs: [user-registration-workflow, shopping-workflow, checkout-workflow]
  steps:
    - name: Complete Workflow Results
      echo: |
        🛒 E-commerce Workflow Test Results:
        =====================================
        
        USER REGISTRATION:
        ✅ User Created: {{outputs.user-registration-workflow.user_email}}
        ✅ Authentication: Token obtained
        ✅ Email Verification: {{outputs.user-registration-workflow.verification_pending ? "Pending (as expected)" : "Issue detected"}}
        
        SHOPPING EXPERIENCE:
        ✅ Product Browse: {{outputs.shopping-workflow.available_products}} products found
        ✅ Add to Cart: Product ID {{outputs.shopping-workflow.first_product_id}} added
        ✅ Cart Verification: {{outputs.shopping-workflow.cart_verified ? "Confirmed" : "Failed"}}
        Cart Total: ${{outputs.shopping-workflow.cart_total}}
        
        CHECKOUT PROCESS:
        {{outputs.checkout-workflow.discount_applied ? "✅ Discount Applied: $" + outputs.checkout-workflow.discount_amount + " off" : "ℹ️ No discount applied"}}
        ✅ Payment Method: Added (ending in 1111)
        ✅ Order Created: Order ID {{outputs.checkout-workflow.order_id}}
        ✅ Order Status: {{outputs.checkout-workflow.order_status}}
        ✅ Order Verified: {{outputs.checkout-workflow.order_verified ? "Confirmed" : "Failed"}}
        Final Total: ${{outputs.checkout-workflow.order_total}}
        
        🎉 Complete e-commerce workflow tested successfully!
        User can register, browse, shop, and checkout without issues.
```

## パフォーマンスと負荷テスト

### レスポンス時間テスト

```yaml
name: API Performance Testing
description: Test API response times and performance characteristics

vars:
  api_base_url: https://api.yourservice.com
  performance_threshold_ms: 1000
  acceptable_threshold_ms: 2000

jobs:
- name: Response Time Performance Tests
  steps:
    - name: Lightweight Endpoint Test
      id: ping
      uses: http
      with:
        url: "{{vars.api_base_url}}/ping"
      test: |
        res.code == 200 &&
        res.time < 500
      outputs:
        ping_time: res.time
        ping_fast: res.time < 200

    - name: Database Query Test
      id: db-query
      uses: http
      with:
        url: "{{vars.api_base_url}}/users?limit=100"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: |
        res.code == 200 &&
        res.time < {{vars.performance_threshold_ms}}
      outputs:
        query_time: res.time
        query_performance: |
          {{res.time < 500 ? "excellent" : 
            res.time < 1000 ? "good" : 
            res.time < 2000 ? "acceptable" : "poor"}}

    - name: Complex Aggregation Test
      id: aggregation
      uses: http
      with:
        url: "{{vars.api_base_url}}/analytics/summary"
        headers:
          Authorization: "Bearer {{vars.api_token}}"
      test: |
        res.code == 200 &&
        res.time < {{vars.acceptable_threshold_ms}}
      outputs:
        aggregation_time: res.time
        aggregation_acceptable: res.time < {{vars.acceptable_threshold_ms}}

    - name: File Upload Test
      id: upload
      uses: http
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
        res.code == 201 &&
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

## ベストプラクティス

### 1. テスト構造

```yaml
# 良い例: 整理されたテスト構造
jobs:
- name: authentication     # 関連テストをグループ化
- name: crud-operations    # 明確なテストカテゴリ
- name: error-handling     # 論理的な整理
- name: performance        # 関心事の分離
```

### 2. データ管理

```yaml
# 良い例: 分離のためにランダムデータを使用
body: |
  {
    "email": "test{{random_str(8)}}@example.com",
    "username": "user_{{unixtime()}}_{{random_str(4)}}"
  }

# 良い例: テストデータのクリーンアップ
- name: Cleanup Test User
  uses: http
  with:
    url: "{{vars.api_base_url}}/users/{{outputs.create.user_id}}"
    method: DELETE
```

### 3. 包括的な検証

```yaml
# 良い例: 複数の側面を検証
test: |
  res.code == 200 &&                           # HTTPステータス
  res.headers["content-type"].contains("json") &&  # コンテンツタイプ
  res.body.json.id != null &&                  # 必須フィールド
  typeof(res.body.json.id) == "number" &&      # データタイプ
  res.body.json.email.contains("@") &&         # データフォーマット
  res.time < 1000                              # パフォーマンス
```

### 4. エラーハンドリング

```yaml
# 良い例: エラーシナリオをテスト
- name: Test Invalid Input
  uses: http
  with:
    body: '{"invalid": "data"}'
  test: res.code == 400
  continue_on_error: true

# 良い例: エラーレスポンスを検証
test: |
  res.code == 400 &&
  res.body.json.error.code == "validation_error" &&
  res.body.json.error.details.length > 0
```

## 次のステップ

APIを包括的にテストできるようになったので、次を探索してください：

- **[エラーハンドリング戦略](../error-handling-strategies/)** - 堅牢なエラーハンドリングパターン
- **[パフォーマンステスト](../performance-testing/)** - 高度な負荷テスト技術
- **[環境管理](../environment-management/)** - テスト設定の管理

APIテストは信頼性の高いサービスに不可欠です。これらのパターンを使用して、問題がプロダクションに到達する前に問題をキャッチする包括的なテストスイートを構築しましょう。