# Execution Model

Understanding Probe's execution model is crucial for designing efficient workflows and troubleshooting execution issues. This guide explores how Probe schedules, executes, and manages workflow components from start to finish.

## Execution Overview

Probe follows a structured execution model that processes workflows in a predictable, deterministic manner:

1. **Workflow Parsing**: Parse and validate YAML configuration
2. **Dependency Resolution**: Build execution graph based on job dependencies
3. **Job Scheduling**: Schedule jobs for execution based on dependencies
4. **Step Execution**: Execute steps sequentially within each job
5. **State Management**: Track execution state and results
6. **Resource Cleanup**: Clean up resources after execution

### Execution Hierarchy

```
Workflow
├── Job 1 (independent)
│   ├── Step 1.1 (sequential)
│   ├── Step 1.2 (sequential)
│   └── Step 1.3 (sequential)
├── Job 2 (independent, parallel to Job 1)
│   ├── Step 2.1 (sequential)
│   └── Step 2.2 (sequential)
└── Job 3 (depends on Job 1 and Job 2)
    ├── Step 3.1 (sequential)
    └── Step 3.2 (sequential)
```

## Job Execution Model

### Independent Job Execution

Jobs without dependencies execute in parallel:

```yaml
name: Parallel Service Check
description: Check multiple services simultaneously

jobs:
  database-check:     # Executes immediately
    name: Database Health
    steps:
      - name: Check Database
        action: http
        with:
          url: "{{env.DB_URL}}/health"
        test: res.status == 200

  api-check:          # Executes in parallel with database-check
    name: API Health
    steps:
      - name: Check API
        action: http
        with:
          url: "{{env.API_URL}}/health"
        test: res.status == 200

  cache-check:        # Executes in parallel with others
    name: Cache Health
    steps:
      - name: Check Cache
        action: http
        with:
          url: "{{env.CACHE_URL}}/health"
        test: res.status == 200
```

**Execution Timeline:**
```
Time 0: Start database-check, api-check, cache-check simultaneously
Time T: All jobs complete (T = max execution time of all jobs)
```

### Dependent Job Execution

Jobs with dependencies wait for prerequisite jobs to complete:

```yaml
name: Staged Deployment Validation
description: Validate deployment in dependency order

jobs:
  infrastructure:     # Executes first
    name: Infrastructure Check
    steps:
      - name: Database Connectivity
        action: http
        with:
          url: "{{env.DB_URL}}/ping"
        test: res.status == 200
        outputs:
          db_healthy: res.status == 200

  services:          # Waits for infrastructure
    name: Service Check
    needs: [infrastructure]
    steps:
      - name: API Service
        action: http
        with:
          url: "{{env.API_URL}}/health"
        test: res.status == 200

  integration:       # Waits for services
    name: Integration Test
    needs: [services]
    steps:
      - name: End-to-End Test
        action: http
        with:
          url: "{{env.API_URL}}/integration-test"
        test: res.status == 200

  notification:      # Waits for integration
    name: Send Notification
    needs: [integration]
    steps:
      - name: Notify Success
        echo: "Deployment validation completed successfully"
```

**Execution Timeline:**
```
Time 0: Start infrastructure job
Time T1: infrastructure completes → start services job
Time T2: services completes → start integration job
Time T3: integration completes → start notification job
Time T4: notification completes → workflow done
```

### Complex Dependency Graphs

Jobs can have multiple dependencies and form complex execution graphs:

```yaml
jobs:
  # Foundation layer (parallel)
  database-setup:
    name: Database Setup
    steps:
      - name: Initialize Database
        outputs:
          db_session_id: "{{random_str(16)}}"

  cache-setup:
    name: Cache Setup
    steps:
      - name: Initialize Cache
        outputs:
          cache_session_id: "{{random_str(16)}}"

  # Service layer (depends on foundation)
  user-service:
    name: User Service Test
    needs: [database-setup, cache-setup]
    steps:
      - name: Test User Service
        outputs:
          user_service_ready: true

  order-service:
    name: Order Service Test
    needs: [database-setup]  # Only needs database
    steps:
      - name: Test Order Service
        outputs:
          order_service_ready: true

  # Integration layer (depends on services)
  integration-test:
    name: Integration Test
    needs: [user-service, order-service]
    steps:
      - name: Test Service Integration
        echo: "Testing integration between user and order services"

  # Reporting layer (depends on everything)
  final-report:
    name: Final Report
    needs: [integration-test]
    steps:
      - name: Generate Report
        echo: |
          Execution Report:
          Database Setup: {{outputs.database-setup ? "✅" : "❌"}}
          Cache Setup: {{outputs.cache-setup ? "✅" : "❌"}}
          User Service: {{outputs.user-service ? "✅" : "❌"}}
          Order Service: {{outputs.order-service ? "✅" : "❌"}}
          Integration Test: {{jobs.integration-test.success ? "✅" : "❌"}}
```

