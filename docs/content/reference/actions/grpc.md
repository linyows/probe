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
| `timeout` | String | No | `30s` | Time limit for the call, including connecting and the reflection lookup. A Go duration such as `"10s"`; any other value is an error |
| `tls` | Boolean | No | `false` | Use TLS |
| `insecure` | Boolean | No | `false` | Skip certificate verification |
| `cert_file` | String | No | - | Client certificate for mutual TLS |
| `key_file` | String | No | - | Client key for mutual TLS |
| `ca_file` | String | No | - | CA certificate used to verify the server |
| `proto` | Object or `false` | No | - | Check the call against the `.proto` files listed in `files`, and fail the step when it breaks them. `false` leaves out a check the job's `defaults` ask for. See [Checking Against .proto Files](#checking-against-proto-files) |

## Response Object

After the call, `res` holds the reply and the gRPC status.

| Field | Type | Description |
|-------|------|-------------|
| `res.body` | Any | Response message, parsed from its JSON form into an object. Field names are in lowerCamelCase, such as `createdAt` |
| `res.rawbody` | String | The response message as unparsed JSON |
| `res.status_code` | String | gRPC status code, such as `OK` or `NOT_FOUND` |
| `res.status_message` | String | Status message |
| `res.metadata` | Object | Headers and trailers the server sent, with the first value of each. A trailer wins over a header of the same name |
| `rt.duration` | String | Round-trip time, such as `"1.2ms"` |
| `rt.sec` | Float | Round-trip time in seconds |
| `status` | Integer | `0` when the status is `OK`, otherwise `1` |
| `req` | Object | The request as it was sent |
| `res.violations` | Array | What the `.proto` files do not allow in the call, empty when they allow all of it. Present only when `proto` is given |

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

## Checking Against .proto Files

The call is made with the server's own definition, as its reflection tells it, which says nothing of whether the server keeps to the definition it was meant to. With `proto`, the call is also checked against `.proto` files, such as those in the repository the service is built from:

```yaml
- name: Users
  defaults:
    grpc:
      addr: "{{vars.grpc_addr}}"
      service: user.v1.UserService
      proto:
        files: [./proto/user/v1/user.proto]
        import_paths: [./proto]
  steps:
    - name: Get a user
      uses: grpc
      with:
        method: GetUser
        body:
          id: "123"
      test: res.status_code == "OK" && res.body.user.name != ""
```

`files` are compiled as `protoc` would with `import_paths`, the directory Probe runs in when none is given; each file is named by its path under the first import path that holds it, as an import of it names it. The files are compiled before the call is made, so a step whose files do not compile fails as an action error without calling anything.

The step fails with the kind `contract_request` when the request body, as JSON, is not one the request message of the files takes, and with the kind `contract_response` when:

- the files declare no such method in the service
- the server's request or response message, or a message in them at any depth, has another name than the files', lacks a field the files declare, or declares a field by the same number with another name, type or cardinality
- a field of the response is not encoded as the files declare its number, such as a string where they declare an `int32`

With `strict: true`, a field the server declares, or sends, that the files do not declare fails the step too. Without it such a field is let through, as protobuf lets a newer peer's fields through. A request body that the server's own definition cannot take either cannot be sent, so nothing is sent, and the step fails as `contract_request`, with `res.status_code` empty, rather than as an action error. `request: false` checks the server and its response alone, for a step that sends what the files do not allow on purpose. A response with a status other than `OK` has no message to check. Each violation is in `res.violations`, the terminal and the reports, with the field, such as `$.user.email`.

proto3 declares no required fields and no ranges, so the files say what shape the messages have rather than which values they may hold.

