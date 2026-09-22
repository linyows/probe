# HTTP Action

The `http` action performs an HTTP request and exposes the response for assertions and outputs.

## Basic Syntax

```yaml
steps:
  - name: Check the API
    uses: http
    with:
      url: "https://api.example.com/health"
      method: GET
    test: res.code == 200
```

## Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | String | Yes | - | Request URL, or the base URL when a method shorthand carries a path |
| `method` | String | Yes | - | HTTP method. Supplied by a method shorthand when one is used |
| `headers` | Object | No | - | Request headers |
| `body` | String or Object | No | - | Request body. An object is serialized as JSON when `content-type` is `application/json` |
| `timeout` | Duration | No | `30s` | Time limit for the whole request, including reading the response |

There are no parameters for redirects or TLS verification. Redirects are followed by default.

### `timeout`

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

### Method Shorthands

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

## Response Object

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

## Response Examples

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

## Common HTTP Patterns

### Authentication

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

### Checking an Error Response

```yaml
  - name: Unknown id returns 404
    uses: http
    with:
      url: "{{vars.api_url}}/users/does-not-exist"
      method: GET
    test: res.code == 404 && res.body.error != null
```

### Retrying a Flaky Endpoint

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
