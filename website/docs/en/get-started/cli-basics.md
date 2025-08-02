---
title: CLI Basics
description: Master the Probe command-line interface
weight: 50
---

# CLI Basics

The Probe command-line interface (CLI) is your primary way to interact with Probe. This guide covers all the essential commands, options, and techniques you need to master.

## Basic Usage

The most basic way to run Probe is:

```bash
probe workflow.yml
```

This executes the workflow defined in `workflow.yml` and displays the results.

## Command Syntax

```bash
probe [options] <workflow-file>
```

- **`workflow-file`**: Path to your YAML workflow file (required)
- **`options`**: Various flags to modify behavior (optional)

## Core Options

### Help and Information

Get help about available options:

```bash
probe --help
# or
probe -h
```

Check the installed version:

```bash
probe --version
```

### Verbose Output

Enable detailed logging to see what's happening under the hood:

```bash
probe --verbose workflow.yml
# or
probe -v workflow.yml
```

Verbose mode shows:
- Detailed HTTP request/response information
- Step execution timing
- Variable resolution details
- Plugin communication logs

**Example verbose output:**
```
[DEBUG] Starting workflow: My Health Check
[DEBUG] --- Step 1: Check Homepage
[DEBUG] Request:
[DEBUG]   method: "GET"
[DEBUG]   url: "https://example.com"
[DEBUG] Response:
[DEBUG]   status: 200
[DEBUG]   body: "<!DOCTYPE html>..."
[DEBUG] RT: 245ms
```

### Response Time Display

Show response times for HTTP requests:

```bash
probe --rt workflow.yml
```

This adds timing information to the output without the full verbosity of `--verbose`.

### Combining Options

You can combine multiple options:

```bash
probe -v --rt workflow.yml
```

## Multiple File Merging

One of Probe's powerful features is the ability to merge multiple YAML files:

```bash
probe base.yml,overrides.yml
```

### Use Cases for File Merging

**1. Environment-specific configuration:**

**base-workflow.yml:**
```yaml
name: API Health Check
jobs:
  health-check:
    steps:
      - name: Check API
        action: http
        with:
          url: "{{env.API_URL}}"
          method: GET
        test: res.status == 200
```

**production.yml:**
```yaml
# Production-specific settings
env:
  API_URL: https://api.production.example.com
```

**staging.yml:**
```yaml
# Staging-specific settings
env:
  API_URL: https://api.staging.example.com
```

Run for different environments:
```bash
# Production
probe base-workflow.yml,production.yml

# Staging
probe base-workflow.yml,staging.yml
```

**2. Shared configuration:**

**common-config.yml:**
```yaml
# Shared HTTP settings
defaults:
  http:
    timeout: 30s
    headers:
      User-Agent: "Probe Health Check v1.0"
```

**api-check.yml:**
```yaml
name: API Monitoring
# Inherits common HTTP settings
jobs:
  # ... job definitions
```

```bash
probe common-config.yml,api-check.yml
```

**3. Override specific values:**

```bash
probe workflow.yml,local-overrides.yml
```

The merge order matters - later files override values from earlier files.

## Working with Environment Variables

Probe can access environment variables in your workflows using the `env` object:

```yaml
steps:
  - name: Connect to Database
    action: http
    with:
      url: "{{env.DATABASE_URL}}"
      headers:
        Authorization: "Bearer {{env.API_TOKEN}}"
```

Set environment variables before running:

```bash
export DATABASE_URL="https://db.example.com"
export API_TOKEN="your-secret-token"
probe workflow.yml
```

## Exit Codes

Probe uses standard exit codes to indicate results:

- **`0`**: Success - all jobs completed successfully
- **`1`**: Failure - one or more jobs failed or error occurred

This makes Probe perfect for use in scripts and CI/CD pipelines:

```bash
#!/bin/bash
if probe health-check.yml; then
    echo "✅ Health check passed"
    deploy_application
else
    echo "❌ Health check failed"
    exit 1
fi
```

## Real-World Examples

### 1. CI/CD Integration

