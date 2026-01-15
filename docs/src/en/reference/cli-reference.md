# CLI Reference

This page provides complete documentation for the Probe command-line interface, including all commands, options, and usage patterns.

## Basic Usage

```bash
probe [workflow-file] [options]
```

## Command Syntax

### Basic Command

Execute a single workflow file:

```bash
probe workflow.yml
```

### File Merging

Execute workflows with configuration merging:

```bash
probe base.yml,environment.yml,overrides.yml
```

Multiple files are merged from left to right, with later files overriding earlier ones.

### Positional Arguments

#### `workflow-path`

**Type:** String (required)  
**Description:** Path to the workflow YAML file, or comma-separated list of files for merging

**Examples:**
```bash
# Single file
probe workflow.yml

# Multiple files (merging)
probe base.yml,production.yml

# Relative paths
probe ./workflows/api-test.yml

# Absolute paths
probe /home/user/workflows/monitoring.yml
```

## Command-Line Options

### `-v, --verbose`

**Type:** Boolean flag  
**Default:** `false`  
**Description:** Enable verbose output showing detailed execution information

**Example:**
```bash
probe -v workflow.yml
probe --verbose workflow.yml
```

**Verbose Output Includes:**
- Step-by-step execution details
- HTTP request/response information
- Template evaluation results
- Timing information
- Debug messages

### `-h, --help`

**Type:** Boolean flag  
**Description:** Show command usage help and exit

**Example:**
```bash
probe -h
probe --help
```

### `--version`

**Type:** Boolean flag
**Description:** Show version information and exit

**Example:**
```bash
probe --version
```

**Output Format:**
```
Probe Version 1.2.3
Build: abc1234
Go Version: go1.20.1
```

### `--graph`

**Type:** Boolean flag
**Default:** `false`
**Description:** Display job dependency graph as ASCII art without executing the workflow

**Example:**
```bash
probe --graph workflow.yml
```

**Output Example:**
```
     [Setup]
        ↓
     [Build]
      ↓   ↓
[Test A] [Test B]
      ↓   ↓
    [Deploy]
```

This is useful for:
- Visualizing workflow structure before execution
- Debugging job dependency configurations
- Documentation and communication

## Environment Variables

The following environment variables affect Probe's behavior:

### `PROBE_CONFIG`

**Type:** String  
**Description:** Path to default configuration file
**Default:** None

```bash
export PROBE_CONFIG=/etc/probe/default.yml
probe workflow.yml  # Will merge with default config
```

### `PROBE_LOG_LEVEL`

**Type:** String  
**Values:** `debug`, `info`, `warn`, `error`  
**Default:** `info`  
**Description:** Set logging level

```bash
export PROBE_LOG_LEVEL=debug
probe workflow.yml
```

### `PROBE_NO_COLOR`

**Type:** Boolean  
**Values:** `true`, `false`, `1`, `0`  
**Default:** `false`  
**Description:** Disable colored output

```bash
export PROBE_NO_COLOR=true
probe workflow.yml
```

### `PROBE_TIMEOUT`

**Type:** Duration  
**Default:** `300s`  
**Description:** Global timeout for workflow execution

```bash
export PROBE_TIMEOUT=600s
probe workflow.yml
```

### `PROBE_PLUGIN_DIR`

**Type:** String  
**Default:** `~/.probe/plugins`  
**Description:** Directory containing custom plugins

```bash
export PROBE_PLUGIN_DIR=/usr/local/lib/probe/plugins
probe workflow.yml
```

## Usage Examples

### Basic Workflow Execution

```bash
# Run a simple health check
probe health-check.yml

# Run with verbose output
probe -v health-check.yml
```

### Environment-Specific Execution

```bash
# Development environment
probe workflow.yml,dev.yml

# Staging environment
probe workflow.yml,staging.yml

# Production environment
probe workflow.yml,prod.yml
```

### Complex Configuration Merging

```bash
# Layer multiple configurations
probe base.yml,region-us.yml,environment-prod.yml,team-overrides.yml
```

### CI/CD Integration

```bash
#!/bin/bash
# deployment-test.sh

set -e

echo "Running deployment validation..."
probe deployment-validation.yml,${ENVIRONMENT}.yml

echo "Running smoke tests..."
probe smoke-tests.yml,${ENVIRONMENT}.yml

echo "All tests passed!"
```

### Docker Integration

```bash
# Run Probe in Docker container
docker run --rm -v $(pwd):/workspace \
  -e API_TOKEN=$API_TOKEN \
  probe:latest workflow.yml

# Docker Compose service
version: '3.8'
services:
  probe:
    image: probe:latest
    volumes:
      - ./workflows:/workflows
    environment:
      - API_TOKEN
      - ENVIRONMENT=production
    command: /workflows/monitoring.yml,/workflows/production.yml
```

### Scheduled Execution

```bash
# Crontab entry for regular monitoring
# Run every 5 minutes
*/5 * * * * /usr/local/bin/probe /opt/workflows/monitoring.yml >> /var/log/probe.log 2>&1

# Systemd timer unit
[Unit]
Description=Probe Monitoring
Requires=probe-monitoring.timer

[Service]
Type=oneshot
ExecStart=/usr/local/bin/probe /opt/workflows/monitoring.yml
User=probe
Group=probe

[Install]
WantedBy=multi-user.target
```

## Exit Codes

Probe uses standard exit codes to indicate execution results:

