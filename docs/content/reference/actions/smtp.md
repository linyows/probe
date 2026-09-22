# SMTP Action

The `smtp` action delivers mail to an SMTP server. It is built for measuring and exercising delivery rather than for sending hand-written notifications: the message body is generated, and its size is set with `length`.

## Basic Syntax

```yaml
steps:
  - name: Send a probe mail
    uses: smtp
    with:
      addr: "localhost:2525"
      from: "sender@example.com"
      to: "recipient@example.com"
      subject: "Delivery probe"
      session: 1
      message: 1
      length: 500
    test: res.code == 0 && res.sent > 0
```

## Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `addr` | String | Yes | - | SMTP server as `host:port` |
| `from` | String | Yes | - | Envelope sender |
| `to` | String | Yes | - | Envelope recipient |
| `subject` | String | No | `""` | Subject line |
| `myhostname` | String | No | - | Hostname used in the `HELO` / `EHLO` command |
| `session` | Integer | No | `1` | Number of SMTP sessions to open |
| `message` | Integer | No | `1` | Messages to send per session |
| `length` | Integer | No | `0` | Size of the generated message body in bytes |

There are no parameters for authentication, TLS, CC/BCC, a custom body or HTML. To include a report in the run output, use the step's `echo`.

## Response Object

| Field | Type | Description |
|-------|------|-------------|
| `res.code` | Integer | `0` when every message was delivered |
| `res.sent` | Integer | Messages delivered |
| `res.failed` | Integer | Messages that failed |
| `res.total` | Integer | Messages attempted |
| `res.error` | String | Error message, when delivery failed |
| `res.maildata` | String | The generated message, when it is text |
| `res.filepath` | String | Path to the generated message, when it is binary |

## SMTP Examples

### Several Sessions and Messages

```yaml
steps:
  - name: Deliver 3 messages over 2 sessions
    id: bulk
    uses: smtp
    with:
      addr: "{{vars.smtp_addr}}"
      from: "{{vars.from_addr}}"
      to: "{{vars.to_addr}}"
      subject: "Bulk delivery test"
      myhostname: probe-client.local
      session: 2
      message: 3
      length: 750
    test: res.code == 0 && res.sent == 6
    outputs:
      sent: res.sent
```

### Reporting the Result

```yaml
  - name: Delivery summary
    uses: hello
    echo: |
      Sent: {{outputs.bulk.sent}}
      Round trip: {{rt.duration}}
```
