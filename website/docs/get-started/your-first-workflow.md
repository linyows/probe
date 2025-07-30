---
title: Your First Workflow
description: Learn to build practical workflows with real-world examples
weight: 40
---

# Your First Workflow

Now that you understand the [core concepts](../understanding-probe/), let's build a practical workflow from scratch. This guide will walk you through creating a comprehensive monitoring workflow for a web application.

## The Scenario

We'll create a workflow that monitors a complete web application stack:

1. **Frontend**: Check if the web application loads correctly
2. **API**: Verify that the REST API is responding
3. **Database**: Test database connectivity through the API
4. **External Services**: Check third-party service integrations

## Step 1: Basic Structure

Let's start with the basic workflow structure:

```yaml
name: Web Application Health Check
description: Comprehensive monitoring for our web application stack

jobs:
  # We'll add jobs here
```

## Step 2: Frontend Monitoring

Add a job to check the frontend application:

```yaml
name: Web Application Health Check
description: Comprehensive monitoring for our web application stack

jobs:
  frontend-check:
    name: Frontend Application Check
    steps:
      - name: Check Homepage
        action: http
        with:
          url: https://myapp.example.com
          method: GET
          headers:
            User-Agent: "Probe Health Check"
        test: res.status == 200 && res.time < 3000
        outputs:
          homepage_response_time: res.time

      - name: Check Critical Page
        action: http
        with:
          url: https://myapp.example.com/dashboard
          method: GET
        test: res.status == 200 || res.status == 302

      - name: Report Frontend Status
        echo: "✅ Frontend is healthy ({{outputs.homepage_response_time}}ms)"
```

## Step 3: API Monitoring

Add a separate job for API monitoring:

```yaml
  api-check:
    name: API Health Check
    steps:
      - name: Check API Health Endpoint
        id: health-check
        action: http
        with:
          url: https://api.myapp.example.com/health
          method: GET
          headers:
            Accept: "application/json"
        test: res.status == 200 && res.json.status == "healthy"
        outputs:
          api_version: res.json.version
          database_status: res.json.database

      - name: Test User Authentication
        action: http
        with:
          url: https://api.myapp.example.com/auth/login
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "username": "healthcheck",
              "password": "{{env.HEALTH_CHECK_PASSWORD}}"
            }
        test: res.status == 200 && res.json.token != null
        outputs:
          auth_token: res.json.token

      - name: Test Authenticated Endpoint
        action: http
        with:
          url: https://api.myapp.example.com/user/profile
          method: GET
          headers:
            Authorization: "Bearer {{outputs.auth_token}}"
        test: res.status == 200

      - name: Report API Status
        echo: "✅ API v{{outputs.api_version}} is healthy"
```

## Step 4: Adding Dependencies

Make the API check depend on successful frontend check:

```yaml
  api-check:
    name: API Health Check
    needs: [frontend-check]  # Wait for frontend to be healthy
    steps:
      # ... existing steps
```

## Step 5: External Service Checks

Add checks for external services:

```yaml
  external-services:
    name: External Services Check
    steps:
      - name: Check Email Service
        action: http
        with:
          url: https://api.sendgrid.com/v3/mail/send
          method: POST
          headers:
            Authorization: "Bearer {{env.SENDGRID_API_KEY}}"
            Content-Type: "application/json"
          body: |
            {
              "from": {"email": "health@myapp.example.com"},
              "subject": "Health Check Test",
              "content": [{"type": "text/plain", "value": "Test"}],
              "personalizations": [{"to": [{"email": "test@myapp.example.com"}]}]
            }
        test: res.status == 202

      - name: Check Payment Gateway
        action: http
        with:
          url: https://api.stripe.com/v1/charges
          method: GET
          headers:
            Authorization: "Bearer {{env.STRIPE_SECRET_KEY}}"
        test: res.status == 200

      - name: Report External Services
        echo: "✅ All external services are responding"
```

## Step 6: Error Handling and Notifications

Add error handling and notification logic:

```yaml
  notification:
    name: Send Notifications
    needs: [frontend-check, api-check, external-services]
    steps:
      - name: Success Notification
        if: jobs.frontend-check.success && jobs.api-check.success && jobs.external-services.success
        echo: |
          🎉 All systems are healthy!
          
          Frontend: ✅ ({{outputs.frontend-check.homepage_response_time}}ms)
          API: ✅ v{{outputs.api-check.api_version}}
          External Services: ✅
          
          Monitoring completed at {{unixtime()}}

      - name: Failure Notification
        if: jobs.frontend-check.failed || jobs.api-check.failed || jobs.external-services.failed
        echo: |
          🚨 ALERT: System health check failed!
          
          Frontend: {{jobs.frontend-check.success ? "✅" : "❌"}}
          API: {{jobs.api-check.success ? "✅" : "❌"}}
          External Services: {{jobs.external-services.success ? "✅" : "❌"}}
          
          Please investigate immediately.
```

## Complete Workflow

Here's the complete workflow file (`health-check.yml`):

