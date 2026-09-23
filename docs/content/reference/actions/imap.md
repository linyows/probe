# IMAP Action

The `imap` action connects to IMAP servers to perform email operations such as reading messages, searching, and mailbox management.

## Basic Syntax

An IMAP step opens a connection and then runs the commands listed in `commands`.

```yaml
vars:
  imap_username: "{{IMAP_USERNAME}}"
  imap_password: "{{IMAP_PASSWORD}}"

steps:
  - name: "Check Email"
    uses: imap
    with:
      host: "imap.example.com"
      port: 993
      username: "{{vars.imap_username}}"
      password: "{{vars.imap_password}}"
      tls: true
      commands:
      - name: "select"
        mailbox: "INBOX"
      - name: "search"
        criteria:
          flags: ["unseen"]
    test: res.code == 0
```

## Parameters

The parameters describe the server to reach, the credentials to present, how the connection is secured, and the list of IMAP commands to run once it is open.

### `host` (required)

**Type:** String  
**Description:** IMAP server hostname or IP address  
**Supports:** Template expressions

```yaml
with:
  host: "imap.gmail.com"
  host: "imap.example.com" 
  host: "{{vars.imap_server}}"
```

### `port` (optional)

**Type:** Integer  
**Default:** `993`  
**Description:** IMAP server port

```yaml
with:
  port: 993   # IMAPS (SSL/TLS)
  port: 143   # IMAP (plain or STARTTLS)
```

### `username` (required)

**Type:** String  
**Description:** IMAP authentication username  
**Supports:** Template expressions

```yaml
vars:
  email_user: "{{EMAIL_USER}}"

with:
  username: "{{vars.email_user}}"
  username: "user@example.com"
```

### `password` (required)

**Type:** String  
**Description:** IMAP authentication password  
**Supports:** Template expressions

```yaml
vars:
  email_password: "{{EMAIL_PASSWORD}}"
  app_password: "{{EMAIL_APP_PASSWORD}}"

with:
  password: "{{vars.email_password}}"
  password: "{{vars.app_password}}"
```

### `tls` (optional)

**Type:** Boolean  
**Default:** `true`  
**Description:** Whether to use TLS/SSL encryption

```yaml
with:
  host: "imap.example.com"
  port: 993
  tls: true     # Use TLS (recommended)
  
with:
  host: "imap.example.com" 
  port: 143
  tls: false    # Plain connection (not recommended)
```

### `timeout` (optional)

**Type:** Duration  
**Default:** `30s`  
**Description:** Connection and operation timeout

```yaml
with:
  timeout: "60s"
  timeout: "2m"
```

### `commands` (required)

**Type:** Array of command objects  
**Description:** IMAP commands to execute sequentially

```yaml
with:
  commands:
  - name: "select"
    mailbox: "INBOX"
  - name: "search"
    criteria:
      since: "today"
  - name: "fetch"
    sequence: "1:5"
    dataitem: "ALL"
```

## IMAP Commands

Each entry in `commands` names one IMAP operation and carries the fields that operation needs. The commands below are the ones the action understands.

### `select` - Select Mailbox

Select a mailbox for read-write operations.

```yaml
- name: "select"
  mailbox: "INBOX"      # Required: mailbox name
- name: "select" 
  mailbox: "Sent"
- name: "select"
  mailbox: "INBOX/Work"
```

### `examine` - Read-only Mailbox Access

Select a mailbox for read-only operations.

```yaml
- name: "examine"
  mailbox: "INBOX"      # Required: mailbox name
```

### `search` - Search Messages

Search messages using various criteria.

```yaml
- name: "search"
  criteria:
    since: "today"           # Date-based search
    flags: ["unseen"]        # Flag-based search  
    headers:                 # Header-based search
      from: "sender@example.com"
      subject: "urgent"
    bodies: ["important"]    # Body text search
    texts: ["meeting"]       # Full-text search
```

### `list` - List Mailboxes

List available mailboxes.

```yaml
- name: "list"
  reference: ""         # Optional: reference name
  pattern: "*"          # Optional: mailbox pattern (default: "*")
- name: "list"
  reference: "INBOX"
  pattern: "INBOX/*"
```

### `fetch` - Fetch Message Data

Retrieve message data using sequence numbers.

```yaml
- name: "fetch"
  sequence: "1:5"       # Required: sequence range
  dataitem: "ALL"       # Required: data items to fetch
- name: "fetch"
  sequence: "*"         # Latest message
  dataitem: "ENVELOPE FLAGS"
```

## Response Object

The IMAP action provides a `res` object with the following structure:

| Property | Type | Description |
|----------|------|-------------|
| `code` | Integer | Operation result (0 = success, non-zero = error) |
| `data` | Object | Command results organized by command type |
| `error` | String | Error message if operation failed |

## IMAP Examples

Providers differ in port, TLS and credentials. The example below is a working configuration for Gmail.

### Gmail Configuration

Gmail requires an app password rather than the account password, and TLS on port 993.

```yaml
vars:
  gmail_username: "{{GMAIL_USERNAME}}"
  gmail_app_password: "{{GMAIL_APP_PASSWORD}}"

steps:
  - name: "Check Gmail Inbox"
    uses: imap
    with:
      host: "imap.gmail.com"
      port: 993
      username: "{{vars.gmail_username}}"
      password: "{{vars.gmail_app_password}}"  # Use App Password
      tls: true
      commands:
      - name: "select"
        mailbox: "INBOX"
      - name: "search"
        criteria:
          flags: ["unseen"]
          since: "today"
      - name: "fetch"
        sequence: "*"
        dataitem: "ENVELOPE FLAGS"
    test: res.code == 0
    outputs:
      unread_count: res.data.search.count
      latest_sender: res.data.fetch.messages__0__from
```
