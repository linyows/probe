# Actions Reference

This page provides comprehensive documentation for all built-in Probe actions, including their parameters, response formats, and usage examples.

## Overview

Actions are the building blocks of Probe workflows. They perform specific tasks like making HTTP requests, sending emails, or executing custom logic. All actions return structured response data that can be used in tests and outputs.

### Built-in Actions

- **[http](#http-action)** - Make HTTP/HTTPS requests and validate responses
- **[db](#database-action)** - Execute database queries on MySQL, PostgreSQL, and SQLite
- **[browser](#browser-action)** - Automate web browsers using ChromeDP
- **[shell](#shell-action)** - Execute shell commands and scripts securely
- **[smtp](#smtp-action)** - Send email notifications and alerts
- **[imap](#imap-action)** - Connect to IMAP servers and manage email operations
- **[ssh](/reference/actions/ssh)** - Run commands on a remote host over SSH
- **[grpc](#grpc-action)** - Call gRPC services by reflection
- **[embedded](#embedded-action)** - Run another workflow as a step
- **[mail-latency](#mail-latency-action)** - Measure delivery latency from a Maildir
- **[hello](#hello-action)** - Simple test action for development and debugging

## HTTP Action

The `http` action performs an HTTP request and exposes the response for assertions and outputs.

### Basic Syntax

```yaml
steps:
  - name: Check the API
    uses: http
    with:
      url: "https://api.example.com/health"
      method: GET
    test: res.code == 200
```

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | String | Yes | - | Request URL, or the base URL when a method shorthand carries a path |
| `method` | String | Yes | - | HTTP method. Supplied by a method shorthand when one is used |
| `headers` | Object | No | - | Request headers |
| `body` | String or Object | No | - | Request body. An object is serialized as JSON when `content-type` is `application/json` |
| `timeout` | Duration | No | `30s` | Time limit for the whole request, including reading the response |

There are no parameters for redirects or TLS verification. Redirects are followed by default.

#### `timeout`

`timeout` accepts a duration string such as `10s` or `1m30s`, or a plain number of seconds. `0` removes the limit.

```yaml
  - name: Slow endpoint
    uses: http
    with:
      url: "{{vars.api_url}}/report"
      method: GET
      timeout: 60s
    test: res.code == 200
```

A request that runs out of time fails the step with a `Client.Timeout exceeded` error.

Set it once for a job through `defaults`:

```yaml
jobs:
  - name: API checks
    defaults:
      http:
        timeout: 5s
    steps:
      - name: Health
        uses: http
        with:
          get: /health
        test: res.code == 200
```

The step's own `timeout` is a separate, outer limit on each attempt of the action, defaulting to 5m. `with.timeout` bounds the HTTP request; the step `timeout` bounds the action call that wraps it, and is what stops an action that hangs without returning.

#### Method Shorthands

`get`, `head`, `post`, `put`, `patch`, `delete`, `connect`, `options` and `trace` set the method and the path in one key. The value is either a full URL or a path resolved against `url`, which makes it convenient with a job's `defaults`.

```yaml
jobs:
  - name: API checks
    defaults:
      http:
        url: "{{vars.api_url}}"
        headers:
          accept: application/json
    steps:
      - name: List users
        uses: http
        with:
          get: /users
        test: res.code == 200

      - name: Create a user
        uses: http
        with:
          post: /users
          headers:
            content-type: application/json
          body:
            name: "{{vars.user_name}}"
        test: res.code == 201
```

### Response Object

| Field | Type | Description |
|-------|------|-------------|
| `res.code` | Integer | Status code, such as `200` |
| `res.status` | String | Status line, such as `"200 OK"` |
| `res.headers` | Object | Response headers, keyed by canonical name such as `Content-Type` |
| `res.body` | Any | Response body. Parsed into an object or array when the response is JSON, otherwise the raw string |
| `res.rawbody` | String | The unparsed body, present when the body was parsed as JSON |
| `res.filepath` | String | Path to the saved file when the response is binary |
| `rt.duration` | String | Round-trip time, such as `"120ms"` |
| `rt.sec` | Float | Round-trip time in seconds |
| `status` | Integer | `0` when the status code is 2xx, `1` otherwise |

### Response Examples

For a JSON response, the fields are read straight off `res.body`:

```yaml
    test: |
      res.code == 200 &&
      res.headers["Content-Type"] contains "application/json" &&
      res.body.status == "ok" &&
      len(res.body.items) > 0
    outputs:
      first_id: res.body.items[0].id
      elapsed_ms: rt.sec * 1000
```

For a text or HTML response, `res.body` is the string itself:

```yaml
    test: |
      res.code == 200 &&
      res.body contains "<title>" &&
      len(res.body) > 100
```

### Common HTTP Patterns

#### Authentication

```yaml
  - name: Log in
    id: auth
    uses: http
    with:
      url: "{{vars.api_url}}/login"
      method: POST
      headers:
        content-type: application/json
      body:
        user: "{{vars.user}}"
        password: "{{vars.password}}"
    test: res.code == 200
    outputs:
      token: res.body.access_token

  - name: Call a protected endpoint
    uses: http
    with:
      url: "{{vars.api_url}}/me"
      method: GET
      headers:
        authorization: "Bearer {{outputs.auth.token}}"
    test: res.code == 200
```

#### Checking an Error Response

```yaml
  - name: Unknown id returns 404
    uses: http
    with:
      url: "{{vars.api_url}}/users/does-not-exist"
      method: GET
    test: res.code == 404 && res.body.error != null
```

#### Retrying a Flaky Endpoint

```yaml
  - name: Eventually consistent read
    uses: http
    retry:
      max_attempts: 5
      interval: 2s
    with:
      url: "{{vars.api_url}}/orders/{{outputs.create.order_id}}"
      method: GET
    test: res.code == 200
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
      params: [true]
    test: res.code == 0 && res.rows_affected > 0
```

### Parameters

#### `dsn` (required)

**Type:** String  
**Description:** Database connection string with automatic driver detection  
**Supports:** Template expressions

```yaml
# MySQL
vars:
  db_pass: "{{DB_PASS}}"

with:
  dsn: "mysql://user:password@localhost:3306/database"
  dsn: "mysql://{{vars.db_user}}:{{vars.db_pass}}@{{vars.db_host}}/{{vars.db_name}}"

# PostgreSQL
vars:
  pg_user: "{{PG_USER}}"
  pg_pass: "{{PG_PASS}}"
  pg_host: "{{PG_HOST}}"
  pg_db: "{{PG_DB}}"

with:
  dsn: "postgres://user:password@localhost:5432/database?sslmode=disable"
  dsn: "postgres://{{vars.pg_user}}:{{vars.pg_pass}}@{{vars.pg_host}}/{{vars.pg_db}}"

# SQLite
with:
  dsn: "file:./testdata/sqlite.db"
  dsn: "file:/absolute/path/to/database.db"
  dsn: "file:{{vars.data_dir}}/app.db"
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

#### `params` (optional)

**Type:** Array of mixed values (String, Number, Boolean)  
**Description:** Query parameters for prepared statements  
**Supports:** Template expressions

```yaml
with:
  query: "SELECT * FROM users WHERE id = ? AND active = ?"
  params: [123, true, "{{vars.user_email}}"]
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
      params: [true]
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
      params: ["John Doe", "john@example.com"]
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
    params: ["Engineering"]
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
    params: ["admin"]
  test: res.code == 0

# PostgreSQL array operations
- name: "Array Query"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost:5432/database"
    query: "SELECT name FROM users WHERE tags && $1"
    params: ['{"admin","moderator"}']
  test: res.code == 0
```

#### SQLite Examples

```yaml
# SQLite with file creation
- name: "SQLite Query"
  uses: db
  with:
    dsn: "file:./testdata/sqlite.db"
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
    dsn: "file::memory:"
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
    params: [999999]
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
    params: ["Test User"]
  test: res.code == 0

- name: "Commit Transaction"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: "COMMIT"
  test: res.code == 0
```

## Browser Action

The `browser` action automates web browsers using ChromeDP, providing comprehensive web automation capabilities for testing, scraping, and interaction with web applications.

### Basic Syntax

```yaml
steps:
  - name: "Navigate to Website"
    uses: browser
    with:
      action: navigate
      url: "https://example.com"
      headless: true
      timeout: 30s
    test: res.code == 0
```

### Parameters

#### `action` (required)

**Type:** String  
**Description:** The browser action to perform  
**Values:** 
- **Navigation:** `navigate`
- **Text/Content:** `text`, `value`, `get_html`
- **Attributes:** `get_attribute`
- **Interactions:** `click`, `double_click`, `right_click`, `hover`, `focus`
- **Input:** `type`, `send_keys`, `select`
- **Forms:** `submit`
- **Scrolling:** `scroll`
- **Screenshots:** `screenshot`, `capture_screenshot`, `full_screenshot`
- **Waiting:** `wait_visible`, `wait_not_visible`, `wait_ready`, `wait_text`, `wait_enabled`

#### `url` (optional)

**Type:** String  
**Description:** URL to navigate to (required for navigate action)  
**Supports:** Template expressions

```yaml
with:
  action: navigate
  url: "https://example.com"
  url: "{{vars.base_url}}/login"
```

#### `selector` (optional)

**Type:** String  
**Description:** CSS selector for targeting elements  
**Supports:** Template expressions

```yaml
with:
  action: get_text
  selector: "h1"
  selector: "#main-title"
  selector: ".article-content p:first-child"
```

#### `value` (optional)

**Type:** String  
**Description:** Value to type or text to wait for  
**Supports:** Template expressions

```yaml
with:
  action: type
  selector: "#email"
  value: "user@example.com"
  value: "{{vars.username}}"
```

#### `attribute` (optional)

**Type:** String  
**Description:** Attribute name to retrieve (required for get_attribute action)

```yaml
with:
  action: get_attribute
  selector: "a"
  attribute: "href"
```

#### `headless` (optional)

**Type:** Boolean  
**Default:** `true`  
**Description:** Whether to run browser in headless mode

```yaml
with:
  action: navigate
  url: "https://example.com"
  headless: false  # Show browser window
```

#### `timeout` (optional)

**Type:** Duration  
**Default:** `30s`  
**Description:** Action timeout

```yaml
with:
  action: wait_visible
  selector: ".loading"
  timeout: "60s"
```

### Response Object

The browser action provides a `res` object with action-specific properties:

#### Common Properties

| Property | Type | Description |
|----------|------|-------------|
| `code` | Integer | Result code (0 = success, non-zero = error) |
| `results` | Object | Action-specific results (text, values, etc.) |

#### Navigation Response

| Property | Type | Description |
|----------|------|-------------|
| `url` | String | URL that was navigated to |
| `time_ms` | String | Navigation time in milliseconds |

#### Text/Attribute Response

| Property | Type | Description |
|----------|------|-------------|
| `selector` | String | CSS selector used |
| `text` | String | Extracted text content (get_text) |
| `attribute` | String | Attribute name (get_attribute) |
| `value` | String | Attribute value (get_attribute) |
| `exists` | String | "true" if attribute exists |

#### Screenshot Response

| Property | Type | Description |
|----------|------|-------------|
| `screenshot` | String | Base64-encoded screenshot |
| `size_bytes` | String | Screenshot size in bytes |

### Browser Actions

#### Navigate to URL

```yaml
- name: "Open Website"
  uses: browser
  with:
    action: navigate
    url: "https://example.com"
    headless: true
  test: res.code == 0
  outputs:
    load_time: rt.sec * 1000
```

#### Extract Text Content

```yaml
- name: "Get Page Title"
  uses: browser
  with:
    action: text
    selector: "h1"
  test: res.code == 0 && res.results.text != ""
  outputs:
    page_title: res.results.text

- name: "Get Input Value"
  uses: browser
  with:
    action: value
    selector: "#username"
  test: res.code == 0
  outputs:
    current_username: res.results.value

- name: "Get Element HTML"
  uses: browser
  with:
    action: get_html
    selector: ".article-content"
  test: res.code == 0
  outputs:
    article_html: res.results.get_html
```

#### Get Element Attributes

```yaml
- name: "Extract Links"
  uses: browser
  with:
    action: get_attribute
    selector: "a.download-link"
    attribute: "href"
  test: res.code == 0 && res.exists == "true"
  outputs:
    download_url: res.results.value
```

#### Form Interactions

```yaml
# Fill form fields
- name: "Enter Email"
  uses: browser
  with:
    action: type
    selector: "#email"
    value: "user@example.com"
  test: res.code == 0

# Click buttons
- name: "Click Submit"
  uses: browser
  with:
    action: click
    selector: "#submit-btn"
  test: res.code == 0

# Submit forms
- name: "Submit Form"
  uses: browser
  with:
    action: submit
    selector: "form"
  test: res.code == 0
```

#### Wait for Elements

```yaml
# Wait for element to appear
- name: "Wait for Results"
  uses: browser
  with:
    action: wait_visible
    selector: ".search-results"
    timeout: "10s"
  test: res.code == 0

# Wait for specific text
- name: "Wait for Success Message"
  uses: browser
  with:
    action: wait_text
    selector: ".status"
    value: "Success"
  test: res.code == 0
```

#### Capture Screenshots

```yaml
- name: "Take Screenshot"
  uses: browser
  with:
    action: screenshot
  test: res.code == 0
  outputs:
    screenshot_data: res.screenshot
    screenshot_size: res.size_bytes
```

### Advanced Usage Examples

#### Login Flow

```yaml
vars:
  login_url: "{{LOGIN_URL}}"
  username: "{{USERNAME}}"
  password: "{{PASSWORD}}"

steps:
  - name: "Navigate to Login"
    uses: browser
    with:
      action: navigate
      url: "{{vars.login_url}}"
    test: res.code == 0

  - name: "Enter Username"
    uses: browser
    with:
      action: type
      selector: "#username"
      value: "{{vars.username}}"
    test: res.code == 0

  - name: "Enter Password"
    uses: browser
    with:
      action: type
      selector: "#password"
      value: "{{vars.password}}"
    test: res.code == 0

  - name: "Submit Login"
    uses: browser
    with:
      action: click
      selector: "#login-button"
    test: res.code == 0

  - name: "Wait for Dashboard"
    uses: browser
    with:
      action: wait_visible
      selector: ".dashboard"
      timeout: "15s"
    test: res.code == 0
```

#### Data Extraction

```yaml
steps:
  - name: "Navigate to Data Page"
    uses: browser
    with:
      action: navigate
      url: "https://example.com/data"
    test: res.code == 0

  - name: "Wait for Table"
    uses: browser
    with:
      action: wait_visible
      selector: "table"
    test: res.code == 0

  - name: "Count Rows"
    uses: browser
    with:
      action: get_elements
      selector: "table tr"
    test: res.code == 0 && res.count != "0"
    outputs:
      row_count: res.count

  - name: "Extract First Cell"
    uses: browser
    with:
      action: get_text
      selector: "table tr:first-child td:first-child"
    test: res.code == 0
    outputs:
      first_cell: res.results.text
```

#### E2E Testing

```yaml
steps:
  - name: "Load Application"
    uses: browser
    with:
      action: navigate
      url: "https://app.example.com"
    test: res.code == 0

  - name: "Fill Contact Form"
    uses: browser
    with:
      action: type
      selector: "#contact-name"
      value: "John Doe"
    test: res.code == 0

  - name: "Fill Email"
    uses: browser
    with:
      action: type
      selector: "#contact-email"
      value: "john@example.com"
    test: res.code == 0

  - name: "Fill Message"
    uses: browser
    with:
      action: type
      selector: "#contact-message"
      value: "Hello from automated test"
    test: res.code == 0

  - name: "Submit Form"
    uses: browser
    with:
      action: submit
      selector: "#contact-form"
    test: res.code == 0

  - name: "Verify Success"
    uses: browser
    with:
      action: wait_text
      selector: ".success-message"
      value: "Thank you"
      timeout: "10s"
    test: res.code == 0

  - name: "Take Success Screenshot"
    uses: browser
    with:
      action: screenshot
    test: res.code == 0
```

### Error Handling

```yaml
- name: "Browser Action with Error Handling"
  uses: browser
  with:
    action: click
    selector: "#may-not-exist"
    timeout: "5s"
  test: res.code == 0 || (res.success == "false" && res.error | contains("not found"))
  outputs:
    click_success: res.code == 0
    error_type: |
      {{res.code == 0 ? "none" :
        res.error | contains("timeout") ? "timeout" :
        res.error | contains("not found") ? "element_not_found" :
        "unknown"}}
```

### Performance Considerations

- **Headless Mode**: Use `headless: true` (default) for faster execution
- **Timeouts**: Set appropriate timeouts to prevent hanging
- **Resource Usage**: Browser actions consume more resources than other actions
- **Screenshots**: Large screenshots consume significant memory

### Security Features

The browser action implements several security measures:

- **Sandboxed Execution**: ChromeDP runs in a sandboxed environment
- **Timeout Protection**: Prevents indefinite hanging
- **URL Validation**: Validates URLs before navigation
- **Resource Limits**: Built-in resource usage limits

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
vars:
  api_url: "{{API_URL}}"

with:
  cmd: "echo 'Hello World'"
  cmd: "npm run {{vars.build_script}}"
  cmd: "curl -f {{vars.api_url}}/health"
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
vars:
  production_api_url: "{{PRODUCTION_API_URL}}"

with:
  cmd: "npm run build"
  env:
    NODE_ENV: "production"
    API_URL: "{{vars.production_api_url}}"
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

The `smtp` action delivers mail to an SMTP server. It is built for measuring and exercising delivery rather than for sending hand-written notifications: the message body is generated, and its size is set with `length`.

### Basic Syntax

```yaml
steps:
  - name: Send a probe mail
    uses: smtp
    with:
      addr: "localhost:2525"
      from: "sender@example.com"
      to: "recipient@example.com"
      subject: "Delivery probe"
      session: 1
      message: 1
      length: 500
    test: res.code == 0 && res.sent > 0
```

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `addr` | String | Yes | - | SMTP server as `host:port` |
| `from` | String | Yes | - | Envelope sender |
| `to` | String | Yes | - | Envelope recipient |
| `subject` | String | No | `""` | Subject line |
| `myhostname` | String | No | - | Hostname used in the `HELO` / `EHLO` command |
| `session` | Integer | No | `1` | Number of SMTP sessions to open |
| `message` | Integer | No | `1` | Messages to send per session |
| `length` | Integer | No | `0` | Size of the generated message body in bytes |

There are no parameters for authentication, TLS, CC/BCC, a custom body or HTML. To include a report in the run output, use the step's `echo`.

### Response Object

| Field | Type | Description |
|-------|------|-------------|
| `res.code` | Integer | `0` when every message was delivered |
| `res.sent` | Integer | Messages delivered |
| `res.failed` | Integer | Messages that failed |
| `res.total` | Integer | Messages attempted |
| `res.error` | String | Error message, when delivery failed |
| `res.maildata` | String | The generated message, when it is text |
| `res.filepath` | String | Path to the generated message, when it is binary |

### SMTP Examples

#### Several Sessions and Messages

```yaml
steps:
  - name: Deliver 3 messages over 2 sessions
    id: bulk
    uses: smtp
    with:
      addr: "{{vars.smtp_addr}}"
      from: "{{vars.from_addr}}"
      to: "{{vars.to_addr}}"
      subject: "Bulk delivery test"
      myhostname: probe-client.local
      session: 2
      message: 3
      length: 750
    test: res.code == 0 && res.sent == 6
    outputs:
      sent: res.sent
```

#### Reporting the Result

```yaml
  - name: Delivery summary
    uses: hello
    echo: |
      Sent: {{outputs.bulk.sent}}
      Round trip: {{rt.duration}}
```


## IMAP Action

The `imap` action connects to IMAP servers to perform email operations such as reading messages, searching, and mailbox management.

### Basic Syntax

```yaml
vars:
  imap_username: "{{IMAP_USERNAME}}"
  imap_password: "{{IMAP_PASSWORD}}"

steps:
  - name: "Check Email"
    uses: imap
    with:
      host: "imap.example.com"
      port: 993
      username: "{{vars.imap_username}}"
      password: "{{vars.imap_password}}"
      tls: true
      commands:
      - name: "select"
        mailbox: "INBOX"
      - name: "search"
        criteria:
          flags: ["unseen"]
    test: res.code == 0
```

### Parameters

#### `host` (required)

**Type:** String  
**Description:** IMAP server hostname or IP address  
**Supports:** Template expressions

```yaml
with:
  host: "imap.gmail.com"
  host: "imap.example.com" 
  host: "{{vars.imap_server}}"
```

#### `port` (optional)

**Type:** Integer  
**Default:** `993`  
**Description:** IMAP server port

```yaml
with:
  port: 993   # IMAPS (SSL/TLS)
  port: 143   # IMAP (plain or STARTTLS)
```

#### `username` (required)

**Type:** String  
**Description:** IMAP authentication username  
**Supports:** Template expressions

```yaml
vars:
  email_user: "{{EMAIL_USER}}"

with:
  username: "{{vars.email_user}}"
  username: "user@example.com"
```

#### `password` (required)

**Type:** String  
**Description:** IMAP authentication password  
**Supports:** Template expressions

```yaml
vars:
  email_password: "{{EMAIL_PASSWORD}}"
  app_password: "{{EMAIL_APP_PASSWORD}}"

with:
  password: "{{vars.email_password}}"
  password: "{{vars.app_password}}"
```

#### `tls` (optional)

**Type:** Boolean  
**Default:** `true`  
**Description:** Whether to use TLS/SSL encryption

```yaml
with:
  host: "imap.example.com"
  port: 993
  tls: true     # Use TLS (recommended)
  
with:
  host: "imap.example.com" 
  port: 143
  tls: false    # Plain connection (not recommended)
```

#### `timeout` (optional)

**Type:** Duration  
**Default:** `30s`  
**Description:** Connection and operation timeout

```yaml
with:
  timeout: "60s"
  timeout: "2m"
```

#### `commands` (required)

**Type:** Array of command objects  
**Description:** IMAP commands to execute sequentially

```yaml
with:
  commands:
  - name: "select"
    mailbox: "INBOX"
  - name: "search"
    criteria:
      since: "today"
  - name: "fetch"
    sequence: "1:5"
    dataitem: "ALL"
```

### IMAP Commands

#### `select` - Select Mailbox

Select a mailbox for read-write operations.

```yaml
- name: "select"
  mailbox: "INBOX"      # Required: mailbox name
- name: "select" 
  mailbox: "Sent"
- name: "select"
  mailbox: "INBOX/Work"
```

#### `examine` - Read-only Mailbox Access

Select a mailbox for read-only operations.

```yaml
- name: "examine"
  mailbox: "INBOX"      # Required: mailbox name
```

#### `search` - Search Messages

Search messages using various criteria.

```yaml
- name: "search"
  criteria:
    since: "today"           # Date-based search
    flags: ["unseen"]        # Flag-based search  
    headers:                 # Header-based search
      from: "sender@example.com"
      subject: "urgent"
    bodies: ["important"]    # Body text search
    texts: ["meeting"]       # Full-text search
```

#### `list` - List Mailboxes

List available mailboxes.

```yaml
- name: "list"
  reference: ""         # Optional: reference name
  pattern: "*"          # Optional: mailbox pattern (default: "*")
- name: "list"
  reference: "INBOX"
  pattern: "INBOX/*"
```

#### `fetch` - Fetch Message Data

Retrieve message data using sequence numbers.

```yaml
- name: "fetch"
  sequence: "1:5"       # Required: sequence range
  dataitem: "ALL"       # Required: data items to fetch
- name: "fetch"
  sequence: "*"         # Latest message
  dataitem: "ENVELOPE FLAGS"
```

### Response Object

The IMAP action provides a `res` object with the following structure:

| Property | Type | Description |
|----------|------|-------------|
| `code` | Integer | Operation result (0 = success, non-zero = error) |
| `data` | Object | Command results organized by command type |
| `error` | String | Error message if operation failed |

### IMAP Examples

#### Gmail Configuration

```yaml
vars:
  gmail_username: "{{GMAIL_USERNAME}}"
  gmail_app_password: "{{GMAIL_APP_PASSWORD}}"

steps:
  - name: "Check Gmail Inbox"
    uses: imap
    with:
      host: "imap.gmail.com"
      port: 993
      username: "{{vars.gmail_username}}"
      password: "{{vars.gmail_app_password}}"  # Use App Password
      tls: true
      commands:
      - name: "select"
        mailbox: "INBOX"
      - name: "search"
        criteria:
          flags: ["unseen"]
          since: "today"
      - name: "fetch"
        sequence: "*"
        dataitem: "ENVELOPE FLAGS"
    test: res.code == 0
    outputs:
      unread_count: res.data.search.count
      latest_sender: res.data.fetch.messages__0__from
```

## Hello Action

The `hello` action does nothing but succeed. It is useful as a placeholder, for a step whose only job is an `echo`, and for trying out expressions.

### Basic Syntax

```yaml
steps:
  - name: Report
    uses: hello
    echo: "Checked {{outputs.health.endpoint}}"
```

### Parameters

The action takes no parameters of its own. Whatever is given in `with` is echoed back on `res`, which makes it a convenient way to publish computed values.

```yaml
steps:
  - name: Build a summary
    id: summary
    uses: hello
    with:
      run_id: "{{vars.run_id}}"
      checked_at: "{{now().Format('2006-01-02T15:04:05Z07:00')}}"
    outputs:
      run_id: res.run_id
      checked_at: res.checked_at
```

### Response Object

| Property | Type | Description |
|----------|------|-------------|
| `res.<key>` | Any | Every key passed in `with` |
| `res.status` | Integer | Always `0` |
| `status` | Integer | Always `0` |

## gRPC Action

The `grpc` action calls a gRPC method. The service definition is resolved through server reflection, so no `.proto` file is needed at run time.

### Basic Syntax

```yaml
- name: Get a user
  uses: grpc
  with:
    addr: "grpc.example.com:443"
    service: "user.v1.UserService"
    method: "GetUser"
    tls: true
    body: |
      {"id": "123"}
  test: res.status_code == "OK"
```

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `addr` | String | Yes | - | Host and port of the gRPC server |
| `service` | String | Yes | - | Fully qualified service name |
| `method` | String | Yes | - | Method name |
| `body` | String | No | `""` | Request message as JSON |
| `metadata` | Object | No | `{}` | Request metadata (the gRPC equivalent of headers) |
| `timeout` | String | No | - | Request timeout, such as `"10s"` |
| `tls` | Boolean | No | `false` | Use TLS |
| `insecure` | Boolean | No | `false` | Skip certificate verification |
| `cert_file` | String | No | - | Client certificate for mutual TLS |
| `key_file` | String | No | - | Client key for mutual TLS |
| `ca_file` | String | No | - | CA certificate used to verify the server |

### Response Object

| Field | Type | Description |
|-------|------|-------------|
| `res.body` | String | Response message as JSON |
| `res.status_code` | String | gRPC status code, such as `OK` or `NOT_FOUND` |
| `res.status_message` | String | Status message |
| `res.metadata` | Object | Response metadata |
| `rt` | String | Round-trip time |
| `req` | Object | The request as it was sent |

## Embedded Action

The `embedded` action runs a **job file** as a single step, which lets shared setup or checks live in their own file and be reused from several workflows.

The file is a job, not a workflow: it holds `name`, `steps` and optionally `defaults` - there is no `jobs` key in it.

### Basic Syntax

**auth.yml:**
```yaml
name: Authentication
steps:
  - name: Get token
    id: get_token
    uses: shell
    with:
      cmd: echo "0123456789"
    test: res.code == 0
    outputs:
      mytoken: replace(res.stdout, '\n', '')
```

**workflow.yml:**
```yaml
jobs:
- name: Main
  steps:
    - name: Authenticate
      id: auth
      uses: embedded
      with:
        path: "./auth.yml"
        vars:
          environment: "{{vars.environment}}"
      test: res.code == 0
      outputs:
        token: res.outputs.mytoken

    - name: Call the API
      uses: http
      with:
        method: GET
        url: "{{vars.api_url}}/me"
        headers:
          authorization: "Bearer {{outputs.auth.token}}"
      test: res.code == 200
```

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | String | Yes | - | Path to the job file. Resolved against the current working directory, not the workflow file |
| `vars` | Object | No | `{}` | Variables passed to the embedded job, read there as `vars.<name>` |

### Response Object

| Field | Type | Description |
|-------|------|-------------|
| `res.code` | Integer | `0` when every step of the embedded job passed |
| `res.outputs` | Object | The outputs published by the embedded job's steps, keyed by output name |
| `res.report` | String | The embedded job's report, which is also nested into the parent report |
| `res.error` | String | Error message, when the embedded job failed |
| `rt` | Object | Time spent running the embedded job |

`res.outputs` is keyed by output name, so a value published as `mytoken` is read as `res.outputs.mytoken`.


## Mail Latency Action

The `mail-latency` action reads messages from a Maildir, computes the delivery latency of each one from its `Received` headers, and writes the result as a CSV file.

### Basic Syntax

```yaml
- name: Measure delivery latency
  uses: mail-latency
  with:
    mail_dir: "/var/mail/probe/new"
    output_dir: "./reports"
  test: res.code == 0
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `mail_dir` | String | Yes | Directory holding the messages to measure |
| `output_dir` | String | Yes | Directory the CSV file is written to |

### Response Object

| Field | Type | Description |
|-------|------|-------------|
| `res.output_file` | String | Path of the written CSV file, named `mail-latency.<timestamp>.csv` |
| `res.status` | Integer | `0` on success |
| `rt` | String | Time spent measuring |

## Action Error Handling

A step fails when its `test` is false or when the action itself returns an error. The remaining steps of the job still run, the job is marked failed, and jobs that list it in `needs` are skipped.

There is no switch to ignore a failure. When a check should not fail the workflow, record its result as an output instead of asserting it.

```yaml
steps:
  - name: Required check
    uses: http
    with:
      method: GET
      url: "{{vars.api_url}}/health"
    test: res.code == 200

  - name: Optional check
    id: optional
    uses: http
    with:
      method: GET
      url: "{{vars.api_url}}/experimental"
    outputs:
      available: res.code == 200
      detail: res.code >= 400 ? res.status : ""

  - name: Report
    uses: hello
    echo: "Experimental endpoint: {{outputs.optional.available ? \"available\" : outputs.optional.detail}}"
```

Use `retry` for a transient failure and `timeout` for a step that may hang:

```yaml
  - name: Flaky endpoint
    uses: http
    timeout: 10s
    retry:
      max_attempts: 3
      interval: 2s
    with:
      method: GET
      url: "{{vars.api_url}}/flaky"
    test: res.code == 200
```


## Performance Considerations

- Jobs without a `needs` relation run in parallel, so independent checks do not queue behind each other.
- `rt.sec` and `rt.duration` measure the action's round trip, not the whole step.
- A large response body is held in memory; a binary body is written to a file and reported as `res.filepath`.
- `repeat` with `async: true` runs the repetitions of a job concurrently, which is the way to generate load.

```yaml
jobs:
  - name: Load test
    repeat:
      count: 50
      async: true
    steps:
      - name: Ping
        uses: http
        timeout: 2s
        with:
          method: GET
          url: "{{vars.api_url}}/ping"
        test: res.code == 200 && rt.sec < 0.5
```

## See Also

- **[YAML Configuration](/reference/yaml-configuration)** - Complete YAML syntax reference
- **[Built-in Functions](/reference/built-in-functions)** - Expression functions for use with actions
- **[SSH Action](/reference/actions/ssh)** - Running commands on a remote host
- **[Concepts: Actions](/guide/concepts/actions)** - Action system architecture
- **[How-tos: API Testing](/guide/how-tos/api-testing)** - Practical HTTP action examples
