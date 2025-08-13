# Jobs and Steps

Jobs and steps are the building blocks of Probe workflows. Understanding their mechanics, execution model, and interaction patterns is crucial for building effective automation. This guide explores the detailed behavior and advanced use cases.

## Job Fundamentals

A **job** is a logical grouping of related steps that execute together as a unit. Jobs provide:

- **Isolation**: Each job runs in its own context
- **Parallelism**: Jobs can run simultaneously unless dependencies exist
- **State Management**: Jobs track their execution state and results
- **Output Sharing**: Jobs can produce outputs for other jobs to consume

### Job Structure

```yaml
jobs:
  job-id:                    # Unique identifier (alphanumeric, hyphens, underscores)
    name: Human Readable Name # Optional: Display name
    needs: [other-job]       # Optional: Job dependencies
    if: condition            # Optional: Conditional execution
    continue_on_error: true  # Optional: Continue workflow on failure
    timeout: 300s            # Optional: Job timeout
    steps:                   # Required: Array of steps
      # Step definitions...
```

### Job Lifecycle

Jobs progress through several states during execution:

1. **Pending**: Job is queued for execution
2. **Running**: Job is actively executing steps
3. **Success**: All steps completed successfully
4. **Failed**: One or more steps failed
5. **Skipped**: Job was skipped due to conditions
6. **Cancelled**: Job was cancelled due to timeout or error

### Job Dependencies

Use the `needs` keyword to create execution dependencies:

```yaml
jobs:
  setup:
    name: Environment Setup
    steps:
      - name: Initialize Database
        action: http
        with:
          url: "{{env.DB_API}}/init"
        test: res.status == 200
        outputs:
          db_session_id: res.json.session_id

  test-suite-a:
    name: API Test Suite A
    needs: [setup]           # Wait for setup to complete
    steps:
      - name: Test User API
        action: http
        with:
          url: "{{env.API_URL}}/users"
          headers:
            X-Session-ID: "{{outputs.setup.db_session_id}}"
        test: res.status == 200

  test-suite-b:
    name: API Test Suite B
    needs: [setup]           # Also depends on setup
    steps:
      - name: Test Order API
        action: http
        with:
          url: "{{env.API_URL}}/orders"
          headers:
            X-Session-ID: "{{outputs.setup.db_session_id}}"
        test: res.status == 200

  cleanup:
    name: Environment Cleanup
    needs: [test-suite-a, test-suite-b]  # Wait for both test suites
    steps:
      - name: Clean Database
        action: http
        with:
          url: "{{env.DB_API}}/cleanup"
          headers:
            X-Session-ID: "{{outputs.setup.db_session_id}}"
        test: res.status == 200
```

### Conditional Job Execution

Jobs can execute conditionally based on other job results:

```yaml
jobs:
  health-check:
    name: Basic Health Check
    steps:
      - name: Ping Service
        id: ping
        action: http
        with:
          url: "{{env.SERVICE_URL}}/ping"
        test: res.status == 200
        outputs:
          service_responsive: res.status == 200

  detailed-check:
    name: Detailed Health Check
    if: jobs.health-check.success && outputs.health-check.service_responsive
    steps:
      - name: Deep Health Check
        action: http
        with:
          url: "{{env.SERVICE_URL}}/health/detailed"
        test: res.status == 200

  recovery:
    name: Service Recovery
    if: jobs.health-check.failed
    steps:
      - name: Restart Service
        action: http
        with:
          url: "{{env.ADMIN_URL}}/restart"
          method: POST
        test: res.status == 200

  notification:
    name: Send Notifications
    needs: [health-check]
    if: jobs.health-check.failed || jobs.recovery.executed
    steps:
      - name: Alert Team
        echo: |
          Service Status Alert:
          Health Check: {{jobs.health-check.success ? "✅" : "❌"}}
          Recovery Attempted: {{jobs.recovery.executed ? "Yes" : "No"}}
          Recovery Successful: {{jobs.recovery.success ? "✅" : "❌"}}
```

## Step Fundamentals

A **step** is the smallest unit of execution in Probe. Each step performs a specific action and can:

- Execute actions (HTTP requests, commands, etc.)
- Test results with assertions
- Produce outputs for other steps
- Echo messages to the console
- Execute conditionally

### Step Structure

```yaml
steps:
  - name: Step Name          # Required: Descriptive name
    id: step-id             # Optional: Unique identifier for referencing
    action: http            # Optional: Action to execute
    with:                   # Optional: Action parameters
      url: https://api.example.com
      method: GET
    test: res.status == 200 # Optional: Test condition
    outputs:                # Optional: Data to pass to other steps
      response_time: res.time
      user_count: res.json.total_users
    echo: "Message"         # Optional: Display message
    if: condition           # Optional: Conditional execution
    continue_on_error: false # Optional: Continue on step failure
    timeout: 30s            # Optional: Step timeout
```

