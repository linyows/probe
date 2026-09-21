# YAML Configuration Reference

This page documents every key Probe reads from a workflow file, the expression context each field is evaluated in, and the validation rules that apply.

## Workflow Structure

```yaml
name: string                  # Required: workflow name
description: string           # Optional: what the workflow does
vars:                         # Optional: workflow variables
  key: value
jobs:                         # Required: a list of jobs
  - name: string              # Required: job name
    id: string                # Optional: job id, used by needs
    needs: [job-id, ...]      # Optional: job dependencies
    skipif: expression        # Optional: skip the job when true
    defaults:                 # Optional: default `with` values per action
      http:
        url: string
    repeat:                   # Optional: run the job repeatedly
      count: integer
      interval: duration
    steps:                    # Required: a list of steps
      - name: string          # Optional: step name
        id: string            # Optional: step id, required to publish outputs
        uses: string          # Required: action name
        with:                 # Optional: action parameters
          key: value
        test: expression      # Optional: assertion
        echo: string          # Optional: text added to the report
        vars:                 # Optional: step variables
          key: value
        outputs:              # Optional: values published for later steps
          key: expression
        skipif: expression    # Optional: skip the step when true
        wait: duration        # Optional: wait before running the step
        timeout: duration     # Optional: step timeout
        iteration:            # Optional: run the step once per entry
          - key: value
        retry:                # Optional: retry on failure
          max_attempts: integer
          interval: duration
          initial_delay: duration
```

`jobs` is a **list**, not a mapping. A workflow that writes `jobs:` as a mapping of job ids fails to load.

There is no top-level `env` key and no top-level `defaults` key. Environment variables are read through `vars`, and `defaults` belongs to a job.

## Top-Level Properties

### `name`

**Type:** String (required)  
**Description:** Name of the workflow, shown at the top of the report.

```yaml
name: "API Health Check"
```

### `description`

**Type:** String (optional)  
**Description:** Longer explanation of what the workflow does.

```yaml
description: |
  Checks the production API:
  - endpoint availability
  - response times
```

### `vars`

**Type:** Object (optional)  
**Description:** Variables available to every job and step as `vars.<name>`.

`vars` is the only place where environment variables are visible, and they are referenced by their bare name. Values are evaluated once, before the first job starts.

```yaml
vars:
  # Read an environment variable
  api_url: "{{API_URL}}"

  # With a fallback
  timeout: "{{REQUEST_TIMEOUT ?? '30s'}}"

  # Computed at load time
  run_id: "{{random_str(8)}}"

  # Nested values are supported
  auth:
    user: "{{API_USER}}"
```

Expressions inside a step cannot read environment variables directly - there is no `env` in the expression context. Put the variable in `vars` and read `vars.<name>`.

## Jobs

### Job Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | String | Yes | Job name, shown in the report. Supports template expressions |
| `id` | String | No | Identifier used by another job's `needs`. Generated automatically when omitted |
| `needs` | Array | No | Ids of jobs that must finish first |
| `steps` | Array | Yes | The steps to run |
| `skipif` | Expression | No | Skip the whole job when the expression is true |
| `defaults` | Object | No | Default `with` values, keyed by action name |
| `repeat` | Object | No | Run the job repeatedly |

There is no `if`, `continue_on_error` or `timeout` at the job level.

#### `needs`

Jobs without dependencies start in parallel. `needs` refers to job **ids**, so a job that others depend on needs an explicit `id`.

```yaml
jobs:
  - id: setup
    name: Setup
    steps:
      - name: Prepare
        uses: hello
        echo: "ready"

  - name: Test
    needs: [setup]
    steps:
      - name: Run
        uses: hello
        echo: "testing"
```

#### `skipif`

A boolean expression. The job is skipped when it evaluates to true.

```yaml
jobs:
  - name: Production only
    skipif: vars.environment != "production"
    steps:
      - name: Check
        uses: http
        with:
          method: GET
          url: "{{vars.api_url}}/health"
```

#### `defaults`

Default parameters merged into the `with` of every step in the job that uses the matching action. A value set on the step wins.

```yaml
jobs:
  - name: API checks
    defaults:
      http:
        url: "{{vars.api_url}}"
        headers:
          authorization: "Bearer {{vars.token}}"
          accept: application/json
    steps:
      - name: Health
        uses: http
        with:
          get: /health        # resolved against the default url
        test: res.code == 200
```

The `http` action requires a method. Either set `method` explicitly, or use the shorthand key `get` / `post` / `put` / `delete` / `patch`, whose value is a full URL or a path resolved against `url`.

```yaml
      - name: Explicit method
        uses: http
        with:
          url: "{{vars.api_url}}/health"
          method: GET
```

#### `repeat`

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `count` | Integer | Yes | Number of runs. Capped by `PROBE_MAX_REPEAT_COUNT` (default 10000) |
| `interval` | Duration | No | Wait between runs |
| `async` | Boolean | No | Run the repetitions concurrently |

```yaml
jobs:
  - name: Poll until ready
    repeat:
      count: 10
      interval: 5s
    steps:
      - name: Check
        uses: http
        with:
          method: GET
          url: "{{vars.api_url}}/status"
        test: res.code == 200
```

## Steps

