---
title: Environment Variables Reference
description: Complete reference for all Probe environment variables
weight: 50
---

# Environment Variables Reference

This page provides comprehensive documentation for all environment variables that control Probe's behavior, configuration, and execution.

## Overview

Probe uses environment variables for:

- **Runtime Configuration** - Control logging, timeouts, and behavior
- **Authentication** - API keys, tokens, and credentials
- **Integration** - CI/CD systems, monitoring tools
- **Customization** - Plugin directories, default configurations

Environment variables can be set at the system level, in CI/CD pipelines, or defined within workflow files using the `env` section.

## Runtime Configuration Variables

### `PROBE_LOG_LEVEL`

**Type:** String  
**Values:** `debug`, `info`, `warn`, `error`  
**Default:** `info`  
**Description:** Controls the verbosity of Probe's logging output

```bash
# Enable debug logging
export PROBE_LOG_LEVEL=debug
probe workflow.yml

# Reduce to warnings and errors only
export PROBE_LOG_LEVEL=warn
probe workflow.yml
```

**Output Examples:**

```bash
# info level (default)
2023-09-01 12:30:00 [INFO] Starting workflow: API Health Check
2023-09-01 12:30:01 [INFO] Job 'health-check' completed successfully

# debug level
2023-09-01 12:30:00 [DEBUG] Loading workflow file: workflow.yml
2023-09-01 12:30:00 [DEBUG] Parsing YAML configuration
2023-09-01 12:30:00 [INFO] Starting workflow: API Health Check
2023-09-01 12:30:00 [DEBUG] Starting job: health-check
2023-09-01 12:30:00 [DEBUG] Executing step: Check API Status
2023-09-01 12:30:01 [DEBUG] HTTP request: GET https://api.example.com/health
2023-09-01 12:30:01 [DEBUG] HTTP response: 200 OK (345ms)
2023-09-01 12:30:01 [INFO] Job 'health-check' completed successfully
```

### `PROBE_NO_COLOR`

**Type:** Boolean  
**Values:** `true`, `false`, `1`, `0`  
**Default:** `false`  
**Description:** Disables colored output in terminal

```bash
# Disable colors (useful for CI/CD logs)
export PROBE_NO_COLOR=true
probe workflow.yml

# Force color output (override terminal detection)
export PROBE_NO_COLOR=false
probe workflow.yml
```

### `PROBE_TIMEOUT`

**Type:** Duration  
**Default:** `300s` (5 minutes)  
**Description:** Global timeout for entire workflow execution

```bash
# Set 10-minute timeout
export PROBE_TIMEOUT=600s
probe long-running-workflow.yml

# Set 30-second timeout for quick tests
export PROBE_TIMEOUT=30s
probe quick-health-check.yml
```

### `PROBE_CONFIG`

**Type:** String (file path)  
**Default:** None  
**Description:** Path to default configuration file that gets merged with all workflows

```bash
# Use global defaults
export PROBE_CONFIG=/etc/probe/defaults.yml
probe workflow.yml  # Merges with defaults.yml

# User-specific defaults
export PROBE_CONFIG=~/.probe/defaults.yml
probe workflow.yml
```

**Example default configuration file:**

```yaml
# /etc/probe/defaults.yml
env:
  # Common environment variables
  USER_AGENT: "Probe Monitor v1.0"
  DEFAULT_TIMEOUT: "30s"

defaults:
  http:
    timeout: "{{env.DEFAULT_TIMEOUT}}"
    headers:
      User-Agent: "{{env.USER_AGENT}}"
      Accept: "application/json"
```

### `PROBE_PLUGIN_DIR`

**Type:** String (directory path)  
**Default:** `~/.probe/plugins`  
**Description:** Directory containing custom action plugins

```bash
# Use system-wide plugins
export PROBE_PLUGIN_DIR=/usr/local/lib/probe/plugins
probe workflow.yml

# Use project-specific plugins
export PROBE_PLUGIN_DIR=./plugins
probe workflow.yml
```

**Plugin directory structure:**

```
/usr/local/lib/probe/plugins/
├── custom-http/
│   └── custom-http-plugin
├── database/
│   └── db-plugin
└── notification/
    └── notification-plugin
```

## Authentication Variables

### API Authentication

Common patterns for API authentication in workflows:

#### `API_TOKEN` / `API_KEY`

```bash
# Bearer token authentication
export API_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

```yaml
# workflow.yml
env:
  API_TOKEN: "{{env.API_TOKEN}}"

defaults:
  http:
    headers:
      Authorization: "Bearer {{env.API_TOKEN}}"
```

#### `USERNAME` / `PASSWORD`

```bash
# Basic authentication credentials
export API_USERNAME="admin"
export API_PASSWORD="secret123"
```

```yaml
# workflow.yml
env:
  USERNAME: "{{env.API_USERNAME}}"
  PASSWORD: "{{env.API_PASSWORD}}"

steps:
  - name: "Authenticated Request"
    action: http
    with:
      url: "https://api.example.com/protected"
      headers:
        Authorization: "Basic {{base64(env.USERNAME + ':' + env.PASSWORD)}}"
