# gRPC Action

The `grpc` action calls a gRPC method. The service definition is resolved through server reflection, so no `.proto` file is needed at run time.

## Basic Syntax

A gRPC step names the server, the service, the method, and the request body.

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

## Parameters

The fields below describe the call. All of them accept template expressions.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `addr` | String | Yes | - | Host and port of the gRPC server |
| `service` | String | Yes | - | Fully qualified service name |
| `method` | String | Yes | - | Method name |
| `body` | String or Object | No | `""` | Request message as JSON. An object is serialized as JSON |
| `metadata` | Object | No | `{}` | Request metadata (the gRPC equivalent of headers) |
| `timeout` | String | No | `30s` | Time limit for the call, including connecting and the reflection lookup. A Go duration such as `"10s"` |
| `tls` | Boolean | No | `false` | Use TLS |
| `insecure` | Boolean | No | `false` | Skip certificate verification |
| `cert_file` | String | No | - | Client certificate for mutual TLS |
| `key_file` | String | No | - | Client key for mutual TLS |
| `ca_file` | String | No | - | CA certificate used to verify the server |

## Response Object

After the call, `res` holds the reply and the gRPC status.

| Field | Type | Description |
|-------|------|-------------|
| `res.body` | Any | Response message, parsed from its JSON form into an object. Field names are in lowerCamelCase, such as `createdAt` |
| `res.rawbody` | String | The response message as unparsed JSON |
| `res.status_code` | String | gRPC status code, such as `OK` or `NOT_FOUND` |
| `res.status_message` | String | Status message |
| `res.metadata` | Object | Response metadata |
| `rt.duration` | String | Round-trip time, such as `"1.2ms"` |
| `rt.sec` | Float | Round-trip time in seconds |
| `status` | Integer | `0` when the status is `OK`, otherwise `1` |
| `req` | Object | The request as it was sent |

`res.status_code` is the canonical name of the status the call ended with: `OK`, `CANCELLED`, `UNKNOWN`, `INVALID_ARGUMENT`, `DEADLINE_EXCEEDED`, `NOT_FOUND`, `ALREADY_EXISTS`, `PERMISSION_DENIED`, `RESOURCE_EXHAUSTED`, `FAILED_PRECONDITION`, `ABORTED`, `OUT_OF_RANGE`, `UNIMPLEMENTED`, `INTERNAL`, `UNAVAILABLE`, `DATA_LOSS` or `UNAUTHENTICATED`. A status other than `OK` is the server's answer, so the step goes on to its test, which can expect it. Only a call that gets no status from the server ends the step with an error: one to a server that cannot be reached or whose reflection does not list the service, or one that runs out of `timeout` or loses its connection before the server answers.

## Usage Examples

### Expecting an Error Status

When an error is the expected outcome, the test checks the status code.

```yaml
steps:
  - name: Unknown user returns NOT_FOUND
    uses: grpc
    with:
      addr: "{{vars.grpc_addr}}"
      service: "user.v1.UserService"
      method: "GetUser"
      tls: true
      body: |
        {"id": "does-not-exist"}
    test: res.status_code == "NOT_FOUND"
```
