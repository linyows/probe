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
- **[browser](#browser-action)** - Automate web browsers using ChromeDP
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
vars:
  api_url: "{{API_URL}}"
  graphql_url: "{{GRAPHQL_URL}}"
  user_id: "{{USER_ID}}"

# JSON API
steps:
  - name: "JSON Request"
    action: http
    with:
      url: "{{vars.api_url}}/json"
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
      url: "{{vars.api_url}}/xml"
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
      url: "{{vars.graphql_url}}"
      method: "POST"
      headers:
        Content-Type: "application/json"
      body: |
        {
          "query": "query { user(id: \"{{vars.user_id}}\") { name email } }"
        }
```

#### File Upload

```yaml
vars:
  api_url: "{{API_URL}}"

steps:
  - name: "File Upload"
    action: http
    with:
      url: "{{vars.api_url}}/upload"
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
  dsn: "sqlite://./testdata/sqlite.db"
  dsn: "sqlite:///absolute/path/to/database.db"
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
    dsn: "sqlite://./testdata/sqlite.db"
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
    load_time: res.time_ms
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
  continue_on_error: true
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

The `smtp` action sends email notifications and alerts through SMTP servers.

### Basic Syntax

```yaml
vars:
  smtp_user: "{{SMTP_USER}}"
  smtp_pass: "{{SMTP_PASS}}"

steps:
  - name: "Send Alert"
    action: smtp
    with:
      host: "smtp.gmail.com"
      username: "{{vars.smtp_user}}"
      password: "{{vars.smtp_pass}}"
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
vars:
  smtp_username: "{{SMTP_USERNAME}}"

with:
  username: "{{vars.smtp_username}}"
  username: "alerts@example.com"
```

#### `password` (required)

**Type:** String  
**Description:** SMTP authentication password  
**Supports:** Template expressions

```yaml
vars:
  smtp_password: "{{SMTP_PASSWORD}}"
  email_app_password: "{{EMAIL_APP_PASSWORD}}"

with:
  password: "{{vars.smtp_password}}"
  password: "{{vars.email_app_password}}"
```

#### `from` (required)

**Type:** String  
**Description:** Sender email address  
**Supports:** Template expressions

```yaml
vars:
  from_email: "{{FROM_EMAIL}}"

with:
  from: "alerts@example.com"
  from: "{{vars.from_email}}"
  from: "Probe Monitor <probe@example.com>"
```

#### `to` (required)

**Type:** Array of strings  
**Description:** Recipient email addresses  
**Supports:** Template expressions

```yaml
vars:
  alert_email: "{{ALERT_EMAIL}}"

with:
  to: ["admin@example.com"]
  to: ["user1@example.com", "user2@example.com"]
  to: ["{{vars.alert_email}}"]"
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
vars:
  service_name: "{{SERVICE_NAME}}"

with:
  subject: "Alert: Service Down"
  subject: "{{vars.service_name}} Status: {{outputs.health-check.status}}"
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
vars:
  gmail_username: "{{GMAIL_USERNAME}}"
  gmail_app_password: "{{GMAIL_APP_PASSWORD}}"

steps:
  - name: "Send Gmail Alert"
    action: smtp
    with:
      host: "smtp.gmail.com"
      port: 587
      username: "{{vars.gmail_username}}"
      password: "{{vars.gmail_app_password}}"  # Use app password, not account password
      from: "{{vars.gmail_username}}"
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
vars:
  o365_username: "{{O365_USERNAME}}"
  o365_password: "{{O365_PASSWORD}}"

steps:
  - name: "Send Office 365 Alert"
    action: smtp
    with:
      host: "smtp.office365.com"
      port: 587
      username: "{{vars.o365_username}}"
      password: "{{vars.o365_password}}"
      from: "{{vars.o365_username}}"
      to: ["team@company.com"]
      subject: "System Alert"
      body: "Alert message content"
      tls: true
```

#### HTML Email with Multiple Recipients

```yaml
vars:
  smtp_host: "{{SMTP_HOST}}"
  smtp_user: "{{SMTP_USER}}"
  smtp_pass: "{{SMTP_PASS}}"

steps:
  - name: "HTML Status Report"
    action: smtp
    with:
      host: "{{vars.smtp_host}}"
      port: 587
      username: "{{vars.smtp_user}}"
      password: "{{vars.smtp_pass}}"
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
vars:
  user_name: "{{USER_NAME}}"

  message: "Hello {{vars.user_name}}"
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
vars:
  user: "{{USER}}"

      message: "User: {{vars.user}}, Time: {{unixtime()}}"
    test: res.message | contains(vars.user)
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

vars:
  smtp_user: "{{SMTP_USER}}"
  smtp_pass: "{{SMTP_PASS}}"

```yaml
steps:
  - name: "SMTP with Error Handling"
    action: smtp
    with:
      host: "smtp.example.com"
      username: "{{vars.smtp_user}}"
      password: "{{vars.smtp_pass}}"
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
vars:
  api_url: "{{API_URL}}"

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
          url: "{{vars.api_url}}/ping"
          timeout: "2s"
        test: res.status == 200 && res.time < 500
```

### SMTP Action Performance

- **Connection reuse:** SMTP connections are established per action
- **Batch emails:** Consider grouping recipients to reduce connections
- **TLS overhead:** TLS negotiation adds latency

```yaml
# Efficient email notification
vars:
  smtp_user: "{{SMTP_USER}}"
  smtp_pass: "{{SMTP_PASS}}"

steps:
  - name: "Batch Notification"
    action: smtp
    with:
      host: "smtp.example.com"
      username: "{{vars.smtp_user}}"
      password: "{{vars.smtp_pass}}"
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