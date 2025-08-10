<p align="right"><a href="https://github.com/linyows/dewy/blob/main/README.md">English</a> | 日本語</p>

<br><br><br><br><br><br>

<p align="center">
  <img alt="PROBE" src="https://github.com/linyows/probe/blob/main/misc/probe.svg" width="200">
</p>

<br><br><br><br><br><br>

<p align="center">
  <a href="https://github.com/linyows/probe/actions/workflows/build.yml">
    <img alt="GitHub Workflow Status" src="https://img.shields.io/github/actions/workflow/status/linyows/probe/build.yml?branch=main&style=for-the-badge&labelColor=666666">
  </a>
  <a href="https://github.com/linyows/probe/releases">
    <img src="http://img.shields.io/github/release/linyows/probe.svg?style=for-the-badge&labelColor=666666&color=DDDDDD" alt="GitHub Release">
  </a>
  <a href="http://godoc.org/github.com/linyows/probe">
    <img src="http://img.shields.io/badge/go-docs-blue.svg?style=for-the-badge&labelColor=666666&color=DDDDDD" alt="Go Documentation">
  </a>
  <a href="https://deepwiki.com/linyows/probe">
    <img src="http://img.shields.io/badge/deepwiki-docs-purple.svg?style=for-the-badge&labelColor=666666&color=DDDDDD" alt="Deepwiki Documentation">
  </a>
</p>

テスト、監視、自動化タスクのために設計された強力なYAMLベースのワークフロー自動化ツールです。Probeはプラグインベースのアクションを使用してワークフローを実行し、高い柔軟性と拡張性を提供します。

![Architecture](/misc/probe-architecture.svg)

クイックスタート
-----------

```yaml
name: API ヘルスチェック
jobs:
- name: API ステータス確認
  steps:
  - name: API に ping
    uses: http
    with:
      url: https://api.example.com
      get: /health
    test: res.code == 200
```

```bash
probe --workflow health-check.yml
```

機能
--------

- **シンプルなYAML構文**: 読みやすいワークフロー定義
- **プラグインアーキテクチャ**: HTTP、データベース、ブラウザ、Shell、SMTP、Helloアクションが組み込まれており、拡張可能
- **ジョブ依存関係**: `needs`で実行順序を制御
- **ステップアウトプット**: `outputs`を使用してステップとジョブ間でデータを共有
- **繰り返し実行**: 設定可能な間隔でジョブを繰り返し
- **イテレーション**: 異なる変数セットでステップを実行
- **式エンジン**: 強力なテンプレートと条件評価
- **テストサポート**: 組み込みアサーションシステム
- **タイミング制御**: 待機条件と遅延
- **リッチアウトプット**: バッファされた一貫した出力フォーマット
- **セキュリティ**: タイムアウト保護付きの安全な式評価

インストール
------------

### Goを使用
```bash
go install github.com/linyows/probe/cmd/probe@latest
```

### ソースから
```bash
git clone https://github.com/linyows/probe.git
cd probe
go build -o probe ./cmd/probe
```

使用方法
-----

### 基本的な使用方法
```bash
# ワークフローを実行
probe --workflow ./workflow.yml

# 詳細出力
probe --workflow ./workflow.yml --verbose

# レスポンス時間を表示
probe --workflow ./workflow.yml --rt
```

### CLIオプション
- `--workflow <path>`: YAMLワークフローファイルを指定
- `--verbose`: 詳細出力を有効化
- `--rt`: レスポンス時間を表示
- `--help`: ヘルプ情報を表示

ワークフロー構文
---------------

### 基本構造
```yaml
name: ワークフロー名
description: オプションの説明
vars:
  global_var: value
jobs:
- name: ジョブ名
  steps:
  - name: ステップ名
    uses: アクション名
    with:
      param: value
```

### ジョブ
ジョブはデフォルトで並列実行されます。依存関係には`needs`を使用してください：

```yaml
jobs:
- name: セットアップ
  id: setup
  steps:
  - name: 初期化
    uses: http
    with:
      post: /setup

- name: テスト
  needs: [setup]  # セットアップ完了を待機
  steps:
  - name: テスト実行
    uses: http
    with:
      get: /test
```

### ステップ
ジョブ内のステップは順次実行されます：

```yaml
steps:
- name: ログイン
  id: login
  uses: http
  with:
    post: /auth/login
    body:
      username: admin
      password: secret
  outputs:
    token: res.body.token

- name: プロフィール取得
  uses: http
  with:
    get: /profile
    headers:
      authorization: "Bearer {{outputs.login.token}}"
  test: res.code == 200
```

### 変数と式
動的な値には`{{expression}}`構文を使用してください：

```yaml
vars:
  api_url: https://api.example.com
  user_id: 123

steps:
- name: ユーザー取得
  uses: http
  with:
    url: "{{vars.api_url}}"
    get: "/users/{{vars.user_id}}"
  test: res.body.id == vars.user_id
```

