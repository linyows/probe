# アクションリファレンス

このページでは、すべての組み込みProbeアクションの包括的なドキュメントを提供します。パラメータ、レスポンス形式、使用例を含みます。

## 概要

アクションはProbeワークフローの構成要素です。HTTPリクエストの送信、メールの送信、カスタムロジックの実行など、特定のタスクを実行します。すべてのアクションはテストと出力で使用できる構造化されたレスポンスデータを返します。

### 組み込みアクション

- **[http](#httpアクション)** - HTTP/HTTPSリクエストの送信とレスポンスの検証
- **[db](#データベースアクション)** - MySQL、PostgreSQL、SQLiteでのデータベースクエリの実行
- **[browser](#ブラウザアクション)** - ChromeDPを使用したWebブラウザの自動化
- **[shell](#シェルアクション)** - シェルコマンドとスクリプトの安全な実行
- **[smtp](#smtpアクション)** - メール通知とアラートの送信
- **[hello](#helloアクション)** - 開発とデバッグ用のシンプルなテストアクション

## HTTPアクション

`http`アクションはHTTP/HTTPSリクエストを実行し、テストと検証のための詳細なレスポンス情報を提供します。

### 基本的な構文

```yaml
steps:
  - name: "API Request"
    uses: http
    with:
      url: "https://api.example.com/endpoint"
      method: "GET"
    test: res.code == 200
```

### パラメータ

#### `url` (必須)

**型:** String  
**説明:** リクエストを送信するURL  
**サポート:** テンプレート式

```yaml
vars:
  api_base_url: "{{API_BASE_URL ?? 'https://api.example.com'}}"

with:
  url: "https://api.example.com/users"
  url: "{{vars.api_base_url}}/v1/health"
  url: "https://api.example.com/users/{{outputs.auth.user_id}}"
```

#### `method` (オプション)

**型:** String  
**デフォルト:** `GET`  
**値:** `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`

```yaml
with:
  url: "https://api.example.com/users"
  method: "POST"
```

#### `headers` (オプション)

**型:** Object  
**説明:** リクエストに含めるHTTPヘッダー  
**サポート:** 値でのテンプレート式

```yaml
vars:
  api_token: "{{API_TOKEN}}"

with:
  url: "https://api.example.com/users"
  headers:
    Authorization: "Bearer {{vars.api_token}}"
    Content-Type: "application/json"
    User-Agent: "Probe Monitor v1.0"
    X-Request-ID: "{{unixtime()}}"
```

#### `body` (オプション)

**型:** String  
**説明:** リクエストボディコンテンツ  
**サポート:** テンプレート式と複数行文字列

```yaml
# JSONボディ
vars:
  user_name: "{{USER_NAME}}"
  user_email: "{{USER_EMAIL}}"

with:
  url: "https://api.example.com/users"
  method: "POST"
  headers:
    Content-Type: "application/json"
  body: |
    {
      "name": "{{vars.user_name}}",
      "email": "{{vars.user_email}}",
      "active": true
    }

# フォームデータ
with:
  url: "https://api.example.com/form"
  method: "POST"
  headers:
    Content-Type: "application/x-www-form-urlencoded"
  body: "name={{vars.user_name}}&email={{vars.user_email}}"

# テンプレート式
with:
  url: "https://api.example.com/users"
  method: "PUT"
  body: "{{outputs.user-data.json | tojson}}"
```

#### `timeout` (オプション)

**型:** Duration  
**デフォルト:** `defaults.http.timeout`を継承、または`30s`  
**説明:** リクエストタイムアウト

```yaml
with:
  url: "https://api.example.com/slow-endpoint"
  timeout: "60s"
```

#### `follow_redirects` (オプション)

**型:** Boolean  
**デフォルト:** `defaults.http.follow_redirects`を継承、または`true`  
**説明:** HTTPリダイレクトに従うかどうか

```yaml
with:
  url: "https://example.com/redirect"
  follow_redirects: false
```

#### `verify_ssl` (オプション)

**型:** Boolean  
**デフォルト:** `defaults.http.verify_ssl`を継承、または`true`  
**説明:** SSL証明書を検証するかどうか

```yaml
with:
  url: "https://self-signed.example.com/api"
  verify_ssl: false
```

#### `max_redirects` (オプション)

**型:** Integer  
**デフォルト:** `defaults.http.max_redirects`を継承、または`10`  
**説明:** 従うリダイレクトの最大数

```yaml
with:
  url: "https://example.com/many-redirects"
  max_redirects: 3
```

### レスポンスオブジェクト

HTTPアクションは次のプロパティを持つ`res`オブジェクトを提供します：

| プロパティ | 型 | 説明 |
|----------|------|-------------|
| `code` | Integer | HTTPステータスコード (200, 404, 500など) |
| `time` | Integer | レスポンス時間（ミリ秒） |
| `body_size` | Integer | レスポンスボディサイズ（バイト） |
| `headers` | Object | レスポンスヘッダーのキー値ペア |
| `body.json` | Object | 解析されたJSONレスポンス（有効なJSONの場合のみ） |
| `body.text` | String | レスポンスボディのテキスト |

### レスポンス例

```yaml
steps:
  - name: "API Test"
    id: api-test
    uses: http
    with:
      url: "https://jsonplaceholder.typicode.com/users/1"
    test: |
      res.code == 200 &&
      res.time < 2000 &&
      res.body.json.id == 1 &&
      res.body.json.name != ""
    outputs:
      user_id: res.body.json.id
      user_name: res.body.json.name
      response_time: res.time
      content_type: res.headers["Content-Type"]
```

### 一般的なHTTPパターン

#### 認証

```yaml
vars:
  api_url: "{{API_URL}}"
  access_token: "{{ACCESS_TOKEN}}"
  username: "{{USERNAME}}"
  password: "{{PASSWORD}}"
  api_key: "{{API_KEY}}"

# Bearerトークン
steps:
  - name: "Authenticated Request"
    uses: http
    with:
      url: "{{vars.api_url}}/protected"
      headers:
        Authorization: "Bearer {{vars.access_token}}"

# Basic認証
  - name: "Basic Auth Request"
    uses: http
    with:
      url: "{{vars.api_url}}/basic"
      headers:
        Authorization: "Basic {{encode_base64(vars.username + ':' + vars.password)}}"

# APIキー
  - name: "API Key Request"
    uses: http
    with:
      url: "{{vars.api_url}}/data"
      headers:
        X-API-Key: "{{vars.api_key}}"
```

#### コンテンツタイプ

```yaml
vars:
  api_url: "{{API_URL}}"
  graphql_url: "{{GRAPHQL_URL}}"
  user_id: "{{USER_ID}}"

# JSON API
steps:
  - name: "JSON Request"
    uses: http
    with:
      url: "{{vars.api_url}}/json"
      method: "POST"
      headers:
        Content-Type: "application/json"
      body: |
        {
          "key": "value",
          "timestamp": "{{unixtime()}}"
        }

# XMLリクエスト
  - name: "XML Request"
    uses: http
    with:
      url: "{{vars.api_url}}/xml"
      method: "POST"
      headers:
        Content-Type: "application/xml"
      body: |
        <?xml version="1.0"?>
        <data>
          <key>value</key>
        </data>

# GraphQLクエリ
  - name: "GraphQL Query"
    uses: http
    with:
      url: "{{vars.graphql_url}}"
      method: "POST"
      headers:
        Content-Type: "application/json"
      body: |
        {
          "query": "query { user(id: \"{{vars.user_id}}\") { name email } }"
        }
```

#### ファイルアップロード

```yaml
vars:
  api_url: "{{API_URL}}"

steps:
  - name: "File Upload"
    uses: http
    with:
      url: "{{vars.api_url}}/upload"
      method: "POST"
      headers:
        Content-Type: "multipart/form-data"
      body: |
        --boundary123
        Content-Disposition: form-data; name="file"; filename="test.txt"
        Content-Type: text/plain
        
        File content here
        --boundary123--
```

## データベースアクション

`db`アクションはMySQL、PostgreSQL、SQLiteデータベースでSQLクエリを実行し、包括的な結果処理とエラーレポートを提供します。

### 基本的な構文

```yaml
steps:
  - name: "Database Query"
    uses: db
    with:
      dsn: "mysql://user:password@localhost:3306/database"
      query: "SELECT * FROM users WHERE active = ?"
      params: [true]
    test: res.code == 0 && res.rows_affected > 0
```

### パラメータ

#### `dsn` (必須)

**型:** String  
**説明:** 自動ドライバー検出付きのデータベース接続文字列  
**サポート:** テンプレート式

```yaml
# MySQL
vars:
  db_pass: "{{DB_PASS}}"

with:
  dsn: "mysql://user:password@localhost:3306/database"
  dsn: "mysql://{{vars.db_user}}:{{vars.db_pass}}@{{vars.db_host}}/{{vars.db_name}}"

# PostgreSQL
vars:
  pg_user: "{{PG_USER}}"
  pg_pass: "{{PG_PASS}}"
  pg_host: "{{PG_HOST}}"
  pg_db: "{{PG_DB}}"

with:
  dsn: "postgres://user:password@localhost:5432/database?sslmode=disable"
  dsn: "postgres://{{vars.pg_user}}:{{vars.pg_pass}}@{{vars.pg_host}}/{{vars.pg_db}}"

# SQLite
with:
  dsn: "file:./testdata/sqlite.db"
  dsn: "file:/absolute/path/to/database.db"
  dsn: "file:{{vars.data_dir}}/app.db"
```

#### `query` (必須)

**型:** String  
**説明:** 実行するSQLクエリ  
**サポート:** テンプレート式と複数行文字列

```yaml
with:
  query: "SELECT * FROM users"
  query: "INSERT INTO logs (message, timestamp) VALUES (?, NOW())"
  query: |
    SELECT u.name, u.email, p.title 
    FROM users u 
    JOIN profiles p ON u.id = p.user_id 
    WHERE u.active = ? AND u.created_at > ?
```

#### `params` (オプション)

**型:** 混合値の配列 (String, Number, Boolean)  
**説明:** プリペアドステートメント用のクエリパラメータ  
**サポート:** テンプレート式

```yaml
with:
  query: "SELECT * FROM users WHERE id = ? AND active = ?"
  params: [123, true, "{{vars.user_email}}"]
```

#### `timeout` (オプション)

**型:** Duration  
**デフォルト:** `30s`  
**説明:** クエリ実行タイムアウト

```yaml
with:
  query: "SELECT COUNT(*) FROM large_table"
  timeout: "60s"
```

### レスポンスオブジェクト

データベースアクションは次のプロパティを持つ`res`オブジェクトを提供します：

| プロパティ | 型 | 説明 |
|----------|------|-------------|
| `code` | Integer | 操作結果 (0 = 成功, 1 = エラー) |
| `rows_affected` | Integer | クエリによって影響を受けた行数 |
| `rows` | Array | SELECTステートメントのクエリ結果（オブジェクトとして） |
| `error` | String | 操作が失敗した場合のエラーメッセージ |

### レスポンス例

#### SELECTクエリレスポンス

```yaml
steps:
  - name: "Fetch Users"
    id: fetch-users
    uses: db
    with:
      dsn: "mysql://user:pass@localhost/db"
      query: "SELECT id, name, email FROM users WHERE active = ?"
      params: [true]
    test: res.code == 0 && res.rows_affected > 0
    outputs:
      user_count: res.rows_affected
      first_user_id: res.rows[0].id
      first_user_name: res.rows[0].name
```

#### INSERT/UPDATEクエリレスポンス

```yaml
steps:
  - name: "Insert User"
    uses: db
    with:
      dsn: "postgres://user:pass@localhost/db"
      query: "INSERT INTO users (name, email) VALUES ($1, $2)"
      params: ["John Doe", "john@example.com"]
    test: res.code == 0 && res.rows_affected == 1
```

### データベース固有の機能

#### MySQL例

```yaml
# 接続オプション付きMySQL
- name: "MySQL Query"
  uses: db
  with:
    dsn: "mysql://user:pass@tcp(localhost:3306)/database?charset=utf8mb4&parseTime=true"
    query: "SELECT VERSION() as mysql_version, NOW() as current_time"
  test: res.code == 0

# MySQLストアドプロシージャ
- name: "Call Procedure"
  uses: db
  with:
    dsn: "mysql://user:pass@localhost:3306/database"
    query: "CALL GetUsersByDepartment(?)"
    params: ["Engineering"]
  test: res.code == 0
```

#### PostgreSQL例

```yaml
# JSON操作付きPostgreSQL
- name: "JSON Query"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost:5432/database?sslmode=disable"
    query: |
      SELECT name, data->>'role' as role, data->'preferences' as prefs
      FROM users 
      WHERE data ? 'role' AND data->>'role' = $1
    params: ["admin"]
  test: res.code == 0

# PostgreSQL配列操作
- name: "Array Query"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost:5432/database"
    query: "SELECT name FROM users WHERE tags && $1"
    params: ['{"admin","moderator"}']
  test: res.code == 0
```

#### SQLite例

```yaml
# ファイル作成付きSQLite
- name: "SQLite Query"
  uses: db
  with:
    dsn: "file:./testdata/sqlite.db"
    query: |
      CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        email TEXT UNIQUE,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
      )
  test: res.code == 0

# インメモリデータベースSQLite
- name: "Memory Database"
  uses: db
  with:
    dsn: "file::memory:"
    query: "CREATE TABLE temp_data (id INTEGER, value TEXT)"
  test: res.code == 0
```

### 一般的なクエリパターン

#### データ検証クエリ

```yaml
- name: "Check Data Integrity"
  uses: db
  with:
    dsn: "mysql://user:pass@localhost/db"
    query: |
      SELECT 
        COUNT(*) as total_users,
        COUNT(CASE WHEN active = 1 THEN 1 END) as active_users,
        COUNT(CASE WHEN email IS NULL THEN 1 END) as missing_emails
      FROM users
  test: |
    res.code == 0 && 
    res.rows[0].total_users > 0 && 
    res.rows[0].missing_emails == 0
```

#### パフォーマンス監視

```yaml
- name: "Database Performance Check"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: |
      SELECT 
        schemaname, 
        tablename, 
        seq_scan, 
        seq_tup_read, 
        idx_scan, 
        idx_tup_fetch
      FROM pg_stat_user_tables 
      WHERE seq_scan > 1000
    timeout: "10s"
  test: res.code == 0
  outputs:
    high_seq_scan_tables: res.rows_affected
```

#### バッチ操作

```yaml
- name: "Batch Insert"
  uses: db
  with:
    dsn: "mysql://user:pass@localhost/db"
    query: |
      INSERT INTO audit_log (action, table_name, record_id, timestamp) VALUES
      ('CREATE', 'users', 123, NOW()),
      ('UPDATE', 'profiles', 456, NOW()),
      ('DELETE', 'sessions', 789, NOW())
  test: res.code == 0 && res.rows_affected == 3
```

### セキュリティ機能

データベースアクションはいくつかのセキュリティ対策を実装しています：

- **プリペアドステートメント**: すべてのパラメータ化クエリでプリペアドステートメントを使用してSQLインジェクションを防止
- **接続文字列マスキング**: ログと出力でパスワードをマスク
- **タイムアウト保護**: 長時間実行されるクエリのハングを防止
- **ドライバー検証**: 承認されたデータベースドライバーのみをサポート
- **DSN検証**: 実行前に接続文字列形式を検証

### エラーハンドリング

一般的なエラーシナリオと処理パターン：

```yaml
- name: "Database with Error Handling"
  uses: db
  with:
    dsn: "mysql://user:pass@localhost/db"
    query: "SELECT * FROM users WHERE id = ?"
    params: [999999]
  test: |
    res.code == 0 ? true :
    res.error | contains("connection") ? false :
    res.error | contains("not found") ? true :
    false
  outputs:
    query_success: res.code == 0
    error_type: |
      {{res.code == 0 ? "none" :
        res.error | contains("connection") ? "connection" :
        res.error | contains("syntax") ? "syntax" :
        "unknown"}}
```

### トランザクション例

アクションは直接トランザクションをサポートしませんが、データベース固有のトランザクション構文を使用できます：

```yaml
# PostgreSQLトランザクション
- name: "Begin Transaction"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: "BEGIN"
  test: res.code == 0

- name: "Insert Data"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: "INSERT INTO users (name) VALUES ($1)"
    params: ["Test User"]
  test: res.code == 0

- name: "Commit Transaction"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: "COMMIT"
  test: res.code == 0
```

## ブラウザアクション

`browser`アクションはChromeDPを使用してWebブラウザを自動化し、テスト、スクレイピング、Webアプリケーションとの相互作用のための包括的なWeb自動化機能を提供します。

### 基本的な構文

```yaml
steps:
  - name: "Navigate to Website"
    uses: browser
    with:
      action: navigate
      url: "https://example.com"
      headless: true
      timeout: 30s
    test: res.code == 0
```

### パラメータ

#### `action` (必須)

**型:** String  
**説明:** 実行するブラウザアクション  
**値:** 
- **ナビゲーション:** `navigate`
- **テキスト/コンテンツ:** `text`, `value`, `get_html`
- **属性:** `get_attribute`
- **インタラクション:** `click`, `double_click`, `right_click`, `hover`, `focus`
- **入力:** `type`, `send_keys`, `select`
- **フォーム:** `submit`
- **スクロール:** `scroll`
- **スクリーンショット:** `screenshot`, `capture_screenshot`, `full_screenshot`
- **待機:** `wait_visible`, `wait_not_visible`, `wait_ready`, `wait_text`, `wait_enabled`

#### `url` (オプション)

**型:** String  
**説明:** ナビゲートするURL（navigateアクションに必須）  
**サポート:** テンプレート式

```yaml
with:
  action: navigate
  url: "https://example.com"
  url: "{{vars.base_url}}/login"
```

#### `selector` (オプション)

**型:** String  
**説明:** 要素をターゲットするためのCSSセレクタ  
**サポート:** テンプレート式

```yaml
with:
  action: get_text
  selector: "h1"
  selector: "#main-title"
  selector: ".article-content p:first-child"
```

#### `value` (オプション)

**型:** String  
**説明:** タイプする値または待機するテキスト  
**サポート:** テンプレート式

```yaml
with:
  action: type
  selector: "#email"
  value: "user@example.com"
  value: "{{vars.username}}"
```

#### `attribute` (オプション)

**型:** String  
**説明:** 取得する属性名（get_attributeアクションに必須）

```yaml
with:
  action: get_attribute
  selector: "a"
  attribute: "href"
```

#### `headless` (オプション)

**型:** Boolean  
**デフォルト:** `true`  
**説明:** ヘッドレスモードでブラウザを実行するかどうか

```yaml
with:
  action: navigate
  url: "https://example.com"
  headless: false  # ブラウザウィンドウを表示
```

#### `timeout` (オプション)

**型:** Duration  
**デフォルト:** `30s`  
**説明:** アクションタイムアウト

```yaml
with:
  action: wait_visible
  selector: ".loading"
  timeout: "60s"
```

### レスポンスオブジェクト

ブラウザアクションはアクション固有のプロパティを持つ`res`オブジェクトを提供します：

#### 共通プロパティ

| プロパティ | 型 | 説明 |
|----------|------|-------------|
| `code` | Integer | 結果コード (0 = 成功, 非ゼロ = エラー) |
| `results` | Object | アクション固有の結果 (テキスト、値など) |

#### ナビゲーションレスポンス

| プロパティ | 型 | 説明 |
|----------|------|-------------|
| `url` | String | ナビゲートしたURL |
| `time_ms` | String | ナビゲーション時間（ミリ秒） |

#### テキスト/属性レスポンス

| プロパティ | 型 | 説明 |
|----------|------|-------------|
| `selector` | String | 使用されたCSSセレクタ |
| `text` | String | 抽出されたテキストコンテンツ (get_text) |
| `attribute` | String | 属性名 (get_attribute) |
| `value` | String | 属性値 (get_attribute) |
| `exists` | String | 属性が存在する場合"true" |

#### スクリーンショットレスポンス

| プロパティ | 型 | 説明 |
|----------|------|-------------|
| `screenshot` | String | Base64エンコードされたスクリーンショット |
| `size_bytes` | String | スクリーンショットサイズ（バイト） |

### ブラウザアクション

#### URLへのナビゲート

```yaml
- name: "Open Website"
  uses: browser
  with:
    action: navigate
    url: "https://example.com"
    headless: true
  test: res.code == 0
  outputs:
    load_time: res.time_ms
```

#### テキストコンテンツの抽出

```yaml
- name: "Get Page Title"
  uses: browser
  with:
    action: text
    selector: "h1"
  test: res.code == 0 && res.results.text != ""
  outputs:
    page_title: res.results.text

- name: "Get Input Value"
  uses: browser
  with:
    action: value
    selector: "#username"
  test: res.code == 0
  outputs:
    current_username: res.results.value

- name: "Get Element HTML"
  uses: browser
  with:
    action: get_html
    selector: ".article-content"
  test: res.code == 0
  outputs:
    article_html: res.results.get_html
```

#### 要素属性の取得

```yaml
- name: "Extract Links"
  uses: browser
  with:
    action: get_attribute
    selector: "a.download-link"
    attribute: "href"
  test: res.code == 0 && res.exists == "true"
  outputs:
    download_url: res.results.value
```

#### フォームインタラクション

```yaml
# フォームフィールドの入力
- name: "Enter Email"
  uses: browser
  with:
    action: type
    selector: "#email"
    value: "user@example.com"
  test: res.code == 0

# ボタンのクリック
- name: "Click Submit"
  uses: browser
  with:
    action: click
    selector: "#submit-btn"
  test: res.code == 0

# フォームの送信
- name: "Submit Form"
  uses: browser
  with:
    action: submit
    selector: "form"
  test: res.code == 0
```

#### 要素の待機

```yaml
# 要素の表示を待機
- name: "Wait for Results"
  uses: browser
  with:
    action: wait_visible
    selector: ".search-results"
    timeout: "10s"
  test: res.code == 0

# 特定のテキストを待機
- name: "Wait for Success Message"
  uses: browser
  with:
    action: wait_text
    selector: ".status"
    value: "Success"
  test: res.code == 0
```

#### スクリーンショットの撮影

```yaml
- name: "Take Screenshot"
  uses: browser
  with:
    action: screenshot
  test: res.code == 0
  outputs:
    screenshot_data: res.screenshot
    screenshot_size: res.size_bytes
```

### 高度な使用例

#### ログインフロー

```yaml
vars:
  login_url: "{{LOGIN_URL}}"
  username: "{{USERNAME}}"
  password: "{{PASSWORD}}"

steps:
  - name: "Navigate to Login"
    uses: browser
    with:
      action: navigate
      url: "{{vars.login_url}}"
    test: res.code == 0

  - name: "Enter Username"
    uses: browser
    with:
      action: type
      selector: "#username"
      value: "{{vars.username}}"
    test: res.code == 0

  - name: "Enter Password"
    uses: browser
    with:
      action: type
      selector: "#password"
      value: "{{vars.password}}"
    test: res.code == 0

  - name: "Submit Login"
    uses: browser
    with:
      action: click
      selector: "#login-button"
    test: res.code == 0

  - name: "Wait for Dashboard"
    uses: browser
    with:
      action: wait_visible
      selector: ".dashboard"
      timeout: "15s"
    test: res.code == 0
```

#### データ抽出

```yaml
steps:
  - name: "Navigate to Data Page"
    uses: browser
    with:
      action: navigate
      url: "https://example.com/data"
    test: res.code == 0

  - name: "Wait for Table"
    uses: browser
    with:
      action: wait_visible
      selector: "table"
    test: res.code == 0

  - name: "Count Rows"
    uses: browser
    with:
      action: get_elements
      selector: "table tr"
    test: res.code == 0 && res.count != "0"
    outputs:
      row_count: res.count

  - name: "Extract First Cell"
    uses: browser
    with:
      action: get_text
      selector: "table tr:first-child td:first-child"
    test: res.code == 0
    outputs:
      first_cell: res.results.text
```

#### E2Eテスト

```yaml
steps:
  - name: "Load Application"
    uses: browser
    with:
      action: navigate
      url: "https://app.example.com"
    test: res.code == 0

  - name: "Fill Contact Form"
    uses: browser
    with:
      action: type
      selector: "#contact-name"
      value: "John Doe"
    test: res.code == 0

  - name: "Fill Email"
    uses: browser
    with:
      action: type
      selector: "#contact-email"
      value: "john@example.com"
    test: res.code == 0

  - name: "Fill Message"
    uses: browser
    with:
      action: type
      selector: "#contact-message"
      value: "Hello from automated test"
    test: res.code == 0

  - name: "Submit Form"
    uses: browser
    with:
      action: submit
      selector: "#contact-form"
    test: res.code == 0

  - name: "Verify Success"
    uses: browser
    with:
      action: wait_text
      selector: ".success-message"
      value: "Thank you"
      timeout: "10s"
    test: res.code == 0

  - name: "Take Success Screenshot"
    uses: browser
    with:
      action: screenshot
    test: res.code == 0
```

### エラーハンドリング

```yaml
- name: "Browser Action with Error Handling"
  uses: browser
  with:
    action: click
    selector: "#may-not-exist"
    timeout: "5s"
  test: res.code == 0 || (res.success == "false" && res.error | contains("not found"))
  continue_on_error: true
  outputs:
    click_success: res.code == 0
    error_type: |
      {{res.code == 0 ? "none" :
        res.error | contains("timeout") ? "timeout" :
        res.error | contains("not found") ? "element_not_found" :
        "unknown"}}
```

### パフォーマンスの考慮事項

- **ヘッドレスモード**: より高速な実行のため`headless: true`（デフォルト）を使用
- **タイムアウト**: ハングを防ぐために適切なタイムアウトを設定
- **リソース使用量**: ブラウザアクションは他のアクションよりも多くのリソースを消費
- **スクリーンショット**: 大きなスクリーンショットは大量のメモリを消費

### セキュリティ機能

ブラウザアクションはいくつかのセキュリティ対策を実装しています：

- **サンドボックス実行**: ChromeDPはサンドボックス環境で実行
- **タイムアウト保護**: 無限ハングを防止
- **URL検証**: ナビゲーション前にURLを検証
- **リソース制限**: 組み込みのリソース使用制限

## シェルアクション

`shell`アクションはシェルコマンドとスクリプトを安全に実行し、包括的な出力キャプチャとエラーハンドリングを提供します。

### 基本的な構文

```yaml
steps:
  - name: "Execute Build Script"
    uses: shell
    with:
      cmd: "npm run build"
    test: res.code == 0
```

### パラメータ

#### `cmd` (必須)

**型:** String  
**説明:** 実行するシェルコマンド  
**サポート:** テンプレート式

```yaml
vars:
  api_url: "{{API_URL}}"

with:
  cmd: "echo 'Hello World'"
  cmd: "npm run {{vars.build_script}}"
  cmd: "curl -f {{vars.api_url}}/health"
```

#### `shell` (オプション)

**型:** String  
**デフォルト:** `/bin/sh`  
**許可値:** `/bin/sh`, `/bin/bash`, `/bin/zsh`, `/bin/dash`, `/usr/bin/sh`, `/usr/bin/bash`, `/usr/bin/zsh`, `/usr/bin/dash`

```yaml
with:
  cmd: "echo $0"
  shell: "/bin/bash"
```

#### `workdir` (オプション)

**型:** String  
**説明:** コマンド実行用の作業ディレクトリ（絶対パス必須）  
**サポート:** テンプレート式

```yaml
with:
  cmd: "pwd && ls -la"
  workdir: "/app/src"
  workdir: "{{vars.project_path}}"
```

#### `timeout` (オプション)

**型:** String または Duration  
**デフォルト:** `30s`  
**形式:** Go duration形式 (`30s`, `5m`, `1h`) または数値 (秒)

```yaml
with:
  cmd: "npm test"
  timeout: "10m"
  timeout: "300"  # 300秒
```

#### `env` (オプション)

**型:** Object  
**説明:** コマンドに設定する環境変数  
**サポート:** 値でのテンプレート式

```yaml
vars:
  production_api_url: "{{PRODUCTION_API_URL}}"

with:
  cmd: "npm run build"
  env:
    NODE_ENV: "production"
    API_URL: "{{vars.production_api_url}}"
    BUILD_VERSION: "{{vars.version}}"
```

### レスポンス形式

```yaml
res:
  code: 0                    # 終了コード (0 = 成功)
  stdout: "Build successful" # 標準出力
  stderr: ""                 # 標準エラー出力

req:
  cmd: "npm run build"       # 元のコマンド
  shell: "/bin/sh"          # 使用されたシェル
  workdir: "/app"           # 作業ディレクトリ
  timeout: "30s"            # タイムアウト設定
  env:                      # 環境変数
    NODE_ENV: "production"
```

### 使用例

#### 基本的なコマンド実行

```yaml
- name: "System Information"
  uses: shell
  with:
    cmd: "uname -a"
  test: res.code == 0
```

#### ビルドとテストパイプライン

```yaml
- name: "Install Dependencies"
  uses: shell
  with:
    cmd: "npm ci"
    workdir: "/app"
    timeout: "5m"
  test: res.code == 0

- name: "Run Tests"
  uses: shell
  with:
    cmd: "npm test"
    workdir: "/app"
    env:
      NODE_ENV: "test"
      CI: "true"
  test: res.code == 0 && (res.stdout | contains("All tests passed"))
```

#### 環境固有のデプロイ

```yaml
vars:
  target_env: "{{TARGET_ENV}}"
  deploy_key: "{{DEPLOY_KEY}}"

- name: "Deploy to Environment"
  uses: shell
  with:
    cmd: "./deploy.sh {{vars.target_env}}"
    workdir: "/deploy"
    shell: "/bin/bash"
    timeout: "15m"
    env:
      DEPLOY_KEY: "{{vars.deploy_key}}"
      TARGET_ENV: "{{vars.target_env}}"
  test: res.code == 0
```

#### エラーハンドリングとデバッグ

```yaml
- name: "Service Health Check"
  uses: shell
  with:
    cmd: "curl -f http://localhost:8080/health || echo 'Service down'"
  test: res.code == 0 || (res.stderr | contains("Service down"))

- name: "Debug Failed Build"
  uses: shell
  with:
    cmd: "npm run build:debug"
  # デバッグ出力をキャプチャするために失敗を許可
  outputs:
    debug_info: res.stderr
```

### セキュリティ機能

シェルアクションはいくつかのセキュリティ対策を実装しています：

- **シェルパス制限**: 承認されたシェル実行ファイルのみを許可
- **作業ディレクトリ検証**: 絶対パスとディレクトリの存在を確保
- **タイムアウト保護**: 無限実行を防止
- **環境変数フィルタリング**: 環境変数の受け渡しを安全に処理
- **出力サニタイゼーション**: コマンド出力を安全にキャプチャして返す

### エラーハンドリング

一般的な終了コードとその意味：

- **0**: 成功
- **1**: 一般的なエラー
- **2**: シェル組み込みコマンドの誤用
- **126**: コマンドを実行できない（権限拒否）
- **127**: コマンドが見つからない
- **130**: Ctrl+Cでスクリプトが終了
- **255**: 終了ステータスが範囲外

```yaml
- name: "Handle Different Exit Codes"
  uses: shell
  with:
    cmd: "some_command_that_might_fail"
  test: |
    res.code == 0 ? true :
    res.code == 127 ? (res.stderr | contains("not found")) :
    res.code < 128
```

## SMTPアクション

`smtp`アクションはSMTPサーバーを通じてメール通知とアラートを送信します。

### 基本的な構文

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

### パラメータ

#### `host` (必須)

**型:** String  
**説明:** SMTPサーバーのホスト名またはIPアドレス

```yaml
with:
  host: "smtp.gmail.com"
  host: "mail.example.com"
  host: "127.0.0.1"
```

#### `port` (オプション)

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

#### `username` (必須)

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

#### `password` (必須)

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

#### `from` (必須)

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

#### `to` (必須)

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

#### `cc` (オプション)

**型:** 文字列の配列  
**説明:** カーボンコピー受信者

```yaml
with:
  to: ["admin@example.com"]
  cc: ["team@example.com", "manager@example.com"]
```

#### `bcc` (オプション)

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

#### `body` (必須)

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

#### `html` (オプション)

**型:** Boolean  
**デフォルト:** `false`  
**説明:** 本文にHTMLコンテンツが含まれているかどうか

```yaml
with:
  subject: "HTML Alert"
  body: "<h1>Alert</h1><p>Service is <strong>down</strong></p>"
  html: true
```

#### `tls` (オプション)

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

### レスポンスオブジェクト

SMTPアクションは次のプロパティを持つ`res`オブジェクトを提供します：

| プロパティ | 型 | 説明 |
|----------|------|-------------|
| `success` | Boolean | メールが正常に送信されたかどうか |
| `message_id` | String | 一意のメッセージ識別子（サーバーから提供された場合） |
| `time` | Integer | メール送信にかかった時間（ミリ秒） |

### SMTP例

#### Gmail設定

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

#### Office 365設定

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

#### 複数受信者でのHTMLメール

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

## Helloアクション

`hello`アクションは開発、デバッグ、ワークフロー検証に使用される簡単なテストアクションです。

### 基本的な構文

```yaml
steps:
  - name: "Test Hello"
    uses: hello
    with:
      message: "Hello, World!"
```

### パラメータ

#### `message` (オプション)

**型:** String  
**デフォルト:** `"Hello from Probe!"`  
**説明:** 表示するメッセージ  
**サポート:** テンプレート式

```yaml
with:
  message: "Hello, World!"
  message: "Current time: {{unixtime()}}"
vars:
  user_name: "{{USER_NAME}}"

  message: "Hello {{vars.user_name}}"
```

#### `delay` (オプション)

**型:** Duration  
**デフォルト:** `0s`  
**説明:** 完了前の人為的な遅延

```yaml
with:
  message: "Delayed hello"
  delay: "2s"
```

### レスポンスオブジェクト

helloアクションは次のプロパティを持つ`res`オブジェクトを提供します：

| プロパティ | 型 | 説明 |
|----------|------|-------------|
| `message` | String | 表示されたメッセージ |
| `time` | Integer | かかった時間（ミリ秒）（遅延を含む） |
| `timestamp` | String | アクション完了時のISO 8601タイムスタンプ |

### Hello例

#### 基本テスト

```yaml
steps:
  - name: "Simple Test"
    uses: hello
    test: res.message != ""
    outputs:
      test_time: res.time
```

#### タイミングテスト

```yaml
steps:
  - name: "Timing Test"
    uses: hello
    with:
      message: "Testing timing"
      delay: "1s"
    test: res.time >= 1000 && res.time < 1100
```

#### テンプレートテスト

```yaml
steps:
  - name: "Template Test"
    uses: hello
    with:
vars:
  user: "{{USER}}"

      message: "User: {{vars.user}}, Time: {{unixtime()}}"
    test: res.message | contains(vars.user)
    outputs:
      rendered_message: res.message
```

## アクションエラーハンドリング

### 一般的なエラーシナリオ

すべてのアクションはさまざまな理由で失敗する可能性があります。一般的な失敗モードを理解することで、堅牢なワークフローの作成に役立ちます。

#### HTTPアクションエラー

```yaml
steps:
  - name: "HTTP with Error Handling"
    uses: http
    with:
      url: "https://api.example.com/endpoint"
    test: |
      res.code >= 200 && res.code < 300
    continue_on_error: false
    outputs:
      success: res.code >= 200 && res.code < 300
      error_message: |
        {{res.code >= 400 ? "Client error: " + res.code : 
          res.code >= 500 ? "Server error: " + res.code : ""}}
```

#### SMTPアクションエラー

```yaml
vars:
  smtp_user: "{{SMTP_USER}}"
  smtp_pass: "{{SMTP_PASS}}"

steps:
  - name: "SMTP with Error Handling"
    uses: smtp
    with:
      host: "smtp.example.com"
      username: "{{vars.smtp_user}}"
      password: "{{vars.smtp_pass}}"
      from: "test@example.com"
      to: ["admin@example.com"]
      subject: "Test"
      body: "Test message"
    test: res.success == true
    continue_on_error: true
    outputs:
      email_sent: res.success
      send_time: res.time
```

## パフォーマンスの考慮事項

### HTTPアクションパフォーマンス

- **コネクションプーリング:** HTTPアクションは可能な場合接続を再利用
- **タイムアウト:** ハングを防ぐために適切なタイムアウトを設定
- **レスポンスサイズ:** 大きなレスポンスはより多くのメモリを消費
- **同時リクエスト:** 複数のHTTPアクションは並列実行可能

```yaml
# パフォーマンス最適化されたHTTP設定
vars:
  api_url: "{{API_URL}}"

jobs:
- name: performance-test
  defaults:
    http:
      timeout: "10s"
      follow_redirects: true
      max_redirects: 3
  steps:
    - name: "Quick Health Check"
      uses: http
      with:
        url: "{{vars.api_url}}/ping"
        timeout: "2s"
      test: res.code == 200 && res.time < 500
```

### SMTPアクションパフォーマンス

- **接続再利用:** SMTP接続はアクションごとに確立
- **バッチメール:** 接続を減らすために受信者をグループ化することを検討
- **TLSオーバーヘッド:** TLSネゴシエーションは遅延を追加

```yaml
# 効率的なメール通知
vars:
  smtp_user: "{{SMTP_USER}}"
  smtp_pass: "{{SMTP_PASS}}"

steps:
  - name: "Batch Notification"
    uses: smtp
    with:
      host: "smtp.example.com"
      username: "{{vars.smtp_user}}"
      password: "{{vars.smtp_pass}}"
      from: "alerts@example.com"
      to: ["admin1@example.com", "admin2@example.com", "admin3@example.com"]
      subject: "Batch Alert"
      body: "Single email to multiple recipients"
```

## 関連項目

- **[YAML設定](../yaml-configuration/)** - 完全なYAML構文リファレンス
- **[組み込み関数](../built-in-functions/)** - アクションで使用する式関数
- **[概念: アクション](../../concepts/actions/)** - アクションシステムアーキテクチャ
- **[ハウツー: APIテスト](../../how-tos/api-testing/)** - 実用的なHTTPアクション例
- **[ハウツー: エラーハンドリング](../../how-tos/error-handling-strategies/)** - エラーハンドリングパターン