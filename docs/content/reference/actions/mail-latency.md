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
  test: res.code == 0
```

## Parameters

Both directories are required: one to read from, one to write to.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `mail_dir` | String | Yes | Directory holding the messages to measure |
| `output_dir` | String | Yes | Directory the CSV file is written to |

## Response Object

The result points at the file that was written.

| Field | Type | Description |
|-------|------|-------------|
| `res.output_file` | String | Path of the written CSV file, named `mail-latency.<timestamp>.csv` |
| `res.status` | Integer | `0` on success |
| `rt` | String | Time spent measuring |