**Execution Timeline:**
```
Time 0: Start database-setup, cache-setup (parallel)
Time T1: Both foundation jobs complete → start user-service, order-service
Time T2: Both service jobs complete → start integration-test
Time T3: integration-test completes → start final-report
Time T4: final-report completes → workflow done
```

## Step Execution Model

### Sequential Step Execution

Within a job, steps execute sequentially in the order defined:

```yaml
jobs:
  user-workflow:
    name: User Management Workflow
    steps:
      - name: Step 1 - Create User
        id: create
        action: http
        with:
          url: "{{env.API_URL}}/users"
          method: POST
          body: '{"name": "Test User", "email": "test@example.com"}'
        test: res.status == 201
        outputs:
          user_id: res.json.user.id

      - name: Step 2 - Verify User
        id: verify
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.create.user_id}}"
        test: res.status == 200
        outputs:
          user_verified: true

      - name: Step 3 - Update User
        id: update
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.create.user_id}}"
          method: PUT
          body: '{"name": "Updated User"}'
        test: res.status == 200

      - name: Step 4 - Delete User
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.create.user_id}}"
          method: DELETE
        test: res.status == 204

      - name: Step 5 - Confirm Deletion
        action: http
        with:
          url: "{{env.API_URL}}/users/{{outputs.create.user_id}}"
        test: res.status == 404
```

**Step Execution Order:**
```
Step 1 → Step 2 → Step 3 → Step 4 → Step 5
```

Each step waits for the previous step to complete before starting.

### Conditional Step Execution

Steps can be skipped based on conditions, but the evaluation order remains sequential:

```yaml
steps:
  - name: Primary Service Check
    id: primary
    action: http
    with:
      url: "{{env.PRIMARY_URL}}/health"
    test: res.status == 200
    continue_on_error: true
    outputs:
      primary_healthy: res.status == 200

  - name: Backup Service Check
    if: "!outputs.primary.primary_healthy"  # Only if primary failed
    id: backup
    action: http
    with:
      url: "{{env.BACKUP_URL}}/health"
    test: res.status == 200
    outputs:
      backup_healthy: res.status == 200

  - name: Success Path
    if: outputs.primary.primary_healthy     # Only if primary succeeded
    echo: "Primary service is healthy"

  - name: Fallback Path
    if: "!outputs.primary.primary_healthy && outputs.backup.backup_healthy"
    echo: "Primary failed, but backup is healthy"

  - name: Failure Path
    if: "!outputs.primary.primary_healthy && (!outputs.backup || !outputs.backup.backup_healthy)"
    echo: "Both primary and backup services failed"

  - name: Always Runs
    echo: "This step always executes"
```

**Conditional Execution Flow:**
```
1. Primary Service Check (always runs)
2. Backup Service Check (conditional - only if primary failed)
3. Success Path (conditional - only if primary succeeded)
4. Fallback Path (conditional - only if primary failed AND backup succeeded)
5. Failure Path (conditional - only if both failed)
6. Always Runs (always runs)
```

## State Management

### Job State Tracking

Probe tracks comprehensive state information for each job:

```yaml
# Job states available for reference:
jobs:
  example-job:
    steps:
      - name: Example Step
        echo: "Job states can be referenced from other jobs"

  dependent-job:
    needs: [example-job]
    steps:
      - name: Check Job States
        echo: |
          Job State Information:
          
          Job executed: {{jobs.example-job.executed}}      # true if job ran
          Job success: {{jobs.example-job.success}}        # true if all steps passed
          Job failed: {{jobs.example-job.failed}}          # true if any step failed
          Job skipped: {{jobs.example-job.skipped}}         # true if job was skipped
          
          Step count: {{jobs.example-job.steps.length}}    # number of steps
          Passed steps: {{jobs.example-job.passed_steps}}  # count of passed steps
          Failed steps: {{jobs.example-job.failed_steps}}  # count of failed steps
```

### Step State and Output Management

Each step produces state and output information:

```yaml
steps:
  - name: API Test Step
    id: api-test
    action: http
    with:
      url: "{{env.API_URL}}/test"
    test: res.status == 200 && res.time < 1000
    outputs:
      response_time: res.time
      status_code: res.status
      api_healthy: res.status == 200

  - name: Reference Previous Step
    echo: |
      Previous Step Information:
      
      Step executed: {{steps.api-test.executed}}           # true if step ran
      Step success: {{steps.api-test.success}}             # true if test passed
      Step failed: {{steps.api-test.failed}}               # true if test failed
      Step skipped: {{steps.api-test.skipped}}             # true if step was skipped
      
      Step outputs:
      Response time: {{outputs.api-test.response_time}}ms
      Status code: {{outputs.api-test.status_code}}
      API healthy: {{outputs.api-test.api_healthy}}
```

### Cross-Job State References

Jobs can reference state from other jobs:

```yaml
jobs:
  health-check:
    name: Health Check
    steps:
      - name: Check Service
        outputs:
          service_healthy: true

  performance-test:
    name: Performance Test
    needs: [health-check]
    if: jobs.health-check.success  # Only run if health check passed
    steps:
      - name: Load Test
        outputs:
          avg_response_time: 250

  reporting:
    name: Generate Report
    needs: [health-check, performance-test]
    steps:
      - name: Status Report
        echo: |
          System Status Report:
          
          Health Check: {{jobs.health-check.success ? "✅ Passed" : "❌ Failed"}}
          Performance Test: {{
            jobs.performance-test.executed ? 
              (jobs.performance-test.success ? "✅ Passed" : "❌ Failed") : 
              "⏸️ Skipped"
          }}
          
          {{jobs.health-check.success && jobs.performance-test.success ? 
            "Average Response Time: " + outputs.performance-test.avg_response_time + "ms" : 
            "Performance data not available"}}
```

## Timing and Performance

### Execution Timing

Probe tracks timing information at multiple levels:

```yaml
jobs:
  timing-example:
    name: Timing Example
    steps:
      - name: Quick Operation
        id: quick
        action: http
        with:
          url: "{{env.API_URL}}/ping"
        test: res.status == 200
        outputs:
          ping_time: res.time

      - name: Slow Operation
        id: slow
        action: http
        with:
          url: "{{env.API_URL}}/complex-query"
        test: res.status == 200
        outputs:
          query_time: res.time

      - name: Timing Summary
        echo: |
          Operation Timing:
          
          Quick operation: {{outputs.quick.ping_time}}ms
          Slow operation: {{outputs.slow.query_time}}ms
          Total step time: {{outputs.quick.ping_time + outputs.slow.query_time}}ms
          
          Performance classification:
          Quick: {{outputs.quick.ping_time < 100 ? "Excellent" : (outputs.quick.ping_time < 500 ? "Good" : "Slow")}}
          Slow: {{outputs.slow.query_time < 1000 ? "Fast" : (outputs.slow.query_time < 5000 ? "Acceptable" : "Too Slow")}}
```

### Timeout Management

Configure timeouts at different levels:

```yaml
jobs:
  timeout-management:
    name: Timeout Management Example
    timeout: 300s  # Job-level timeout (5 minutes)
    steps:
      - name: Quick API Call
        action: http
        with:
          url: "{{env.API_URL}}/quick"
          timeout: 10s  # Step-level timeout
        test: res.status == 200

      - name: Database Query
        action: http
        with:
          url: "{{env.DB_API}}/complex-query"
          timeout: 60s  # Longer timeout for complex operation
        test: res.status == 200

      - name: External Service Call
        action: http
        with:
          url: "{{env.EXTERNAL_API}}/data"
          timeout: 30s  # External services may be slower
        test: res.status == 200
        continue_on_error: true  # Don't fail workflow if external service is slow
```

### Parallel Execution Optimization

Optimize workflow execution through effective parallelization:

```yaml
name: Optimized Parallel Execution
description: Efficiently organize jobs for maximum parallelism

jobs:
  # Tier 1: Independent foundation checks (all parallel)
  database-check:
    name: Database Health
    steps:
      - name: DB Connection Test
        action: http
        with:
          url: "{{env.DB_URL}}/ping"
        test: res.status == 200

  cache-check:
    name: Cache Health
    steps:
      - name: Cache Connection Test
        action: http
        with:
          url: "{{env.CACHE_URL}}/ping"
        test: res.status == 200

  network-check:
    name: Network Connectivity
    steps:
      - name: External API Test
        action: http
        with:
          url: "{{env.EXTERNAL_API}}/ping"
        test: res.status == 200

  # Tier 2: Service-level checks (parallel, depend on infrastructure)
  user-service-test:
    name: User Service Test
    needs: [database-check, cache-check]
    steps:
      - name: User API Test
        action: http
        with:
          url: "{{env.USER_API}}/health"
        test: res.status == 200

  order-service-test:
    name: Order Service Test
    needs: [database-check]
    steps:
      - name: Order API Test
        action: http
        with:
          url: "{{env.ORDER_API}}/health"
        test: res.status == 200

  notification-service-test:
    name: Notification Service Test
    needs: [network-check]
    steps:
      - name: Notification API Test
        action: http
        with:
          url: "{{env.NOTIFICATION_API}}/health"
        test: res.status == 200

  # Tier 3: Integration tests (depend on services)
  user-order-integration:
    name: User-Order Integration
    needs: [user-service-test, order-service-test]
    steps:
      - name: Integration Test
        action: http
        with:
          url: "{{env.API_URL}}/integration/user-order"
        test: res.status == 200

  # Tier 4: Final validation (depends on integration)
  end-to-end-test:
    name: End-to-End Test
    needs: [user-order-integration, notification-service-test]
    steps:
      - name: Complete Workflow Test
        action: http
        with:
          url: "{{env.API_URL}}/e2e/complete-workflow"
        test: res.status == 200
```