### 組み込み関数
```yaml
test: |
  res.code == 200 &&
  match_json(res.body, vars.expected) &&
  random_int(10) > 5
```

利用可能な関数：
- `match_json(actual, expected)`: 正規表現サポート付きJSONパターンマッチング
- `diff_json(actual, expected)`: JSONオブジェクト間の差分を表示
- `random_int(max)`: ランダムな整数を生成
- `random_str(length)`: ランダムな文字列を生成
- `unixtime()`: 現在のUnixタイムスタンプ

アウトプット管理
--------------

ステップとジョブ間でデータを共有：

```yaml
- name: トークン取得
  id: auth
  uses: http
  with:
    post: /auth
  outputs:
    token: res.body.access_token
    expires: res.body.expires_in

- name: トークン使用
  uses: http
  with:
    get: /protected
    headers:
      authorization: "Bearer {{outputs.auth.token}}"
```

イテレーション
-----------

異なる変数セットでステップを実行：

```yaml
- name: 複数ユーザーテスト
  uses: http
  with:
    post: /users
    body:
      name: "{{vars.name}}"
      role: "{{vars.role}}"
  test: res.code == 201
  iter:
  - {name: "Alice", role: "admin"}
  - {name: "Bob", role: "user"}
  - {name: "Carol", role: "editor"}
```

ジョブ繰り返し
-----------

間隔を指定してジョブ全体を繰り返し：

```yaml
- name: ヘルスチェック
  repeat:
    count: 10
    interval: 30s
  steps:
  - name: Ping
    uses: http
    with:
      get: /health
    test: res.code == 200
```

条件付き実行
----------

条件に基づいてステップをスキップ：

```yaml
- name: 条件付きステップ
  uses: http
  with:
    get: /api/data
  skipif: vars.skip_test == true
  test: res.code == 200
```

タイミングと遅延
-----------

```yaml
- name: 待機してチェック
  uses: http
  with:
    get: /status
  wait: 5s  # 実行前に待機
  test: res.code == 200
```

組み込みアクション
----------------

### HTTPアクション
```yaml
- name: HTTPリクエスト
  uses: http
  with:
    url: https://api.example.com
    method: POST  # GET, POST, PUT, DELETE など
    headers:
      content-type: application/json
      authorization: Bearer token
    body:
      key: value
    timeout: 30s
```

### Shellアクション
```yaml
- name: ビルドスクリプト実行
  uses: shell
  with:
    cmd: "npm run build"
    workdir: "/app"
    shell: "/bin/bash"
    timeout: "5m"
    env:
      NODE_ENV: production
  test: res.code == 0
```

### データベースアクション
```yaml
- name: データベースクエリ
  uses: db
  with:
    dsn: "mysql://user:password@localhost:3306/database"
    query: "SELECT * FROM users WHERE active = ?"
    params: [true]
    timeout: 30s
  test: res.code == 0 && res.rows_affected > 0
```

対応データベース:
- **MySQL**: `mysql://user:pass@host:port/database`
- **PostgreSQL**: `postgres://user:pass@host:port/database?sslmode=disable`
- **SQLite**: `file:./testdata/sqlite.db` または `/absolute/path/database`

### ブラウザアクション
```yaml
- name: ウェブ自動化
  uses: browser
  with:
    action: navigate
    url: "https://example.com"
    headless: true
    timeout: 30s
  test: res.success == "true"
```

対応アクション:
- **navigate**: URLへのナビゲーション
- **text**: 要素からのテキスト内容抽出
- **value**: 入力フィールドの値取得
- **get_attribute**: 要素の属性値取得
- **get_html**: 要素からのHTML内容抽出
- **click**: 要素のクリック
- **double_click**: 要素のダブルクリック
- **right_click**: 要素の右クリック
- **hover**: 要素へのホバー
- **focus**: 要素へのフォーカス設定
- **type** / **send_keys**: 入力フィールドへのテキスト入力
- **select**: ドロップダウンオプションの選択
- **submit**: フォーム送信
- **scroll**: 要素をビューにスクロール
- **screenshot**: 要素のスクリーンショット撮影
- **capture_screenshot**: ページ全体のスクリーンショット撮影
- **full_screenshot**: 品質設定付きページ全体のスクリーンショット撮影
- **wait_visible**: 要素が表示されるまで待機
- **wait_not_visible**: 要素が非表示になるまで待機
- **wait_ready**: ページ準備完了まで待機
- **wait_text**: 特定テキスト表示まで待機
- **wait_enabled**: 要素が有効になるまで待機

### SMTPアクション
```yaml
- name: メール送信
  uses: smtp
  with:
    addr: smtp.example.com:587
    from: sender@example.com
    to: recipient@example.com
    subject: テストメール
    body: メール内容
    my-hostname: localhost
```

### Helloアクション（テスト用）
```yaml
- name: テストアクション
  uses: hello
  with:
    name: World
```

