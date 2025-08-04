---
title: Actions Reference
description: Complete reference for all built-in actions and their parameters
weight: 30
---

# Actions Reference

This page provides comprehensive documentation for all built-in Probe actions, including their parameters, response formats, and usage examples.

## Overview

Actions are the building blocks of Probe workflows. They perform specific tasks like making HTTP requests, sending emails, or executing custom logic. All actions return structured response data that can be used in tests and outputs.

### Built-in Actions

- **[http](#http-action)** - Make HTTP/HTTPS requests and validate responses
- **[db](#database-action)** - Execute database queries on MySQL, PostgreSQL, and SQLite
- **[shell](#shell-action)** - Execute shell commands and scripts securely
- **[smtp](#smtp-action)** - Send email notifications and alerts  
- **[hello](#hello-action)** - Simple test action for development and debugging

## HTTP Action

The `http` action performs HTTP/HTTPS requests and provides detailed response information for testing and validation.

### Basic Syntax

```yaml
steps:
  - name: "API Request"
    action: http
    with:
      url: "https://api.example.com/endpoint"
      method: "GET"
    test: res.status == 200
```

### Parameters

#### `url` (required)

**Type:** String  
**Description:** The URL to make the request to  
**Supports:** Template expressions

```yaml
vars:
  api_base_url: "{{API_BASE_URL ?? 'https://api.example.com'}}"

with:
  url: "https://api.example.com/users"
  url: "{{vars.api_base_url}}/v1/health"
  url: "https://api.example.com/users/{{outputs.auth.user_id}}"
```

#### `method` (optional)

**Type:** String  
**Default:** `GET`  
**Values:** `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`

```yaml
with:
  url: "https://api.example.com/users"
  method: "POST"
```

#### `headers` (optional)

**Type:** Object  
**Description:** HTTP headers to include with the request  
**Supports:** Template expressions in values

```yaml
vars:
  api_token: "{{API_TOKEN}}"

with:
  url: "https://api.example.com/users"
  headers:
    Authorization: "Bearer {{vars.api_token}}"
    Content-Type: "application/json"
    User-Agent: "Probe Monitor v1.0"
    X-Request-ID: "{{unixtime()}}"
```

#### `body` (optional)

**Type:** String  
**Description:** Request body content  
**Supports:** Template expressions and multi-line strings

```yaml
# JSON body
vars:
  user_name: "{{USER_NAME}}"
  user_email: "{{USER_EMAIL}}"

with:
  url: "https://api.example.com/users"
  method: "POST"
  headers:
    Content-Type: "application/json"
  body: |
    {
      "name": "{{vars.user_name}}",
      "email": "{{vars.user_email}}",
      "active": true
    }

# Form data
with:
  url: "https://api.example.com/form"
  method: "POST"
  headers:
    Content-Type: "application/x-www-form-urlencoded"
  body: "name={{vars.user_name}}&email={{vars.user_email}}"

# Template expression
with:
  url: "https://api.example.com/users"
  method: "PUT"
  body: "{{outputs.user-data.json | tojson}}"
```

#### `timeout` (optional)

**Type:** Duration  
**Default:** Inherits from `defaults.http.timeout` or `30s`  
**Description:** Request timeout

```yaml
with:
  url: "https://api.example.com/slow-endpoint"
  timeout: "60s"
```

#### `follow_redirects` (optional)

**Type:** Boolean  
**Default:** Inherits from `defaults.http.follow_redirects` or `true`  
**Description:** Whether to follow HTTP redirects

```yaml
with:
  url: "https://example.com/redirect"
  follow_redirects: false
```

#### `verify_ssl` (optional)

**Type:** Boolean  
**Default:** Inherits from `defaults.http.verify_ssl` or `true`  
**Description:** Whether to verify SSL certificates

```yaml
with:
  url: "https://self-signed.example.com/api"
  verify_ssl: false
```

#### `max_redirects` (optional)

**Type:** Integer  
**Default:** Inherits from `defaults.http.max_redirects` or `10`  
**Description:** Maximum number of redirects to follow

```yaml
with:
  url: "https://example.com/many-redirects"
  max_redirects: 3
```

### Response Object

The HTTP action provides a `res` object with the following properties:

| Property | Type | Description |
|----------|------|-------------|
| `status` | Integer | HTTP status code (200, 404, 500, etc.) |
| `time` | Integer | Response time in milliseconds |
| `body_size` | Integer | Response body size in bytes |
| `headers` | Object | Response headers as key-value pairs |
| `json` | Object | Parsed JSON response (only if valid JSON) |
| `text` | String | Response body as text |

### Response Examples

```yaml
steps:
  - name: "API Test"
    id: api-test
    action: http
    with:
      url: "https://jsonplaceholder.typicode.com/users/1"
    test: |
      res.status == 200 &&
      res.time < 2000 &&
      res.json.id == 1 &&
      res.json.name != ""
    outputs:
      user_id: res.json.id
      user_name: res.json.name
      response_time: res.time
      content_type: res.headers["Content-Type"]
```

### Common HTTP Patterns

#### Authentication

```yaml
vars:
  api_url: "{{API_URL}}"
  access_token: "{{ACCESS_TOKEN}}"
  username: "{{USERNAME}}"
  password: "{{PASSWORD}}"
  api_key: "{{API_KEY}}"

# Bearer token
steps:
  - name: "Authenticated Request"
    uses: http
    with:
      url: "{{vars.api_url}}/protected"
      headers:
        Authorization: "Bearer {{vars.access_token}}"

# Basic auth
  - name: "Basic Auth Request"
    uses: http
    with:
      url: "{{vars.api_url}}/basic"
      headers:
        Authorization: "Basic {{encode_base64(vars.username + ':' + vars.password)}}"

# API key
  - name: "API Key Request"
    uses: http
    with:
      url: "{{vars.api_url}}/data"
      headers:
        X-API-Key: "{{vars.api_key}}"
```

#### Content Types

```yaml
# JSON API
steps:
  - name: "JSON Request"
    action: http
    with:
      url: "{{env.API_URL}}/json"
      method: "POST"
      headers:
        Content-Type: "application/json"
      body: |
        {
          "key": "value",
          "timestamp": "{{iso8601()}}"
        }

# XML Request
  - name: "XML Request"
    action: http
    with:
      url: "{{env.API_URL}}/xml"
      method: "POST"
      headers:
        Content-Type: "application/xml"
      body: |
        <?xml version="1.0"?>
        <data>
          <key>value</key>
        </data>

# GraphQL Query
  - name: "GraphQL Query"
    action: http
    with:
      url: "{{env.GRAPHQL_URL}}"
      method: "POST"
      headers:
        Content-Type: "application/json"
      body: |
        {
          "query": "query { user(id: \"{{env.USER_ID}}\") { name email } }"
        }
```

#### File Upload

```yaml
steps:
  - name: "File Upload"
    action: http
    with:
      url: "{{env.API_URL}}/upload"
      method: "POST"
      headers:
        Content-Type: "multipart/form-data"
      body: |
        --boundary123
        Content-Disposition: form-data; name="file"; filename="test.txt"
        Content-Type: text/plain
        
        File content here
        --boundary123--
```

## Database Action

The `db` action executes SQL queries on MySQL, PostgreSQL, and SQLite databases, providing comprehensive result handling and error reporting.

### Basic Syntax

```yaml
steps:
  - name: "Database Query"
    uses: db
    with:
      dsn: "mysql://user:password@localhost:3306/database"
      query: "SELECT * FROM users WHERE active = ?"
      params__0: true
    test: res.code == 0 && res.rows_affected > 0
```

### Parameters

#### `dsn` (required)

**Type:** String  
**Description:** Database connection string with automatic driver detection  
**Supports:** Template expressions

```yaml
# MySQL
with:
  dsn: "mysql://user:password@localhost:3306/database"
  dsn: "mysql://{{vars.db_user}}:{{env.DB_PASS}}@{{vars.db_host}}/{{vars.db_name}}"

# PostgreSQL  
with:
  dsn: "postgres://user:password@localhost:5432/database?sslmode=disable"
  dsn: "postgres://{{env.PG_USER}}:{{env.PG_PASS}}@{{env.PG_HOST}}/{{env.PG_DB}}"

# SQLite
with:
  dsn: "sqlite:///absolute/path/to/database.db"
  dsn: "sqlite://./relative/path/to/database.db"
  dsn: "sqlite://{{vars.data_dir}}/app.db"
```

#### `query` (required)

**Type:** String  
**Description:** SQL query to execute  
**Supports:** Template expressions and multi-line strings

```yaml
with:
  query: "SELECT * FROM users"
  query: "INSERT INTO logs (message, timestamp) VALUES (?, NOW())"
  query: |
    SELECT u.name, u.email, p.title 
    FROM users u 
    JOIN profiles p ON u.id = p.user_id 
    WHERE u.active = ? AND u.created_at > ?
```

#### `params__N` (optional)

**Type:** Mixed (String, Number, Boolean)  
**Description:** Query parameters for prepared statements (N starts from 0)  
**Supports:** Template expressions

```yaml
with:
  query: "SELECT * FROM users WHERE id = ? AND active = ?"
  params__0: 123
  params__1: true
  params__2: "{{vars.user_email}}"
```

#### `timeout` (optional)

**Type:** Duration  
**Default:** `30s`  
**Description:** Query execution timeout

```yaml
with:
  query: "SELECT COUNT(*) FROM large_table"
  timeout: "60s"
```

### Response Object

The database action provides a `res` object with the following properties:

| Property | Type | Description |
|----------|------|-------------|
| `code` | Integer | Operation result (0 = success, 1 = error) |
| `rows_affected` | Integer | Number of rows affected by the query |
| `rows` | Array | Query results for SELECT statements (as objects) |
| `error` | String | Error message if operation failed |

### Response Examples

#### SELECT Query Response

```yaml
steps:
  - name: "Fetch Users"
    id: fetch-users
    uses: db
    with:
      dsn: "mysql://user:pass@localhost/db"
      query: "SELECT id, name, email FROM users WHERE active = ?"
      params__0: true
    test: res.code == 0 && res.rows_affected > 0
    outputs:
      user_count: res.rows_affected
      first_user_id: res.rows__0__id
      first_user_name: res.rows__0__name
```

#### INSERT/UPDATE Query Response

```yaml
steps:
  - name: "Insert User"
    uses: db
    with:
      dsn: "postgres://user:pass@localhost/db"
      query: "INSERT INTO users (name, email) VALUES ($1, $2)"
      params__0: "John Doe"
      params__1: "john@example.com"
    test: res.code == 0 && res.rows_affected == 1
```

### Database-Specific Features

#### MySQL Examples

```yaml
# MySQL with connection options
- name: "MySQL Query"
  uses: db
  with:
    dsn: "mysql://user:pass@tcp(localhost:3306)/database?charset=utf8mb4&parseTime=true"
    query: "SELECT VERSION() as mysql_version, NOW() as current_time"
  test: res.code == 0

# MySQL stored procedure
- name: "Call Procedure"
  uses: db
  with:
    dsn: "mysql://user:pass@localhost:3306/database"
    query: "CALL GetUsersByDepartment(?)"
    params__0: "Engineering"
  test: res.code == 0
```

#### PostgreSQL Examples

```yaml
# PostgreSQL with JSON operations
- name: "JSON Query"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost:5432/database?sslmode=disable"
    query: |
      SELECT name, data->>'role' as role, data->'preferences' as prefs
      FROM users 
      WHERE data ? 'role' AND data->>'role' = $1
    params__0: "admin"
  test: res.code == 0

# PostgreSQL array operations
- name: "Array Query"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost:5432/database"
    query: "SELECT name FROM users WHERE tags && $1"
    params__0: '{"admin","moderator"}'
  test: res.code == 0
```

#### SQLite Examples

```yaml
# SQLite with file creation
- name: "SQLite Query"
  uses: db
  with:
    dsn: "sqlite://./data/app.db"
    query: |
      CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        email TEXT UNIQUE,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
      )
  test: res.code == 0

# SQLite with in-memory database
- name: "Memory Database"
  uses: db
  with:
    dsn: "sqlite://:memory:"
    query: "CREATE TABLE temp_data (id INTEGER, value TEXT)"
  test: res.code == 0
```

### Common Query Patterns

#### Data Validation Queries

```yaml
- name: "Check Data Integrity"
  uses: db
  with:
    dsn: "mysql://user:pass@localhost/db"
    query: |
      SELECT 
        COUNT(*) as total_users,
        COUNT(CASE WHEN active = 1 THEN 1 END) as active_users,
        COUNT(CASE WHEN email IS NULL THEN 1 END) as missing_emails
      FROM users
  test: |
    res.code == 0 && 
    res.rows__0__total_users > 0 && 
    res.rows__0__missing_emails == 0
```

#### Performance Monitoring

```yaml
- name: "Database Performance Check"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: |
      SELECT 
        schemaname, 
        tablename, 
        seq_scan, 
        seq_tup_read, 
        idx_scan, 
        idx_tup_fetch
      FROM pg_stat_user_tables 
      WHERE seq_scan > 1000
    timeout: "10s"
  test: res.code == 0
  outputs:
    high_seq_scan_tables: res.rows_affected
```

#### Batch Operations

```yaml
- name: "Batch Insert"
  uses: db
  with:
    dsn: "mysql://user:pass@localhost/db"
    query: |
      INSERT INTO audit_log (action, table_name, record_id, timestamp) VALUES
      ('CREATE', 'users', 123, NOW()),
      ('UPDATE', 'profiles', 456, NOW()),
      ('DELETE', 'sessions', 789, NOW())
  test: res.code == 0 && res.rows_affected == 3
```

### Security Features

The database action implements several security measures:

- **Prepared Statements**: All parameterized queries use prepared statements to prevent SQL injection
- **Connection String Masking**: Passwords are masked in logs and output
- **Timeout Protection**: Prevents long-running queries from hanging
- **Driver Validation**: Only supports approved database drivers
- **DSN Validation**: Validates connection string format before execution

### Error Handling

Common error scenarios and handling patterns:

```yaml
- name: "Database with Error Handling"
  uses: db
  with:
    dsn: "mysql://user:pass@localhost/db"
    query: "SELECT * FROM users WHERE id = ?"
    params__0: 999999
  test: |
    res.code == 0 ? true :
    res.error | contains("connection") ? false :
    res.error | contains("not found") ? true :
    false
  outputs:
    query_success: res.code == 0
    error_type: |
      {{res.code == 0 ? "none" :
        res.error | contains("connection") ? "connection" :
        res.error | contains("syntax") ? "syntax" :
        "unknown"}}
```

### Transaction Examples

While the action doesn't directly support transactions, you can use database-specific transaction syntax:

```yaml
# PostgreSQL transaction
- name: "Begin Transaction"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: "BEGIN"
  test: res.code == 0

- name: "Insert Data"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: "INSERT INTO users (name) VALUES ($1)"
    params__0: "Test User"
  test: res.code == 0

- name: "Commit Transaction"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: "COMMIT"
  test: res.code == 0
```

## Shell Action

The `shell` action executes shell commands and scripts securely, providing comprehensive output capture and error handling.

### Basic Syntax

```yaml
steps:
  - name: "Execute Build Script"
    uses: shell
    with:
      cmd: "npm run build"
    test: res.code == 0
```

### Parameters

#### `cmd` (required)

**Type:** String  
**Description:** The shell command to execute  
**Supports:** Template expressions

```yaml
with:
  cmd: "echo 'Hello World'"
  cmd: "npm run {{vars.build_script}}"
  cmd: "curl -f {{env.API_URL}}/health"
```

#### `shell` (optional)

**Type:** String  
**Default:** `/bin/sh`  
**Allowed Values:** `/bin/sh`, `/bin/bash`, `/bin/zsh`, `/bin/dash`, `/usr/bin/sh`, `/usr/bin/bash`, `/usr/bin/zsh`, `/usr/bin/dash`

```yaml
with:
  cmd: "echo $0"
  shell: "/bin/bash"
```

#### `workdir` (optional)

**Type:** String  
**Description:** Working directory for command execution (must be absolute path)  
**Supports:** Template expressions

```yaml
with:
  cmd: "pwd && ls -la"
  workdir: "/app/src"
  workdir: "{{vars.project_path}}"
```

#### `timeout` (optional)

**Type:** String or Duration  
**Default:** `30s`  
**Format:** Go duration format (`30s`, `5m`, `1h`) or plain number (seconds)

```yaml
with:
  cmd: "npm test"
  timeout: "10m"
  timeout: "300"  # 300 seconds
```

#### `env` (optional)

**Type:** Object  
**Description:** Environment variables to set for the command  
**Supports:** Template expressions in values

```yaml
with:
  cmd: "npm run build"
  env:
    NODE_ENV: "production"
    API_URL: "{{env.PRODUCTION_API_URL}}"
    BUILD_VERSION: "{{vars.version}}"
```

### Response Format

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

### Usage Examples

#### Basic Command Execution

```yaml
- name: "System Information"
  uses: shell
  with:
    cmd: "uname -a"
  test: res.code == 0
```

#### Build and Test Pipeline

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

#### Environment-specific Deployment

```yaml
- name: "Deploy to Environment"
  uses: shell
  with:
    cmd: "./deploy.sh {{env.TARGET_ENV}}"
    workdir: "/deploy"
    shell: "/bin/bash"
    timeout: "15m"
    env:
      DEPLOY_KEY: "{{env.DEPLOY_KEY}}"
      TARGET_ENV: "{{env.TARGET_ENV}}"
  test: res.code == 0
```

#### Error Handling and Debugging

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

### Security Features

The shell action implements several security measures:

- **Shell Path Restriction**: Only allows approved shell executables
- **Working Directory Validation**: Ensures absolute paths and directory existence
- **Timeout Protection**: Prevents infinite execution
- **Environment Variable Filtering**: Safely handles environment variable passing
- **Output Sanitization**: Safely captures and returns command output

### Error Handling

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

## SMTP Action

The `smtp` action sends email notifications and alerts through SMTP servers.

### Basic Syntax

```yaml
steps:
  - name: "Send Alert"
    action: smtp
    with:
      host: "smtp.gmail.com"
      username: "{{env.SMTP_USER}}"
      password: "{{env.SMTP_PASS}}"
      from: "alerts@example.com"
      to: ["admin@example.com"]
      subject: "Service Alert"
      body: "Service is down"
```

### Parameters

#### `host` (required)

**Type:** String  
**Description:** SMTP server hostname or IP address

```yaml
with:
  host: "smtp.gmail.com"
  host: "mail.example.com"
  host: "127.0.0.1"
```

#### `port` (optional)

**Type:** Integer  
**Default:** `587`  
**Description:** SMTP server port

```yaml
with:
  host: "smtp.gmail.com"
  port: 587    # TLS/STARTTLS
  port: 465    # SSL
  port: 25     # Plain
```

#### `username` (required)

**Type:** String  
**Description:** SMTP authentication username  
**Supports:** Template expressions

```yaml
with:
  username: "{{env.SMTP_USERNAME}}"
  username: "alerts@example.com"
```

#### `password` (required)

**Type:** String  
**Description:** SMTP authentication password  
**Supports:** Template expressions

```yaml
with:
  password: "{{env.SMTP_PASSWORD}}"
  password: "{{env.EMAIL_APP_PASSWORD}}"
```

#### `from` (required)

**Type:** String  
**Description:** Sender email address  
**Supports:** Template expressions

```yaml
with:
  from: "alerts@example.com"
  from: "{{env.FROM_EMAIL}}"
  from: "Probe Monitor <probe@example.com>"
```

#### `to` (required)

**Type:** Array of strings  
**Description:** Recipient email addresses  
**Supports:** Template expressions

```yaml
with:
  to: ["admin@example.com"]
  to: ["user1@example.com", "user2@example.com"]
  to: ["{{env.ALERT_EMAIL}}"]
```

#### `cc` (optional)

**Type:** Array of strings  
**Description:** Carbon copy recipients

```yaml
with:
  to: ["admin@example.com"]
  cc: ["team@example.com", "manager@example.com"]
```

#### `bcc` (optional)

**Type:** Array of strings  
**Description:** Blind carbon copy recipients

```yaml
with:
  to: ["admin@example.com"]
  bcc: ["audit@example.com"]
```

#### `subject` (required)

**Type:** String  
**Description:** Email subject line  
**Supports:** Template expressions

```yaml
with:
  subject: "Alert: Service Down"
  subject: "{{env.SERVICE_NAME}} Status: {{outputs.health-check.status}}"
  subject: "Daily Report - {{date('2006-01-02')}}"
```

#### `body` (required)

**Type:** String  
**Description:** Email body content  
**Supports:** Template expressions and multi-line strings

```yaml
with:
  body: "Simple text message"
  
  # Multi-line text
  body: |
    Service Alert Report
    
    Status: {{outputs.check.status}}
    Timestamp: {{iso8601()}}
    Response Time: {{outputs.check.time}}ms
    
    Please investigate immediately.

  # HTML email (set html: true)
  body: |
    <html>
    <body>
      <h1>Service Alert</h1>
      <p>Status: <strong>{{outputs.check.status}}</strong></p>
      <p>Time: {{iso8601()}}</p>
    </body>
    </html>
```

#### `html` (optional)

**Type:** Boolean  
**Default:** `false`  
**Description:** Whether the body contains HTML content

```yaml
with:
  subject: "HTML Alert"
  body: "<h1>Alert</h1><p>Service is <strong>down</strong></p>"
  html: true
```

#### `tls` (optional)

**Type:** Boolean  
**Default:** `true`  
**Description:** Whether to use TLS/STARTTLS encryption

```yaml
with:
  host: "smtp.example.com"
  port: 587
  tls: true     # Use STARTTLS
  
with:
  host: "smtp.example.com"
  port: 465
  tls: false    # Use SSL (port 465 typically uses implicit SSL)
```

### Response Object

The SMTP action provides a `res` object with the following properties:

| Property | Type | Description |
|----------|------|-------------|
| `success` | Boolean | Whether the email was sent successfully |
| `message_id` | String | Unique message identifier (if provided by server) |
| `time` | Integer | Time taken to send email in milliseconds |

### SMTP Examples

#### Gmail Configuration

```yaml
steps:
  - name: "Send Gmail Alert"
    action: smtp
    with:
      host: "smtp.gmail.com"
      port: 587
      username: "{{env.GMAIL_USERNAME}}"
      password: "{{env.GMAIL_APP_PASSWORD}}"  # Use app password, not account password
      from: "{{env.GMAIL_USERNAME}}"
      to: ["admin@example.com"]
      subject: "Probe Alert - {{date('15:04')}}"
      body: |
        Alert from Probe workflow.
        
        Details:
        - Workflow: {{workflow.name}}
        - Time: {{iso8601()}}
        - Status: Failed
```

#### Office 365 Configuration

```yaml
steps:
  - name: "Send Office 365 Alert"
    action: smtp
    with:
      host: "smtp.office365.com"
      port: 587
      username: "{{env.O365_USERNAME}}"
      password: "{{env.O365_PASSWORD}}"
      from: "{{env.O365_USERNAME}}"
      to: ["team@company.com"]
      subject: "System Alert"
      body: "Alert message content"
      tls: true
```

#### HTML Email with Multiple Recipients

```yaml
steps:
  - name: "HTML Status Report"
    action: smtp
    with:
      host: "{{env.SMTP_HOST}}"
      port: 587
      username: "{{env.SMTP_USER}}"
      password: "{{env.SMTP_PASS}}"
      from: "reports@example.com"
      to: ["admin@example.com", "ops@example.com"]
      cc: ["manager@example.com"]
      subject: "Daily Health Report - {{date('2006-01-02')}}"
      html: true
      body: |
        <html>
        <head><title>Health Report</title></head>
        <body>
          <h1>Daily Health Report</h1>
          <table border="1">
            <tr><th>Service</th><th>Status</th><th>Response Time</th></tr>
            <tr><td>API</td><td style="color: {{outputs.api.success ? 'green' : 'red'}}">{{outputs.api.status}}</td><td>{{outputs.api.time}}ms</td></tr>
            <tr><td>Database</td><td style="color: {{outputs.db.success ? 'green' : 'red'}}">{{outputs.db.status}}</td><td>{{outputs.db.time}}ms</td></tr>
          </table>
          <p>Generated at {{iso8601()}}</p>
        </body>
        </html>
```

## Hello Action

The `hello` action is a simple test action used for development, debugging, and workflow validation.

### Basic Syntax

```yaml
steps:
  - name: "Test Hello"
    action: hello
    with:
      message: "Hello, World!"
```

### Parameters

#### `message` (optional)

**Type:** String  
**Default:** `"Hello from Probe!"`  
**Description:** Message to display  
**Supports:** Template expressions

```yaml
with:
  message: "Hello, World!"
  message: "Current time: {{iso8601()}}"
  message: "Hello {{env.USER_NAME}}"
```

#### `delay` (optional)

**Type:** Duration  
**Default:** `0s`  
**Description:** Artificial delay before completing

```yaml
with:
  message: "Delayed hello"
  delay: "2s"
```

### Response Object

The hello action provides a `res` object with the following properties:

| Property | Type | Description |
|----------|------|-------------|
| `message` | String | The message that was displayed |
| `time` | Integer | Time taken in milliseconds (including delay) |
| `timestamp` | String | ISO 8601 timestamp when action completed |

### Hello Examples

#### Basic Test

```yaml
steps:
  - name: "Simple Test"
    action: hello
    test: res.message != ""
    outputs:
      test_time: res.time
```

#### Timing Test

```yaml
steps:
  - name: "Timing Test"
    action: hello
    with:
      message: "Testing timing"
      delay: "1s"
    test: res.time >= 1000 && res.time < 1100
```

#### Template Testing

```yaml
steps:
  - name: "Template Test"
    action: hello
    with:
      message: "User: {{env.USER}}, Time: {{unixtime()}}"
    test: res.message | contains(env.USER)
    outputs:
      rendered_message: res.message
```

## Action Error Handling

### Common Error Scenarios

All actions can fail for various reasons. Understanding common failure modes helps with writing robust workflows.

#### HTTP Action Errors

```yaml
steps:
  - name: "HTTP with Error Handling"
    action: http
    with:
      url: "https://api.example.com/endpoint"
    test: |
      res.status >= 200 && res.status < 300
    continue_on_error: false
    outputs:
      success: res.status >= 200 && res.status < 300
      error_message: |
        {{res.status >= 400 ? "Client error: " + res.status : 
          res.status >= 500 ? "Server error: " + res.status : ""}}
```

#### SMTP Action Errors

```yaml
steps:
  - name: "SMTP with Error Handling"
    action: smtp
    with:
      host: "smtp.example.com"
      username: "{{env.SMTP_USER}}"
      password: "{{env.SMTP_PASS}}"
      from: "test@example.com"
      to: ["admin@example.com"]
      subject: "Test"
      body: "Test message"
    test: res.success == true
    continue_on_error: true
    outputs:
      email_sent: res.success
      send_time: res.time
```

## Performance Considerations

### HTTP Action Performance

- **Connection pooling:** HTTP actions reuse connections when possible
- **Timeouts:** Set appropriate timeouts to prevent hanging
- **Response size:** Large responses consume more memory
- **Concurrent requests:** Multiple HTTP actions can run in parallel

```yaml
# Performance-optimized HTTP configuration
defaults:
  http:
    timeout: "10s"
    follow_redirects: true
    max_redirects: 3

jobs:
  performance-test:
    steps:
      - name: "Quick Health Check"
        action: http
        with:
          url: "{{env.API_URL}}/ping"
          timeout: "2s"
        test: res.status == 200 && res.time < 500
```

### SMTP Action Performance

- **Connection reuse:** SMTP connections are established per action
- **Batch emails:** Consider grouping recipients to reduce connections
- **TLS overhead:** TLS negotiation adds latency

```yaml
# Efficient email notification
steps:
  - name: "Batch Notification"
    action: smtp
    with:
      host: "smtp.example.com"
      username: "{{env.SMTP_USER}}"
      password: "{{env.SMTP_PASS}}"
      from: "alerts@example.com"
      to: ["admin1@example.com", "admin2@example.com", "admin3@example.com"]
      subject: "Batch Alert"
      body: "Single email to multiple recipients"
```

## See Also

- **[YAML Configuration](../yaml-configuration/)** - Complete YAML syntax reference
- **[Built-in Functions](../built-in-functions/)** - Expression functions for use with actions
- **[Concepts: Actions](../../concepts/actions/)** - Action system architecture
- **[How-tos: API Testing](../../how-tos/api-testing/)** - Practical HTTP action examples
- **[How-tos: Error Handling](../../how-tos/error-handling-strategies/)** - Error handling patterns