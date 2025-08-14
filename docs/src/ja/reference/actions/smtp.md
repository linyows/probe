# SMTPアクション

`smtp`アクションはSMTPサーバーを通じてメール通知とアラートを送信します。

## 基本的な構文

```yaml
vars:
  smtp_user: "{{SMTP_USER}}"
  smtp_pass: "{{SMTP_PASS}}"

steps:
  - name: "Send Alert"
    uses: smtp
    with:
      host: "smtp.gmail.com"
      username: "{{vars.smtp_user}}"
      password: "{{vars.smtp_pass}}"
      from: "alerts@example.com"
      to: ["admin@example.com"]
      subject: "Service Alert"
      body: "Service is down"
```

## パラメータ

### `host` (必須)

**型:** String  
**説明:** SMTPサーバーのホスト名またはIPアドレス

```yaml
with:
  host: "smtp.gmail.com"
  host: "mail.example.com"
  host: "127.0.0.1"
```

### `port` (オプション)

**型:** Integer  
**デフォルト:** `587`  
**説明:** SMTPサーバーポート

```yaml
with:
  host: "smtp.gmail.com"
  port: 587    # TLS/STARTTLS
  port: 465    # SSL
  port: 25     # Plain
```

### `username` (必須)

**型:** String  
**説明:** SMTP認証ユーザー名  
**サポート:** テンプレート式

```yaml
vars:
  smtp_username: "{{SMTP_USERNAME}}"

with:
  username: "{{vars.smtp_username}}"
  username: "alerts@example.com"
```

### `password` (必須)

**型:** String  
**説明:** SMTP認証パスワード  
**サポート:** テンプレート式

```yaml
vars:
  smtp_password: "{{SMTP_PASSWORD}}"
  email_app_password: "{{EMAIL_APP_PASSWORD}}"

with:
  password: "{{vars.smtp_password}}"
  password: "{{vars.email_app_password}}"
```

### `from` (必須)

**型:** String  
**説明:** 送信者メールアドレス  
**サポート:** テンプレート式

```yaml
vars:
  from_email: "{{FROM_EMAIL}}"

with:
  from: "alerts@example.com"
  from: "{{vars.from_email}}"
  from: "Probe Monitor <probe@example.com>"
```

### `to` (必須)

**型:** 文字列の配列  
**説明:** 受信者メールアドレス  
**サポート:** テンプレート式

```yaml
vars:
  alert_email: "{{ALERT_EMAIL}}"

with:
  to: ["admin@example.com"]
  to: ["user1@example.com", "user2@example.com"]
  to: ["{{vars.alert_email}}"]
```

### `cc` (オプション)

**型:** 文字列の配列  
**説明:** カーボンコピー受信者

```yaml
with:
  to: ["admin@example.com"]
  cc: ["team@example.com", "manager@example.com"]
```

### `bcc` (オプション)

**型:** 文字列の配列  
**説明:** ブラインドカーボンコピー受信者

```yaml
with:
  to: ["admin@example.com"]
  bcc: ["audit@example.com"]
```

#### `subject` (必須)

**型:** String  
**説明:** メール件名行  
**サポート:** テンプレート式

```yaml
vars:
  service_name: "{{SERVICE_NAME}}"

with:
  subject: "Alert: Service Down"
  subject: "{{vars.service_name}} Status: {{outputs.health-check.status}}"
  subject: "Daily Report - {{unixtime() | date('2006-01-02')}}"
```

### `body` (必須)

**型:** String  
**説明:** メール本文コンテンツ  
**サポート:** テンプレート式と複数行文字列

```yaml
with:
  body: "Simple text message"

  # 複数行テキスト
  body: |
    Service Alert Report

    Status: {{outputs.check.status}}
    Timestamp: {{unixtime()}}
    Response Time: {{outputs.check.time}}ms

    Please investigate immediately.

  # HTMLメール (html: trueを設定)
  body: |
    <html>
    <body>
      <h1>Service Alert</h1>
      <p>Status: <strong>{{outputs.check.status}}</strong></p>
      <p>Time: {{unixtime()}}</p>
    </body>
    </html>
```

