<p align="right">English | <a href="https://github.com/linyows/probe/blob/main/README.ja.md">日本語</a></p>

<br><br><br><br><br><br>

<p align="center">
  <img alt="PROBE" src="https://github.com/linyows/probe/blob/main/misc/probe.svg" width="200">
</p>

<br><br><br><br><br><br>

<p align="center">
  <a href="https://github.com/linyows/probe/actions/workflows/build.yml">
    <img alt="GitHub Workflow Status" src="https://img.shields.io/github/actions/workflow/status/linyows/probe/build.yml?branch=main&style=for-the-badge&labelColor=666666">
  </a>
  <a href="https://github.com/linyows/probe/releases">
    <img src="http://img.shields.io/github/release/linyows/probe.svg?style=for-the-badge&labelColor=666666&color=DDDDDD" alt="GitHub Release">
  </a>
  <a href="http://godoc.org/github.com/linyows/probe">
    <img src="http://img.shields.io/badge/go-docs-blue.svg?style=for-the-badge&labelColor=666666&color=DDDDDD" alt="Go Documentation">
  </a>
  <a href="https://deepwiki.com/linyows/probe">
    <img src="http://img.shields.io/badge/deepwiki-docs-purple.svg?style=for-the-badge&labelColor=666666&color=DDDDDD" alt="Deepwiki Documentation">
  </a>
</p>

A powerful YAML-based workflow automation tool designed for testing, monitoring, and automation tasks. Probe uses plugin-based actions to execute workflows, making it highly flexible and extensible.

![Architecture](/misc/probe-architecture.svg)

Quick Start
-----------

```yaml
name: API Health Check
jobs:
- name: Check API Status
  steps:
  - name: Ping API
    uses: http
    with:
      url: https://api.example.com
      get: /health
    test: res.code == 200
```

```bash
probe health-check.yml
```

Features
--------

- **Simple YAML Syntax**: Easy-to-read workflow definitions
- **Plugin Architecture**: Built-in HTTP, Database, Browser, Shell, SSH, SMTP, IMAP, and Hello actions with extensibility
- **Job Dependencies**: Control execution order with `needs`
- **Step Outputs**: Share data between steps and jobs using `outputs`
- **Repetition**: Repeat jobs with configurable intervals
- **Iteration**: Execute steps with different variable sets
- **Expression Engine**: Powerful templating and condition evaluation
- **Testing Support**: Built-in assertion system
- **Timing Controls**: Wait conditions and delays
- **Rich Output**: Buffered, consistent output formatting
- **Security**: Safe expression evaluation with timeout protection

Installation
------------

### Using Go
```bash
go install github.com/linyows/probe/cmd/probe@latest
```

### From Source
```bash
git clone https://github.com/linyows/probe.git
cd probe
go build -o probe ./cmd/probe
```

Usage
-----

### Basic Usage
```bash
# Run a workflow
probe ./workflow.yml

# Verbose output
probe ./workflow.yml --verbose

# Show response times
probe ./workflow.yml --rt
```

### CLI Options
- `<workflow>`: Specify YAML workflow file path
- `--verbose`: Enable detailed output
- `--rt`: Show response times
- `--help`: Show help information

Workflow Syntax
---------------

### Basic Structure
```yaml
name: Workflow Name
description: Optional description
vars:
  global_var: value
jobs:
- name: Job Name
  steps:
  - name: Step Name
    uses: action_name
    with:
      param: value
```

### Jobs
Jobs are executed in parallel by default. Use `needs` for dependencies:

```yaml
jobs:
- name: Setup
  id: setup
  steps:
  - name: Initialize
    uses: http
    with:
      post: /setup

- name: Test
  needs: [setup]  # Wait for setup to complete
  steps:
  - name: Run test
    uses: http
    with:
      get: /test
```

### Steps
Steps within a job execute sequentially:

