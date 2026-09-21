# YAML設定リファレンス

Probe がワークフローファイルから読み取るキー、式が評価されるコンテキスト、バリデーションの規則をまとめます。

## ワークフローの構造

```yaml
name: string                  # 必須: ワークフロー名
description: string           # 任意: ワークフローの説明
vars:                         # 任意: ワークフロー変数
  key: value
jobs:                         # 必須: ジョブのリスト
  - name: string              # 必須: ジョブ名
    id: string                # 任意: ジョブ ID。needs から参照する
    needs: [job-id, ...]      # 任意: 依存するジョブ
    skipif: expression        # 任意: 真ならジョブをスキップ
    defaults:                 # 任意: アクションごとの with の既定値
      http:
        url: string
    repeat:                   # 任意: ジョブを繰り返す
      count: integer
      interval: duration
    steps:                    # 必須: ステップのリスト
      - name: string          # 任意: ステップ名
        id: string            # 任意: ステップ ID。outputs の公開に必要
        uses: string          # 必須: アクション名
        with:                 # 任意: アクション引数
          key: value
        test: expression      # 任意: アサーション
        echo: string          # 任意: レポートに出力する文字列
        vars:                 # 任意: ステップ変数
          key: value
        outputs:              # 任意: 後続へ渡す値
          key: expression
        skipif: expression    # 任意: 真ならステップをスキップ
        wait: duration        # 任意: 実行前の待機
        timeout: duration     # 任意: ステップのタイムアウト
        iteration:            # 任意: 要素ごとにステップを繰り返す
          - key: value
        retry:                # 任意: 失敗時のリトライ
          max_attempts: integer
          interval: duration
          initial_delay: duration
```

`jobs` は**リスト**です。ジョブ ID をキーにしたマップで書くと読み込みに失敗します。

トップレベルの `env` と `defaults` はありません。環境変数は `vars` を通して読み、`defaults` はジョブに書きます。

## トップレベルのプロパティ

### `name`

**型:** String（必須）
**説明:** ワークフロー名。レポートの先頭に表示されます。

```yaml
name: "API Health Check"
```

### `description`

**型:** String（任意）
**説明:** ワークフローの説明。

```yaml
description: |
  本番 API を確認します。
  - エンドポイントの死活
  - レスポンスタイム
```

### `vars`

**型:** Object（任意）
**説明:** すべてのジョブとステップから `vars.<name>` で参照できる変数です。

環境変数が見えるのは `vars` の中だけで、変数名をそのまま書いて参照します。値は最初のジョブが始まる前に一度だけ評価されます。

```yaml
vars:
  # 環境変数を読む
  api_url: "{{API_URL}}"

  # デフォルト付き
  timeout: "{{REQUEST_TIMEOUT ?? '30s'}}"

  # 読み込み時に計算する
  run_id: "{{random_str(8)}}"

  # ネストした値も書ける
  auth:
    user: "{{API_USER}}"
```

ステップの式から環境変数を直接読むことはできません。式のコンテキストに `env` は存在しないため、`vars` に置いて `vars.<name>` で参照します。

## ジョブ

### ジョブのプロパティ

| プロパティ | 型 | 必須 | 説明 |
|---|---|---|---|
| `name` | String | 必須 | ジョブ名。テンプレート式が使えます |
| `id` | String | 任意 | 他ジョブの `needs` から参照する識別子。省略時は自動採番されます |
| `needs` | Array | 任意 | 先に完了している必要があるジョブの ID |
| `steps` | Array | 必須 | 実行するステップ |
| `skipif` | Expression | 任意 | 真ならジョブ全体をスキップ |
| `defaults` | Object | 任意 | アクション名をキーとした `with` の既定値 |
| `repeat` | Object | 任意 | ジョブを繰り返す |

ジョブに `if`、`continue_on_error`、`timeout` はありません。

#### `needs`

依存のないジョブは並列に開始します。`needs` が参照するのはジョブの **ID** なので、依存される側には明示的な `id` が必要です。