### `html` (オプション)

**型:** Boolean  
**デフォルト:** `false`  
**説明:** 本文にHTMLコンテンツが含まれているかどうか

```yaml
with:
  subject: "HTML Alert"
  body: "<h1>Alert</h1><p>Service is <strong>down</strong></p>"
  html: true
```

### `tls` (オプション)

**型:** Boolean  
**デフォルト:** `true`  
**説明:** TLS/STARTTLS暗号化を使用するかどうか

```yaml
with:
  host: "smtp.example.com"
  port: 587
  tls: true     # STARTTLSを使用

with:
  host: "smtp.example.com"
  port: 465
  tls: false    # SSL使用 (ポート465は通常暗黙的SSLを使用)
```

## レスポンスオブジェクト

SMTPアクションは次のプロパティを持つ`res`オブジェクトを提供します：

 | プロパティ   | 型      | 説明                                                 |
 | ----------   | ------  | -------------                                        |
 | `success`    | Boolean | メールが正常に送信されたかどうか                     |
 | `message_id` | String  | 一意のメッセージ識別子（サーバーから提供された場合） |
 | `time`       | Integer | メール送信にかかった時間（ミリ秒）                   |

## SMTP例

### Gmail設定

```yaml
vars:
  gmail_username: "{{GMAIL_USERNAME}}"
  gmail_app_password: "{{GMAIL_APP_PASSWORD}}"

steps:
  - name: "Send Gmail Alert"
    uses: smtp
    with:
      host: "smtp.gmail.com"
      port: 587
      username: "{{vars.gmail_username}}"
      password: "{{vars.gmail_app_password}}"  # アカウントパスワードではなくアプリパスワードを使用
      from: "{{vars.gmail_username}}"
      to: ["admin@example.com"]
      subject: "Probe Alert - {{unixtime() | date('15:04')}}"
      body: |
        Alert from Probe workflow.

        Details:
        - Workflow: {{workflow.name}}
        - Time: {{unixtime()}}
        - Status: Failed
```

### Office 365設定

```yaml
vars:
  o365_username: "{{O365_USERNAME}}"
  o365_password: "{{O365_PASSWORD}}"

steps:
  - name: "Send Office 365 Alert"
    uses: smtp
    with:
      host: "smtp.office365.com"
      port: 587
      username: "{{vars.o365_username}}"
      password: "{{vars.o365_password}}"
      from: "{{vars.o365_username}}"
      to: ["team@company.com"]
      subject: "System Alert"
      body: "Alert message content"
      tls: true
```

### 複数受信者でのHTMLメール

```yaml
vars:
  smtp_host: "{{SMTP_HOST}}"
  smtp_user: "{{SMTP_USER}}"
  smtp_pass: "{{SMTP_PASS}}"

steps:
  - name: "HTML Status Report"
    uses: smtp
    with:
      host: "{{vars.smtp_host}}"
      port: 587
      username: "{{vars.smtp_user}}"
      password: "{{vars.smtp_pass}}"
      from: "reports@example.com"
      to: ["admin@example.com", "ops@example.com"]
      cc: ["manager@example.com"]
      subject: "Daily Health Report - {{unixtime() | date('2006-01-02')}}"
      html: true
      body: |
        <html>
        <head><title>Health Report</title></head>
        <body>
          <h1>Daily Health Report</h1>
          <table border="1">
            <tr><th>Service</th><th>Status</th><th>Response Time</th></tr>
            <tr><td>API</td><td style="color: {{outputs.api.success ? 'green' : 'red'}}">{{outputs.api.status}}</td><td>{{outputs.api.time}}ms</td></tr>
            <tr><td>Database</td><td style="color: {{outputs.db.success ? 'green' : 'red'}}">{{outputs.db.status}}</td><td>{{outputs.db.time}}ms</td></tr>
          </table>
          <p>Generated at {{unixtime()}}</p>
        </body>
        </html>
```

