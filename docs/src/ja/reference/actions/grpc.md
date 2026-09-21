# gRPCアクション

`grpc` アクションは gRPC のメソッドを呼び出します。サービス定義はサーバーリフレクションで解決するため、実行時に `.proto` ファイルは必要ありません。

## 基本的な構文

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

| パラメータ | 型 | 必須 | デフォルト | 説明 |
|---|---|---|---|---|
| `addr` | String | 必須 | - | gRPC サーバーのホストとポート |
| `service` | String | 必須 | - | 完全修飾のサービス名 |
| `method` | String | 必須 | - | メソッド名 |
| `body` | String | 任意 | `""` | リクエストメッセージ（JSON） |
| `metadata` | Object | 任意 | `{}` | リクエストメタデータ（HTTP のヘッダーに相当） |
| `timeout` | String | 任意 | - | タイムアウト（`"10s"` など） |
| `tls` | Boolean | 任意 | `false` | TLS を使う |
| `insecure` | Boolean | 任意 | `false` | 証明書の検証をスキップする |
| `cert_file` | String | 任意 | - | mTLS 用のクライアント証明書 |
| `key_file` | String | 任意 | - | mTLS 用のクライアント鍵 |
| `ca_file` | String | 任意 | - | サーバー検証に使う CA 証明書 |

## レスポンスオブジェクト

| プロパティ | 型 | 説明 |
|---|---|---|
| `res.body` | String | レスポンスメッセージ（JSON） |
| `res.status_code` | String | gRPC のステータスコード（`OK`、`NOT_FOUND` など） |
| `res.status_message` | String | ステータスメッセージ |
| `res.metadata` | Object | レスポンスメタデータ |
| `rt` | Object | レスポンスタイム |
| `req` | Object | 送信したリクエスト |

## 使用例

### メタデータを付けて呼び出す

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

- **[変数](./variables)** - ステップで使える変数
- **[YAML設定](../yaml-configuration)** - ステップのプロパティ
