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
| `body` | StringまたはObject | 任意 | - | リクエストボディ。`content-type`がJSONの型のとき、オブジェクトはJSONにシリアライズされます。JSONの型とは、`charset`などのパラメータの有無を問わない`application/json`と、`+json`で終わる型です |
| `timeout` | Duration | 任意 | `30s` | レスポンスの読み取りまで含めた、リクエスト全体の制限時間 |
| `basic_auth` | Object | 任意 | - | HTTP Basic認証の`username`と`password`。`Authorization`ヘッダーとして送ります |
| `form` | Object | 任意 | - | フォームのフィールド。`application/x-www-form-urlencoded`のボディとして送ります。[フォームの送信](#フォームの送信)を参照 |
| `multipart` | Object | 任意 | - | フォームのフィールドとファイル。`multipart/form-data`のボディとして送ります。[ファイルのアップロード](#ファイルのアップロード)を参照 |

リダイレクトとTLS検証を指定するパラメータはありません。リダイレクトは既定で追跡します。

`headers`で指定しない限り、すべてのリクエストは`Accept: */*`と`User-Agent: probe-http/1.0.0`を送ります。ヘッダー名は大文字と小文字を区別せずに比較するため、`user-agent: my-agent`と書けば既定値を置き換えます。

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

### フォームの送信

`form`は、HTMLのフォームと同じく、フィールドを`application/x-www-form-urlencoded`のボディとして送ります。フィールドの値には文字列、数値、真偽値を書けます。リストを書くと、その値ごとに同じ名前のフィールドを送ります。

```yaml
  - name: Log in with a form
    uses: http
    with:
      post: /login
      form:
        user: "{{vars.user}}"
        password: "{{vars.password}}"
        scope: [read, write]
    test: res.code == 302
```

フィールドは名前順に並べてパーセントエンコードします。`req.body`には送った内容がそのまま入ります。

### ファイルのアップロード

`multipart`は、テキストのフィールドとファイルを`multipart/form-data`のボディとして送ります。文字列、数値、真偽値で書いたフィールドはテキストのフィールドです。マップで書いたフィールドはファイルで、中身は`file`のパスから読むか、`content`に直接書きます。リストを書くと、その値ごとに同じ名前のフィールドを送るので、1つの名前で複数のファイルを送れます。

```yaml
  - name: Upload an avatar
    uses: http
    with:
      post: /api/images
      multipart:
        title: avatar
        image:
          file: ./fixtures/logo.png
        attachments:
          - file: ./fixtures/terms.pdf
          - content: "a,b\n1,2\n"
            filename: data.csv
            content_type: text/csv
    test: res.code == 201
```

ファイルには次のキーを書けます。

| キー | 説明 |
|---|---|
| `file` | 送るファイルのパス。ほかのアクションのパスと同じく、カレントディレクトリからの相対パスです |
| `content` | `file`の代わりに送る中身 |
| `filename` | ファイルとともに送るファイル名。既定は`file`のベース名、`content`の場合はフィールド名です |
| `content_type` | ファイルのメディアタイプ。既定は`filename`の拡張子から決まるもので、決まらなければ`application/octet-stream`です |

テキストのフィールドを先に、ファイルを後に、それぞれ名前順で送ります。S3へのPOSTアップロードのように、ファイルより前にあるフィールドしか読まないサーバーがあるためです。リストの値は書いた順に送ります。ファイルはステップを実行するたびに読むので、リトライしたステップはその時点のファイルを送ります。

ボディは長さを付けて送り、表示はしません。`req.body`は空になり、代わりに`req.multipart`に書いた内容が入ります。

`form`と`multipart`はそれぞれ`Content-Type`ヘッダーを設定し、`headers`やジョブの`defaults`で指定したもの（`application/json`など）を置き換えます。どちらも`body`とは同時に指定できず、互いに同時に指定することもできません。

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

HTTP Basic認証を受け付けるサーバーには、`basic_auth`でユーザー名とパスワードを渡します。`basic_auth`はそこから`Authorization`ヘッダーを組み立てます。ジョブの`defaults`に書けば、ジョブのすべてのリクエストに適用されます。

```yaml
- name: Admin API
  defaults:
    http:
      url: "{{vars.api_url}}"
      basic_auth:
        username: "{{vars.admin_user}}"
        password: "{{vars.admin_password}}"
  steps:
    - name: List users
      uses: http
      with:
        get: /admin/users
      test: res.code == 200 && len(res.body) > 0
```

ユーザー名にはコロンを含められません。そこでユーザー名が終わったと解釈されるためです。パスワードは空でも構いません。送れるのはどちらか一方なので、`basic_auth`と`authorization`ヘッダーは同時に指定できません。パスワードと、そこから組み立てたヘッダーは、ほかの資格情報と同じく出力では伏せられます。

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