```bash
# In your CI/CD pipeline
probe smoke-tests.yml
if [ $? -eq 0 ]; then
    echo "Smoke tests passed, proceeding with deployment"
else
    echo "Smoke tests failed, aborting deployment"
    exit 1
fi
```

### 2. Cron Job Monitoring

```bash
# In crontab - run every 5 minutes
*/5 * * * * /usr/local/bin/probe /opt/monitoring/health-check.yml >> /var/log/probe.log 2>&1
```

### 3. Development Testing

```bash
# Quick test during development
probe -v api-tests.yml,local-config.yml
```

### 4. Load Testing

```bash
# Run performance tests with timing
probe --rt --verbose load-test.yml
```

## Output Interpretation

Understanding Probe's output helps you quickly identify issues:

### Successful Execution

```
My Health Check
Monitoring application health

⏺ Frontend Check (Completed in 0.45s)
  ⎿ 0. ✔︎  Check Homepage (234ms)
     1. ✔︎  Check API Health (156ms)

Total workflow time: 0.45s ✔︎ All jobs succeeded
```

**Key indicators:**
- **⏺**: Job completed
- **✔︎**: Step succeeded
- **Green text**: Success status
- **Total time**: Overall execution time

### Failed Execution

```
My Health Check
Monitoring application health

⏺ Frontend Check (Failed in 1.23s)
  ⎿ 0. ✘  Check Homepage (1.23s)
              request: map[string]interface {}{"method":"GET", "url":"https://example.com"}
              response: map[string]interface {}{"status":500, "body":"Internal Server Error"}
     1. ⏭  Check API Health (skipped)

Total workflow time: 1.23s ✘ 1 job(s) failed
```

**Key indicators:**
- **✘**: Step failed
- **⏭**: Step skipped (due to previous failure)
- **Red text**: Failure status
- **Request/Response**: Debug information for failed HTTP requests

### Partial Success

```
Multi-Service Check
Checking multiple services

⏺ Critical Services (Completed in 0.67s)
  ⎿ 0. ✔︎  Database Check (234ms)
     1. ✔︎  API Check (156ms)

⏺ Optional Services (Failed in 2.34s)
  ⎿ 0. ✘  External API Check (2.34s)

Total workflow time: 2.34s ✘ 1 job(s) failed
```

## Troubleshooting Common Issues

### File Not Found

```
[ERROR] workflow is required
```

**Solution:** Make sure you're providing a valid file path:
```bash
probe ./workflows/health-check.yml
```

### YAML Syntax Errors

```
[ERROR] yaml: line 5: mapping values are not allowed in this context
```

**Solution:** Check your YAML syntax:
- Use spaces, not tabs for indentation
- Ensure proper key-value syntax with colons
- Quote strings containing special characters

### Permission Errors

```
[ERROR] permission denied
```

**Solution:** Ensure the workflow file is readable:
```bash
chmod +r workflow.yml
```

### Network Timeouts

If steps hang or timeout:
- Use `--verbose` to see detailed network information
- Check network connectivity to target URLs
- Consider adding timeout values in your HTTP actions

## Best Practices

### 1. Use Descriptive File Names

```bash
# Good
probe api-health-check.yml
probe database-migration-test.yml

# Not so good
probe test.yml
probe workflow.yml
```

### 2. Organize with Directories

```bash
# Organize workflows by purpose
probe monitoring/health-check.yml
probe deployment/smoke-tests.yml
probe maintenance/cleanup.yml
```

### 3. Use Configuration Files

Keep secrets and environment-specific values in separate files:

```bash
probe workflow.yml,configs/production.yml
```

### 4. Validate Before Production

Always test with verbose mode first:

```bash
# Test thoroughly
probe -v new-workflow.yml,test-config.yml

# Deploy to production
probe new-workflow.yml,production-config.yml
```

## What's Next?

Now that you've mastered the CLI basics, you can:

1. **[Explore How-tos](../../how-tos/)** - Learn specific patterns and use cases
2. **[Check the Reference](../../reference/)** - Deep dive into all available options
3. **[Try the Tutorials](../../tutorials/)** - Follow step-by-step guides for common scenarios

The CLI is your gateway to Probe's power. With these basics mastered, you're ready to build sophisticated monitoring and automation workflows.