### Step Types

#### 1. Action Steps

Execute specific actions like HTTP requests:

```yaml
- name: Check User API
  action: http
  with:
    url: "{{env.API_URL}}/users/{{env.TEST_USER_ID}}"
    method: GET
    headers:
      Authorization: "Bearer {{env.API_TOKEN}}"
      Accept: "application/json"
  test: res.status == 200 && res.json.user.active == true
  outputs:
    user_id: res.json.user.id
    user_email: res.json.user.email
    last_login: res.json.user.last_login
```

#### 2. Echo Steps

Display messages or computed values:

```yaml
- name: Display Results
  echo: |
    Test Results Summary:
    
    User ID: {{outputs.previous-step.user_id}}
    Email: {{outputs.previous-step.user_email}}
    Last Login: {{outputs.previous-step.last_login}}
    
    Response Time: {{outputs.previous-step.response_time}}ms
    Test Completed: {{unixtime()}}
```

#### 3. Hybrid Steps

Combine actions with echo messages:

```yaml
- name: Test and Report
  action: http
  with:
    url: "{{env.API_URL}}/status"
  test: res.status == 200
  echo: |
    API Status Check:
    Status Code: {{res.status}}
    Response Time: {{res.time}}ms
    API Version: {{res.json.version}}
```

### Step Execution Flow

Steps within a job execute sequentially by default:

```yaml
jobs:
  sequential-test:
    name: Sequential Step Execution
    steps:
      - name: Step 1 - Setup
        id: setup
        action: http
        with:
          url: "{{env.API_URL}}/setup"
        test: res.status == 200
        outputs:
          session_id: res.json.session_id

      - name: Step 2 - Execute Test
        id: test
        action: http
        with:
          url: "{{env.API_URL}}/test"
          headers:
            X-Session-ID: "{{outputs.setup.session_id}}"
        test: res.status == 200
        outputs:
          test_result: res.json.result

      - name: Step 3 - Cleanup
        action: http
        with:
          url: "{{env.API_URL}}/cleanup"
          headers:
            X-Session-ID: "{{outputs.setup.session_id}}"
        test: res.status == 200

      - name: Step 4 - Report
        echo: "Test completed with result: {{outputs.test.test_result}}"
```

### Conditional Step Execution

Steps can execute based on conditions:

```yaml
steps:
  - name: Primary Health Check
    id: primary
    action: http
    with:
      url: "{{env.PRIMARY_SERVICE_URL}}/health"
    test: res.status == 200
    continue_on_error: true
    outputs:
      primary_healthy: res.status == 200

  - name: Backup Service Check
    if: "!outputs.primary.primary_healthy"
    action: http
    with:
      url: "{{env.BACKUP_SERVICE_URL}}/health"
    test: res.status == 200
    outputs:
      backup_healthy: res.status == 200

  - name: Success Report
    if: outputs.primary.primary_healthy || outputs.backup.backup_healthy
    echo: |
      Service Status: ✅ Healthy
      Primary: {{outputs.primary.primary_healthy ? "Online" : "Offline"}}
      Backup: {{outputs.backup.backup_healthy ? "Online" : "N/A"}}

  - name: Failure Report
    if: "!outputs.primary.primary_healthy && !outputs.backup.backup_healthy"
    echo: "🚨 CRITICAL: Both primary and backup services are down!"
```

## Advanced Patterns

### 1. Error Recovery Patterns

Implement robust error handling with recovery steps:

```yaml
jobs:
  resilient-check:
    name: Resilient Service Check
    steps:
      - name: Attempt Primary Connection
        id: primary-attempt
        action: http
        with:
          url: "{{env.SERVICE_URL}}/api/v1/health"
          timeout: 10s
        test: res.status == 200
        continue_on_error: true
        outputs:
          primary_success: res.status == 200

      - name: Try Alternative Endpoint
        if: "!outputs.primary-attempt.primary_success"
        id: alt-attempt
        action: http
        with:
          url: "{{env.SERVICE_URL}}/api/v2/health"
          timeout: 15s
        test: res.status == 200
        continue_on_error: true
        outputs:
          alt_success: res.status == 200

      - name: Fallback to Legacy Endpoint
        if: "!outputs.primary-attempt.primary_success && !outputs.alt-attempt.alt_success"
        id: legacy-attempt
        action: http
        with:
          url: "{{env.SERVICE_URL}}/health"
          timeout: 20s
        test: res.status == 200
        continue_on_error: true
        outputs:
          legacy_success: res.status == 200

      - name: Final Status Report
        echo: |
          Service Health Check Results:
          
          Primary API (v1): {{outputs.primary-attempt.primary_success ? "✅" : "❌"}}
          Alternative API (v2): {{outputs.alt-attempt.alt_success ? "✅" : "❌"}}
          Legacy API: {{outputs.legacy-attempt.legacy_success ? "✅" : "❌"}}
          
          Overall Status: {{
            outputs.primary-attempt.primary_success || 
            outputs.alt-attempt.alt_success || 
            outputs.legacy-attempt.legacy_success ? "HEALTHY" : "DOWN"
          }}
```