**Execution Visualization:**
```
Time 0-T1: database-check, cache-check, network-check (parallel)
Time T1-T2: user-service-test, order-service-test, notification-service-test (parallel)
Time T2-T3: user-order-integration
Time T3-T4: end-to-end-test
```

## Error Propagation and Recovery

### Error Propagation Model

Understanding how errors propagate through the execution model:

```yaml
jobs:
  critical-foundation:
    name: Critical Foundation
    steps:
      - name: Critical Check
        action: http
        with:
          url: "{{env.CRITICAL_SERVICE}}/health"
        test: res.status == 200
        # Default: continue_on_error: false (stops job on failure)

  dependent-service:
    name: Dependent Service
    needs: [critical-foundation]  # Won't execute if foundation fails
    steps:
      - name: Service Test
        action: http
        with:
          url: "{{env.SERVICE_URL}}/test"
        test: res.status == 200

  resilient-check:
    name: Resilient Check
    # No dependencies - always executes
    steps:
      - name: Independent Check
        action: http
        with:
          url: "{{env.INDEPENDENT_SERVICE}}/health"
        test: res.status == 200
        continue_on_error: true  # Job continues even if step fails

  conditional-cleanup:
    name: Conditional Cleanup
    needs: [critical-foundation, dependent-service, resilient-check]
    if: jobs.critical-foundation.failed || jobs.dependent-service.failed
    steps:
      - name: Cleanup Failed State
        echo: |
          Cleaning up after failures:
          Critical Foundation: {{jobs.critical-foundation.success ? "✅" : "❌"}}
          Dependent Service: {{jobs.dependent-service.executed ? (jobs.dependent-service.success ? "✅" : "❌") : "⏸️"}}
          Resilient Check: {{jobs.resilient-check.success ? "✅" : "❌"}}
```

### Recovery Execution Model

Implement recovery workflows that execute based on failure patterns:

```yaml
jobs:
  primary-workflow:
    name: Primary Workflow
    steps:
      - name: Main Process
        id: main
        action: http
        with:
          url: "{{env.API_URL}}/main-process"
        test: res.status == 200
        continue_on_error: true
        outputs:
          process_successful: res.status == 200

  recovery-workflow:
    name: Recovery Workflow
    if: jobs.primary-workflow.failed
    steps:
      - name: Diagnose Failure
        id: diagnose
        action: http
        with:
          url: "{{env.API_URL}}/diagnostics"
        test: res.status == 200
        outputs:
          diagnosis: res.json.issue_type

      - name: Automated Recovery
        if: outputs.diagnose.diagnosis == "temporary_failure"
        action: http
        with:
          url: "{{env.API_URL}}/recovery/auto"
          method: POST
        test: res.status == 200

      - name: Manual Recovery Alert
        if: outputs.diagnose.diagnosis == "critical_failure"
        echo: "🚨 Critical failure detected - manual intervention required"

  validation-workflow:
    name: Validation Workflow
    needs: [primary-workflow, recovery-workflow]
    if: jobs.primary-workflow.success || jobs.recovery-workflow.success
    steps:
      - name: Validate Final State
        action: http
        with:
          url: "{{env.API_URL}}/validate"
        test: res.status == 200
        outputs:
          system_healthy: res.status == 200

  final-report:
    name: Final Report
    needs: [validation-workflow]
    steps:
      - name: Execution Summary
        echo: |
          Workflow Execution Summary:
          
          Primary workflow: {{jobs.primary-workflow.success ? "✅ Successful" : "❌ Failed"}}
          Recovery executed: {{jobs.recovery-workflow.executed ? "Yes" : "No"}}
          Recovery successful: {{jobs.recovery-workflow.executed ? (jobs.recovery-workflow.success ? "✅ Yes" : "❌ No") : "N/A"}}
          Final validation: {{jobs.validation-workflow ? (jobs.validation-workflow.success ? "✅ Passed" : "❌ Failed") : "⏸️ Skipped"}}
          
          Overall result: {{
            jobs.validation-workflow.success ? "✅ System operational" :
            jobs.recovery-workflow.executed ? "⚠️ System recovered with issues" :
            "❌ System failed"
          }}
```

