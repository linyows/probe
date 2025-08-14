# YAML設定リファレンス

このページでは、利用可能なすべてのオプション、データ型、検証ルールを含む、ProbeのYAML設定構文の完全なドキュメントを提供します。

## ワークフロー構造

Probeワークフローの基本構造：

```yaml
name: string                    # 必須: ワークフロー名
description: string             # オプション: ワークフローの説明
vars:                          # オプション: 変数（環境変数を含む）
  KEY: "{{ENV_VAR ?? 'default'}}"
jobs:                          # 必須: ジョブ定義
- name: "Job Name"
  defaults:                    # オプション: デフォルト設定
    http:
      timeout: duration
      headers:
        KEY: value
  steps:
    # ステップ設定
```

## トップレベルプロパティ

### `name`

**型:** String（必須）  
**説明:** ワークフローの人間が読める名前  
**制約:** 空でない必要があります

```yaml
name: "API Health Check"
name: "Production Monitoring Workflow"
```

### `description`

**型:** String（オプション）  
**説明:** ワークフローの目的の詳細な説明  
**サポート:** YAMLリテラルブロック構文を使用した複数行文字列

```yaml
description: "Monitors the health of production APIs"

# 複数行の説明
description: |
  このワークフローでは以下を含む包括的なヘルスチェックを実行します：
  - APIエンドポイントの検証
  - データベース接続テスト
  - パフォーマンス監視
```

### `vars`

**型:** Object（オプション）  
**説明:** すべてのジョブとステップで利用可能な変数  
**キー形式:** 有効な変数名（英数字 + アンダースコア）  
**値の型:** String、number、boolean

```yaml
vars:
  API_BASE_URL: "https://api.example.com"
  TIMEOUT_SECONDS: 30
  DEBUG_MODE: true
  USER_AGENT: "Probe Monitor v1.0"
```

**環境変数の解決:**
```yaml
vars:
  # 静的値
  api_url: "https://api.example.com"
  
  # 外部環境変数を参照
  db_password: "{{DATABASE_PASSWORD}}"
  
  # デフォルト値
  timeout: "{{REQUEST_TIMEOUT ?? '30s'}}"
  
  # 計算値
  build_info: "Build {{BUILD_NUMBER ?? 'unknown'}} at {{unixtime()}}"
```

## ジョブ設定

### ジョブ構造

```yaml
jobs:
- name: string                  # オプション: 人間が読めるジョブ名
  needs: [job-name, ...]       # オプション: ジョブ依存関係
  if: expression               # オプション: 条件付き実行
  continue_on_error: boolean   # オプション: ジョブ失敗時でもワークフローを継続
  timeout: duration            # オプション: ジョブタイムアウト
  defaults:                    # オプション: このジョブのデフォルト設定
    http:
      timeout: duration
  steps:                       # 必須: ステップの配列
    # ステップ設定
```

### ジョブプロパティ

#### `name`

**型:** String（オプション）  
**説明:** ジョブの人間が読める名前  
**デフォルト:** 指定されない場合はジョブIDを使用

```yaml
jobs:
- name: "API Health Check"
```

#### `needs`

**型:** 文字列の配列（オプション）  
**説明:** このジョブが実行される前に完了する必要があるジョブ名のリスト  
**制約:** 参照されるジョブが存在する必要があります

```yaml
jobs:
- name: setup
  # setupジョブ
  
- name: test
  needs: [setup]              # 単一依存関係
  
- name: cleanup
  needs: [setup, test]        # 複数依存関係
```

#### `if`

**型:** 式文字列（オプション）  
**説明:** ジョブを実行するかどうかを決定する条件式  
**コンテキスト:** 環境変数と他のジョブ結果へのアクセス

```yaml
vars:
  environment: "{{ENVIRONMENT}}"

jobs:
- name: production-only
  if: vars.environment == "production"

- name: cleanup
  if: jobs.test.failed

- name: notification
  if: jobs.test.success || jobs.fallback.success
```

#### `continue_on_error`

**型:** Boolean（オプション）  
**デフォルト:** `false`  
**説明:** このジョブが失敗した場合にワークフローを継続するかどうか

```yaml
jobs:
- name: critical-test
  continue_on_error: false    # 失敗時にワークフロー停止（デフォルト）

- name: optional-check
  continue_on_error: true     # 失敗してもワークフローを継続
```

#### `timeout`

**型:** Duration（オプション）  
**説明:** このジョブが実行できる最大時間  
**形式:** 期間文字列（例：`30s`, `5m`, `1h`）

```yaml
jobs:
- name: quick-check
  timeout: "30s"

- name: comprehensive-test
  timeout: "10m"
```

