# 式とテンプレート

式とテンプレートは Probe ワークフローの動的な心臓部です。条件付きロジック、データ変換、動的設定を可能にします。このガイドでは、式システム、テンプレート構文、高度な使用パターンについて詳しく説明します。

## 式システム概要

Probe は2つのタイプの式を使用します：

1. **テンプレート式** (`{{expression}}`) - 動的値の挿入用
2. **テスト式** (`expression`) - ブール条件と検証用

両方とも、セキュリティ強化とカスタム関数を備えた [expr](https://github.com/antonmedv/expr) ベースの同じ基礎式エンジンを使用しています。

## テンプレート式

テンプレート式は `{{}}` 構文を使用して文字列に動的値を挿入します。

### 基本テンプレート構文

```yaml
# シンプルな変数置換
- name: Greet User
  echo: "Hello {{vars.USERNAME}}!"

# ネストされたデータへのアクセス
- name: API Request
  uses: http
  with:
    url: "{{vars.API_BASE_URL}}/users/{{outputs.auth.user_id}}"
    headers:
      Authorization: "Bearer {{outputs.auth.access_token}}"

# 複雑な式
- name: Dynamic Configuration
  echo: "Environment: {{vars.NODE_ENV || 'development'}}, Users: {{outputs.api.user_count || 0}}"
```

### テンプレート式コンテキスト

テンプレート式はいくつかのデータソースにアクセスできます：

#### 環境変数 (`env`)
```yaml
variables:
  api_url: "{{vars.API_URL}}"                    # 環境変数
  port: "{{vars.PORT || '3000'}}"               # デフォルト値付き
  debug_mode: "{{vars.DEBUG == 'true'}}"        # ブール変換
```

#### ステップ出力 (`outputs`)
```yaml
steps:
  - name: Get User Info
    id: user-info
    uses: http
    with:
      url: "{{vars.API_URL}}/user/current"
    outputs:
      user_id: res.body.json.id
      user_name: res.body.json.name
      user_email: res.body.json.email

  - name: Send Welcome Email
    action: smtp
    with:
      to: ["{{outputs.user-info.user_email}}"]
      subject: "Welcome {{outputs.user-info.user_name}}!"
      body: "Your user ID is: {{outputs.user-info.user_id}}"
```

#### ジョブ出力 (ジョブ間参照)
```yaml
jobs:
  setup:
    steps:
      - name: Initialize
        outputs:
          session_id: "{{random_str(16)}}"

  main-test:
    needs: [setup]
    steps:
      - name: Use Session
        uses: http
        with:
          headers:
            X-Session-ID: "{{outputs.setup.session_id}}"
```

### 高度なテンプレートパターン

#### 条件付き値
```yaml
# 三項演算子
- name: Environment-specific URL
  echo: "URL: {{vars.NODE_ENV == 'production' ? 'https://api.prod.com' : 'https://api.dev.com'}}"

# Null 合体
- name: Default Configuration
  echo: "Timeout: {{vars.TIMEOUT || '30s'}}"
```

#### 文字列操作
```yaml
# 文字列連結
- name: Build File Path
  echo: "File: {{vars.BASE_PATH}}/{{vars.FILE_NAME}}.{{vars.FILE_EXT}}"

# 文字列メソッド（限定サポート）
- name: Format Output
  echo: "User: {{outputs.user.name.upper()}} ({{outputs.user.email.lower()}})"
```

#### 算術演算
```yaml
# 数学的演算
- name: Calculate Metrics
  echo: |
    Performance Metrics:
    Average Response Time: {{(outputs.test1.time + outputs.test2.time + outputs.test3.time) / 3}}ms
    Total Requests: {{outputs.test1.requests + outputs.test2.requests + outputs.test3.requests}}
    Success Rate: {{(outputs.successful.count / outputs.total.count) * 100}}%
```

#### 複雑なデータアクセス
```yaml
# 配列アクセス
- name: Process User List
  echo: "First user: {{outputs.users.list[0].name}}"

# オブジェクトプロパティアクセス
- name: Nested Data Access
  echo: "Database: {{outputs.config.database.host}}:{{outputs.config.database.port}}"
```

## テスト式

テスト式は `test` と `if` ステートメントで使用されるブール条件です。

### 基本テスト構文

```yaml
# シンプルなステータスチェック
- name: Health Check
  uses: http
  with:
    url: "{{vars.API_URL}}/health"
  test: res.code == 200

# 複雑な条件
- name: Comprehensive API Test
  uses: http
  with:
    url: "{{vars.API_URL}}/api/data"
  test: |
    res.code == 200 &&
    res.body.json.success == true &&
    res.body.json.data != null &&
    res.time < 1000
```

### HTTP レスポンステスト

`res` オブジェクトは包括的なレスポンスデータを提供します：

```yaml
# ステータスコードテスト
test: res.code == 200
test: res.code >= 200 && res.code < 300
test: res.code in [200, 201, 202]

# レスポンス時間テスト
test: res.time < 1000                           # 1秒未満
test: res.time >= 100 && res.time <= 500      # 100-500ms の間

# レスポンスサイズテスト
test: res.body_size > 0                        # コンテンツあり
test: res.body_size < 1048576                  # 1MB 未満

# ヘッダーテスト
test: res.headers["content-type"] == "application/json"
test: res.headers["x-rate-limit-remaining"] > "10"

# JSON レスポンステスト
test: res.body.json.status == "success"
test: res.body.json.data.users.length > 0
test: res.body.json.error == null

# テキストレスポンステスト
test: res.text.contains("Success")
test: res.text.startsWith("<!DOCTYPE html>")
test: res.text.length > 100
```

### 高度なテスト条件

#### 正規表現
```yaml
# レスポンステキストのパターンマッチング
test: res.text.matches("user-\\d+@example\\.com")

# JSON フィールドパターン検証
test: res.body.json.user.email.matches("[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}")
```

#### 配列とオブジェクトのテスト
```yaml
# 配列テスト
test: res.body.json.users.length == 5
test: res.body.json.tags.contains("production")
test: res.body.json.permissions.all(p -> p.active == true)
test: res.body.json.items.any(item -> item.price > 100)

# オブジェクトプロパティテスト
test: res.body.json.user.has("id") && res.body.json.user.has("email")
test: res.body.json.config.database.host != null
```

#### 複雑な論理条件
```yaml
# 複数条件検証
test: |
  (res.code == 200 && res.body.json.success == true) ||
  (res.code == 202 && res.body.json.processing == true)

# ネストした条件検証
test: |
  res.code == 200 &&
  res.body.json.data != null &&
  (
    (res.body.json.data.type == "user" && res.body.json.data.user.active == true) ||
    (res.body.json.data.type == "system" && res.body.json.data.system.healthy == true)
  )
```

## 組み込み関数

Probe は一般的な操作のためのいくつかの組み込み関数を提供します。

### ランダム関数

#### `random_int(max)`
ランダムな整数を生成：

```yaml
# ランダムなユーザーID生成
- name: Create Test User
  uses: http
  with:
    url: "{{vars.API_URL}}/users"
    method: POST
    body: |
      {
        "id": {{random_int(999999)}},
        "name": "TestUser{{random_int(1000)}}",
        "group": {{random_int(10)}}
      }
```

#### `random_str(length)`
ランダムな文字列を生成：

```yaml
# ユニークな識別子を生成
- name: Create Session
  outputs:
    session_id: "session_{{random_str(16)}}"
    transaction_id: "txn_{{random_str(12)}}"
    correlation_id: "{{random_str(32)}}"

# テストデータ生成
- name: Create Test Record
  uses: http
  with:
    body: |
      {
        "username": "user_{{random_str(8)}}",
        "email": "test_{{random_str(6)}}@example.com",
        "api_key": "{{random_str(40)}}"
      }
```

### 時間関数

#### `unixtime()`
現在の Unix タイムスタンプを取得：

```yaml
# リクエストにタイムスタンプを追加
- name: Timestamped Request
  uses: http
  with:
    url: "{{vars.API_URL}}/events"
    method: POST
    body: |
      {
        "event": "test_execution",
        "timestamp": {{unixtime()}},
        "execution_id": "exec_{{unixtime()}}_{{random_str(8)}}"
      }

# 時間ベースのテスト
- name: Check Timestamp
  uses: http
  with:
    url: "{{vars.API_URL}}/status"
  test: res.body.json.server_time >= {{unixtime() - 300}}  # 過去5分以内
```

### カスタム関数使用パターン

#### ユニークテストデータ生成
```yaml
jobs:
  user-lifecycle-test:
    steps:
      - name: Create Unique User
        id: create-user
        uses: http
        with:
          url: "{{vars.API_URL}}/users"
          method: POST
          body: |
            {
              "username": "testuser_{{unixtime()}}_{{random_str(6)}}",
              "email": "test_{{random_str(8)}}@example.com",
              "password": "{{random_str(16)}}",
              "user_id": {{random_int(1000000)}}
            }
        test: res.code == 201
        outputs:
          user_id: res.body.json.user.id
          username: res.body.json.user.username

      - name: Verify User Creation
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.create-user.user_id}}"
        test: |
          res.code == 200 &&
          res.body.json.user.username == "{{outputs.create-user.username}}"

      - name: Clean Up User
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.create-user.user_id}}"
          method: DELETE
        test: res.code == 204
```

#### セッションと関連ID
```yaml
jobs:
  distributed-trace-test:
    steps:
      - name: Initialize Trace
        id: trace
        echo: "Starting distributed trace"
        outputs:
          trace_id: "trace_{{unixtime()}}_{{random_str(16)}}"
          correlation_id: "corr_{{random_str(32)}}"

      - name: Service A Call
        uses: http
        with:
          url: "{{vars.SERVICE_A_URL}}/process"
          headers:
            X-Trace-ID: "{{outputs.trace.trace_id}}"
            X-Correlation-ID: "{{outputs.trace.correlation_id}}"
        test: res.code == 200

      - name: Service B Call
        uses: http
        with:
          url: "{{vars.SERVICE_B_URL}}/process"
          headers:
            X-Trace-ID: "{{outputs.trace.trace_id}}"
            X-Correlation-ID: "{{outputs.trace.correlation_id}}"
        test: res.code == 200

      - name: Verify Trace Correlation
        uses: http
        with:
          url: "{{vars.TRACING_URL}}/traces/{{outputs.trace.trace_id}}"
        test: |
          res.code == 200 &&
          res.body.json.spans.length >= 2 &&
          res.body.json.correlation_id == "{{outputs.trace.correlation_id}}"
```

## 条件付きロジックパターン

### ステップレベル条件

```yaml
steps:
  - name: Check Primary Service
    id: primary
    uses: http
    with:
      url: "{{vars.PRIMARY_URL}}/health"
    test: res.code == 200
    continue_on_error: true
    outputs:
      primary_healthy: res.code == 200

  - name: Check Secondary Service
    if: "!outputs.primary.primary_healthy"
    id: secondary
    uses: http
    with:
      url: "{{vars.SECONDARY_URL}}/health"
    test: res.code == 200
    outputs:
      secondary_healthy: res.code == 200

  - name: Success Path
    if: outputs.primary.primary_healthy || outputs.secondary.secondary_healthy
    echo: "At least one service is healthy"

  - name: Failure Path
    if: "!outputs.primary.primary_healthy && (!outputs.secondary || !outputs.secondary.secondary_healthy)"
    echo: "All services are down!"
```

### ジョブレベル条件

```yaml
jobs:
  health-check:
    steps:
      - name: Basic Health Check
        outputs:
          healthy: res.code == 200

  detailed-analysis:
    if: jobs.health-check.failed
    steps:
      - name: Deep Diagnostic
        uses: http
        with:
          url: "{{vars.API_URL}}/diagnostics"

  performance-test:
    if: jobs.health-check.success
    steps:
      - name: Load Test
        uses: http
        with:
          url: "{{vars.API_URL}}/load-test"
```

### 環境ベースの条件

```yaml
steps:
  - name: Development Setup
    if: vars.NODE_ENV == "development"
    echo: "Running in development mode"

  - name: Production Validation
    if: vars.NODE_ENV == "production"
    uses: http
    with:
      url: "{{vars.API_URL}}/production-check"
    test: res.code == 200

  - name: Feature Flag Check
    if: vars.FEATURE_FLAGS.contains("new-api")
    uses: http
    with:
      url: "{{vars.API_URL}}/v2/endpoint"
```

## セキュリティ考慮事項

### 式のセキュリティ機能

Probe はいくつかのセキュリティ対策を実装しています：

1. **式長制限**: リソース枯渇を防止
2. **危険な関数ブロック**: システム関数へのアクセスをブロック
3. **環境変数フィルタリング**: 機密変数へのアクセスを制限
4. **タイムアウト保護**: 式での無限ループを防止

### 安全な式パターン

```yaml
# 良い例: 安全な環境変数アクセス
- name: Safe Config
  echo: "API URL: {{vars.API_URL}}"

# 良い例: 制限されたデータアクセス
- name: Safe Data Access
  test: res.body.json.users.length <= 1000

# 避ける: 無制限の操作
# test: res.body.json.data.some_huge_array.all(item -> expensive_operation(item))

# 良い例: シンプルな条件
- name: Simple Validation
  test: res.code == 200 && res.body.json.success == true

# 避ける: 複雑なネスト式
# test: deeply.nested.complex.expression.with.many.operations()
```

### 機密データの処理

```yaml
# 良い例: シークレットに環境変数を使用
- name: Authenticated Request
  uses: http
  with:
    headers:
      Authorization: "Bearer {{vars.API_TOKEN}}"

# 良い例: 機密データのログを避ける
- name: Login Test
  uses: http
  with:
    body: |
      {
        "username": "{{vars.TEST_USERNAME}}",
        "password": "{{vars.TEST_PASSWORD}}"
      }
  # 機密レスポンスデータを出力しない
  outputs:
    login_successful: res.code == 200
    # NG: auth_token: res.body.json.token (ログに露出する)
```

## パフォーマンス最適化

### 効率的な式の記述

```yaml
# 良い例: シンプルで直接的な式
test: res.code == 200

# 良い例: && による早期終了
test: res.code == 200 && res.body.json.success == true

# 避ける: 式での複雑な計算
# test: expensive_calculation(res.body.json.large_dataset) == expected_value

# 良い例: 複雑な値を事前計算
outputs:
  user_count: res.body.json.users.length
  active_users: res.body.json.users.filter(u -> u.active == true).length
```

### テンプレート最適化

```yaml
# 良い例: シンプルなテンプレート置換
echo: "User {{outputs.user.name}} logged in"

# 良い例: 最小限の文字列操作
url: "{{vars.BASE_URL}}/users/{{outputs.user.id}}"

# 避ける: 複雑なテンプレート式
# echo: "{{complex_calculation(outputs.data) + another_operation(vars.CONFIG)}}"
```

## 式のデバッグ

### よくある問題と解決方法

#### テンプレート式エラー
```yaml
# エラー: JSON でクォート不足
body: |
  {
    "name": {{outputs.user.name}}      # エラー: クォートなし
  }

# 解決方法: 適切な JSON クォート
body: |
  {
    "name": "{{outputs.user.name}}"    # 正解: 文字列をクォート
  }
```

#### テスト式デバッグ
```yaml
# 詳細モードでデバッグ
probe -v workflow.yml

# デバッグ出力を追加
- name: Debug Values
  echo: |
    Debug Information:
    Status: {{res.code}}
    Response Time: {{res.time}}
    JSON Success: {{res.body.json.success}}
    Headers: {{res.headers}}
```

#### Null 値の処理
```yaml
# 良い例: 潜在的な null 値を処理
test: res.body.json.user != null && res.body.json.user.active == true

# 良い例: デフォルト値を使用
echo: "User count: {{outputs.api.user_count || 0}}"

# 良い例: アクセス前に存在をチェック
test: res.body.json.has("data") && res.body.json.data.has("users")
```

## ベストプラクティス

### 1. 式をシンプルに保つ
```yaml
# 良い例: シンプルで読みやすい式
test: res.code == 200 && res.time < 1000

# 避ける: 過度に複雑な式
# test: (res.code >= 200 && res.code < 300) && (res.time < (vars.MAX_TIME || 1000)) && (res.body.json.data.items.filter(i -> i.active && i.validated).length > 0)
```

### 2. 意味のある変数名を使用
```yaml
# 良い例: 説明的な出力名
outputs:
  user_id: res.body.json.user.id
  auth_token: res.body.json.access_token
  expires_at: res.body.json.expires_in

# 避ける: 汎用的な名前
outputs:
  data1: res.body.json.user.id
  value: res.body.json.access_token
```

### 3. エッジケースを処理
```yaml
# 良い例: 防御的プログラミング
test: |
  res.code == 200 &&
  res.body.json != null &&
  res.body.json.users != null &&
  res.body.json.users.length > 0

# 良い例: デフォルト値を提供
echo: "Processing {{outputs.api.item_count || 0}} items"
```

### 4. 複雑な式を文書化
```yaml
- name: Complex Business Logic Validation
  uses: http
  with:
    url: "{{vars.API_URL}}/business-data"
  # テストは以下を検証:
  # 1. レスポンスが成功 (200)
  # 2. 処理時間が許容範囲 (< 2s)
  # 3. データ整合性が維持 (必須フィールド存在)
  # 4. ビジネスルールが満たされる (アクティブユーザー > 0、売上 > 閾値)
  test: |
    res.code == 200 &&
    res.time < 2000 &&
    res.body.json.users != null &&
    res.body.json.revenue != null &&
    res.body.json.users.filter(u -> u.active == true).length > 0 &&
    res.body.json.revenue > 1000
```

## 次のステップ

式とテンプレートを理解したら、以下を探索してください：

1. **[データフロー](../data-flow/)** - ワークフローを通してデータがどのように移動するかを学ぶ
2. **[テストとアサーション](../testing-and-assertions/)** - 検証技術をマスターする
3. **[ハウツー](../../how-tos/)** - 実用的な式使用パターンを見る

式とテンプレートは Probe の動的エンジンです。これらの概念をマスターして、柔軟でデータ駆動の自動化ワークフローを構築しましょう。