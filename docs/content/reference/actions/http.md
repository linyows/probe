# HTTP Action

The `http` action performs an HTTP request and exposes the response for assertions and outputs.

## Basic Syntax

An HTTP step needs a URL and a method, and asserts on the response in `test`.

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

The fields below describe the request. All of them accept template expressions.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | String | Yes | - | Request URL, or the base URL when a method shorthand carries a path |
| `method` | String | Yes | - | HTTP method. Supplied by a method shorthand when one is used |
| `headers` | Object | No | - | Request headers |
| `body` | String or Object | No | - | Request body. An object is serialized as JSON when `content-type` is a JSON type: `application/json`, with or without parameters such as `charset`, or one ending in `+json` |
| `timeout` | Duration | No | `30s` | Time limit for the whole request, including reading the response |
| `basic_auth` | Object | No | - | `username` and `password` for HTTP Basic authentication, sent as the `Authorization` header |
| `form` | Object | No | - | Form fields, sent as an `application/x-www-form-urlencoded` body. See [Sending a Form](#sending-a-form) |
| `multipart` | Object | No | - | Form fields and files, sent as a `multipart/form-data` body. See [Uploading Files](#uploading-files) |

There are no parameters for redirects or TLS verification. Redirects are followed by default.

Every request sends `Accept: */*` and `User-Agent: probe-http/1.0.0` unless `headers` sets them. Header names are compared without regard to case, so `user-agent: my-agent` replaces the default.

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

### Sending a Form

`form` sends its fields as an `application/x-www-form-urlencoded` body, as an HTML form does. A field takes a string, a number or a boolean, and a list sends the field once for each of its values.

```yaml
  - name: Log in with a form
    uses: http
    with:
      post: /login
      form:
        user: "{{vars.user}}"
        password: "{{vars.password}}"
        scope: [read, write]
    test: res.code == 302
```

The fields are sorted by name and percent-encoded, and `req.body` shows them as they were sent.

### Uploading Files

`multipart` sends text fields and files as a `multipart/form-data` body. A field written as a string, a number or a boolean is a text field. A field written as a map is a file, read from `file` or given inline as `content`. A list sends the field once for each of its values, so several files can go under one name.

```yaml
  - name: Upload an avatar
    uses: http
    with:
      post: /api/images
      multipart:
        title: avatar
        image:
          file: ./fixtures/logo.png
        attachments:
          - file: ./fixtures/terms.pdf
          - content: "a,b\n1,2\n"
            filename: data.csv
            content_type: text/csv
    test: res.code == 201
```

A file takes these keys:

| Key | Description |
|-----|-------------|
| `file` | Path to the file to send, relative to the working directory, as the paths of other actions are |
| `content` | The content to send, in place of `file` |
| `filename` | Filename sent with the file. Defaults to the base name of `file`, or the field name for `content` |
| `content_type` | Media type of the file. Defaults to the one the extension of `filename` names, or `application/octet-stream` |

Text fields are sent first and files after them, each sorted by name, since some servers, such as S3 for a POST upload, read only the fields that come before the file. The values of a list keep their order. A file is read each time the step runs, so a retried step sends the file as it is then.

The body is sent with its length, and is not shown: `req.body` is empty and `req.multipart` shows what was written instead.

`form` and `multipart` each set the `Content-Type` header, which replaces one given in `headers` or in a job's `defaults`, such as `application/json`. Neither can be given together with `body`, nor with each other.

## Response Object

After the request, `res` holds what came back.

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

The fields above combine into a few shapes that come up in most workflows: carrying a token between steps, asserting on an error response, and retrying a request that is not yet consistent.

### Authentication

The token is captured as an output of the login step and read by the steps that follow.

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

A server that takes HTTP Basic authentication is given the username and password with `basic_auth`, which builds the `Authorization` header from them. Set in a job's `defaults`, it applies to every request of the job.

```yaml
- name: Admin API
  defaults:
    http:
      url: "{{vars.api_url}}"
      basic_auth:
        username: "{{vars.admin_user}}"
        password: "{{vars.admin_password}}"
  steps:
    - name: List users
      uses: http
      with:
        get: /admin/users
      test: res.code == 200 && len(res.body) > 0
```

A username must not contain a colon, which would end it early, and a password may be empty. `basic_auth` cannot be given together with an `authorization` header, since only one of them could be sent. The password and the header built from it are hidden in the output as other credentials are.

### Checking an Error Response

Here the error is the expected result, so the test asserts on the status and the error body.

```yaml
  - name: Unknown id returns 404
    uses: http
    with:
      url: "{{vars.api_url}}/users/does-not-exist"
      method: GET
    test: res.code == 404 && res.body.error != null
```

### Retrying a Flaky Endpoint

`retry` repeats the step until the test passes or the attempts run out.

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