## Resource Management

### Plugin Lifecycle Management

Probe manages action plugins throughout workflow execution:

```yaml
jobs:
  plugin-intensive-workflow:
    name: Plugin Intensive Workflow
    steps:
      # HTTP plugin loaded for this step
      - name: API Test
        action: http
        with:
          url: "{{env.API_URL}}/test"
        test: res.status == 200

      # SMTP plugin loaded for this step
      - name: Send Notification
        action: smtp
        with:
          host: "{{env.SMTP_HOST}}"
          to: ["admin@company.com"]
          subject: "Test Completed"
          body: "API test completed successfully"

      # Hello plugin loaded for this step
      - name: Debug Message
        action: hello
        with:
          message: "Debug checkpoint reached"

      # HTTP plugin reused (already loaded)
      - name: Follow-up API Test
        action: http
        with:
          url: "{{env.API_URL}}/follow-up"
        test: res.status == 200
```

Plugin lifecycle:
1. Plugin loaded when first action is encountered
2. Plugin reused for subsequent actions of same type
3. Plugin cleaned up after job completion

### Memory and Performance Optimization

Probe optimizes execution for performance and resource usage:

```yaml
jobs:
  optimized-workflow:
    name: Performance Optimized Workflow
    steps:
      # Efficient: Direct property access
      - name: User Data Collection
        id: user-data
        action: http
        with:
          url: "{{env.API_URL}}/users"
        test: res.status == 200
        outputs:
          user_count: res.json.total_users    # Extract specific value
          first_user_id: res.json.users[0].id # Direct array access
          # Avoid: large_user_list: res.json.users (stores entire array)

      # Efficient: Conditional processing
      - name: Process Large Dataset
        if: outputs.user-data.user_count < 1000  # Only process if manageable size
        action: http
        with:
          url: "{{env.API_URL}}/users/batch-process"
        test: res.status == 200

      # Efficient: Scoped outputs
      - name: Summary Generation
        echo: |
          Processing Summary:
          Total users: {{outputs.user-data.user_count}}
          First user: {{outputs.user-data.first_user_id}}
          Batch processed: {{steps.process-large-dataset.executed ? "Yes" : "No"}}
          # Efficient output without storing unnecessary data
```

## Best Practices

### 1. Dependency Design

```yaml
# Good: Logical dependency grouping
jobs:
  infrastructure:    # Foundation layer
  application:       # Depends on infrastructure
    needs: [infrastructure]
  integration:       # Depends on application
    needs: [application]

# Avoid: Unnecessary dependencies
jobs:
  independent-check-1:
  independent-check-2:
    needs: [independent-check-1]  # Unnecessary if truly independent
```

### 2. Error Handling Strategy

```yaml
# Good: Strategic error handling
- name: Critical Operation
  test: res.status == 200
  continue_on_error: false      # Fail fast for critical operations

- name: Optional Operation
  test: res.status == 200
  continue_on_error: true       # Continue for optional operations
```

### 3. Output Efficiency

```yaml
# Good: Efficient outputs
outputs:
  essential_data: res.json.id
  computed_value: res.json.items.length
  status_flag: res.status == 200

# Avoid: Storing large objects
outputs:
  # entire_response: res.json  # Could be very large
```

### 4. Execution Flow Documentation

```yaml
name: Well-Documented Workflow
description: |
  Execution flow:
  1. Infrastructure validation (parallel)
  2. Service health checks (parallel, depends on infrastructure)
  3. Integration testing (sequential, depends on services)
  4. Reporting (depends on all previous stages)
  
  Expected execution time: 2-5 minutes
  Critical path: infrastructure → services → integration → reporting
```

## What's Next?

Now that you understand the execution model, explore:

1. **[File Merging](../file-merging/)** - Learn configuration composition techniques
2. **[How-tos](../../how-tos/)** - See practical execution patterns in action
3. **[Reference](../../reference/)** - Detailed syntax and configuration reference

Understanding the execution model helps you design efficient, predictable workflows that make optimal use of parallelism and handle failures gracefully.