```yaml
steps:
- name: Login
  id: login
  uses: http
  with:
    post: /auth/login
    body:
      username: admin
      password: secret
  outputs:
    token: res.body.token

- name: Get Profile
  uses: http
  with:
    get: /profile
    headers:
      authorization: "Bearer {{outputs.login.token}}"
  test: res.code == 200
```

### Variables and Expressions
Use `{{expression}}` syntax for dynamic values:

```yaml
vars:
  api_url: https://api.example.com
  user_id: 123

steps:
- name: Get User
  uses: http
  with:
    url: "{{vars.api_url}}"
    get: "/users/{{vars.user_id}}"
  test: res.body.id == vars.user_id
```

### Built-in Functions
```yaml
test: |
  res.code == 200 &&
  match_json(res.body, vars.expected) &&
  random_int(10) > 5
```

Available functions:
- `match_json(actual, expected)`: JSON pattern matching with regex support
- `diff_json(actual, expected)`: Show differences between JSON objects
- `random_int(max)`: Generate random integer
- `random_str(length)`: Generate random string
- `unixtime()`: Current Unix timestamp

### Output Management
Share data between steps and jobs:

```yaml
- name: Get Token
  id: auth
  uses: http
  with:
    post: /auth
  outputs:
    token: res.body.access_token
    expires: res.body.expires_in

- name: Use Token
  uses: http
  with:
    get: /protected
    headers:
      authorization: "Bearer {{outputs.auth.token}}"
```

### Iteration
Execute steps with different variable sets:

```yaml
- name: Test Multiple Users
  uses: http
  with:
    post: /users
    body:
      name: "{{vars.name}}"
      role: "{{vars.role}}"
  test: res.code == 201
  iteration:
  - {name: "Alice", role: "admin"}
  - {name: "Bob", role: "user"}
  - {name: "Carol", role: "editor"}
```

### Job Repetition
Repeat entire jobs with intervals:

```yaml
- name: Health Check
  repeat:
    count: 10
    interval: 30s
  steps:
  - name: Ping
    uses: http
    with:
      get: /health
    test: res.code == 200
```

### Conditional Execution
Skip steps based on conditions:

```yaml
- name: Conditional Step
  uses: http
  with:
    get: /api/data
  skipif: vars.skip_test == true
  test: res.code == 200
```

### Timing and Delays
```yaml
- name: Wait and Check
  uses: http
  with:
    get: /status
  wait: 5s  # Wait before execution
  test: res.code == 200
```

Built-in Actions
----------------

### HTTP Action
```yaml
- name: HTTP Request
  uses: http
  with:
    url: https://api.example.com
    # GET, POST, PUT, DELETE, etc.
    post: /
    headers:
      content-type: application/json
      authorization: Bearer token
    body:
      key: value
    timeout: 30s
```

### SMTP Action
```yaml
- name: Send Email
  uses: smtp
  with:
    addr: smtp.example.com:587
    from: sender@example.com
    to: recipient@example.com
    subject: Test Email
    body: Email content
    my-hostname: localhost
```

### Database Action
```yaml
- name: Database Query
  uses: db
  with:
    dsn: "mysql://user:password@localhost:3306/database"
    query: "SELECT * FROM users WHERE active = ?"
    params: [true]
    timeout: 30s
  test: res.code == 0 && res.rows_affected > 0
```

Supported databases:
- **MySQL**: `mysql://user:pass@host:port/database`
- **PostgreSQL**: `postgres://user:pass@host:port/database?sslmode=disable`
- **SQLite**: `file:./testdata/sqlite.db` or `/absolute/path/database`

### Browser Action
```yaml
- name: Web Automation
  uses: browser
  with:
    action: navigate
    url: "https://example.com"
    headless: true
    timeout: 30s
  test: res.success == "true"
```