```yaml
jobs:
  - id: setup
    name: Setup
    steps:
      - name: Prepare
        uses: hello
        echo: "ready"

  - name: Test
    needs: [setup]
    steps:
      - name: Run
        uses: hello
        echo: "testing"
```

#### `skipif`

ブール式です。真と評価されるとジョブをスキップします。

```yaml
jobs:
  - name: Production only
    skipif: vars.environment != "production"
    steps:
      - name: Check
        uses: http
        with:
          url: "{{vars.api_url}}/health"
          method: GET
```

#### `defaults`

ジョブ内の該当アクションを使うステップの `with` にマージされる既定値です。ステップ側の指定が優先されます。

```yaml
jobs:
  - name: API checks
    defaults:
      http:
        url: "{{vars.api_url}}"
        headers:
          authorization: "Bearer {{vars.token}}"
          accept: application/json
    steps:
      - name: Health
        uses: http
        with:
          get: /health        # 既定の url からの相対パス
        test: res.code == 200
```

`http` アクションはメソッドの指定が必須です。`method` を明示するか、`get` / `post` / `put` / `delete` / `patch` といった省略記法を使います。省略記法の値は完全な URL か、`url` からの相対パスです。

```yaml
      - name: Explicit method
        uses: http
        with:
          url: "{{vars.api_url}}/health"
          method: GET
```

#### `repeat`

| プロパティ | 型 | 必須 | 説明 |
|---|---|---|---|
| `count` | Integer | 必須 | 実行回数。上限は `PROBE_MAX_REPEAT_COUNT`（既定 10000） |
| `interval` | Duration | 任意 | 実行間隔 |
| `async` | Boolean | 任意 | 繰り返しを並行実行する |

```yaml
jobs:
  - name: Poll until ready
    repeat:
      count: 10
      interval: 5s
    steps:
      - name: Check
        uses: http
        with:
          url: "{{vars.api_url}}/status"
          method: GET
        test: res.code == 200
```

## ステップ

### ステップのプロパティ

| プロパティ | 型 | 必須 | 説明 |
|---|---|---|---|
| `uses` | String | 必須 | 実行するアクション名（`http`、`shell` など） |
| `name` | String | 任意 | ステップ名 |
| `id` | String | 任意 | このステップの `outputs` の名前空間になる識別子 |
| `with` | Object | 任意 | アクション引数 |
| `test` | Expression | 任意 | アサーション。偽ならステップは失敗します |
| `echo` | String | 任意 | レポートに出力する文字列 |
| `vars` | Object | 任意 | ステップ内だけの変数 |
| `outputs` | Object | 任意 | 後続のステップやジョブへ渡す値 |
| `skipif` | Expression | 任意 | 真ならステップをスキップ |
| `wait` | Duration | 任意 | 実行前の待機時間 |
| `timeout` | Duration | 任意 | ステップのタイムアウト（既定 5m） |
| `iteration` | Array | 任意 | 要素ごとにステップを繰り返す。値は `vars` から参照します |
| `retry` | Object | 任意 | 失敗時のリトライ |

キーは `action` ではなく `uses`、条件は `if` ではなく `skipif` です。

#### `outputs`

後続のステップやジョブへ渡す値です。**`id` を持つステップだけが outputs を公開します。** `id` が無いと `outputs` ブロックは捨てられます。

値はテンプレートではなく式なので、波カッコは書きません。

```yaml
steps:
  - name: Log in
    id: auth
    uses: http
    with:
      url: "{{vars.api_url}}/login"
      method: POST
    test: res.code == 200
    outputs:
      token: res.body.access_token
      user_id: res.body.user.id
```

後続からはステップ ID で名前空間を指定するか、出力名だけで参照します。

```yaml
      headers:
        Authorization: "Bearer {{outputs.auth.token}}"
        X-User: "{{outputs.user_id}}"
```

ハイフンを含む ID は式の識別子として解釈できないため、ブラケットで参照します。

```yaml
    echo: "{{outputs['create-user'].user_id}}"
```

#### `retry`

