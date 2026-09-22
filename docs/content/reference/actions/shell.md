# Shell Action

The `shell` action executes shell commands and scripts securely, providing comprehensive output capture and error handling.

## Basic Syntax

```yaml
steps:
  - name: "Execute Build Script"
    uses: shell
    with:
      cmd: "npm run build"
    test: res.code == 0
```

## Parameters

### `cmd` (required)

**Type:** String  
**Description:** The shell command to execute  
**Supports:** Template expressions

```yaml
vars:
  api_url: "{{API_URL}}"

with:
  cmd: "echo 'Hello World'"
  cmd: "npm run {{vars.build_script}}"
  cmd: "curl -f {{vars.api_url}}/health"
```

### `shell` (optional)

**Type:** String  
**Default:** `/bin/sh`  
**Allowed Values:** `/bin/sh`, `/bin/bash`, `/bin/zsh`, `/bin/dash`, `/usr/bin/sh`, `/usr/bin/bash`, `/usr/bin/zsh`, `/usr/bin/dash`

```yaml
with:
  cmd: "echo $0"
  shell: "/bin/bash"
```

### `workdir` (optional)

**Type:** String  
**Description:** Working directory for command execution (must be absolute path)  
**Supports:** Template expressions

```yaml
with:
  cmd: "pwd && ls -la"
  workdir: "/app/src"
  workdir: "{{vars.project_path}}"
```

### `timeout` (optional)

**Type:** String or Duration  
**Default:** `30s`  
**Format:** Go duration format (`30s`, `5m`, `1h`) or plain number (seconds)

```yaml
with:
  cmd: "npm test"
  timeout: "10m"
  timeout: "300"  # 300 seconds
```

### `env` (optional)

**Type:** Object  
**Description:** Environment variables to set for the command  
**Supports:** Template expressions in values

```yaml
vars:
  production_api_url: "{{PRODUCTION_API_URL}}"

with:
  cmd: "npm run build"
  env:
    NODE_ENV: "production"
    API_URL: "{{vars.production_api_url}}"
    BUILD_VERSION: "{{vars.version}}"
```

## Response Format

```yaml
res:
  code: 0                    # Exit code (0 = success)
  stdout: "Build successful" # Standard output
  stderr: ""                 # Standard error output

req:
  cmd: "npm run build"       # Original command
  shell: "/bin/sh"          # Shell used
  workdir: "/app"           # Working directory
  timeout: "30s"            # Timeout setting
  env:                      # Environment variables
    NODE_ENV: "production"
```

## Usage Examples

### Basic Command Execution

```yaml
- name: "System Information"
  uses: shell
  with:
    cmd: "uname -a"
  test: res.code == 0
```

### Build and Test Pipeline

```yaml
- name: "Install Dependencies"
  uses: shell
  with:
    cmd: "npm ci"
    workdir: "/app"
    timeout: "5m"
  test: res.code == 0

- name: "Run Tests"
  uses: shell
  with:
    cmd: "npm test"
    workdir: "/app"
    env:
      NODE_ENV: "test"
      CI: "true"
  test: res.code == 0 && (res.stdout | contains("All tests passed"))
```

### Environment-specific Deployment

```yaml
vars:
  target_env: "{{TARGET_ENV}}"
  deploy_key: "{{DEPLOY_KEY}}"

- name: "Deploy to Environment"
  uses: shell
  with:
    cmd: "./deploy.sh {{vars.target_env}}"
    workdir: "/deploy"
    shell: "/bin/bash"
    timeout: "15m"
    env:
      DEPLOY_KEY: "{{vars.deploy_key}}"
      TARGET_ENV: "{{vars.target_env}}"
  test: res.code == 0
```

### Error Handling and Debugging

```yaml
- name: "Service Health Check"
  uses: shell
  with:
    cmd: "curl -f http://localhost:8080/health || echo 'Service down'"
  test: res.code == 0 || (res.stderr | contains("Service down"))

- name: "Debug Failed Build"
  uses: shell
  with:
    cmd: "npm run build:debug"
  # Allow failure to capture debug output
  outputs:
    debug_info: res.stderr
```

## Security Features

The shell action implements several security measures:

- **Shell Path Restriction**: Only allows approved shell executables
- **Working Directory Validation**: Ensures absolute paths and directory existence
- **Timeout Protection**: Prevents infinite execution
- **Environment Variable Filtering**: Safely handles environment variable passing
- **Output Sanitization**: Safely captures and returns command output

## Error Handling

Common exit codes and their meanings:

- **0**: Success
- **1**: General error
- **2**: Misuse of shell builtins
- **126**: Command cannot execute (permission denied)
- **127**: Command not found
- **130**: Script terminated by Ctrl+C
- **255**: Exit status out of range

```yaml
- name: "Handle Different Exit Codes"
  uses: shell
  with:
    cmd: "some_command_that_might_fail"
  test: |
    res.code == 0 ? true :
    res.code == 127 ? (res.stderr | contains("not found")) :
    res.code < 128
```