### 2. Data Collection and Aggregation

Collect data across multiple steps for analysis:

```yaml
jobs:
  performance-analysis:
    name: Performance Analysis
    steps:
      - name: Test Homepage
        id: homepage
        action: http
        with:
          url: "{{env.BASE_URL}}/"
        test: res.status == 200
        outputs:
          homepage_time: res.time
          homepage_size: res.body_size

      - name: Test API Endpoint
        id: api
        action: http
        with:
          url: "{{env.BASE_URL}}/api/users"
        test: res.status == 200
        outputs:
          api_time: res.time
          api_size: res.body_size

      - name: Test Search Function
        id: search
        action: http
        with:
          url: "{{env.BASE_URL}}/search?q=test"
        test: res.status == 200
        outputs:
          search_time: res.time
          search_size: res.body_size

      - name: Performance Summary
        echo: |
          Performance Analysis Results:
          
          Homepage:
            Response Time: {{outputs.homepage.homepage_time}}ms
            Size: {{outputs.homepage.homepage_size}} bytes
            
          API Endpoint:
            Response Time: {{outputs.api.api_time}}ms
            Size: {{outputs.api.api_size}} bytes
            
          Search Function:
            Response Time: {{outputs.search.search_time}}ms
            Size: {{outputs.search.search_size}} bytes
            
          Average Response Time: {{
            (outputs.homepage.homepage_time + 
             outputs.api.api_time + 
             outputs.search.search_time) / 3
          }}ms
          
          Total Data Transfer: {{
            outputs.homepage.homepage_size + 
            outputs.api.api_size + 
            outputs.search.search_size
          }} bytes
```

### 3. Dynamic Step Configuration

Configure steps based on runtime conditions:

```yaml
jobs:
  adaptive-monitoring:
    name: Adaptive Monitoring
    steps:
      - name: Determine Environment
        id: env-detect
        action: http
        with:
          url: "{{env.SERVICE_URL}}/config"
        test: res.status == 200
        outputs:
          environment: res.json.environment
          feature_flags: res.json.features
          monitoring_level: res.json.monitoring.level

      - name: Basic Health Check
        action: http
        with:
          url: "{{env.SERVICE_URL}}/health"
        test: res.status == 200

      - name: Detailed Monitoring
        if: outputs.env-detect.monitoring_level == "detailed"
        action: http
        with:
          url: "{{env.SERVICE_URL}}/metrics"
        test: res.status == 200
        outputs:
          cpu_usage: res.json.system.cpu_percent
          memory_usage: res.json.system.memory_percent

      - name: Feature-Specific Tests
        if: outputs.env-detect.feature_flags.beta_features == true
        action: http
        with:
          url: "{{env.SERVICE_URL}}/beta/features"
        test: res.status == 200

      - name: Production Alerts
        if: outputs.env-detect.environment == "production" && (outputs.detailed.cpu_usage > 80 || outputs.detailed.memory_usage > 90)
        echo: |
          🚨 PRODUCTION ALERT: High resource usage detected!
          CPU: {{outputs.detailed.cpu_usage}}%
          Memory: {{outputs.detailed.memory_usage}}%
```

## Step and Job Identification

### Step IDs

Use `id` to reference steps from other parts of the workflow:

```yaml
steps:
  - name: User Authentication Test
    id: auth-test                    # Define ID for referencing
    action: http
    with:
      url: "{{env.API_URL}}/auth/login"
      method: POST
      body: |
        {
          "username": "testuser",
          "password": "{{env.TEST_PASSWORD}}"
        }
    test: res.status == 200
    outputs:
      auth_token: res.json.token
      user_id: res.json.user.id

  - name: User Profile Test
    action: http
    with:
      url: "{{env.API_URL}}/users/{{outputs.auth-test.user_id}}"  # Reference by ID
      headers:
        Authorization: "Bearer {{outputs.auth-test.auth_token}}"   # Reference by ID
    test: res.status == 200
```

### Job References

Reference job results from other jobs:

```yaml
jobs:
  database-check:
    name: Database Connectivity
    steps:
      - name: Test Database
        action: http
        with:
          url: "{{env.DB_API}}/ping"
        test: res.status == 200

  api-check:
    name: API Functionality
    needs: [database-check]
    steps:
      - name: Test API
        if: jobs.database-check.success    # Reference job success
        action: http
        with:
          url: "{{env.API_URL}}/health"
        test: res.status == 200

      - name: Skip Message
        if: jobs.database-check.failed     # Reference job failure
        echo: "Skipping API test due to database connectivity issues"
```