#### `defaults`

**型:** Object（オプション）  
**説明:** 上書きされない限り、すべてのアクションに適用されるデフォルト設定

##### `defaults.http`

HTTP固有のデフォルト設定：

```yaml
jobs:
- name: api-tests
  defaults:
    http:
      timeout: "30s"                    # HTTPアクションのデフォルトタイムアウト
      follow_redirects: true            # HTTPリダイレクトに従う
      verify_ssl: true                  # SSL証明書を検証
      max_redirects: 5                  # 最大リダイレクト数
      headers:                          # すべてのHTTPリクエストのデフォルトヘッダー
        User-Agent: "Probe Monitor"
        Accept: "application/json"
        Authorization: "Bearer {{vars.api_token}}"
```

**サポートされるHTTPデフォルト:**

| プロパティ | 型 | デフォルト | 説明 |
|----------|------|---------|-------------|
| `timeout` | Duration | `30s` | リクエストタイムアウト |
| `follow_redirects` | Boolean | `true` | HTTPリダイレクトに従う |
| `verify_ssl` | Boolean | `true` | SSL証明書を検証 |
| `max_redirects` | Integer | `10` | 従う最大リダイレクト数 |
| `headers` | Object | `{}` | デフォルトヘッダー |

## ステップ設定

### ステップ構造

```yaml
steps:
  - name: string                    # 必須: ステップ名
    id: string                      # オプション: 参照用ステップ識別子
    uses: string                    # オプション: 実行するアクション
    with:                          # オプション: アクションパラメータ
      parameter: value
    test: expression               # オプション: テスト条件
    outputs:                       # オプション: 出力定義
      key: expression
    if: expression                 # オプション: 条件付き実行
    continue_on_error: boolean     # オプション: ステップ失敗時に継続
    timeout: duration              # オプション: ステップタイムアウト
```

### ステッププロパティ

#### `name`

**型:** String（必須）  
**説明:** ステップの人間が読める名前

```yaml
steps:
  - name: "Check API Health"
  - name: "Validate User Authentication"
```

#### `id`

**型:** String（オプション）  
**説明:** ステップ出力を参照するための一意の識別子  
**制約:** ジョブ内で一意、英数字 + ハイフン/アンダースコア

```yaml
steps:
  - name: "Get Auth Token"
    id: auth
    # ... ステップ設定
  
  - name: "Use Auth Token"
    uses: http
    with:
      headers:
        Authorization: "Bearer {{outputs.auth.token}}"
```

#### `uses`

**型:** String（オプション）  
**説明:** 実行するアクションプラグイン  
**組み込みアクション:** `http`, `hello`, `smtp`

```yaml
steps:
  - name: "HTTP Request"
    uses: http
  
  - name: "Send Email"
    uses: smtp
  
  - name: "Test Plugin"
    uses: hello
```

#### `with`

**型:** Object（オプション）  
**説明:** アクションに渡されるパラメータ  
**構造:** アクションタイプによって異なる

**HTTPアクションパラメータ:**
```yaml
steps:
  - name: "API Request"
    uses: http
    with:
      url: "https://api.example.com/users"     # 必須
      method: "GET"                            # オプション、デフォルト: GET
      headers:                                 # オプション
        Authorization: "Bearer {{vars.TOKEN}}"
        Content-Type: "application/json"
      body: |                                  # オプション
        {
          "name": "Test User"
        }
      timeout: "30s"                          # オプション
      follow_redirects: true                   # オプション
      verify_ssl: true                        # オプション
      max_redirects: 5                        # オプション
```

**SMTPアクションパラメータ:**
```yaml
vars:
  smtp_user: "{{SMTP_USER}}"
  smtp_pass: "{{SMTP_PASS}}"
  service_name: "{{SERVICE_NAME}}"

steps:
  - name: "Send Notification"
    uses: smtp
    with:
      host: "smtp.gmail.com"                  # 必須
      port: 587                               # オプション、デフォルト: 587
      username: "{{vars.smtp_user}}"          # 必須
      password: "{{vars.smtp_pass}}"          # 必須
      from: "alerts@example.com"              # 必須
      to: ["admin@example.com"]               # 必須
      cc: ["team@example.com"]                # オプション
      bcc: ["audit@example.com"]              # オプション
      subject: "Alert: {{vars.service_name}}" # 必須
      body: "Service alert message"           # 必須
      html: false                             # オプション、デフォルト: false
      tls: true                              # オプション、デフォルト: true
```

**Helloアクションパラメータ:**
```yaml
steps:
  - name: "Test Hello"
    uses: hello
    with:
      message: "Test message"                 # オプション
      delay: "1s"                            # オプション
```

