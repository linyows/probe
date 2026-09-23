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
| `body` | String | No | `""` | Request message as JSON |
| `metadata` | Object | No | `{}` | Request metadata (the gRPC equivalent of headers) |
| `timeout` | String | No | - | Request timeout, such as `"10s"` |
| `tls` | Boolean | No | `false` | Use TLS |
| `insecure` | Boolean | No | `false` | Skip certificate verification |
| `cert_file` | String | No | - | Client certificate for mutual TLS |
| `key_file` | String | No | - | Client key for mutual TLS |
| `ca_file` | String | No | - | CA certificate used to verify the server |

## Response Object

After the call, `res` holds the reply and the gRPC status.

| Field | Type | Description |
|-------|------|-------------|
| `res.body` | String | Response message as JSON |
| `res.status_code` | String | gRPC status code, such as `OK` or `NOT_FOUND` |
| `res.status_message` | String | Status message |
| `res.metadata` | Object | Response metadata |
| `rt` | String | Round-trip time |
| `req` | Object | The request as it was sent |
