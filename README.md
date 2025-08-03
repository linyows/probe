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
probe --workflow health-check.yml
```

Features
--------

- **Simple YAML Syntax**: Easy-to-read workflow definitions
- **Plugin Architecture**: Built-in HTTP, Shell, SMTP, and Hello actions with extensibility
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
probe --workflow ./workflow.yml

# Verbose output
probe --workflow ./workflow.yml --verbose

# Show response times
probe --workflow ./workflow.yml --rt
```

### CLI Options
- `--workflow <path>`: Specify YAML workflow file
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
  iter:
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
    method: POST  # GET, POST, PUT, DELETE, etc.
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
- `vars.*`: Workflow and step variables
- `env.*`: Environment variables
- `res.*`: Previous step response
- `req.*`: Previous step request
- `outputs.*`: Step outputs
- `steps[i].*`: Historical step data

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
- `PROBE_*`: Custom environment variables accessible via `env.*`

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
probe --workflow test.yml --verbose
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