```yaml
name: Web Application Health Check
description: Comprehensive monitoring for our web application stack

jobs:
  frontend-check:
    name: Frontend Application Check
    steps:
      - name: Check Homepage
        action: http
        with:
          url: https://myapp.example.com
          method: GET
          headers:
            User-Agent: "Probe Health Check"
        test: res.status == 200 && res.time < 3000
        outputs:
          homepage_response_time: res.time

      - name: Check Critical Page
        action: http
        with:
          url: https://myapp.example.com/dashboard
          method: GET
        test: res.status == 200 || res.status == 302

      - name: Report Frontend Status
        echo: "✅ Frontend is healthy ({{outputs.homepage_response_time}}ms)"

  api-check:
    name: API Health Check
    needs: [frontend-check]
    steps:
      - name: Check API Health Endpoint
        id: health-check
        action: http
        with:
          url: https://api.myapp.example.com/health
          method: GET
          headers:
            Accept: "application/json"
        test: res.status == 200 && res.json.status == "healthy"
        outputs:
          api_version: res.json.version
          database_status: res.json.database

      - name: Test User Authentication
        action: http
        with:
          url: https://api.myapp.example.com/auth/login
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "username": "healthcheck",
              "password": "{{env.HEALTH_CHECK_PASSWORD}}"
            }
        test: res.status == 200 && res.json.token != null
        outputs:
          auth_token: res.json.token

      - name: Test Authenticated Endpoint
        action: http
        with:
          url: https://api.myapp.example.com/user/profile
          method: GET
          headers:
            Authorization: "Bearer {{outputs.auth_token}}"
        test: res.status == 200

      - name: Report API Status
        echo: "✅ API v{{outputs.api_version}} is healthy"

  external-services:
    name: External Services Check
    steps:
      - name: Check Email Service
        action: http
        with:
          url: https://api.sendgrid.com/v3/mail/send
          method: POST
          headers:
            Authorization: "Bearer {{env.SENDGRID_API_KEY}}"
            Content-Type: "application/json"
          body: |
            {
              "from": {"email": "health@myapp.example.com"},
              "subject": "Health Check Test",
              "content": [{"type": "text/plain", "value": "Test"}],
              "personalizations": [{"to": [{"email": "test@myapp.example.com"}]}]
            }
        test: res.status == 202

      - name: Check Payment Gateway
        action: http
        with:
          url: https://api.stripe.com/v1/charges
          method: GET
          headers:
            Authorization: "Bearer {{env.STRIPE_SECRET_KEY}}"
        test: res.status == 200

      - name: Report External Services
        echo: "✅ All external services are responding"

  notification:
    name: Send Notifications
    needs: [frontend-check, api-check, external-services]
    steps:
      - name: Success Notification
        if: jobs.frontend-check.success && jobs.api-check.success && jobs.external-services.success
        echo: |
          🎉 All systems are healthy!
          
          Frontend: ✅ ({{outputs.frontend-check.homepage_response_time}}ms)
          API: ✅ v{{outputs.api-check.api_version}}
          External Services: ✅
          
          Monitoring completed at {{unixtime()}}

      - name: Failure Notification
        if: jobs.frontend-check.failed || jobs.api-check.failed || jobs.external-services.failed
        echo: |
          🚨 ALERT: System health check failed!
          
          Frontend: {{jobs.frontend-check.success ? "✅" : "❌"}}
          API: {{jobs.api-check.success ? "✅" : "❌"}}
          External Services: {{jobs.external-services.success ? "✅" : "❌"}}
          
          Please investigate immediately.
```

## Running the Workflow

### Set Environment Variables

First, set up your environment variables:

```bash
export HEALTH_CHECK_PASSWORD="your-test-password"
export SENDGRID_API_KEY="your-sendgrid-key"
export STRIPE_SECRET_KEY="your-stripe-key"
```

### Execute the Workflow

Run the workflow:

```bash
probe health-check.yml
```

### Use Verbose Mode for Debugging

For detailed output during development:

```bash
probe -v health-check.yml
```

## Making It Production-Ready

### 1. Environment-Specific Configuration

Create environment-specific config files:

**production.yml:**
```yaml
# Override URLs for production
variables:
  frontend_url: https://app.mycompany.com
  api_url: https://api.mycompany.com
```

**staging.yml:**
```yaml
# Override URLs for staging
variables:
  frontend_url: https://staging.mycompany.com
  api_url: https://api-staging.mycompany.com
```

Run with environment-specific config:
```bash
probe health-check.yml,production.yml
```

### 2. Add Retry Logic

```yaml
- name: Check Critical Service
  action: http
  with:
    url: https://critical-service.example.com
    method: GET
    retry_count: 3
    retry_delay: 5s
  test: res.status == 200
```

### 3. Set up Monitoring Schedule

Use cron to run regularly:
```bash
# Add to crontab - run every 5 minutes
*/5 * * * * /usr/local/bin/probe /path/to/health-check.yml
```

## What You've Learned

In this guide, you've learned how to:

- ✅ Structure a multi-job workflow
- ✅ Use job dependencies with `needs`
- ✅ Pass data between steps using `outputs`
- ✅ Handle authentication in API calls
- ✅ Implement conditional logic with `if`
- ✅ Use environment variables for configuration
- ✅ Create comprehensive error handling
- ✅ Merge configuration files for different environments

## Next Steps

Ready to dive deeper? Here are your next steps:

1. **[Master the CLI](../cli-basics/)** - Learn all command-line options
2. **[Explore How-tos](../../how-tos/)** - See specific use cases and patterns
3. **[Browse the Reference](../../reference/)** - Deep dive into all available features

The workflow you've built is a solid foundation. You can extend it by adding more checks, integrating with monitoring systems, or customizing it for your specific application stack.