## Performance Optimization

### 1. Parallel Job Execution

Structure jobs to run in parallel when possible:

```yaml
jobs:
  # These jobs can run in parallel (no dependencies)
  frontend-test:
    name: Frontend Tests
    steps:
      - name: Test UI Components
        action: http
        with:
          url: "{{env.FRONTEND_URL}}"
        test: res.status == 200

  backend-test:
    name: Backend Tests
    steps:
      - name: Test API Endpoints
        action: http
        with:
          url: "{{env.BACKEND_URL}}/api"
        test: res.status == 200

  database-test:
    name: Database Tests
    steps:
      - name: Test Database Connection
        action: http
        with:
          url: "{{env.DB_URL}}/health"
        test: res.status == 200

  # This job waits for all parallel jobs to complete
  integration-test:
    name: Integration Tests
    needs: [frontend-test, backend-test, database-test]
    steps:
      - name: End-to-End Test
        action: http
        with:
          url: "{{env.APP_URL}}/integration-test"
        test: res.status == 200
```

### 2. Efficient Resource Usage

Optimize step execution for better resource utilization:

```yaml
jobs:
  efficient-monitoring:
    name: Efficient Resource Monitoring
    steps:
      # Use timeouts to prevent hanging
      - name: Quick Health Check
        action: http
        with:
          url: "{{env.SERVICE_URL}}/ping"
          timeout: 5s                    # Short timeout for ping
        test: res.status == 200

      # Conditional expensive operations
      - name: Detailed Analysis
        if: outputs.previous.response_time > 1000  # Only if response is slow
        action: http
        with:
          url: "{{env.SERVICE_URL}}/detailed-metrics"
          timeout: 30s                   # Longer timeout for detailed analysis
        test: res.status == 200

      # Batch related operations
      - name: Batch Status Check
        action: http
        with:
          url: "{{env.SERVICE_URL}}/batch-status"
          method: POST
          body: |
            {
              "checks": [
                {"type": "health", "endpoint": "/health"},
                {"type": "metrics", "endpoint": "/metrics"},
                {"type": "version", "endpoint": "/version"}
              ]
            }
        test: res.status == 200 && res.json.all_passed == true
```

## Best Practices

### 1. Job Granularity

Strike the right balance in job size:

```yaml
# Good: Focused, cohesive jobs
jobs:
  authentication-tests:
    name: Authentication System Tests
    steps:
      - name: Test Login
      - name: Test Logout
      - name: Test Token Refresh
      - name: Test Password Reset

  user-management-tests:
    name: User Management Tests
    steps:
      - name: Test User Creation
      - name: Test User Update
      - name: Test User Deletion

# Avoid: Overly granular jobs
jobs:
  test-login:           # Too granular
    steps:
      - name: Test Login
  test-logout:          # Each should be a step, not a job
    steps:
      - name: Test Logout

# Avoid: Monolithic jobs
jobs:
  all-tests:            # Too broad
    steps:
      - name: Test Login
      - name: Test Database
      - name: Test Email
      - name: Test Files
      # ... 50 more unrelated steps
```

### 2. Clear Step Names

Use descriptive, action-oriented step names:

```yaml
steps:
  # Good: Clear, specific names
  - name: Verify User Registration API Returns 201
  - name: Test Database Connection Pool Health
  - name: Validate JWT Token Expiration Logic
  - name: Check Email Service Rate Limiting

  # Avoid: Vague or generic names
  - name: Test API           # Too vague
  - name: Check Thing        # Not descriptive
  - name: Step 1             # No context
```

### 3. Proper Error Handling

Implement appropriate error handling strategies:

```yaml
steps:
  # Critical step - fail fast
  - name: Verify Database Connectivity
    action: http
    with:
      url: "{{env.DB_URL}}/ping"
    test: res.status == 200
    continue_on_error: false        # Default: fail the job

  # Non-critical step - continue on failure
  - name: Update Usage Analytics
    action: http
    with:
      url: "{{env.ANALYTICS_URL}}/update"
    test: res.status == 200
    continue_on_error: true         # Continue even if this fails

  # Recovery step
  - name: Log Failure Details
    if: steps.previous.failed
    echo: "Analytics update failed, but continuing with main workflow"
```

## What's Next?

Now that you understand jobs and steps in detail, explore:

1. **[Actions](../actions/)** - Learn about the action system and available plugins
2. **[Expressions and Templates](../expressions-and-templates/)** - Master dynamic configuration and testing
3. **[Data Flow](../data-flow/)** - Understand how data moves through workflows

Jobs and steps are the execution engine of Probe. Master these concepts to build efficient, reliable, and maintainable automation workflows.