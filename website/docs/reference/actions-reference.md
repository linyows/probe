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
with:
  url: "https://api.example.com/users"
  url: "{{env.API_BASE_URL}}/v1/health"
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
with:
  url: "https://api.example.com/users"
  headers:
    Authorization: "Bearer {{env.API_TOKEN}}"
    Content-Type: "application/json"
    User-Agent: "Probe Monitor v1.0"
    X-Request-ID: "{{uuid()}}"
```

#### `body` (optional)

**Type:** String  
**Description:** Request body content  
**Supports:** Template expressions and multi-line strings

```yaml
# JSON body
with:
  url: "https://api.example.com/users"
  method: "POST"
  headers:
    Content-Type: "application/json"
  body: |
    {
      "name": "{{env.USER_NAME}}",
      "email": "{{env.USER_EMAIL}}",
      "active": true
    }

# Form data
with:
  url: "https://api.example.com/form"
  method: "POST"
  headers:
    Content-Type: "application/x-www-form-urlencoded"
  body: "name={{env.USER_NAME}}&email={{env.USER_EMAIL}}"

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
# Bearer token
steps:
  - name: "Authenticated Request"
    action: http
    with:
      url: "{{env.API_URL}}/protected"
      headers:
        Authorization: "Bearer {{env.ACCESS_TOKEN}}"

# Basic auth
  - name: "Basic Auth Request"
    action: http
    with:
      url: "{{env.API_URL}}/basic"
      headers:
        Authorization: "Basic {{base64(env.USERNAME + ':' + env.PASSWORD)}}"

# API key
  - name: "API Key Request"
    action: http
    with:
      url: "{{env.API_URL}}/data"
      headers:
        X-API-Key: "{{env.API_KEY}}"
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