#### `test`

**型:** 式文字列（オプション）  
**説明:** ステップの成功/失敗を決定するブール式  
**コンテキスト:** `res`オブジェクトを通じてアクションレスポンスにアクセス

```yaml
steps:
  - name: "API Health Check"
    uses: http
    with:
      url: "{{vars.API_URL}}/health"
    test: res.code == 200
  
  - name: "Complex Validation"
    uses: http
    with:
      url: "{{vars.API_URL}}/data"
    test: |
      res.code == 200 &&
      res.body.json.success == true &&
      res.body.json.data | length > 0 &&
      res.time < 1000
```

**レスポンスオブジェクトプロパティ:**

HTTPアクションの場合、`res`オブジェクトには以下が含まれます：

| プロパティ | 型 | 説明 |
|----------|------|-------------|
| `code` | Integer | HTTPステータスコード |
| `time` | Integer | レスポンス時間（ミリ秒） |
| `body_size` | Integer | レスポンスボディサイズ（バイト） |
| `headers` | Object | レスポンスヘッダー |
| `body.json` | Object | 解析されたJSONレスポンス（該当する場合） |
| `body.text` | String | レスポンスボディのテキスト |

#### `outputs`

**型:** Object（オプション）  
**説明:** 後のステップで使用するためにステップから抽出された名前付きの値  
**キー形式:** 有効な識別子名  
**値の型:** 式文字列

```yaml
steps:
  - name: "Get User Data"
    id: user-data
    uses: http
    with:
      url: "{{vars.API_URL}}/users/1"
    test: res.code == 200
    outputs:
      user_id: res.body.json.id
      user_name: res.body.json.name
      user_email: res.body.json.email
      response_time: res.time
      is_active: res.body.json.active == true
      full_name: "{{res.body.json.first_name}} {{res.body.json.last_name}}"
```

#### `if`

**型:** 式文字列（オプション）  
**説明:** ステップを実行するかどうかを決定する条件式

```yaml
steps:
  - name: "Production Only Step"
    if: "{{vars.ENVIRONMENT}}" == "production"
    uses: http
    with:
      url: "{{vars.PROD_API_URL}}/check"
  
  - name: "Retry on Failure"
    if: steps.previous-step.failed
    uses: http
    with:
      url: "{{vars.FALLBACK_URL}}/retry"
```

#### `continue_on_error`

**型:** Boolean（オプション）  
**デフォルト:** `false`  
**説明:** このステップが失敗した場合にジョブを継続するかどうか

```yaml
steps:
  - name: "Critical Step"
    uses: http
    with:
      url: "{{vars.CRITICAL_URL}}/check"
    continue_on_error: false      # 失敗時にジョブ停止（デフォルト）
  
  - name: "Optional Step"
    uses: http
    with:
      url: "{{vars.OPTIONAL_URL}}/info"
    continue_on_error: true       # 失敗してもジョブを継続
```

#### `timeout`

**型:** Duration（オプション）  
**説明:** このステップが実行できる最大時間

```yaml
steps:
  - name: "Quick Check"
    timeout: "5s"
    uses: http
    with:
      url: "{{vars.API_URL}}/ping"
  
  - name: "Long Running Process"
    timeout: "5m"
    uses: http
    with:
      url: "{{vars.API_URL}}/long-process"
```

## データ型

### Duration

期間文字列は時間の期間を指定します：

**形式:** `<数値><単位>`  
**単位:** `ns`, `us`, `ms`, `s`, `m`, `h`

```yaml
# 例
timeout: "30s"          # 30秒
timeout: "5m"           # 5分
timeout: "1h30m"        # 1時間30分
timeout: "500ms"        # 500ミリ秒
```

### 式文字列

式文字列はカスタム関数を持つGoテンプレート構文を使用します：

**テンプレート式:** `{{expression}}`  
**テスト式:** プレーンブール式

```yaml
# テンプレート式（値用）
url: "{{vars.BASE_URL}}/api/{{vars.VERSION}}"
message: "Hello {{outputs.user.name}}"

# テスト式（条件用）
test: res.code == 200 && res.time < 1000
if: "{{vars.ENVIRONMENT}}" == "production"
```

### 環境変数参照

式で環境変数を参照：

```yaml
vars:
  API_URL: "{{EXTERNAL_API_URL}}"           # 外部環境変数を参照
  TIMEOUT: "{{REQUEST_TIMEOUT ?? '30s'}}"   # デフォルト値付き
  DEBUG: "{{DEBUG_MODE == 'true'}}"         # ブール変換
```

## 検証ルール

### ワークフロー検証

