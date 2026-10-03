# gRPCアクション

`grpc`アクションはgRPCのメソッドを呼び出します。サービス定義はサーバーリフレクションで解決するため、実行時に`.proto`ファイルは必要ありません。

## 基本的な構文

gRPCのステップでは、サーバー、サービス、メソッド、リクエストの内容を指定します。

```yaml
steps:
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

## パラメータ

以下のフィールドで呼び出し内容を指定します。いずれもテンプレート式を書けます。

| パラメータ | 型 | 必須 | デフォルト | 説明 |
|---|---|---|---|---|
| `addr` | String | 必須 | - | gRPCサーバーのホストとポート |
| `service` | String | 必須 | - | 完全修飾のサービス名 |
| `method` | String | 必須 | - | メソッド名 |
| `body` | StringまたはObject | 任意 | `""` | リクエストメッセージ（JSON）。オブジェクトはJSONにシリアライズされます |
| `metadata` | Object | 任意 | `{}` | リクエストメタデータ（HTTPのヘッダーに相当） |
| `timeout` | String | 任意 | `30s` | 接続とリフレクションによる解決を含めた、呼び出しの制限時間。`"10s"`のようなGoのduration形式 |
| `tls` | Boolean | 任意 | `false` | TLSを使う |
| `insecure` | Boolean | 任意 | `false` | 証明書の検証をスキップする |
| `cert_file` | String | 任意 | - | mTLS用のクライアント証明書 |
| `key_file` | String | 任意 | - | mTLS用のクライアント鍵 |
| `ca_file` | String | 任意 | - | サーバー検証に使うCA証明書 |

## レスポンスオブジェクト

呼び出しの後、`res`に応答とgRPCのステータスが入ります。

| プロパティ | 型 | 説明 |
|---|---|---|
| `res.body` | Any | レスポンスメッセージ。JSON形式から解析したオブジェクトで、フィールド名は`createdAt`のようなlowerCamelCase |
| `res.rawbody` | String | 解析前のJSON形式のレスポンスメッセージ |
| `res.status_code` | String | gRPCのステータスコード（`OK`、`NOT_FOUND`など） |
| `res.status_message` | String | ステータスメッセージ |
| `res.metadata` | Object | レスポンスメタデータ |
| `rt.duration` | String | ラウンドトリップ時間（例: `"1.2ms"`） |
| `rt.sec` | Float | ラウンドトリップ時間（秒） |
| `status` | Integer | ステータスが`OK`のとき`0`、それ以外は`1` |
| `req` | Object | 送信したリクエスト |

`res.status_code`は、呼び出しが終わったときのステータスの正式名です。`OK`、`CANCELLED`、`UNKNOWN`、`INVALID_ARGUMENT`、`DEADLINE_EXCEEDED`、`NOT_FOUND`、`ALREADY_EXISTS`、`PERMISSION_DENIED`、`RESOURCE_EXHAUSTED`、`FAILED_PRECONDITION`、`ABORTED`、`OUT_OF_RANGE`、`UNIMPLEMENTED`、`INTERNAL`、`UNAVAILABLE`、`DATA_LOSS`、`UNAUTHENTICATED`のいずれかになります。`OK`以外のステータスもサーバーの応答なので、ステップはそのままテストに進み、テストでそのステータスを期待できます。接続できないサーバーや、リフレクションにサービスが載っていない場合のように、ステータスが得られなかった呼び出しだけが、エラーとしてステップを終わらせます。

## 使用例

以下では、メタデータを付けて呼び出す場合と、エラーが返ることを検証する場合の書き方を示します。

### メタデータを付けて呼び出す

認証情報などはメタデータとして渡します。

```yaml
steps:
  - name: Authenticated call
    id: get-user
    uses: grpc
    with:
      addr: "{{vars.grpc_addr}}"
      service: "user.v1.UserService"
      method: "GetUser"
      tls: true
      timeout: "10s"
      metadata:
        authorization: "Bearer {{vars.token}}"
      body: |
        {"id": "{{vars.user_id}}"}
    test: res.status_code == "OK"
    outputs:
      user: res.body
```

### エラーを検証する

ここではエラーが期待される結果です。そのためステータスコードを検証します。

```yaml
steps:
  - name: Unknown user returns NOT_FOUND
    uses: grpc
    with:
      addr: "{{vars.grpc_addr}}"
      service: "user.v1.UserService"
      method: "GetUser"
      tls: true
      body: |
        {"id": "does-not-exist"}
    test: res.status_code == "NOT_FOUND"
```

## 関連項目

- **[変数](/ja/reference/actions/variables)** - ステップで使える変数
- **[YAML設定](/ja/reference/yaml-configuration)** - ステップのプロパティ
