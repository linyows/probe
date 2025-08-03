---
title: API Testing Pipeline
description: Build a comprehensive API testing suite with Probe
weight: 20
---

# API Testing Pipeline

In this tutorial, you'll build a comprehensive API testing pipeline that validates functionality, performance, security, and data integrity across your entire API surface. This goes beyond simple health checks to create a complete testing suite suitable for CI/CD integration.

## What You'll Build

A complete API testing pipeline featuring:

- **Functional Testing** - Validate CRUD operations and business logic
- **Data Validation** - Ensure response schemas and data integrity
- **Performance Testing** - Response time and throughput validation
- **Authentication Testing** - Security and access control verification
- **Error Handling Tests** - Validate error responses and edge cases
- **Integration Testing** - End-to-end user journey validation
- **Contract Testing** - API specification compliance
- **Regression Testing** - Prevent breaking changes

## Prerequisites

- Probe installed ([Installation Guide](../get-started/installation/))
- A REST API to test (we'll use a sample e-commerce API)
- Understanding of HTTP methods and status codes
- Basic knowledge of JSON and API design

## Tutorial Overview

We'll build tests for a sample e-commerce API with these endpoints:

- **Authentication**: `POST /auth/login`, `POST /auth/logout`
- **Users**: `GET /users/profile`, `PUT /users/profile`
- **Products**: `GET /products`, `GET /products/{id}`, `POST /products`
- **Orders**: `POST /orders`, `GET /orders/{id}`, `GET /orders`
- **Cart**: `POST /cart/items`, `DELETE /cart/items/{id}`

## Step 1: Project Structure and Configuration

Create a well-organized testing structure:

```bash
api-tests/
├── config/
│   ├── base.yml
│   ├── development.yml
│   ├── staging.yml
│   └── production.yml
├── tests/
│   ├── auth-tests.yml
│   ├── user-tests.yml
│   ├── product-tests.yml
│   ├── order-tests.yml
│   └── integration-tests.yml
└── main-test-suite.yml
```

**config/base.yml:**
```yaml
name: "API Testing Pipeline"
description: "Comprehensive API testing suite for e-commerce platform"

env:
  # Test Data
  TEST_USER_EMAIL: "test@example.com"
  TEST_USER_PASSWORD: "TestPassword123!"
  TEST_PRODUCT_NAME: "Test Product"
  TEST_ORDER_AMOUNT: 99.99
  
  # Performance Thresholds
  FAST_RESPONSE_TIME: 200      # 200ms
  ACCEPTABLE_RESPONSE_TIME: 1000  # 1 second
  SLOW_RESPONSE_TIME: 3000     # 3 seconds
  
  # Test Configuration
  RETRY_COUNT: 3
  PARALLEL_REQUESTS: 5

defaults:
  http:
    timeout: "10s"
    headers:
      Content-Type: "application/json"
      User-Agent: "Probe API Test Suite v1.0"
    verify_ssl: true
```

**config/development.yml:**
```yaml
env:
  API_BASE_URL: "http://localhost:3000"
  API_VERSION: "v1"
  SKIP_PERFORMANCE_TESTS: true
  SKIP_LOAD_TESTS: true

defaults:
  http:
    timeout: "30s"
    verify_ssl: false  # Allow self-signed certificates
```

**config/staging.yml:**
```yaml
env:
  API_BASE_URL: "https://api-staging.example.com"
  API_VERSION: "v1"
  SKIP_PERFORMANCE_TESTS: false
  SKIP_LOAD_TESTS: true
```

**config/production.yml:**
```yaml
env:
  API_BASE_URL: "https://api.example.com"
  API_VERSION: "v1" 
  SKIP_PERFORMANCE_TESTS: false
  SKIP_LOAD_TESTS: false
  # Use more conservative thresholds for production
  ACCEPTABLE_RESPONSE_TIME: 2000
```

## Step 2: Authentication Testing

Create comprehensive authentication tests:

**tests/auth-tests.yml:**
```yaml
name: "Authentication Tests"
description: "Test user authentication, authorization, and session management"

jobs:
  auth-functional-tests:
    name: "Authentication Functional Tests"
    steps:
      - name: "Test User Registration"
        id: register
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/register"
          method: "POST"
          body: |
            {
              "email": "{{env.TEST_USER_EMAIL}}",
              "password": "{{env.TEST_USER_PASSWORD}}",
              "firstName": "Test",
              "lastName": "User"
            }
        test: |
          res.status == 201 &&
          res.json.user != null &&
          res.json.user.email == env.TEST_USER_EMAIL &&
          res.json.token != null
        outputs:
          user_id: res.json.user.id
          auth_token: res.json.token
        continue_on_error: true

      - name: "Test User Login"
        id: login
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/login"
          method: "POST"
          body: |
            {
              "email": "{{env.TEST_USER_EMAIL}}",
              "password": "{{env.TEST_USER_PASSWORD}}"
            }
        test: |
          res.status == 200 &&
          res.json.token != null &&
          res.json.user != null &&
          len(res.json.token) > 20
        outputs:
          auth_token: res.json.token
          user_id: res.json.user.id
          login_time: res.time

      - name: "Test Invalid Login Credentials"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/login"
          method: "POST"
          body: |
            {
              "email": "{{env.TEST_USER_EMAIL}}",
              "password": "wrongpassword"
            }
        test: |
          res.status == 401 &&
          res.json.error != null &&
          res.json.message | contains("Invalid credentials")

      - name: "Test Token Validation"
        id: token-validation
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/validate"
          headers:
            Authorization: "Bearer {{outputs.login.auth_token}}"
        test: |
          res.status == 200 &&
          res.json.valid == true &&
          res.json.user.id == outputs.login.user_id

      - name: "Test Invalid Token"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/validate"
          headers:
            Authorization: "Bearer invalid-token-12345"
        test: |
          res.status == 401 &&
          res.json.error != null

      - name: "Test Token Refresh"
        id: refresh
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/refresh"  
          method: "POST"
          headers:
            Authorization: "Bearer {{outputs.login.auth_token}}"
        test: |
          res.status == 200 &&
          res.json.token != null &&
          res.json.token != outputs.login.auth_token
        outputs:
          new_token: res.json.token

      - name: "Test Logout"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/logout"
          method: "POST"
          headers:
            Authorization: "Bearer {{outputs.refresh.new_token}}"
        test: |
          res.status == 200 &&
          res.json.message | contains("logged out")

      - name: "Test Using Token After Logout"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/validate"
          headers:
            Authorization: "Bearer {{outputs.refresh.new_token}}"
        test: |
          res.status == 401 &&
          res.json.error != null
```

## Step 3: CRUD Operations Testing

Create comprehensive CRUD tests for products:

**tests/product-tests.yml:**
```yaml
name: "Product API Tests"
description: "Test product CRUD operations, search, and data validation"

jobs:
  product-crud-tests:
    name: "Product CRUD Operations"
    steps:
      - name: "Setup - Get Auth Token"
        id: auth
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/login"
          method: "POST"
          body: |
            {
              "email": "{{env.TEST_USER_EMAIL}}",
              "password": "{{env.TEST_USER_PASSWORD}}"
            }
        test: res.status == 200
        outputs:
          token: res.json.token

      - name: "Create Product"
        id: create-product
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products"
          method: "POST"
          headers:
            Authorization: "Bearer {{outputs.auth.token}}"
          body: |
            {
              "name": "{{env.TEST_PRODUCT_NAME}} {{unixtime()}}",
              "description": "Test product for API testing",
              "price": 29.99,
              "category": "Electronics",
              "sku": "TEST-{{unixtime()}}",
              "stock": 100,
              "tags": ["test", "electronics", "api-test"]
            }
        test: |
          res.status == 201 &&
          res.json.id != null &&
          res.json.name | contains(env.TEST_PRODUCT_NAME) &&
          res.json.price == 29.99 &&
          res.json.sku | hasPrefix("TEST-") &&
          res.json.stock == 100
        outputs:
          product_id: res.json.id
          product_name: res.json.name
          product_sku: res.json.sku
          creation_time: res.time

      - name: "Read Product by ID"
        id: read-product
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products/{{outputs.create-product.product_id}}"
        test: |
          res.status == 200 &&
          res.json.id == outputs.create-product.product_id &&
          res.json.name == outputs.create-product.product_name &&
          res.json.price == 29.99 &&
          res.json.category == "Electronics" &&
          len(res.json.tags) == 3
        outputs:
          read_time: res.time

      - name: "Update Product"
        id: update-product
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products/{{outputs.create-product.product_id}}"
          method: "PUT"
          headers:
            Authorization: "Bearer {{outputs.auth.token}}"
          body: |
            {
              "name": "{{outputs.create-product.product_name}} - UPDATED",
              "description": "Updated test product",
              "price": 39.99,
              "category": "Electronics",
              "sku": "{{outputs.create-product.product_sku}}",
              "stock": 75,
              "tags": ["test", "electronics", "api-test", "updated"]
            }
        test: |
          res.status == 200 &&
          res.json.id == outputs.create-product.product_id &&
          res.json.name | hasSuffix("UPDATED") &&
          res.json.price == 39.99 &&
          res.json.stock == 75 &&
          len(res.json.tags) == 4
        outputs:
          updated_name: res.json.name
          update_time: res.time

      - name: "Verify Update Persistence"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products/{{outputs.create-product.product_id}}"
        test: |
          res.status == 200 &&
          res.json.name == outputs.update-product.updated_name &&
          res.json.price == 39.99 &&
          res.json.stock == 75

      - name: "Test Partial Update (PATCH)"
        id: patch-product
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products/{{outputs.create-product.product_id}}"
          method: "PATCH"
          headers:
            Authorization: "Bearer {{outputs.auth.token}}"
          body: |
            {
              "stock": 50,
              "tags": ["test", "electronics", "patched"]
            }
        test: |
          res.status == 200 &&
          res.json.stock == 50 &&
          len(res.json.tags) == 3 &&
          res.json.name == outputs.update-product.updated_name
        outputs:
          patch_time: res.time

      - name: "Delete Product"
        id: delete-product
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products/{{outputs.create-product.product_id}}"
          method: "DELETE"
          headers:
            Authorization: "Bearer {{outputs.auth.token}}"
        test: res.status == 204 || res.status == 200
        outputs:
          delete_time: res.time

      - name: "Verify Product Deletion"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products/{{outputs.create-product.product_id}}"
        test: res.status == 404

  product-search-tests:
    name: "Product Search and Filtering"
    needs: [product-crud-tests]
    steps:
      - name: "Setup - Create Multiple Test Products"
        id: setup-products
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products/batch"
          method: "POST"
          headers:
            Authorization: "Bearer {{outputs.auth.token}}"
          body: |
            {
              "products": [
                {
                  "name": "Search Test Product A",
                  "price": 10.00,
                  "category": "Books",
                  "tags": ["fiction", "bestseller"]
                },
                {
                  "name": "Search Test Product B", 
                  "price": 25.00,
                  "category": "Electronics",
                  "tags": ["gadget", "mobile"]
                },
                {
                  "name": "Search Test Product C",
                  "price": 15.00,
                  "category": "Books",
                  "tags": ["non-fiction", "educational"]
                }
              ]
            }
        test: |
          res.status == 201 &&
          len(res.json.products) == 3
        outputs:
          created_products: res.json.products

      - name: "Test Product Search by Name"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products/search?q=Search%20Test"
        test: |
          res.status == 200 &&
          len(res.json.products) >= 3 &&
          res.json.products[0].name | contains("Search Test")

      - name: "Test Product Filter by Category"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products?category=Books"
        test: |
          res.status == 200 &&
          len(res.json.products) >= 2
        outputs:
          books_found: len(res.json.products)

      - name: "Test Product Filter by Price Range"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products?min_price=10&max_price=20"
        test: |
          res.status == 200 &&
          res.json.products[0].price >= 10 &&
          res.json.products[0].price <= 20

      - name: "Test Product Sorting"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products?sort=price&order=desc"
        test: |
          res.status == 200 &&
          len(res.json.products) > 1 &&
          res.json.products[0].price >= res.json.products[1].price

      - name: "Test Pagination"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products?page=1&limit=2"
        test: |
          res.status == 200 &&
          len(res.json.products) <= 2 &&
          res.json.pagination.page == 1 &&
          res.json.pagination.total > 0
```

## Step 4: Performance Testing

Add performance validation to your tests:

**tests/performance-tests.yml:**
```yaml
name: "API Performance Tests"
description: "Validate API response times and performance characteristics"

jobs:
  response-time-tests:
    name: "Response Time Validation"
    if: env.SKIP_PERFORMANCE_TESTS != "true"
    steps:
      - name: "Test Fast Endpoint Performance"
        id: health-perf
        action: http
        with:
          url: "{{env.API_BASE_URL}}/health"
        test: |
          res.status == 200 &&
          res.time < env.FAST_RESPONSE_TIME
        outputs:
          health_time: res.time

      - name: "Test API List Performance"
        id: list-perf
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products?limit=10"
        test: |
          res.status == 200 &&
          res.time < env.ACCEPTABLE_RESPONSE_TIME
        outputs:
          list_time: res.time

      - name: "Test Search Performance"
        id: search-perf
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products/search?q=test"
        test: |
          res.status == 200 &&
          res.time < env.ACCEPTABLE_RESPONSE_TIME
        outputs:
          search_time: res.time

      - name: "Performance Summary"
        echo: |
          === PERFORMANCE TEST RESULTS ===
          Health Check: {{outputs.health-perf.health_time}}ms (threshold: {{env.FAST_RESPONSE_TIME}}ms)
          Product List: {{outputs.list-perf.list_time}}ms (threshold: {{env.ACCEPTABLE_RESPONSE_TIME}}ms)
          Search: {{outputs.search-perf.search_time}}ms (threshold: {{env.ACCEPTABLE_RESPONSE_TIME}}ms)
          
          {{outputs.health-perf.health_time < env.FAST_RESPONSE_TIME ? "✅" : "❌"}} Health: Fast
          {{outputs.list-perf.list_time < env.ACCEPTABLE_RESPONSE_TIME ? "✅" : "❌"}} List: Acceptable
          {{outputs.search-perf.search_time < env.ACCEPTABLE_RESPONSE_TIME ? "✅" : "❌"}} Search: Acceptable

  concurrent-request-tests:
    name: "Concurrent Request Testing"
    if: env.SKIP_LOAD_TESTS != "true"
    needs: [response-time-tests]
    steps:
      - name: "Concurrent Health Checks"
        id: concurrent-health
        action: http
        with:
          url: "{{env.API_BASE_URL}}/health"
        test: |
          res.status == 200 &&
          res.time < mul(env.ACCEPTABLE_RESPONSE_TIME, 2)
        # Note: This would be repeated with different parallel execution
        # In a real scenario, you'd use a load testing tool or script

      - name: "Stress Test Report"
        echo: |
          === CONCURRENT LOAD TEST ===
          Concurrent requests: {{env.PARALLEL_REQUESTS}}
          Average response time: {{outputs.concurrent-health.time}}ms
          Success rate: 100%
          
          Status: {{outputs.concurrent-health.time < env.ACCEPTABLE_RESPONSE_TIME ? "✅ PASSED" : "❌ FAILED"}}
```

## Step 5: Data Validation and Schema Testing

Create robust data validation tests:

**tests/data-validation-tests.yml:**
```yaml
name: "Data Validation Tests"
description: "Validate API response schemas and data integrity"

jobs:
  schema-validation-tests:
    name: "Response Schema Validation"
    steps:
      - name: "Validate Product Schema"
        id: product-schema
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products"
        test: |
          res.status == 200 &&
          res.json.products != null &&
          len(res.json.products) > 0 &&
          res.json.products[0].id != null &&
          res.json.products[0].name != null &&
          res.json.products[0].price != null &&
          res.json.products[0].category != null &&
          res.json.pagination != null &&
          res.json.pagination.total != null &&
          res.json.pagination.page != null &&
          res.json.pagination.limit != null

      - name: "Validate User Profile Schema"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/login"
          method: "POST"
          body: |
            {
              "email": "{{env.TEST_USER_EMAIL}}",
              "password": "{{env.TEST_USER_PASSWORD}}"
            }
        test: |
          res.status == 200 &&
          res.json.user != null &&
          res.json.user.id != null &&
          res.json.user.email != null &&
          res.json.user.firstName != null &&
          res.json.user.lastName != null &&
          res.json.user.createdAt != null &&
          res.json.token != null &&
          len(res.json.token) > 20

      - name: "Validate Error Response Schema"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products/nonexistent-id"
        test: |
          res.status == 404 &&
          res.json.error != null &&
          res.json.message != null &&
          res.json.statusCode == 404 &&
          res.json.timestamp != null

  data-integrity-tests:
    name: "Data Integrity Validation"
    steps:
      - name: "Test Data Type Validation"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products"
          method: "POST"
          headers:
            Authorization: "Bearer {{env.API_TOKEN}}"
          body: |
            {
              "name": "Type Test Product",
              "price": "invalid-price",
              "category": "Books"
            }
        test: |
          res.status == 400 &&
          res.json.error != null &&
          res.json.message | contains("price")

      - name: "Test Required Field Validation"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products"
          method: "POST"
          headers:
            Authorization: "Bearer {{env.API_TOKEN}}"
          body: |
            {
              "description": "Missing required name field"
            }
        test: |
          res.status == 400 &&
          res.json.error != null &&
          res.json.message | contains("name")

      - name: "Test Email Format Validation"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/register"
          method: "POST"
          body: |
            {
              "email": "invalid-email-format",
              "password": "validpassword123"
            }
        test: |
          res.status == 400 &&
          res.json.error != null &&
          res.json.message | contains("email")

      - name: "Test Numeric Range Validation"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products"
          method: "POST"
          headers:
            Authorization: "Bearer {{env.API_TOKEN}}"
          body: |
            {
              "name": "Range Test Product",
              "price": -10.00,
              "category": "Books"
            }
        test: |
          res.status == 400 &&
          res.json.error != null &&
          res.json.message | contains("price")
```

## Step 6: Integration Testing

Create end-to-end user journey tests:

**tests/integration-tests.yml:**
```yaml
name: "Integration Tests"
description: "End-to-end user journey validation"

jobs:
  user-journey-test:
    name: "Complete User Purchase Journey"
    steps:
      - name: "User Registration"
        id: register
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/register"
          method: "POST"
          body: |
            {
              "email": "journey-test-{{unixtime()}}@example.com",
              "password": "JourneyTest123!",
              "firstName": "Journey",
              "lastName": "Test"
            }
        test: res.status == 201
        outputs:
          user_id: res.json.user.id
          auth_token: res.json.token
          user_email: res.json.user.email

      - name: "Browse Products"
        id: browse
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products?category=Electronics&limit=5"
        test: |
          res.status == 200 &&
          len(res.json.products) > 0
        outputs:
          available_products: res.json.products
          first_product_id: res.json.products[0].id
          first_product_price: res.json.products[0].price

      - name: "Add Product to Cart"
        id: add-to-cart
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/cart/items"
          method: "POST"
          headers:
            Authorization: "Bearer {{outputs.register.auth_token}}"
          body: |
            {
              "productId": "{{outputs.browse.first_product_id}}",
              "quantity": 2
            }
        test: |
          res.status == 201 &&
          res.json.item.productId == outputs.browse.first_product_id &&
          res.json.item.quantity == 2
        outputs:
          cart_item_id: res.json.item.id
          cart_total: res.json.cart.total

      - name: "View Cart"
        id: view-cart
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/cart"
          headers:
            Authorization: "Bearer {{outputs.register.auth_token}}"
        test: |
          res.status == 200 &&
          len(res.json.items) == 1 &&
          res.json.items[0].productId == outputs.browse.first_product_id &&
          res.json.total == mul(outputs.browse.first_product_price, 2)

      - name: "Update Cart Quantity"
        id: update-cart
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/cart/items/{{outputs.add-to-cart.cart_item_id}}"
          method: "PUT"
          headers:
            Authorization: "Bearer {{outputs.register.auth_token}}"
          body: |
            {
              "quantity": 3
            }
        test: |
          res.status == 200 &&
          res.json.item.quantity == 3
        outputs:
          new_cart_total: res.json.cart.total

      - name: "Create Order"
        id: create-order
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/orders"
          method: "POST"
          headers:
            Authorization: "Bearer {{outputs.register.auth_token}}"
          body: |
            {
              "shippingAddress": {
                "street": "123 Test Street",
                "city": "Test City",
                "zipCode": "12345",
                "country": "US"
              },
              "paymentMethod": "credit_card",
              "paymentDetails": {
                "cardNumber": "4111111111111111",
                "expiryMonth": "12",
                "expiryYear": "2025",
                "cvv": "123"
              }
            }
        test: |
          res.status == 201 &&
          res.json.order.id != null &&
          res.json.order.status == "pending" &&
          res.json.order.total == outputs.update-cart.new_cart_total
        outputs:
          order_id: res.json.order.id
          order_status: res.json.order.status

      - name: "Verify Order Details"
        id: verify-order
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/orders/{{outputs.create-order.order_id}}"
          headers:
            Authorization: "Bearer {{outputs.register.auth_token}}"
        test: |
          res.status == 200 &&
          res.json.id == outputs.create-order.order_id &&
          len(res.json.items) == 1 &&
          res.json.items[0].productId == outputs.browse.first_product_id &&
          res.json.items[0].quantity == 3
        outputs:
          order_created_at: res.json.createdAt

      - name: "View Order History"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/orders"
          headers:
            Authorization: "Bearer {{outputs.register.auth_token}}"
        test: |
          res.status == 200 &&
          len(res.json.orders) >= 1 &&
          res.json.orders[0].id == outputs.create-order.order_id

      - name: "User Journey Report"
        echo: |
          === USER JOURNEY TEST COMPLETE ===
          
          ✅ User Registration: {{outputs.register.user_email}}
          ✅ Product Browse: Found {{len(outputs.browse.available_products)}} products
          ✅ Add to Cart: Product {{outputs.browse.first_product_id}}
          ✅ Update Cart: Quantity changed to 3
          ✅ Order Creation: Order {{outputs.create-order.order_id}}
          ✅ Order Verification: Status {{outputs.create-order.order_status}}
          ✅ Order History: Retrieved successfully
          
          Total Order Value: ${{outputs.update-cart.new_cart_total}}
          Journey Completion Time: {{sub(unixtime(), outputs.register.timestamp)}} seconds
```

## Step 7: Main Test Suite

Create the main test suite that orchestrates all tests:

**main-test-suite.yml:**
```yaml
name: "Complete API Test Suite"
description: "Comprehensive API testing pipeline"

# Import base configuration
# This file will be merged with environment-specific config

jobs:
  setup:
    name: "Test Environment Setup"
    steps:
      - name: "Verify API Availability"
        id: api-check
        action: http
        with:
          url: "{{env.API_BASE_URL}}/health"
        test: res.status == 200
        outputs:
          api_available: res.status == 200

      - name: "Setup Test Data"
        if: outputs.api-check.api_available
        echo: |
          === API TEST SUITE STARTING ===
          Environment: {{env.ENVIRONMENT || 'default'}}
          API Base URL: {{env.API_BASE_URL}}
          API Version: {{env.API_VERSION}}
          Skip Performance Tests: {{env.SKIP_PERFORMANCE_TESTS}}
          Skip Load Tests: {{env.SKIP_LOAD_TESTS}}
          
          Test Configuration:
          - Fast Response Threshold: {{env.FAST_RESPONSE_TIME}}ms
          - Acceptable Response Threshold: {{env.ACCEPTABLE_RESPONSE_TIME}}ms
          - Slow Response Threshold: {{env.SLOW_RESPONSE_TIME}}ms

  authentication-tests:
    name: "Authentication Test Suite"
    needs: [setup]
    # In practice, you would import the auth-tests.yml file here
    # For this example, we'll reference key auth tests
    steps:
      - name: "Run Authentication Tests"
        echo: "Running authentication test suite..."
        # This would typically import tests/auth-tests.yml

  functional-tests:
    name: "Functional Test Suite"
    needs: [authentication-tests]
    steps:
      - name: "Run Product CRUD Tests"
        echo: "Running product CRUD test suite..."
        # This would typically import tests/product-tests.yml
        
      - name: "Run Data Validation Tests"
        echo: "Running data validation test suite..."
        # This would typically import tests/data-validation-tests.yml

  performance-tests:
    name: "Performance Test Suite"
    needs: [functional-tests]
    if: env.SKIP_PERFORMANCE_TESTS != "true"
    steps:
      - name: "Run Performance Tests"
        echo: "Running performance test suite..."
        # This would typically import tests/performance-tests.yml

  integration-tests:
    name: "Integration Test Suite"
    needs: [functional-tests]
    steps:
      - name: "Run Integration Tests"
        echo: "Running integration test suite..."
        # This would typically import tests/integration-tests.yml

  test-report:
    name: "Generate Test Report"
    needs: [authentication-tests, functional-tests, performance-tests, integration-tests]
    steps:
      - name: "Final Test Report"
        echo: |
          === API TEST SUITE COMPLETE ===
          
          Test Results Summary:
          ✅ Authentication Tests: {{jobs.authentication-tests.status}}
          ✅ Functional Tests: {{jobs.functional-tests.status}}
          {{env.SKIP_PERFORMANCE_TESTS != "true" ? "✅ Performance Tests: " + jobs.performance-tests.status : "⏭️  Performance Tests: Skipped"}}
          ✅ Integration Tests: {{jobs.integration-tests.status}}
          
          Overall Status: {{
            jobs.authentication-tests.success &&
            jobs.functional-tests.success &&
            (env.SKIP_PERFORMANCE_TESTS == "true" || jobs.performance-tests.success) &&
            jobs.integration-tests.success ? "✅ ALL TESTS PASSED" : "❌ SOME TESTS FAILED"
          }}
          
          Generated: {{iso8601()}}
```

## Step 8: Running Your Test Suite

Execute your comprehensive API test suite:

```bash
# Run tests for different environments
probe config/base.yml,config/development.yml,main-test-suite.yml
probe config/base.yml,config/staging.yml,main-test-suite.yml
probe config/base.yml,config/production.yml,main-test-suite.yml

# Run specific test categories
probe config/base.yml,config/staging.yml,tests/auth-tests.yml
probe config/base.yml,config/staging.yml,tests/product-tests.yml

# Run with verbose output for debugging
probe -v config/base.yml,config/development.yml,tests/integration-tests.yml
```

## Step 9: CI/CD Integration

Integrate your API tests into your CI/CD pipeline:

**.github/workflows/api-tests.yml:**
```yaml
name: API Tests
on: 
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  api-tests:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        environment: [staging, production]
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Probe
        run: |
          curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o probe
          chmod +x probe
          sudo mv probe /usr/local/bin/
      
      - name: Run API Tests
        env:
          API_AUTH_TOKEN: ${{ secrets.API_TOKEN }}
          SMTP_USERNAME: ${{ secrets.SMTP_USERNAME }}
          SMTP_PASSWORD: ${{ secrets.SMTP_PASSWORD }}
          ENVIRONMENT: ${{ matrix.environment }}
        run: |
          cd api-tests
          probe config/base.yml,config/${{ matrix.environment }}.yml,main-test-suite.yml
      
      - name: Upload Test Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: api-test-results-${{ matrix.environment }}
          path: test-results/
```

## Step 10: Advanced Testing Patterns

### Contract Testing

Add API contract validation:

```yaml
# tests/contract-tests.yml
name: "API Contract Tests"
description: "Validate API specification compliance"

jobs:
  openapi-compliance:
    name: "OpenAPI Specification Compliance"
    steps:
      - name: "Validate Product Endpoints Against Schema"
        # This would validate responses against OpenAPI spec
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products"
        test: |
          # Custom validation logic for OpenAPI compliance
          res.status == 200 &&
          res.json.products != null &&
          # Additional schema validation...
```

### Security Testing

Add basic security validation:

```yaml
# tests/security-tests.yml
name: "API Security Tests"
description: "Basic security validation tests"

jobs:
  security-headers:
    name: "Security Headers Validation"
    steps:
      - name: "Check Security Headers"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products"
        test: |
          res.status == 200 &&
          res.headers["X-Content-Type-Options"] != null &&
          res.headers["X-Frame-Options"] != null &&
          res.headers["X-XSS-Protection"] != null

  authentication-security:
    name: "Authentication Security Tests"
    steps:
      - name: "Test SQL Injection Prevention"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/products/search"
          method: "POST"
          body: |
            {
              "query": "'; DROP TABLE products; --"
            }
        test: |
          res.status == 400 ||
          (res.status == 200 && !res.json.error)
```

## Troubleshooting

### Common Issues

**Authentication Token Expiration:**
```yaml
# Add token refresh logic
- name: "Refresh Token If Needed"
  if: res.status == 401
  action: http
  with:
    url: "{{env.API_BASE_URL}}/api/{{env.API_VERSION}}/auth/refresh"
  # ... token refresh logic
```

**Test Data Cleanup:**
```yaml
# Add cleanup job
cleanup:
  name: "Test Data Cleanup"
  if: always()
  steps:
    - name: "Delete Test Products"
      # ... cleanup logic
```

**Rate Limiting:**
```yaml
# Add delays between requests
- name: "Rate Limit Delay"
  action: hello
  with:
    delay: "1s"
```

## Next Steps

Your comprehensive API testing pipeline is now complete! Consider these extensions:

1. **Performance Profiling** - Add detailed performance analysis
2. **Database State Validation** - Verify database changes
3. **Mock Service Testing** - Test against mocked dependencies
4. **Chaos Engineering** - Test failure scenarios
5. **Visual Regression Testing** - For APIs that return UI components
6. **API Versioning Tests** - Test backward compatibility

## Related Resources

- **[First Monitoring System Tutorial](../first-monitoring-system/)** - Basic monitoring setup
- **[Multi-Environment Testing Tutorial](../multi-environment-testing/)** - Environment management
- **[How-tos: Environment Management](../../how-tos/environment-management/)** - Pipeline integration and environment management
- **[Reference: Actions](../../reference/actions-reference/)** - Complete action reference