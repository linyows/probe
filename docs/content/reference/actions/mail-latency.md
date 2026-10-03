# Mail Latency Action

The `mail-latency` action reads messages from a Maildir, computes the delivery latency of each one from its `Received` headers, and writes the result as a CSV file.

## Basic Syntax

The action reads the messages in a directory and writes the measurements as a CSV file.

```yaml
- name: Measure delivery latency
  uses: mail-latency
  with:
    mail_dir: "/var/mail/probe/new"
    output_dir: "./reports"
  test: res.status == 0
```

## Parameters

Both directories are required: one to read from, one to write to.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `mail_dir` | String | Yes | Directory holding the messages to measure |
| `output_dir` | String | Yes | Directory the CSV file is written to |

`mail_dir` is read recursively, subdirectories included. A file is measured when it starts with a `Return-Path:` or `Delivered-To:` header, which is how a delivered message usually begins. The action fails, and writes no CSV, when any file in the directory has no `Date` header, or when a measured message has no `Received` header.

Each message becomes one row of the CSV, with these columns:

- `Sent Time Offset (sec)` and `Received Time Offset (sec)`: when the message was sent and received, counted from the earliest `Date` in the directory
- `Sent Time`: the `Date` header
- `Last Received Time` and `First Received Time`: the topmost and the bottom `Received` header
- `End-to-End Latency (sec)`: from `Date` to the topmost `Received`
- `Relay Latency (sec)`: from the bottom `Received` to the topmost one
- `Return Path` and `File Path`: the envelope sender and the file name

## Response Object

The result points at the file that was written.

| Field | Type | Description |
|-------|------|-------------|
| `res.output_file` | String | Path of the written CSV file, named `mail-latency.<timestamp>.csv` |
| `res.status` | Integer | `0` on success |
| `rt.duration` | String | Time spent measuring, such as `"1.2ms"` |
| `rt.sec` | Float | Time spent measuring in seconds |
| `status` | Integer | `0` on success |
