# HTTPアクション

`http`アクションはHTTP/HTTPSリクエストを実行し、テストと検証のための詳細なレスポンス情報を提供します。

## 基本的な構文

```yaml
steps:
- name: API Request
  uses: http
  with:
    url: https://api.example.com
    get: /endpoint
  test: res.code == 200
```

## パラメータ

### `url` (必須)

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

### `method: path` (必須)

**型:** String  
**デフォルト:** `GET`  
**フィールド名:** `get`, `post`, `put`, `patch`, `delete`, `head`, `options`

```yaml
with:
  url: https://api.example.com
  post: /users
```

### `headers` (オプション)

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

### `body` (オプション)

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

### `timeout` (オプション)

**型:** Duration  
**デフォルト:** `defaults.http.timeout`を継承、または`30s`  
**説明:** リクエストタイムアウト

```yaml
with:
  url: "https://api.example.com/slow-endpoint"
  timeout: "60s"
```

### `follow_redirects` (オプション)

**型:** Boolean  
**デフォルト:** `defaults.http.follow_redirects`を継承、または`true`  
**説明:** HTTPリダイレクトに従うかどうか

```yaml
with:
  url: "https://example.com/redirect"
  follow_redirects: false
```

### `verify_ssl` (オプション)

**型:** Boolean  
**デフォルト:** `defaults.http.verify_ssl`を継承、または`true`  
**説明:** SSL証明書を検証するかどうか

```yaml
with:
  url: "https://self-signed.example.com/api"
  verify_ssl: false
```

### `max_redirects` (オプション)

**型:** Integer  
**デフォルト:** `defaults.http.max_redirects`を継承、または`10`  
**説明:** 従うリダイレクトの最大数

```yaml
with:
  url: "https://example.com/many-redirects"
  max_redirects: 3
```

## レスポンスオブジェクト

HTTPアクションは次のプロパティを持つ`res`オブジェクトを提供します：

 | プロパティ  | 型      | 説明                                             |
 | ----------  | ------  | -------------                                    |
 | `code`      | Integer | HTTPステータスコード (200, 404, 500など)         |
 | `time`      | Integer | レスポンス時間（ミリ秒）                         |
 | `body_size` | Integer | レスポンスボディサイズ（バイト）                 |
 | `headers`   | Object  | レスポンスヘッダーのキー値ペア                   |
 | `body.json` | Object  | 解析されたJSONレスポンス（有効なJSONの場合のみ） |
 | `body.text` | String  | レスポンスボディのテキスト                       |

## レスポンス例

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

## 一般的なHTTPパターン

### 認証

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

### コンテンツタイプ

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

### ファイルアップロード

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