- `name`は必須で空でない
- `jobs`は必須で少なくとも1つのジョブを含む
- ジョブ名は一意である必要がある
- `needs`のジョブ名は既存のジョブを参照する必要がある
- ジョブの`needs`に循環依存がない

### ジョブ検証

- 各ジョブには`steps`配列が必要
- ステップ名は必須で説明的である必要がある
- ステップIDはジョブ内で一意である必要がある
- アクション名は有効である必要がある（組み込みまたは利用可能なプラグイン）

### 式の検証

- テンプレート式は有効なGoテンプレート構文を使用する必要がある
- テスト式はブール値に評価される必要がある
- 参照される変数と出力が存在する必要がある
- 関数呼び出しは有効な組み込み関数を使用する必要がある

## 一般的なパターン

### 環境固有設定

```yaml
vars:
  node_env: "{{NODE_ENV}}"
  environment: "{{vars.node_env ?? 'development'}}"
  api_url: |
    {{vars.node_env == "production" ? 
      "https://api.prod.com" : 
      "https://api.dev.com"}}
  timeout: |
    {{vars.node_env == "production" ? "10s" : "30s"}}
```

### 条件付きジョブ実行

```yaml
vars:
  environment: "{{ENVIRONMENT}}"

jobs:
- name: setup
  # 常に実行

- name: development-tests
  if: "{{vars.environment}}" == "development"
  needs: [setup]

- name: production-checks
  if: "{{vars.environment}}" == "production"  
  needs: [setup]

- name: cleanup
  needs: [development-tests, production-checks]
  if: |
    jobs.development-tests.executed || 
    jobs.production-checks.executed
```

### エラーハンドリングと復旧

```yaml
jobs:
- name: primary-test
  continue_on_error: true
  steps:
    - name: "Primary Service Test"
      uses: http
      with:
        url: "{{vars.PRIMARY_URL}}/test"
      continue_on_error: true

- name: fallback-test
  if: jobs.primary-test.failed
  steps:
    - name: "Fallback Service Test"
      uses: http
      with:
        url: "{{vars.FALLBACK_URL}}/test"
```

### ステップ間のデータフロー

```yaml
jobs:
- name: data-processing
  steps:
    - name: "Fetch Data"
      id: fetch
      uses: http
      with:
        url: "{{vars.API_URL}}/data"
      outputs:
        data_count: res.body.json.items | length
        first_item_id: res.body.json.items[0].id
    
    - name: "Process Data"
      uses: http
      with:
        url: "{{vars.API_URL}}/process/{{outputs.fetch.first_item_id}}"
      test: res.code == 200
    
    - name: "Summary"
      uses: echo
      with:
        message: "Processed {{outputs.fetch.data_count}} items"
```

## ベストプラクティス

### YAMLスタイル

```yaml
# 良い例: 一貫したインデント（2スペース）
jobs:
- name: test
  steps:
    - name: "Health Check"
      uses: http

# 良い例: 特殊文字を含む文字列の引用
vars:
  MESSAGE: "Hello, World!"
  PATTERN: "user-\\d+"

# 良い例: 可読性のための複数行文字列
description: |
  このワークフローでは以下を含む包括的なテストを実行します：
  - APIエンドポイントの検証
  - データベース接続
  - パフォーマンスベンチマーク
```

### 命名規則

```yaml
# 良い例: 説明的な名前
name: "Production API Health Check"

jobs:
- name: "User Authentication Test"
    
- name: "Database Connectivity Check"

steps:
  - name: "Verify SSL Certificate Validity"
  - name: "Test User Login Endpoint"
  - name: "Validate Database Connection Pool"
```

### 設定の整理

```yaml
# 良い例: 論理的なグループ化
vars:
  # API設定
  API_BASE_URL: "https://api.example.com"
  API_VERSION: "v1"
  API_TIMEOUT: "30s"
  
  # データベース設定  
  DB_HOST: "localhost"
  DB_PORT: 5432
  
  # 機能フラグ
  ENABLE_CACHING: true
  ENABLE_METRICS: false

jobs:
- name: default
  defaults:
    http:
      timeout: "{{vars.API_TIMEOUT}}"
      headers:
        User-Agent: "Probe Monitor"
        Accept: "application/json"
```

## 関連項目

- **[CLIリファレンス](../cli-reference/)** - コマンドラインオプションと使用法
- **[アクションリファレンス](../actions-reference/)** - 組み込みアクションとパラメータ
- **[組み込み関数](../built-in-functions/)** - 式関数
- **[概念: ワークフロー](../../concepts/workflows/)** - ワークフロー設計パターン
- **[概念: 式とテンプレート](../../concepts/expressions-and-templates/)** - 式言語ガイド