```

### Email Authentication

#### SMTP Configuration

```bash
# Gmail with app password
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT=587
export SMTP_USERNAME="alerts@example.com"
export SMTP_PASSWORD="app-specific-password"

# Office 365
export SMTP_HOST="smtp.office365.com"
export SMTP_PORT=587
export SMTP_USERNAME="alerts@company.com"
export SMTP_PASSWORD="account-password"
```

```yaml
# workflow.yml
defaults:
  smtp:
    host: "{{env.SMTP_HOST}}"
    port: "{{env.SMTP_PORT}}"
    username: "{{env.SMTP_USERNAME}}"
    password: "{{env.SMTP_PASSWORD}}"
    from: "{{env.SMTP_USERNAME}}"
```

## Application Configuration Variables

### Service URLs

```bash
# API endpoints
export API_BASE_URL="https://api.production.com"
export API_VERSION="v2"
export HEALTH_CHECK_URL="${API_BASE_URL}/${API_VERSION}/health"

# Database connections
export DATABASE_URL="postgresql://user:pass@localhost:5432/db"
export REDIS_URL="redis://localhost:6379/0"

# External services
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
export MONITORING_URL="https://monitoring.example.com/api/alerts"
```

### Feature Flags

```bash
# Enable/disable features
export ENABLE_MONITORING=true
export ENABLE_SLACK_NOTIFICATIONS=false
export ENABLE_DETAILED_LOGGING=true
export ENABLE_PERFORMANCE_TRACKING=true
```

```yaml
# workflow.yml
jobs:
  monitoring:
    if: env.ENABLE_MONITORING == "true"
    steps:
      - name: "Performance Check"
        if: env.ENABLE_PERFORMANCE_TRACKING == "true"
        action: http
        with:
          url: "{{env.API_BASE_URL}}/metrics"

  notifications:
    if: env.ENABLE_SLACK_NOTIFICATIONS == "true"
    needs: [monitoring]
    steps:
      - name: "Slack Alert"
        action: http
        with:
          url: "{{env.SLACK_WEBHOOK_URL}}"
          method: "POST"
          body: |
            {
              "text": "Monitoring completed: {{jobs.monitoring.status}}"
            }
```

## CI/CD Integration Variables

### GitHub Actions

```yaml
# .github/workflows/probe.yml
name: API Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Run Probe Tests
        env:
          API_TOKEN: ${{ secrets.API_TOKEN }}
          ENVIRONMENT: ${{ github.ref == 'refs/heads/main' && 'production' || 'staging' }}
          BRANCH_NAME: ${{ github.ref_name }}
          COMMIT_SHA: ${{ github.sha }}
          RUN_ID: ${{ github.run_id }}
        run: probe workflow.yml
```

**Available in workflow:**

```bash
export GITHUB_REPOSITORY="owner/repo"
export GITHUB_REF="refs/heads/main"
export GITHUB_SHA="abc123def456"
export GITHUB_RUN_ID="123456789"
export GITHUB_ACTOR="username"
```

### GitLab CI

```yaml
# .gitlab-ci.yml
probe-test:
  stage: test
  script:
    - probe workflow.yml
  variables:
    API_TOKEN: $API_TOKEN
    ENVIRONMENT: $CI_ENVIRONMENT_NAME
    BRANCH_NAME: $CI_COMMIT_REF_NAME
    COMMIT_SHA: $CI_COMMIT_SHA
    PIPELINE_ID: $CI_PIPELINE_ID
```

**Available in workflow:**

```bash
export CI_PROJECT_NAME="project-name"
export CI_COMMIT_REF_NAME="main"
export CI_COMMIT_SHA="abc123"
export CI_PIPELINE_ID="123456"
export CI_JOB_ID="789012"
```

### Jenkins

```groovy
// Jenkinsfile
pipeline {
    agent any
    environment {
        API_TOKEN = credentials('api-token')
        ENVIRONMENT = "${BRANCH_NAME == 'main' ? 'production' : 'staging'}"
        BUILD_NUMBER = "${BUILD_NUMBER}"
        JOB_NAME = "${JOB_NAME}"
    }
    stages {
        stage('Test') {
            steps {
                sh 'probe workflow.yml'
            }
        }
    }
}
```

**Available in workflow:**

```bash
export BUILD_NUMBER="123"
export JOB_NAME="api-tests"
export WORKSPACE="/var/jenkins_home/workspace/api-tests"
export JENKINS_URL="https://jenkins.example.com"
```

## Environment-Specific Configuration

### Multi-Environment Setup

```bash
# Base configuration (always set)
export API_VERSION="v1"
export DEFAULT_TIMEOUT="30s"

# Environment-specific (set per environment)
case $ENVIRONMENT in
  "production")
    export API_BASE_URL="https://api.prod.com"
    export DATABASE_URL="postgresql://prod-db:5432/app"
    export LOG_LEVEL="warn"
    export ENABLE_MONITORING=true
    ;;
  "staging")  
    export API_BASE_URL="https://api.staging.com"
    export DATABASE_URL="postgresql://staging-db:5432/app"
    export LOG_LEVEL="info"
    export ENABLE_MONITORING=true
    ;;
  "development")
    export API_BASE_URL="http://localhost:3000"
    export DATABASE_URL="postgresql://localhost:5432/app_dev"
    export LOG_LEVEL="debug"
    export ENABLE_MONITORING=false
    ;;