| プロパティ | 型 | 必須 | 説明 |
|---|---|---|---|
| `max_attempts` | Integer | 必須 | 試行回数。1 以上で、上限は `PROBE_MAX_ATTEMPTS`（既定 10000） |
| `interval` | Duration | 任意 | 試行間隔 |
| `initial_delay` | Duration | 任意 | 最初の試行前の待機 |

```yaml
steps:
  - name: Flaky endpoint
    uses: http
    with:
      url: "{{vars.api_url}}/slow"
      method: GET
    test: res.code == 200
    retry:
      max_attempts: 3
      interval: 2s
```

#### `iteration`

要素ごとにステップを繰り返します。各要素のキーは `vars` から参照できます。

```yaml
steps:
  - name: Check {{vars.path}}
    uses: http
    iteration:
      - path: /health
      - path: /metrics
      - path: /version
    with:
      url: "{{vars.api_url}}{{vars.path}}"
      method: GET
    test: res.code == 200
```

## 式のコンテキスト

ステップの式から見えるのは次の名前だけです。

| 名前 | 型 | 説明 |
|---|---|---|
| `vars` | Object | ワークフローの変数にステップの `vars` をマージしたもの |
| `res` | Object | アクションのレスポンス |
| `req` | Object | 送信したリクエスト |
| `rt` | Object | レスポンスタイム。`rt.duration`（文字列）と `rt.sec`（秒、浮動小数点数） |
| `status` | Integer | アクションの終了ステータス。成功は `0` |
| `outputs` | Object | 先行ステップが公開した値 |
| `repeat_index` | Integer | ジョブ繰り返し時の現在のインデックス |

このコンテキストに `env`、`jobs`、`steps` はありません。

### `http` の `res`

| フィールド | 型 | 説明 |
|---|---|---|
| `res.code` | Integer | ステータスコード（例: `200`） |
| `res.status` | String | ステータス行（例: `"200 OK"`） |
| `res.headers` | Object | レスポンスヘッダー。キーは `Content-Type` のような正規形 |
| `res.body` | Any | レスポンスボディ。JSON ならオブジェクトや配列に解析され、それ以外は文字列 |
| `res.rawbody` | String | 解析前のボディ。JSON として解析したときに入ります |

```yaml
    test: |
      res.code == 200 &&
      res.headers["Content-Type"] contains "application/json" &&
      res.body.status == "ok" &&
      rt.sec < 1
```

他のアクションの `res` は[アクションリファレンス](./actions/variables)を参照してください。

## データ型

### Duration

Go の duration 文字列か、秒数の数値で指定します。

```yaml
timeout: "30s"
timeout: "5m"
interval: 10        # 10 秒
```

### 式とテンプレート

**テンプレート式**は文字列の中に書き、評価結果で置き換わります。

```yaml
url: "{{vars.api_url}}/users/{{outputs.auth.user_id}}"
```

**ブール式**は波カッコなしでそのまま書きます。

```yaml
test: res.code == 200 && rt.sec < 2
skipif: vars.environment == "local"
```

`outputs` の値も式なので、波カッコは書きません。

式の中で使える関数は[組み込み関数](./built-in-functions)を参照してください。

## バリデーション規則

- ワークフローには `name` が必要で、`jobs` は空でないリストである必要があります
- 各ジョブには `name` と 1 つ以上のステップが必要です
- 各ステップには `uses` が必要です
- `needs` は存在するジョブ ID を指し、依存関係は循環していない必要があります
- `repeat.count` は 0 以上、`retry.max_attempts` は 1 以上である必要があります
- ステップに `id` が無いと `outputs` は公開されません

## ファイルのマージ

複数のファイルをカンマ区切りで渡すと順に連結されます。複数回定義されたトップレベルのキーは、最後のファイルの値になります。

```bash
probe base.yml,production.yml
```

マージの規則は[ファイルマージ](../guide/concepts/file-merging)を参照してください。

## 関連項目

- **[CLI](./cli-reference)** - コマンドラインオプション
- **[組み込み関数](./built-in-functions)** - 式で使える関数
- **[環境変数](./environment-variables)** - Probe が読む環境変数