Supported actions:
- **navigate**: Navigate to URL
- **text**: Extract text content from elements
- **value**: Get input field values
- **get_attribute**: Get element attribute values
- **get_html**: Extract HTML content from elements
- **click**: Click on elements
- **double_click**: Double-click on elements
- **right_click**: Right-click on elements
- **hover**: Hover over elements
- **focus**: Set focus to elements
- **type** / **send_keys**: Type text into input fields
- **select**: Select dropdown options
- **submit**: Submit forms
- **scroll**: Scroll elements into view
- **screenshot**: Capture element screenshots
- **capture_screenshot**: Capture full page screenshots
- **full_screenshot**: Capture full page screenshots with quality settings
- **wait_visible**: Wait for elements to become visible
- **wait_not_visible**: Wait for elements to become invisible
- **wait_ready**: Wait for page to be ready
- **wait_text**: Wait for specific text to appear
- **wait_enabled**: Wait for elements to become enabled

### Shell Action
```yaml
- name: Run Build Script
  uses: shell
  with:
    cmd: "npm run build"
    workdir: "/app"
    shell: "/bin/bash"
    timeout: "5m"
    env:
      NODE_ENV: production
  test: res.code == 0 && (res.stdout | contains("Build successful"))
```

### SSH Action
```yaml
- name: Deploy Application
  uses: ssh
  with:
    host: "prod.example.com"
    port: 22
    user: "deploy"
    # Authentication methods (use either password or key_file)
    password: "secure_password"
    # key_file: "~/.ssh/deploy_key"
    # key_passphrase: "key_passphrase"  # if key is encrypted
    cmd: |
      cd /opt/myapp
      git pull origin main
      systemctl restart myapp
    timeout: "300s"
    workdir: "/opt/myapp"
    # Environment variables
    env:
      DEPLOY_ENV: production
      APP_VERSION: v1.2.3
    # Security settings
    strict_host_check: true
    known_hosts: "~/.ssh/known_hosts"
  test: res.code == 0 && !contains(res.stderr, "error")
```

### IMAP Action
```yaml
- name: Email Operations
  uses: imap
  with:
    host: "imap.example.com"
    port: 993
    username: "user@example.com"
    password: "password"
    tls: true
    timeout: 30s
    strict_host_check: true
    insecure_skip_tls: false
    commands:
    - name: "select"
      mailbox: "INBOX"
    - name: "search"
      criteria:
        since: "today"
        flags: ["unseen"]
    - name: "fetch"
      sequence: "*"
      dataitem: "ALL"
  test: res.code == 0 && res.data.search.count > 0
```

Supported IMAP commands:
- **select**: Select a mailbox for read-write operations
- **examine**: Select a mailbox for read-only operations  
- **search**: Search messages using criteria
- **uid search**: Search using UID instead of sequence numbers
- **list**: List available mailboxes
- **fetch**: Fetch message data
- **uid fetch**: Fetch using UID instead of sequence numbers
- **store**: Store message flags (basic implementation)
- **uid store**: Store using UID (basic implementation)
- **copy**: Copy messages to another mailbox (basic implementation)
- **uid copy**: Copy using UID (basic implementation)
- **create**: Create a new mailbox
- **delete**: Delete a mailbox
- **rename**: Rename a mailbox
- **subscribe**: Subscribe to a mailbox
- **unsubscribe**: Unsubscribe from a mailbox
- **noop**: No operation (keepalive)

Search criteria support:
- **seq_nums**: Sequence number ranges (e.g., "1:10", "*")
- **uids**: UID ranges for UID search commands
- **since**: Messages received since date ("today", "yesterday", "2024-01-01", "1 hour ago")
- **before**: Messages received before date
- **sent_since**: Messages sent since date
- **sent_before**: Messages sent before date
- **headers**: Header field matches (e.g., {"from": "sender@example.com"})
- **bodies**: Body text contains
- **texts**: Text (headers + body) contains
- **flags**: Message flags (e.g., ["seen", "answered"])
- **not_flags**: Messages without these flags

### Hello Action (Testing)
```yaml
- name: Test Action
  uses: hello
  with:
    name: World
```

Advanced Examples
-----------------

