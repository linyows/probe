# HTTPアクション

`http`アクションはHTTPリクエストを実行し、レスポンスをアサーションやoutputsから参照できるようにします。

## 基本的な構文

HTTPのステップにはURLとメソッドを指定し、レスポンスを`test`で検証します。

```yaml
steps:
  - name: Check the API
    uses: http
    with:
      url: "https://api.example.com/health"
      method: GET
    test: res.code == 200
```

## パラメータ

以下のフィールドでリクエストを指定します。いずれもテンプレート式を書けます。

| パラメータ | 型 | 必須 | デフォルト | 説明 |
|---|---|---|---|---|
| `url` | String | 必須 | - | リクエストURL。メソッド省略記法でパスを渡す場合はベースURL |
| `method` | String | 必須 | - | HTTPメソッド。メソッド省略記法を使う場合はそちらが設定します |
| `headers` | Object | 任意 | - | リクエストヘッダー |
| `body` | StringまたはObject | 任意 | - | リクエストボディ。`content-type`が`application/json`のとき、オブジェクトはJSONにシリアライズされます |
| `timeout` | Duration | 任意 | `30s` | レスポンスの読み取りまで含めた、リクエスト全体の制限時間 |

リダイレクトとTLS検証を指定するパラメータはありません。リダイレクトは既定で追跡します。

### `timeout`

`timeout`には`10s`や`1m30s`のようなduration文字列、または秒数の数値を指定します。`0`を指定すると制限しません。

```yaml
  - name: Slow endpoint
    uses: http
    with:
      url: "{{vars.api_url}}/report"
      method: GET
      timeout: 60s
    test: res.code == 200
```

制限時間を超えたリクエストは`Client.Timeout exceeded`のエラーになり、ステップは失敗します。

ジョブの`defaults`でまとめて指定できます。

```yaml
jobs:
  - name: API checks
    defaults:
      http:
        timeout: 5s
    steps:
      - name: Health
        uses: http
        with:
          get: /health
        test: res.code == 200
```

ステップの`timeout`はこれとは別の、アクション実行1回ごとの外側の制限です（既定5分）。`with.timeout`がHTTPリクエストそのものを、ステップの`timeout`がそれを包むアクション呼び出しを区切ります。応答を返さないまま固まったアクションを止めるのは後者です。

### メソッド省略記法

`get` `head` `post` `put` `patch` `delete` `connect` `options` `trace`は、メソッドとパスを1つのキーで指定します。値は完全なURLか、`url`からの相対パスです。ジョブの`defaults`と組み合わせると簡潔に書けます。

```yaml
jobs:
  - name: API checks
    defaults:
      http:
        url: "{{vars.api_url}}"
        headers:
          accept: application/json
    steps:
      - name: List users
        uses: http
        with:
          get: /users
        test: res.code == 200

      - name: Create a user
        uses: http
        with:
          post: /users
          headers:
            content-type: application/json
          body:
            name: "{{vars.user_name}}"
        test: res.code == 201
```

## レスポンスオブジェクト

リクエストの後、`res`に応答の内容が入ります。

| フィールド | 型 | 説明 |
|---|---|---|
| `res.code` | Integer | ステータスコード（例: `200`） |
| `res.status` | String | ステータス行（例: `"200 OK"`） |
| `res.headers` | Object | レスポンスヘッダー。キーは`Content-Type`のような正規形 |
| `res.body` | Any | レスポンスボディ。JSONならオブジェクトや配列に解析され、それ以外は文字列 |
| `res.rawbody` | String | 解析前のボディ。JSONとして解析したときに入ります |
| `res.filepath` | String | バイナリレスポンスを保存したファイルのパス |
| `rt.duration` | String | ラウンドトリップ時間（例: `"120ms"`） |
| `rt.sec` | Float | ラウンドトリップ時間（秒） |
| `status` | Integer | ステータスコードが2xxなら`0`、それ以外は`1` |

## レスポンス例

JSONレスポンスの値は`res.body`から直接読みます。

```yaml
    test: |
      res.code == 200 &&
      res.headers["Content-Type"] contains "application/json" &&
      res.body.status == "ok" &&
      len(res.body.items) > 0
    outputs:
      first_id: res.body.items[0].id
      elapsed_ms: rt.sec * 1000
```

テキストやHTMLのレスポンスでは、`res.body`は文字列そのものです。

```yaml
    test: |
      res.code == 200 &&
      res.body contains "<title>" &&
      len(res.body) > 100
```

## 一般的なパターン

上記のパラメータの組み合わせは、いくつかの決まった形にまとまります。トークンをステップ間で受け渡す、エラーレスポンスを検証する、まだ反映されていないリクエストをリトライする、といった形です。

### 認証

トークンはログインのステップの出力として取り出し、後続のステップから参照します。

```yaml
  - name: Log in
    id: auth
    uses: http
    with:
      url: "{{vars.api_url}}/login"
      method: POST
      headers:
        content-type: application/json
      body:
        user: "{{vars.user}}"
        password: "{{vars.password}}"
    test: res.code == 200
    outputs:
      token: res.body.access_token

  - name: Call a protected endpoint
    uses: http
    with:
      url: "{{vars.api_url}}/me"
      method: GET
      headers:
        authorization: "Bearer {{outputs.auth.token}}"
    test: res.code == 200
```

Basic認証は`encode_base64`で組み立てます。

```yaml
      headers:
        authorization: "Basic {{encode_base64(vars.user + ':' + vars.password)}}"
```

### エラーレスポンスの検証

ここではエラーが期待される結果です。そのためステータスとエラーのボディを検証します。

```yaml
  - name: Unknown id returns 404
    uses: http
    with:
      url: "{{vars.api_url}}/users/does-not-exist"
      method: GET
    test: res.code == 404 && res.body.error != null
```

### 不安定なエンドポイントのリトライ

`retry`は、テストが通るか試行回数を使い切るまでステップを繰り返します。

```yaml
  - name: Eventually consistent read
    uses: http
    retry:
      max_attempts: 5
      interval: 2s
    with:
      url: "{{vars.api_url}}/orders/{{outputs.create.order_id}}"
      method: GET
    test: res.code == 200
```

## 関連項目

- **[変数](/ja/reference/actions/variables)** - ステップで使える変数
- **[YAML設定](/ja/reference/yaml-configuration)** - ステップのプロパティ
- **[組み込み関数](/ja/reference/built-in-functions)** - 式で使える関数