| Exit Code | Meaning | Description |
|-----------|---------|-------------|
| `0` | Success | All workflow jobs completed successfully |
| `1` | General Error | Workflow failed due to test failures or action errors |
| `2` | Configuration Error | Invalid YAML syntax or configuration |
| `3` | File Not Found | Workflow file(s) could not be found |
| `4` | Permission Error | Insufficient permissions to read files or execute |
| `5` | Network Error | Network connectivity issues |
| `6` | Timeout Error | Workflow execution exceeded timeout |
| `7` | Plugin Error | Plugin loading or execution failed |

### Exit Code Examples

```bash
# Check exit code in scripts
probe workflow.yml
if [ $? -eq 0 ]; then
  echo "Workflow succeeded"
else
  echo "Workflow failed with exit code $?"
fi

# Use in CI/CD pipelines
probe integration-tests.yml || exit 1
```

## Configuration File Search Order

Probe searches for configuration files in the following order:

1. **Command line argument** (explicit file path)
2. **Current directory** (`./probe.yml`, `./probe.yaml`)  
3. **Home directory** (`~/.probe.yml`, `~/.probe.yaml`)
4. **System directory** (`/etc/probe/probe.yml`)
5. **Environment variable** (`$PROBE_CONFIG`)

### Example Search

```bash
# Probe will search in this order:
# 1. ./my-workflow.yml (command line)
# 2. ./probe.yml (current directory)
# 3. ~/.probe.yml (home directory)  
# 4. /etc/probe/probe.yml (system)
# 5. $PROBE_CONFIG (environment)

probe my-workflow.yml
```

## Performance and Resource Usage

### Memory Usage

- **Base memory:** ~10MB for Probe runtime
- **Per workflow:** ~1-5MB depending on complexity
- **Per action:** ~0.1-1MB depending on response size

### Execution Timing

```bash
# Time workflow execution
time probe workflow.yml

# Detailed timing with verbose mode
probe -v workflow.yml 2>&1 | grep "Execution time"
```

### Concurrent Execution

Probe executes jobs in parallel when possible:

```bash
# Jobs without dependencies run concurrently
# Maximum concurrency is typically limited by system resources
# Use verbose mode to see execution pattern
probe -v parallel-workflow.yml
```

## Troubleshooting Commands

### Debug Information

```bash
# Maximum debug output
PROBE_LOG_LEVEL=debug probe -v workflow.yml

# Check version and build info
probe --version

# Validate workflow syntax without execution
probe --dry-run workflow.yml  # (if supported)
```

### Common Issues

**File not found:**
```bash
probe: error: workflow file 'missing.yml' not found
# Check file path and permissions
ls -la missing.yml
```

**Permission denied:**
```bash
probe: error: permission denied reading 'workflow.yml'
# Fix file permissions
chmod 644 workflow.yml
```

**YAML syntax error:**
```bash
probe: error: YAML syntax error at line 15
# Validate YAML syntax
yaml-validator workflow.yml
```

## Integration Examples

### GitHub Actions

```yaml
name: Probe Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Probe
        run: |
          curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o probe
          chmod +x probe
          sudo mv probe /usr/local/bin/
      
      - name: Run Tests
        env:
          API_TOKEN: ${{ secrets.API_TOKEN }}
        run: probe workflow.yml,${GITHUB_REF##*/}.yml
```

### GitLab CI

```yaml
stages:
  - test

probe-test:
  stage: test
  image: alpine:latest
  before_script:
    - apk add --no-cache curl
    - curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o /usr/local/bin/probe
    - chmod +x /usr/local/bin/probe
  script:
    - probe workflow.yml,$CI_ENVIRONMENT_NAME.yml
  variables:
    API_TOKEN: $API_TOKEN
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any
    
    environment {
        API_TOKEN = credentials('api-token')
        PROBE_LOG_LEVEL = 'info'
    }
    
    stages {
        stage('Install Probe') {
            steps {
                sh '''
                    curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o probe
                    chmod +x probe
                    sudo mv probe /usr/local/bin/
                '''
            }
        }
        
        stage('Run Tests') {
            steps {
                sh 'probe workflow.yml,${BRANCH_NAME}.yml'
            }
        }
    }
    
    post {
        always {
            archiveArtifacts artifacts: '*.log', allowEmptyArchive: true
        }
    }
}
```

## Advanced Usage Patterns

### Configuration Templates

```bash
# Use environment variables in file paths
export ENV=production
probe workflow.yml,configs/${ENV}.yml

# Dynamic file selection
WORKFLOW_FILE=$([ "$ENV" = "prod" ] && echo "prod-workflow.yml" || echo "dev-workflow.yml")
probe $WORKFLOW_FILE
```

### Batch Execution

```bash
# Run multiple workflows
for workflow in workflows/*.yml; do
  echo "Running $workflow..."
  probe "$workflow" || echo "Failed: $workflow"
done

# Parallel execution
find workflows/ -name "*.yml" | xargs -P 4 -I {} probe {}
```

### Monitoring Integration

```bash
# Integration with monitoring systems
probe monitoring.yml
RESULT=$?

if [ $RESULT -ne 0 ]; then
  # Send alert to monitoring system
  curl -X POST https://monitoring.example.com/alert \
    -H "Content-Type: application/json" \
    -d '{"message": "Probe workflow failed", "exit_code": '$RESULT'}'
fi
```

## See Also

- **[YAML Configuration](../yaml-configuration/)** - Complete YAML syntax reference
- **[Actions Reference](../actions-reference/)** - Built-in actions and parameters
- **[Environment Variables](../environment-variables/)** - All supported environment variables
- **[How-tos](../../how-tos/)** - Practical usage examples
- **[Error Handling Strategies](../../how-tos/error-handling-strategies/)** - Common issues and solutions