### REST API Testing
```yaml
name: User API Test
vars:
  base_url: https://api.example.com
  admin_token: "{{env.ADMIN_TOKEN}}"

jobs:
- name: User CRUD Operations
  defaults:
    http:
      url: "{{vars.base_url}}"
      headers:
        authorization: "Bearer {{vars.admin_token}}"
        content-type: application/json

  steps:
  - name: Create User
    id: create
    uses: http
    with:
      post: /users
      body:
        name: Test User
        email: test@example.com
    test: res.code == 201
    outputs:
      user_id: res.body.id

  - name: Get User
    uses: http
    with:
      get: "/users/{{outputs.create.user_id}}"
    test: |
      res.code == 200 &&
      res.body.name == "Test User"

  - name: Update User
    uses: http
    with:
      put: "/users/{{outputs.create.user_id}}"
      body:
        name: Updated User
    test: res.code == 200

  - name: Delete User
    uses: http
    with:
      delete: "/users/{{outputs.create.user_id}}"
    test: res.code == 204
```

### Load Testing with Repetition
```yaml
name: Load Test
jobs:
- name: Concurrent Requests
  repeat:
    count: 100
    interval: 100ms
  steps:
  - name: API Call
    uses: http
    with:
      url: https://api.example.com
      get: /endpoint
    test: res.code == 200
```

### Multi-Service Integration
```yaml
name: E2E Test
jobs:
- name: Setup Database
  id: db-setup
  steps:
  - name: Initialize
    uses: http
    with:
      post: http://db-service/init

- name: API Tests
  needs: [db-setup]
  steps:
  - name: Test API
    uses: http
    with:
      get: http://api-service/data
    test: res.code == 200

- name: Cleanup
  needs: [db-setup]
  steps:
  - name: Reset Database
    uses: http
    with:
      post: http://db-service/reset
```

Expression Reference
--------------------

### Context Variables
- `vars.*`: Workflow and step variables (including environment variables defined in vars)
- `res.*`: Previous step response
- `req.*`: Previous step request  
- `outputs.*`: Step outputs

### Response Object
```yaml
res:
  code: 200
  status: "200 OK"
  headers:
    content-type: application/json
  body: {...}  # Parsed JSON or raw string
  rawbody: "..." # Original response body
```

### Operators
- Arithmetic: `+`, `-`, `*`, `/`, `%`
- Comparison: `==`, `!=`, `<`, `<=`, `>`, `>=`
- Logical: `&&`, `||`, `!`
- Ternary: `condition ? true_value : false_value`

Extending Probe
---------------

Probe supports custom actions through Protocol Buffers. Create your own actions to extend functionality beyond the built-in HTTP, SMTP, and Hello actions.

Configuration
-------------

### Environment Variables
- Environment variables can be accessed by defining them in the root `vars` section

### Defaults
Use YAML anchors for common configurations:
```yaml
x_defaults: &api_defaults
  http:
    url: https://api.example.com
    headers:
      authorization: Bearer token

jobs:
- name: Test Job
  defaults: *api_defaults
```

Troubleshooting
---------------

### Common Issues

**Expression Evaluation Errors**
- Check syntax: `{{expression}}` not `{expression}`
- Verify variable names and paths
- Use quotes around string values

**HTTP Action Issues**
- Verify URL format and accessibility
- Check headers and authentication
- Review timeout settings

**Job Dependencies**
- Ensure job IDs are unique
- Check `needs` references
- Avoid circular dependencies

### Debug Output
Use `--verbose` flag for detailed execution information:
```bash
probe test.yml --verbose
```

Best Practices
--------------

1. **Use descriptive names** for workflows, jobs, and steps
2. **Leverage outputs** for data sharing between steps
3. **Implement proper testing** with meaningful assertions
4. **Use defaults** to reduce repetition
5. **Structure workflows** logically with clear dependencies
6. **Handle errors** gracefully with appropriate tests
7. **Use variables** for configuration management

Contributing
------------

We welcome contributions! Please feel free to submit issues, feature requests, or pull requests.

License
-------

This project is licensed under the MIT License - see the LICENSE file for details.

Author
------

[linyows](https://github.com/linyows)

---

For more examples and advanced usage, check the [examples directory](./examples/).
