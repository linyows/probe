---
title: Understanding Probe
description: Learn the core concepts and architecture of Probe
weight: 30
---

# Understanding Probe

Probe is a YAML-based workflow automation tool designed for monitoring, testing, and automation tasks. This guide explains the core concepts you need to understand to effectively use Probe.

## Core Concepts

### Workflows

A **workflow** is the top-level container that defines what Probe should execute. It consists of:

- **Metadata**: Name and description of the workflow
- **Jobs**: One or more jobs that can run in parallel or sequentially
- **Global configuration**: Shared settings that apply to all jobs

```yaml
name: My Workflow
description: What this workflow does
# Jobs go here...
```

### Jobs

A **job** is a collection of steps that execute together. Jobs can:

- Run in parallel with other jobs
- Have dependencies on other jobs
- Share outputs with other jobs
- Have their own configuration and context

```yaml
jobs:
  job-name:
    name: Human-readable job name
    needs: [other-job]  # Optional: wait for other jobs
    steps:
      # Steps go here...
```

### Steps

A **step** is the smallest unit of execution. Each step can:

- Execute an action (HTTP request, email, etc.)
- Run tests to validate results
- Echo messages to the output
- Set outputs for use by other steps
- Have conditional execution logic

```yaml
steps:
  - name: Step Name
    action: http          # The action to execute
    with:                 # Parameters for the action
      url: https://api.example.com
      method: GET
    test: res.status == 200  # Test condition
    outputs:              # Data to pass to other steps
      response_time: res.time
```

### Actions

**Actions** are the building blocks that actually do the work. Probe comes with built-in actions:

- **`http`**: Make HTTP/HTTPS requests
- **`shell`**: Execute shell commands and scripts securely
- **`smtp`**: Send email notifications and alerts
- **`hello`**: Simple greeting action (mainly for testing)

Actions are implemented as plugins, so you can extend Probe with custom actions.

## Workflow Execution Model

### Parallel Execution

By default, jobs run in parallel for maximum efficiency:

```yaml
jobs:
  frontend-check:    # These jobs run
    # ...             # at the same time
  backend-check:     # (in parallel)
    # ...
  database-check:
    # ...
```

### Sequential Execution with Dependencies

Use the `needs` keyword to create dependencies:

```yaml
jobs:
  setup:
    name: Setup Environment
    steps:
      # Setup steps...

  test:
    name: Run Tests
    needs: [setup]     # Wait for 'setup' to complete
    steps:
      # Test steps...

  cleanup:
    name: Clean Up
    needs: [test]      # Wait for 'test' to complete
    steps:
      # Cleanup steps...
```

### Data Flow

Data flows through the workflow using **outputs**:

```yaml
jobs:
  data-fetch:
    steps:
      - name: Get User Info
        action: http
        with:
          url: https://api.example.com/user/123
        outputs:
          user_id: res.json.id
          user_name: res.json.name

  notification:
    needs: [data-fetch]
    steps:
      - name: Send Welcome Email
        echo: "Welcome {{outputs.data-fetch.user_name}}!"
```

## Expression System

Probe uses expressions for dynamic values and testing. Expressions are written using `{{}}` syntax:

### Template Expressions

Use template expressions to insert dynamic values:

```yaml
- name: Greet User
  echo: "Hello {{outputs.previous-step.username}}!"
```

### Test Expressions

Use test expressions to validate results:

```yaml
- name: Check API Response
  action: http
  with:
    url: https://api.example.com/status
  test: res.status == 200 && res.json.healthy == true
```

### Available Variables

In expressions, you have access to:

- **`res`**: Response from the current action
- **`outputs`**: Outputs from previous steps/jobs
- **`env`**: Environment variables
- **Custom functions**: `random_int()`, `random_str()`, `unixtime()`

## File Merging

Probe supports merging multiple YAML files, which is useful for:

- Separating configuration from workflow logic
- Reusing common definitions across workflows
- Environment-specific overrides

```bash
# Merge base workflow with environment-specific config
probe base-workflow.yml,production-config.yml
```

The files are merged in order, with later files overriding values from earlier files.

## Error Handling

Probe provides several mechanisms for handling errors:

### Test Failures

When a test fails, the step is marked as failed:

```yaml
- name: Critical Check
  action: http
  with:
    url: https://critical-api.example.com
  test: res.status == 200  # If this fails, step fails
```

### Conditional Execution

Use the `if` condition to handle failures:

```yaml
- name: Primary Service Check
  id: primary
  action: http
  with:
    url: https://primary-api.example.com
  test: res.status == 200

- name: Fallback Check
  if: steps.primary.failed
  action: http
  with:
    url: https://backup-api.example.com
  test: res.status == 200
```

### Continue on Failure

By default, job execution stops on the first failure. You can change this behavior:

```yaml
- name: Non-Critical Check
  continue_on_error: true
  action: http
  with:
    url: https://optional-service.example.com
  test: res.status == 200
```

## Best Practices

### 1. Use Descriptive Names

```yaml
# Good
- name: Check Production API Health
  action: http
  # ...

# Not so good  
- name: HTTP Check
  action: http
  # ...
```

### 2. Group Related Steps into Jobs

```yaml
jobs:
  infrastructure-check:
    name: Infrastructure Health Check
    steps:
      - name: Check Database
        # ...
      - name: Check Cache
        # ...
      - name: Check Load Balancer
        # ...
```

### 3. Use Outputs for Data Sharing

```yaml
- name: Fetch Configuration
  id: config
  action: http
  with:
    url: https://config-service.example.com
  outputs:
    database_url: res.json.database_url
    
- name: Test Database Connection
  action: http
  with:
    url: "{{outputs.config.database_url}}/health"
```

### 4. Add Meaningful Test Conditions

```yaml
# Good - specific test conditions
test: res.status == 200 && res.json.status == "healthy" && res.time < 1000

# Not so good - generic test
test: res.status == 200
```

## What's Next?

Now that you understand the core concepts, you can:

1. **[Create your first workflow](../your-first-workflow/)** - Build a practical workflow
2. **[Learn CLI basics](../cli-basics/)** - Master the command-line interface
3. **[Explore the reference](../../reference/)** - Deep dive into all available options

The key to mastering Probe is practice. Start with simple workflows and gradually build more complex automation as you become comfortable with the concepts.