esac
```

### Docker Configuration

```dockerfile
# Dockerfile
FROM alpine:latest

# Install Probe
RUN curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o /usr/local/bin/probe && \
    chmod +x /usr/local/bin/probe

# Set default environment variables
ENV PROBE_LOG_LEVEL=info
ENV PROBE_NO_COLOR=true
ENV PROBE_TIMEOUT=300s

COPY workflows/ /workflows/
WORKDIR /workflows

ENTRYPOINT ["probe"]
CMD ["workflow.yml"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  probe:
    build: .
    environment:
      - API_TOKEN=${API_TOKEN}
      - API_BASE_URL=https://api.example.com
      - ENVIRONMENT=production
      - PROBE_LOG_LEVEL=info
    volumes:
      - ./workflows:/workflows
      - ./reports:/reports
    command: monitoring.yml
```

## Security Considerations

### Sensitive Variables

**Never commit sensitive values to version control:**

```bash
# ❌ Bad: Hardcoded in workflow file
env:
  API_TOKEN: "secret-token-here"

# ✅ Good: Reference environment variable
env:
  API_TOKEN: "{{env.API_TOKEN}}"
```

### Variable Management

```bash
# Use secure secret management
export API_TOKEN=$(aws ssm get-parameter --name "/app/api-token" --with-decryption --query 'Parameter.Value' --output text)
export DB_PASSWORD=$(vault kv get -field=password secret/database)

# Use temporary files for complex secrets
echo "$GOOGLE_CREDENTIALS_JSON" > /tmp/gcp-key.json
export GOOGLE_APPLICATION_CREDENTIALS=/tmp/gcp-key.json
```

### Environment Isolation

```bash
# Prefix variables by environment to avoid conflicts
export PROD_API_TOKEN="prod-token"
export STAGING_API_TOKEN="staging-token"  
export DEV_API_TOKEN="dev-token"

# Use environment-specific selection
export API_TOKEN="${ENVIRONMENT}_API_TOKEN"
export API_TOKEN="${!API_TOKEN}"  # Indirect variable expansion
```

## Debugging Environment Variables

### Viewing Available Variables

```bash
# List all environment variables
env | grep -E '^(PROBE_|API_|SMTP_)' | sort

# Check specific variables
echo "API_TOKEN: $API_TOKEN"
echo "PROBE_LOG_LEVEL: $PROBE_LOG_LEVEL"

# Debug in workflow (be careful with sensitive data)
probe -v workflow.yml 2>&1 | grep -i "environment"
```

### Template Debugging

```yaml
# workflow.yml - Debug environment variable expansion
steps:
  - name: "Debug Environment"
    action: hello
    with:
      message: |
        Environment Debug:
        API_URL: {{env.API_BASE_URL}}
        Environment: {{env.ENVIRONMENT}}
        Timeout: {{env.DEFAULT_TIMEOUT || "not set"}}
        
  - name: "Test Variable Access"
    echo: |
      Available variables:
      {{range $key, $value := env}}
        {{$key}}: {{$value}}
      {{end}}
```

## Common Patterns

### Configuration Cascading

```bash
# System defaults
export PROBE_CONFIG=/etc/probe/system.yml

# Team defaults  
export TEAM_CONFIG=/opt/team/defaults.yml

# Project-specific
export PROJECT_CONFIG=./probe-defaults.yml

# Runtime execution with cascading
probe ${PROBE_CONFIG},${TEAM_CONFIG},${PROJECT_CONFIG},workflow.yml
```

### Dynamic Configuration

```bash
# Generate configuration based on environment
WORKFLOW_FILE="workflow-${ENVIRONMENT}.yml"
if [[ ! -f "$WORKFLOW_FILE" ]]; then
  WORKFLOW_FILE="workflow.yml"
fi

probe "$WORKFLOW_FILE"
```

### Conditional Execution

```bash
# Skip certain jobs based on environment
export SKIP_PERFORMANCE_TESTS=$([[ "$ENVIRONMENT" == "development" ]] && echo "true" || echo "false")
export ENABLE_SLACK_ALERTS=$([[ "$ENVIRONMENT" == "production" ]] && echo "true" || echo "false")
```

```yaml
# workflow.yml
jobs:
  performance-tests:
    if: env.SKIP_PERFORMANCE_TESTS != "true"
    # Performance test steps
    
  alerts:
    if: env.ENABLE_SLACK_ALERTS == "true"
    needs: [performance-tests]
    # Alert steps
```

## See Also

- **[CLI Reference](../cli-reference/)** - Command-line environment variable usage
- **[YAML Configuration](../yaml-configuration/)** - Using environment variables in workflows
- **[How-tos: Environment Management](../../how-tos/environment-management/)** - Practical environment management strategies
- **[Concepts: File Merging](../../concepts/file-merging/)** - Configuration composition patterns