### Step Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `uses` | String | Yes | Action to run, such as `http` or `shell` |
| `name` | String | No | Step name, shown in the report |
| `id` | String | No | Identifier that namespaces this step's `outputs` |
| `with` | Object | No | Action parameters |
| `test` | Expression | No | Assertion. The step fails when it is false |
| `echo` | String | No | Text added to the report |
| `vars` | Object | No | Variables local to the step |
| `outputs` | Object | No | Values published for later steps and jobs |
| `skipif` | Expression | No | Skip the step when true |
| `wait` | Duration | No | Wait before running the step |
| `timeout` | Duration | No | Step timeout (default 5m) |
| `iteration` | Array | No | Run the step once per entry, exposed as `vars` |
| `retry` | Object | No | Retry the step on failure |

The key is `uses`, not `action`, and the conditional key is `skipif`, not `if`.

#### `outputs`

Values published for use by later steps and jobs. **A step only publishes outputs when it has an `id`** - without one the `outputs` block is discarded.

Each value is an expression, not a template, so it is written without <span v-pre>`{{ }}`</span>.

```yaml
steps:
  - name: Log in
    id: auth
    uses: http
    with:
      url: "{{vars.api_url}}/login"
      method: POST
    test: res.code == 200
    outputs:
      token: res.body.access_token
      user_id: res.body.user.id
```

Later steps read them either namespaced by step id or by the output name alone:

```yaml
      headers:
        Authorization: "Bearer {{outputs.auth.token}}"
        X-User: "{{outputs.user_id}}"
```

An id containing a hyphen is not a valid identifier in an expression, so it has to be read with brackets:

```yaml
    echo: "{{outputs['create-user'].user_id}}"
```

#### `retry`

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `max_attempts` | Integer | Yes | Total attempts, at least 1. Capped by `PROBE_MAX_ATTEMPTS` (default 10000) |
| `interval` | Duration | No | Wait between attempts |
| `initial_delay` | Duration | No | Wait before the first attempt |

```yaml
steps:
  - name: Flaky endpoint
    uses: http
    with:
      method: GET
      url: "{{vars.api_url}}/slow"
    test: res.code == 200
    retry:
      max_attempts: 3
      interval: 2s
```

#### `iteration`

Runs the step once per entry. Each entry's keys are exposed through `vars`.

```yaml
steps:
  - name: Check {{vars.path}}
    uses: http
    iteration:
      - path: /health
      - path: /metrics
      - path: /version
    with:
      method: GET
      url: "{{vars.api_url}}{{vars.path}}"
    test: res.code == 200
```

## The Expression Context

Inside a step, an expression sees exactly these names:

| Name | Type | Description |
|------|------|-------------|
| `vars` | Object | Workflow variables merged with the step's own `vars` |
| `res` | Object | The action's response |
| `req` | Object | The request as it was sent |
| `rt` | Object | Response time: `rt.duration` (string) and `rt.sec` (float seconds) |
| `status` | Integer | Action exit status, `0` on success |
| `outputs` | Object | Outputs published by earlier steps |
| `repeat_index` | Integer | Current index when the job repeats |

There is no `env`, `jobs` or `steps` in this context.

### The `res` Object for `http`

| Field | Type | Description |
|-------|------|-------------|
| `res.code` | Integer | Status code, such as `200` |
| `res.status` | String | Status line, such as `"200 OK"` |
| `res.headers` | Object | Response headers, keyed by canonical name such as `Content-Type` |
| `res.body` | Any | Response body. Parsed into an object or array when the response is JSON, otherwise the raw string |
| `res.rawbody` | String | The unparsed body, present when the body was parsed as JSON |

```yaml
    test: |
      res.code == 200 &&
      res.headers["Content-Type"] contains "application/json" &&
      res.body.status == "ok" &&
      rt.sec < 1
```

Other actions publish their own `res` fields; see the [Actions Reference](/reference/actions-reference).

## Data Types

### Duration

A duration is either a Go duration string or a plain number of seconds.

```yaml
timeout: "30s"
timeout: "5m"
interval: 10        # 10 seconds
```

### Expressions and Templates

A **template expression** appears inside a string and is replaced by its value:

```yaml
url: "{{vars.api_url}}/users/{{outputs.auth.user_id}}"
```

A **boolean expression** is written bare, without braces:

```yaml
test: res.code == 200 && rt.sec < 2
skipif: vars.environment == "local"
```

`outputs` values are expressions too, so they take no braces.

See [Built-in Functions](/reference/built-in-functions) for what can be called inside an expression.

## Validation Rules

- `name` is required at the workflow level, and `jobs` must be a non-empty list.
- Every job needs a `name` and at least one step.
- Every step needs a `uses`.
- A job's `needs` must refer to ids that exist, and the dependency graph must be acyclic.
- `repeat.count` must be zero or greater, and `retry.max_attempts` at least 1.
- A step must have an `id` for its `outputs` to be published.

## File Merging

Several files can be combined by passing them comma-separated. They are concatenated in order, and a top-level key defined more than once takes the value from the last file:

```bash
probe base.yml,production.yml
```

See [File Merging](/guide/concepts/file-merging) for the merge rules.

## See Also

- **[CLI Reference](/reference/cli-reference)** - Command-line options
- **[Actions Reference](/reference/actions-reference)** - Action parameters and responses
- **[Built-in Functions](/reference/built-in-functions)** - Expression functions
- **[Environment Variables](/reference/environment-variables)** - Variables Probe reads
