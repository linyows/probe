---
title: Quickstart
description: Get up and running with Probe in 5 minutes
weight: 20
---

# Quickstart

This guide will get you up and running with Probe in just a few minutes. You'll create your first workflow and see Probe in action.

## Prerequisites

- [Probe installed](../installation/) on your system
- Basic understanding of YAML syntax

## Your First Workflow

Let's create a simple workflow that checks if a website is responding correctly.

### 1. Create a Workflow File

Create a new file called `my-first-workflow.yml`:

```yaml
name: My First Website Check
description: Check if my website is responding correctly

jobs:
  health-check:
    name: Website Health Check
    steps:
      - name: Check Homepage
        action: http
        with:
          url: https://httpbin.org/status/200
          method: GET
        test: res.status == 200
        
      - name: Check API Endpoint
        action: http
        with:
          url: https://httpbin.org/json
          method: GET
        test: res.status == 200 && res.json.slideshow != null
        
      - name: Success Message
        echo: "✅ All checks passed! Website is healthy."
```

### 2. Run Your Workflow

Execute the workflow using the Probe CLI:

```bash
probe my-first-workflow.yml
```

You should see output similar to:

```
My First Website Check
Check if my website is responding correctly

⏺ Website Health Check (Completed in 1.23s)
  ⎿ 0. ✔︎  Check Homepage (245ms)
     1. ✔︎  Check API Endpoint (189ms)
     2. ✔︎  Success Message
       ✅ All checks passed! Website is healthy.

Total workflow time: 1.23s ✔︎ All jobs succeeded
```

### 3. Understanding What Happened

Let's break down what this workflow does:

1. **Workflow Metadata**: The `name` and `description` provide context about what this workflow does
2. **Job Definition**: The `health-check` job contains the steps to execute
3. **HTTP Actions**: Two steps make HTTP requests to test endpoints
4. **Test Assertions**: Each HTTP step includes a `test` that validates the response
5. **Echo Step**: Displays a success message when all checks pass

## Adding Verbose Output

Want to see more details about what's happening? Run with the verbose flag:

```bash
probe -v my-first-workflow.yml
```

This will show detailed information about each HTTP request and response.

## What's Next?

Now that you've run your first workflow, you can:

1. **[Learn the core concepts](../understanding-probe/)** - Understand workflows, jobs, and steps
2. **[Create your first custom workflow](../your-first-workflow/)** - Build something tailored to your needs
3. **[Explore CLI options](../cli-basics/)** - Learn about all available command-line options

## Common Next Steps

### Check Multiple Environments

Modify your workflow to check different environments:

```yaml
name: Multi-Environment Health Check
description: Check health across multiple environments

jobs:
  production-check:
    name: Production Health Check
    steps:
      - name: Check Production API
        action: http
        with:
          url: https://api.myapp.com/health
          method: GET
        test: res.status == 200

  staging-check:
    name: Staging Health Check
    steps:
      - name: Check Staging API
        action: http
        with:
          url: https://staging-api.myapp.com/health
          method: GET
        test: res.status == 200
```

### Add Error Handling

Include steps that handle failures gracefully:

```yaml
- name: Check Service
  action: http
  with:
    url: https://api.example.com/status
    method: GET
  test: res.status == 200
  
- name: Fallback Check
  if: steps.previous.failed
  action: http
  with:
    url: https://backup-api.example.com/status
    method: GET
  test: res.status == 200
```

## Troubleshooting

### Workflow File Not Found

```
[ERROR] workflow is required
```

Make sure you're providing the correct path to your YAML file.

### Invalid YAML Syntax

```
[ERROR] yaml: line 5: mapping values are not allowed in this context
```

Check your YAML syntax. Common issues include:
- Incorrect indentation (use spaces, not tabs)
- Missing colons after keys
- Unquoted strings with special characters

### Test Failures

If your tests fail, use verbose mode (`-v`) to see the actual response data and debug your test expressions.

Ready to dive deeper? Continue with [Understanding Probe](../understanding-probe/) to learn the core concepts.