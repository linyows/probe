# SSH Action

The SSH Action allows you to connect to remote servers via SSH and execute commands. It's ideal for deployment, remote monitoring, and server management tasks.

## Basic Usage

```yaml
- name: Check server status
  uses: ssh
  with:
    host: "example.com"
    user: "deploy"
    password: "your_password"
    cmd: "uptime && df -h"
  test: res.code == 0
```

## Parameters

### Required Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `host` | string | Target hostname or IP address |
| `user` | string | SSH username |
| `cmd` | string | Command to execute |

### Authentication Parameters (one required)

| Parameter | Type | Description |
|-----------|------|-------------|
| `password` | string | Password authentication |
| `key_file` | string | Private key file path (supports `~` expansion) |
| `key_passphrase` | string | Private key passphrase (for encrypted keys) |

### Optional Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `port` | int | 22 | SSH port number |
| `timeout` | string | "30s" | Command execution timeout |
| `workdir` | string | - | Working directory on remote server |
| `strict_host_check` | bool | true | Strict host key verification |
| `known_hosts` | string | `~/.ssh/known_hosts` | Known hosts file path |

### Environment Variables

Set environment variables using the `env__` prefix:

```yaml
with:
  env__NODE_ENV: production
  env__APP_VERSION: v1.2.3
```

## Return Values

The SSH Action returns the following values:

| Field | Type | Description |
|-------|------|-------------|
| `res.code` | int | Exit code (0 = success) |
| `res.stdout` | string | Standard output |
| `res.stderr` | string | Standard error output |
| `res.status` | int | Execution status (0 = success, 1 = failure) |

## Examples

### Password Authentication

```yaml
- name: Login with password
  uses: ssh
  with:
    host: "server.example.com"
    port: 22
    user: "admin"
    password: "secure_password"
    cmd: "systemctl status nginx"
    timeout: "60s"
  test: res.code == 0
```

### Public Key Authentication

```yaml
- name: Login with SSH key
  uses: ssh
  with:
    host: "prod.example.com"
    user: "deploy"
    key_file: "~/.ssh/deploy_key"
    cmd: "git pull origin main && systemctl restart app"
    workdir: "/opt/myapp"
    timeout: "300s"
  test: res.code == 0
```

### Encrypted Private Key

```yaml
- name: Login with passphrase-protected key
  uses: ssh
  with:
    host: "secure.example.com"
    user: "admin"
    key_file: "~/.ssh/id_rsa"
    key_passphrase: "key_passphrase"
    cmd: "docker ps"
  test: res.code == 0
```

### Using Environment Variables

```yaml
- name: Deploy with environment variables
  uses: ssh
  with:
    host: "app.example.com"
    user: "deploy"
    key_file: "~/.ssh/deploy_key"
    cmd: |
      echo "Deploying version: $APP_VERSION"
      echo "Environment: $DEPLOY_ENV"
      ./deploy.sh
    env__APP_VERSION: "v2.1.0"
    env__DEPLOY_ENV: "production"
    workdir: "/opt/myapp"
    timeout: "600s"
  test: res.code == 0 && contains(res.stdout, "Deploy completed")
```

### Multi-line Commands

```yaml
- name: Collect system information
  uses: ssh
  with:
    host: "monitor.example.com"
    user: "admin"
    password: "admin_pass"
    cmd: |
      echo "=== System Information ==="
      uname -a
      echo "=== Disk Usage ==="
      df -h
      echo "=== Memory Usage ==="
      free -h
      echo "=== Process List ==="
      ps aux | head -10
    timeout: "120s"
  test: res.code == 0
  echo: "{{res.stdout}}"
```

### Error Handling

```yaml
- name: Restart service with error handling
  uses: ssh
  with:
    host: "app.example.com"
    user: "deploy"
    key_file: "~/.ssh/deploy_key"
    cmd: |
      if systemctl is-active --quiet myapp; then
        echo "Stopping service..."
        systemctl stop myapp
      fi
      echo "Starting service..."
      systemctl start myapp
      systemctl is-active myapp
    timeout: "180s"
  test: |
    res.code == 0 && 
    contains(res.stdout, "active") &&
    !contains(res.stderr, "failed")
```

## Security Settings

### Host Key Verification

```yaml
- name: Strict host key verification
  uses: ssh
  with:
    host: "secure.example.com"
    user: "admin"
    key_file: "~/.ssh/id_rsa"
    cmd: "whoami"
    strict_host_check: true
    known_hosts: "~/.ssh/known_hosts"
  test: res.code == 0
```

### Disable Host Key Verification (Development Only)

```yaml
- name: Development environment connection
  uses: ssh
  with:
    host: "dev.example.com"
    user: "developer"
    password: "dev_password"
    cmd: "npm test"
    strict_host_check: false  # Use only in development
  test: res.code == 0
```

## Best Practices

### 1. Authentication Method Selection

**Prefer public key authentication**:
```yaml
# Recommended
key_file: "~/.ssh/deploy_key"

# Avoid if possible
password: "plain_password"
```

### 2. Timeout Configuration

Set appropriate timeout values:
```yaml
# Short commands
timeout: "30s"

# Long operations like deployment
timeout: "600s"
```

### 3. Error Handling

```yaml
test: |
  res.code == 0 && 
  !contains(res.stderr, "error") &&
  !contains(res.stderr, "failed")
```

### 4. Log Safety

Password and private key information is automatically excluded from logs.

### 5. Environment Variable Usage

Use environment variables for sensitive information:
```yaml
vars:
  ssh_password: "{{env.SSH_PASSWORD}}"
  deploy_key: "{{env.DEPLOY_KEY_PATH}}"

steps:
- uses: ssh
  with:
    password: "{{vars.ssh_password}}"
    key_file: "{{vars.deploy_key}}"
```

## Limitations

### No SSH Config File Support

To maintain portability and reproducibility, SSH config files (`~/.ssh/config`) are not supported. All connection parameters must be explicitly defined in the workflow.

### Interactive Commands

Interactive commands that require user input are not supported. All commands must run non-interactively.

## Troubleshooting

### Connection Errors

**Host key verification failed**:
```yaml
# Check known_hosts
known_hosts: "~/.ssh/known_hosts"

# Or disable for development
strict_host_check: false
```

**Authentication failed**:
```yaml
# Check key file path and permissions
key_file: "/home/user/.ssh/id_rsa"  # Absolute path
# or
key_file: "~/.ssh/id_rsa"          # Tilde expansion

# Add passphrase if needed
key_passphrase: "your_passphrase"
```

**Timeout errors**:
```yaml
# Increase timeout
timeout: "300s"
```

### Consecutive Execution Issues

If you experience issues with consecutive connections to the same host, add appropriate delays:

```yaml
- name: First connection
  uses: ssh
  with:
    cmd: "command1"
  
- name: Brief wait
  uses: hello
  with:
    name: "wait"
  wait: 1s
  
- name: Second connection
  uses: ssh
  with:
    cmd: "command2"
```

## Related Actions

- [Shell Action](./shell.md) - Local command execution
- [HTTP Action](./http.md) - REST API communication
- [Environment Variables](../environment-variables.md)