高度な例
-----------------

### REST API テスト
```yaml
name: ユーザーAPI テスト
vars:
  base_url: https://api.example.com
  admin_token: "{{env.ADMIN_TOKEN}}"

jobs:
- name: ユーザーCRUD操作
  defaults:
    http:
      url: "{{vars.base_url}}"
      headers:
        authorization: "Bearer {{vars.admin_token}}"
        content-type: application/json

  steps:
  - name: ユーザー作成
    id: create
    uses: http
    with:
      post: /users
      body:
        name: テストユーザー
        email: test@example.com
    test: res.code == 201
    outputs:
      user_id: res.body.id

  - name: ユーザー取得
    uses: http
    with:
      get: "/users/{{outputs.create.user_id}}"
    test: |
      res.code == 200 &&
      res.body.name == "テストユーザー"

  - name: ユーザー更新
    uses: http
    with:
      put: "/users/{{outputs.create.user_id}}"
      body:
        name: 更新されたユーザー
    test: res.code == 200

  - name: ユーザー削除
    uses: http
    with:
      delete: "/users/{{outputs.create.user_id}}"
    test: res.code == 204
```

### 繰り返しによる負荷テスト
```yaml
name: 負荷テスト
jobs:
- name: 並行リクエスト
  repeat:
    count: 100
    interval: 100ms
  steps:
  - name: API呼び出し
    uses: http
    with:
      url: https://api.example.com
      get: /endpoint
    test: res.code == 200
```

### マルチサービス連携
```yaml
name: E2Eテスト
jobs:
- name: データベースセットアップ
  id: db-setup
  steps:
  - name: 初期化
    uses: http
    with:
      post: http://db-service/init

- name: APIテスト
  needs: [db-setup]
  steps:
  - name: APIテスト
    uses: http
    with:
      get: http://api-service/data
    test: res.code == 200

- name: クリーンアップ
  needs: [db-setup]
  steps:
  - name: データベースリセット
    uses: http
    with:
      post: http://db-service/reset
```

式リファレンス
--------------------

### コンテキスト変数
- `vars.*`: ワークフローおよびステップ変数（varsで定義された環境変数を含む）
- `res.*`: 前のステップのレスポンス
- `req.*`: 前のステップのリクエスト
- `outputs.*`: ステップアウトプット

### レスポンスオブジェクト
```yaml
res:
  code: 200
  status: "200 OK"
  headers:
    content-type: application/json
  body: {...}  # パースされたJSONまたは生の文字列
  rawbody: "..." # 元のレスポンスボディ
```

### 演算子
- 算術: `+`, `-`, `*`, `/`, `%`
- 比較: `==`, `!=`, `<`, `<=`, `>`, `>=`
- 論理: `&&`, `||`, `!`
- 三項演算子: `condition ? true_value : false_value`

Probeの拡張
---------------

ProbeはProtocol Buffersを通じてカスタムアクションをサポートしています。組み込みのHTTP、SMTP、Helloアクションを超えた機能を拡張するために、独自のアクションを作成できます。

設定
-------------

### 環境変数
- 環境変数はroot の `vars` セクションで定義してアクセス可能

### デフォルト設定
共通設定にはYAMLアンカーを使用：
```yaml
x_defaults: &api_defaults
  http:
    url: https://api.example.com
    headers:
      authorization: Bearer token

jobs:
- name: テストジョブ
  defaults: *api_defaults
```

トラブルシューティング
---------------

### よくある問題

**式評価エラー**
- 構文をチェック: `{{expression}}` 、 `{expression}`ではない
- 変数名とパスを確認
- 文字列値は引用符で囲む

**HTTPアクションの問題**
- URLフォーマットとアクセス可能性を確認
- ヘッダーと認証をチェック
- タイムアウト設定を確認

**ジョブ依存関係**
- ジョブIDが一意であることを確認
- `needs`参照をチェック
- 循環依存を避ける

### デバッグ出力
詳細な実行情報には`--verbose`フラグを使用：
```bash
probe --workflow test.yml --verbose
```

ベストプラクティス
--------------

1. **説明的な名前を使用** ワークフロー、ジョブ、ステップに
2. **アウトプットを活用** ステップ間でのデータ共有に
3. **適切なテストを実装** 意味のあるアサーションで
4. **デフォルトを使用** 重複を減らすために
5. **ワークフローを論理的に構造化** 明確な依存関係で
6. **エラーを適切に処理** 適切なテストで
7. **変数を使用** 設定管理のために

コントリビューション
------------

コントリビューションを歓迎します！issue、機能リクエスト、プルリクエストをお気軽に送信してください。

ライセンス
-------

このプロジェクトはMITライセンスの下でライセンスされています - 詳細はLICENSEファイルを参照してください。

作者
------

[linyows](https://github.com/linyows)

---

さらなる例と高度な使用方法については、[examplesディレクトリ](./examples/)をご確認ください。
