# アクション

アクションはProbeの中で実際の作業を実行するコアとなる実行単位です。プラグインとして実装されており、Probeを拡張可能でモジュラーなものにしています。このガイドではアクションシステム、組み込みアクション、プラグインアーキテクチャの使用方法を詳しく説明します。

## アクションシステム概要

Probeのアクションシステムは、以下を提供するプラグインアーキテクチャの上に構築されています：

- **モジュラー性**: 各アクションは独立したプラグインです
- **拡張性**: カスタムアクションを簡単に追加できます
- **分離**: アクションは安定性のために別プロセスで実行されます
- **標準化**: すべてのアクションは同じインターフェースに従います

### アクション実行フロー

1. **プラグイン発見**: Probeが利用可能なアクションプラグインを特定します
2. **プラグイン初期化**: アクションプラグインが別プロセスで開始されます
3. **通信**: ProbeはgRPCを介してプラグインと通信します
4. **実行**: プラグインが要求されたアクションを実行します
5. **レスポンス**: 結果が処理のためにProbeに返されます
6. **クリーンアップ**: プラグインプロセスは使用後に終了されます

## 組み込みアクション

Probeには一般的な使用例をカバーする組み込みアクションがいくつか付属しています。

### HTTPアクション

`http`アクションは最も汎用性があり、HTTP/HTTPSリクエストを行うために最も一般的に使用されるアクションです。

#### 基本的な使用方法

GETリクエストに必要なのはURLだけです。

```yaml
- name: Simple GET Request
  uses: http
  with:
    url: https://api.example.com/users
    method: GET
  test: res.code == 200
```

#### 完全なHTTPアクション リファレンス

このアクションが受け取るフィールドを1つのステップにまとめた例です。

```yaml
- name: Comprehensive HTTP Request
  uses: http
  timeout: 30s                                  # オプション: リクエストタイムアウト
  with:
    url: https://api.example.com/users/123        # 必須: ターゲット URL
    method: POST                                  # オプション: HTTP メソッド (デフォルト: GET)
    headers:                                      # オプション: リクエストヘッダー
      Content-Type: "application/json"
      Authorization: "Bearer {{vars.api_token}}"
      X-Request-ID: "{{random_str(16)}}"
    body: |                                       # オプション: リクエストボディ
      {
        "name": "John Doe",
        "email": "john@example.com",
        "active": true
      }
  test: res.code == 200 && res.body.success == true
  outputs:
    user_id: res.body.user.id
    created_at: res.body.user.created_at
    response_time: (rt.sec * 1000)
```

#### HTTPレスポンス オブジェクト

HTTPアクションは豊富なレスポンスオブジェクトを提供します：

```yaml
# 利用可能なレスポンスプロパティ:
test: |
  res.code == 200 &&                    # HTTP ステータスコード
  (rt.sec * 1000) < 1000 &&                      # レスポンス時間（ミリ秒）
  res.body_size < 10000 &&               # レスポンスボディサイズ（バイト）
  res.headers["Content-Type"] == "application/json" &&  # レスポンスヘッダー
  res.body.success == true &&            # 解析された JSON ボディ（該当する場合）
  res.body contains "success"           # テキストとしてのレスポンスボディ
```

#### 一般的なHTTPパターン

**API認証:**
```yaml
jobs:
- id: api-test
  name: api-test
  steps:
    - name: Authenticate
      id: auth
      uses: http
      with:
        url: "{{vars.api_base_url}}/auth/login"
        method: POST
        headers:
          Content-Type: "application/json"
        body: |
          {
            "username": "{{vars.api_username}}",
            "password": "{{vars.api_password}}"
          }
      test: res.code == 200
      outputs:
        access_token: res.body.access_token
        refresh_token: res.body.refresh_token

    - name: Make Authenticated Request
      uses: http
      with:
        url: "{{vars.api_base_url}}/protected/resource"
        method: GET
        headers:
          Authorization: "Bearer {{outputs.auth.access_token}}"
      test: res.code == 200
```

**ファイルアップロード:**
```yaml
- name: Upload File
  uses: http
  with:
    url: "{{vars.api_url}}/upload"
    method: POST
    headers:
      Content-Type: "multipart/form-data"
    body: |
      --boundary123
      Content-Disposition: form-data; name="file"; filename="test.txt"
      Content-Type: text/plain
      
      This is test file content
      --boundary123--
  test: res.code == 201
```

**GraphQLクエリ:**
```yaml
- name: GraphQL Query
  uses: http
  with:
    url: "{{vars.graphql_endpoint}}"
    method: POST
    headers:
      Content-Type: "application/json"
      Authorization: "Bearer {{vars.graphql_token}}"
    body: |
      {
        "query": "query GetUser($id: ID!) { user(id: $id) { name email active } }",
        "variables": { "id": "{{vars.test_user_id}}" }
      }
  test: res.code == 200 && res.body.data.user != null
  outputs:
    user_name: res.body.data.user.name
    user_email: res.body.data.user.email
```

### Shellアクション

`shell`アクションはワークフロー内でシェルコマンドとスクリプトの安全な実行を可能にします。包括的な出力キャプチャ、タイムアウト保護、環境変数サポートを提供します。

#### 基本的な使用方法

シェルのステップに必要なのはコマンドだけです。

```yaml
- name: Build Application
  uses: shell
  with:
    cmd: "npm run build"
    workdir: "/app"
    timeout: "5m"
  test: res.code == 0
```

#### 完全なShellアクション リファレンス

このアクションが受け取るフィールドを1つのステップにまとめた例です。

```yaml
- name: Deploy Application
  uses: shell
  with:
    cmd: "./deploy.sh production"              # 必須: 実行するコマンド
    shell: "/bin/bash"                        # オプション: 使用するシェル (デフォルト: /bin/sh)
    workdir: "/deploy"                        # オプション: 作業ディレクトリ（絶対パス）
    timeout: "15m"                           # オプション: 実行タイムアウト（デフォルト: 30s）
    env:                                     # オプション: 環境変数
      DEPLOY_ENV: "production"
      API_KEY: "{{vars.production_api_key}}"
      BUILD_VERSION: "{{vars.version}}"
  test: res.code == 0 && (res.stdout | contains("Deploy successful"))
  outputs:
    deploy_time: res.rt
    deploy_log: res.stdout
```

#### Shellレスポンス オブジェクト

終了コードと、標準出力および標準エラー出力をテストから参照できます。

```yaml
# 利用可能なレスポンスプロパティ:
test: |
  res.code == 0 &&                         # 終了コード（0 = 成功）
  res.stdout contains "success" &&        # 標準出力
  res.stderr == "" &&                      # 標準エラー（空 = エラーなし）
  req.cmd == "npm run build" &&           # 元のコマンド
  req.shell == "/bin/bash"                # 実行に使用されたシェル
```

#### 一般的なShellパターン

**ビルドとテストのパイプライン:**
```yaml
jobs:
- id: build-and-test
  name: build-and-test
  steps:
    - name: Install Dependencies
      uses: shell
      with:
        cmd: "npm ci"
        workdir: "/app"
        timeout: "5m"
      test: res.code == 0

    - name: Run Tests
      uses: shell
      with:
        cmd: "npm test"
        workdir: "/app"
        env:
          NODE_ENV: "test"
          CI: "true"
      test: res.code == 0

    - name: Build Application
      uses: shell
      with:
        cmd: "npm run build"
        workdir: "/app"
        env:
          NODE_ENV: "production"
      test: res.code == 0
```

**システムヘルス監視:**
```yaml
- name: Check System Health
  uses: shell
  with:
    cmd: |
      echo "=== System Health Report ===" &&
      echo "CPU Usage: $(top -bn1 | grep Cpu | cut -d' ' -f2)" &&
      echo "Memory: $(free -h | grep Mem)" &&
      echo "Disk: $(df -h /)"
  test: res.code == 0
  outputs:
    health_report: res.stdout
```

#### セキュリティ機能

shellアクションは複数のセキュリティレイヤーを実装します：

- **シェル制限**: 承認されたシェル実行ファイルのみを許可
- **パス検証**: 作業ディレクトリは絶対パスである必要があります
- **タイムアウト保護**: 暴走プロセスを防止
- **環境分離**: 安全な環境変数ハンドリング
- **出力サニタイズ**: コマンド出力の安全なキャプチャ

### Helloアクション

`hello`アクションは主にテストとデモンストレーションに使用されます。プラグイン機能を検証するシンプルな方法を提供します。

```yaml
- name: Test Hello Action
  id: hello
  uses: hello
  with:
    message: "Test message"           # res にそのまま返る
  echo: "{{res.message}}"
  outputs:
    greeting: res.message
```

**Helloアクション レスポンス:** このアクション固有のパラメータはありません。`with`に渡したキーはそのまま`res`に返り、`status`は常に`0`です。

```yaml
test: status == 0 && res.message != null
```

### SMTPアクション

`smtp`アクションは通知とアラートのメール送信機能を有効にします。

```yaml
- name: Send Email Notification
  uses: smtp
  with:
    addr: "smtp.gmail.com              # SMTP サーバーホスト:587                         # SMTP サーバーポート"
    from: alerts@mycompany.com        # 送信者メールアドレス
    to: ["admin@mycompany.com", "team@mycompany.com"]  # 受信者
    subject: "System Alert: {{vars.alert_type}}"       # メール件名
    session: 1
    message: 1
    length: 500
  echo: |                           # メール本文（プレーンテキストまたは HTML）
  test: res.code == "sent"
  outputs:
    message_id: res.message_id
    recipients_count: res.recipients_count
```

**SMTP設定例:**

**Gmail:**
```yaml
with:
  host: smtp.gmail.com
  port: 587
  username: "your-email@gmail.com"
  password: "your-app-password"
  tls: true
```

**AWS SES:**
```yaml
with:
  host: email-smtp.us-east-1.amazonaws.com
  port: 587
  username: "{{vars.aws_ses_username}}"
  password: "{{vars.aws_ses_password}}"
  tls: true
```

**Office 365:**
```yaml
with:
  host: smtp.office365.com
  port: 587
  username: "your-email@company.com"
  password: "{{vars.o365_password}}"
  tls: true
```

## リトライ機能

Probeは全てのアクションで利用可能な統一されたリトライ機能を提供します。この機能により、一時的な障害や起動待機時間が必要なサービスに対してアクションを自動的に再実行できます。

### 基本的なリトライ構文

リトライはステップレベルで設定し、任意のアクション（`http`、`shell`、`db`など）と組み合わせて使用できます：

```yaml
- name: "Service Health Check"
  uses: http
  with:
    method: GET
    url: "http://localhost:8080/health"
  retry:
    max_attempts: 10      # 最大試行回数 (1-100)
    interval: "2s"        # リトライ間隔
    initial_delay: "5s"   # 初回実行前の待機時間（オプション）
  test: res.code == 200
```

### リトライパラメータ

リトライの挙動は3つのパラメータで決まります。試行回数の上限、試行の間隔、そして初回実行前の待機時間です。

#### `max_attempts` (必須)
- **型:** Integer
- **範囲:** 1-100
- **説明:** リトライする最大試行回数

#### `interval` (オプション)
- **型:** StringまたはDuration
- **デフォルト:** `1s`
- **形式:** Go duration形式 (`500ms`, `2s`, `1m`) または数値 (秒)
- **説明:** 各リトライ試行の間隔

#### `initial_delay` (オプション)
- **型:** StringまたはDuration
- **デフォルト:** `0s` (遅延なし)
- **形式:** Go duration形式 (`500ms`, `2s`, `1m`) または数値 (秒)
- **説明:** 最初の試行前の待機時間

### 成功条件

各アクションの成功条件は統一されたステータスシステムに基づいています：

- **成功**: `status`フィールドが`0`の場合
- **失敗**: `status`フィールドが`0`以外の場合

アクション別の成功条件：
- **HTTP**: ステータスコード200-299 → `status: 0`
- **Shell**: 終了コード0 → `status: 0`
- **DB**: クエリ成功 → `status: 0`

### 実行フロー

1. `initial_delay`が指定されている場合、その時間だけ待機
2. アクションを実行
3. `status`が`0`の場合、成功として結果を返す
4. `status`が`0`以外で、まだ試行回数に余裕がある場合：
   - `interval`の時間だけ待機
   - ステップ2に戻る
5. 最大試行回数に達した場合、最後の実行結果を返す

### アクション別の使用例

待つ対象はアクションによって変わります。以下ではHTTP、Shell、DBのそれぞれで、起動や接続の確立を待つ書き方を示します。

#### HTTPリトライ - API起動待機

起動直後のAPIは接続を受け付けないことがあります。応答するまでリクエストを繰り返します。

```yaml
- name: "Wait for API Server"
  uses: http
  timeout: "5s"
  with:
    url: "{{vars.api_base_url}}/health"
    method: GET
  retry:
    max_attempts: 30
    interval: "2s"
    initial_delay: "10s"
  test: res.code == 200 && res.body.status == "healthy"
```

#### Shellリトライ - サービス起動監視

コマンドの終了コードを条件にして、サービスが立ち上がるまで待ちます。

```yaml
- name: "Wait for Database"
  uses: shell
  with:
    cmd: "pg_isready -h postgres -p 5432 -U app"
  retry:
    max_attempts: 60
    interval: "1s"
    initial_delay: "5s"
  test: res.code == 0
```

#### DBリトライ - 接続確立

データベースは接続を受け付けるまでに時間がかかります。接続できるまでクエリを繰り返します。

```yaml
- name: "Database Connection Test"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost:5432/testdb"
    query: "SELECT 1"
  retry:
    max_attempts: 20
    interval: "3s"
  test: res.code == 0
```

### 高度なリトライパターン

リトライのパラメータにも式を書けます。これにより、環境に応じた待ち時間や、条件を満たしたときだけのリトライを表現できます。

#### 段階的な遅延

依存の順に待つジョブを並べれば、起動の順序に沿ったリトライになります。

```yaml
jobs:
- id: staged-startup
  name: staged-startup
  steps:
    - name: "Quick Health Check"
      uses: http
      with:
        method: GET
        url: "{{vars.service_url}}/ping"
      retry:
        max_attempts: 5
        interval: "100ms"
      test: res.code == 200

    - name: "Detailed Health Check"
      uses: http
      with:
        method: GET
        url: "{{vars.service_url}}/health"
      retry:
        max_attempts: 30
        interval: "2s"
        initial_delay: "1s"
      test: res.code == 200 && res.body.database_connected == true
```

#### 条件付きリトライ

リトライのパラメータに式を書けば、環境に応じて待ち方を変えられます。

```yaml
- name: "Environment-Aware Health Check"
  uses: http
  with:
    method: GET
    url: "{{vars.service_url}}/health"
  retry:
    max_attempts: "{{vars.environment == 'production' ? 60 : 10}}"
    interval: "{{vars.environment == 'production' ? '5s' : '1s'}}"
  test: res.code == 200
```

### ベストプラクティス

リトライを設定するときは、1回あたりのタイムアウト、試行回数の上限、初期遅延の3つを対象の性質に合わせます。

#### 1. 適切なタイムアウト設定

1回あたりのタイムアウトは、リトライ間隔より短くします。

```yaml
# 良い例: リトライ間隔より短いタイムアウト
- name: "Quick API Check"
  uses: http
  timeout: "2s"        # 短いタイムアウト
  with:
    method: GET
    url: "{{vars.api_url}}/ping"
  retry:
    max_attempts: 10
    interval: "3s"       # タイムアウトより長い間隔
```

#### 2. 実用的な最大試行回数

試行回数は、待つ対象が実際に立ち上がるまでの時間から決めます。

```yaml
# 良い例: 合理的な試行回数
- name: "Service Startup"
  uses: shell
  with:
    cmd: "service myapp status"
  retry:
    max_attempts: 30     # 30回 × 2秒 = 最大1分待機
    interval: "2s"
```

#### 3. 初期遅延の活用

起動に時間がかかるとわかっている相手には、最初の試行自体を遅らせます。

```yaml
# 良い例: サービス起動時間を考慮した初期遅延
- name: "Database Health Check"
  uses: db
  with:
    dsn: "{{vars.db_dsn}}"
    query: "SELECT 1"
  retry:
    max_attempts: 20
    interval: "3s"
    initial_delay: "15s"  # データベース起動に時間がかかる場合
```

## 高度なアクション使用方法

アクションは単独で呼ぶだけでなく、組み合わせて使います。失敗の扱い、複数のアクションの連結、実行中に求めた値からの設定を扱います。

### アクションでのエラーハンドリング

アクション失敗に対する堅牢なエラーハンドリングを実装します：

```yaml
jobs:
- id: resilient-http-check
  name: resilient-http-check
  steps:
    - name: Primary Endpoint Check
      id: primary
      uses: http
      timeout: 10s
      with:
        url: "{{vars.primary_url}}/health"
        method: GET
      test: res.code == 200
      outputs:
        primary_healthy: res.code == 200
        primary_response_time: (rt.sec * 1000)

    - name: Secondary Endpoint Check
      id: secondary
      uses: http
      timeout: 15s
      with:
        url: "{{vars.secondary_url}}/health"
        method: GET
      test: res.code == 200
      outputs:
        secondary_healthy: res.code == 200
        secondary_response_time: (rt.sec * 1000)

    - name: Alert on Total Failure
      uses: smtp
      with:
        addr: "{{vars.smtp_host}}:587"
        from: "alerts@company.com"
        to: "ops-team@company.com"
        subject: "CRITICAL: All endpoints down"
        session: 1
        message: 1
        length: 500
      echo: |
        CRITICAL ALERT: All monitored endpoints are down
          
        Primary Endpoint: FAILED
        Secondary Endpoint: FAILED
          
        Time: {{unixtime()}}
          
        Immediate investigation required!
```

### アクション構成パターン

アクションを組み合わせて複雑なワークフローを作成します：

```yaml
jobs:
- id: comprehensive-api-test
  name: Comprehensive API Testing
  steps:
    # 1. ヘルスチェック
    - name: Verify API Health
      id: health
      uses: http
      with:
        method: GET
        url: "{{vars.api_url}}/health"
      test: res.code == 200
      outputs:
        api_version: res.body.version
        database_connected: res.body.database.connected

    # 2. 認証テスト
    - name: Test Authentication
      id: auth
      uses: http
      with:
        url: "{{vars.api_url}}/auth/token"
        method: POST
        headers:
          Content-Type: "application/json"
        body: |
          {
            "client_id": "{{vars.client_id}}",
            "client_secret": "{{vars.client_secret}}",
            "grant_type": "client_credentials"
          }
      test: res.code == 200
      outputs:
        access_token: res.body.access_token
        token_expires: res.body.expires_in

    # 3. 機能テスト
    - name: Test Core Functionality
      id: functional
      uses: http
      with:
        url: "{{vars.api_url}}/api/test"
        method: GET
        headers:
          Authorization: "Bearer {{outputs.auth.access_token}}"
      test: res.code == 200 && res.body.test_passed == true
      outputs:
        test_duration: (rt.sec * 1000)
        test_results: res.body.results

    # 4. パフォーマンス検証
    - name: Validate Performance
      uses: smtp
      with:
        addr: "{{vars.smtp_host}}:587"
        from: "performance@company.com"
        to: "dev-team@company.com"
        subject: "Performance Alert: Slow API Response"
        session: 1
        message: 1
        length: 500
      echo: |
        Performance Alert
          
        API Version: {{outputs.health.api_version}}
        Response Time: {{outputs.functional.test_duration}}ms
        Expected: < 2000ms
          
        Please investigate performance degradation.

        功通知
    - name: Success Report
      uses: hello
      echo: |
        ✅ API Test Suite Completed Successfully
          
        Health Check: ✅ (v{{outputs.health.api_version}})
        Authentication: ✅ (expires in {{outputs.auth.token_expires}}s)
        Functionality: ✅ ({{outputs.functional.test_duration}}ms)
        Performance: ✅ (within acceptable limits)
```

### 動的アクション設定

実行時の条件に基づいてアクションを動的に設定します：

```yaml
jobs:
- id: adaptive-monitoring
  name: adaptive-monitoring
  steps:
    - name: Determine Environment
      id: env
      uses: http
      with:
        method: GET
        url: "{{vars.config_service_url}}/environment"
      test: res.code == 200
      outputs:
        environment: res.body.environment
        notification_level: res.body.notifications.level
        smtp_config: res.body.smtp

    - name: Environment-Specific Health Check
      id: health
      uses: http
      timeout: "{{outputs.vars.environment == 'production' ? '5s' : '30s'}}"
      with:
        method: GET
        url: "{{vars.service_url}}/health"
      test: res.code == 200
      outputs:
        service_status: res.body.status
        error_count: res.body.errors

    - name: Conditional Alert
      uses: smtp
      with:
        addr: "{{outputs.vars.smtp_config.host}}:{{outputs.vars.smtp_config.port}}"
        from: "monitoring@company.com"
        to: "{{outputs.vars.environment == 'production' ? ['ops@company.com', 'management@company.com'] : ['dev@company.com']}}"
        subject: "{{outputs.vars.environment == 'production' ? 'PRODUCTION' : 'NON-PROD'}} Alert: Service Errors Detected"
        session: 1
        message: 1
        length: 500
      echo: |
        Service Error Alert
          
        Environment: {{outputs.vars.environment}}
        Service Status: {{outputs.health.service_status}}
        Error Count: {{outputs.health.error_count}}
          
        {{outputs.vars.environment == "production" ? "IMMEDIATE ACTION REQUIRED" : "Please investigate when convenient"}}
```

## プラグインアーキテクチャ詳細

アクションはすべてプラグインです。以下ではProbeとプラグインの通信方法、プラグインの開始と終了、組み込みプラグインの管理を扱います。

### プラグイン通信

Probeはプラグイン通信にgRPCを使用し、以下を提供します：

- **型安全性**: Protocol Buffersによる強い型付け
- **パフォーマンス**: 効率的なバイナリシリアライゼーション
- **クロスランゲージ**: gRPCをサポートする任意の言語でプラグインを作成可能
- **信頼性**: 組み込みのエラーハンドリングとタイムアウト

### プラグインライフサイクル

1. **発見**: Probeは起動時に利用可能なプラグインを発見します
2. **オンデマンド読み込み**: プラグインは必要な時にのみ読み込まれます
3. **プロセス分離**: 各プラグインは独自のプロセスで実行されます
4. **リソース管理**: プラグインプロセスは使用後にクリーンアップされます
5. **エラー分離**: プラグインの障害はProbeをクラッシュさせません

### 組み込みプラグイン管理

Probeは組み込みプラグインを自動的に管理します：

```bash
# 組み込みプラグインは Probe バイナリに埋め込まれています
probe workflow.yml  # 必要なプラグインを自動的に読み込みます

# 組み込みアクションに別途インストールは不要:
# - http
# - hello  
# - smtp
```

## アクションのベストプラクティス

どのアクションを使う場合でも、同じ問いが生じます。どれだけ待つか、失敗時にどうするか、認証情報をどこから得るか、何を検証するか、次のステップに何を渡すかです。

### 1. タイムアウト設定

常に適切なタイムアウトを設定します：

```yaml
# 良い例: 期待されるレスポンス時間に基づく具体的なタイムアウト
- name: Quick Health Check
  uses: http
  timeout: 5s              # クイックping は高速でレスポンスすべき
  with:
    method: GET
    url: "{{vars.api_url}}/ping"

- name: Complex Query
  uses: http
  timeout: 60s             # 複雑な操作にはより多くの時間が必要
  with:
    method: GET
    url: "{{vars.api_url}}/complex-report"
```

### 2. エラーハンドリング戦略

適切なエラーハンドリングを実装します：

```yaml
# 重要なアクション - 高速失敗
- name: Database Connectivity Check
  uses: http
  with:
    method: GET
    url: "{{vars.db_url}}/ping"
  test: res.code == 200

# 非重要なアクション - エラーでも継続
- name: Optional Analytics Update
  uses: http
  with:
    method: GET
    url: "{{vars.analytics_url}}/update"
  test: res.code == 200
```

### 3. 安全な設定

機密データを適切に扱います：

```yaml
# 良い例: シークレットに環境変数を使用
- name: Authenticated Request
  uses: http
  with:
    method: GET
    url: "{{vars.api_url}}/secure"
    headers:
      Authorization: "Bearer {{vars.api_token}}"  # vars から

# 良い例: 安全な SMTP 設定を使用
- name: Send Alert
  uses: smtp
  with:
    addr: "{{vars.smtp_host}}:25"
    from: "probe@example.com"
    to: "ops@example.com"
    session: 1
    message: 1
    length: 500
- name: Bad Example
  uses: http
  with:
    headers:
      Authorization: "Bearer secret-token-123"  # これは決してしてはいけません！
```

### 4. レスポンス検証

アクションレスポンスを徹底的に検証します：

```yaml
- name: Comprehensive API Test
  uses: http
  with:
    method: GET
    url: "{{vars.api_url}}/users"
  test: |
    res.code == 200 &&
    res.headers["Content-Type"] contains "application/json" &&
    res.body.users != null &&
    len(res.body.users) > 0 &&
    (rt.sec * 1000) < 1000
  outputs:
    user_count: len(res.body.users)
    response_time: (rt.sec * 1000)
```

### 5. 意味のある出力

他のステップのために有用な出力を定義します：

```yaml
- name: User Creation Test
  uses: http
  with:
    url: "{{vars.api_url}}/users"
    method: POST
    body: '{"name": "Test User", "email": "test@example.com"}'
  test: res.code == 201
  outputs:
    created_user_id: res.body.user.id
    created_user_email: res.body.user.email
    creation_timestamp: res.body.user.created_at
    response_time: (rt.sec * 1000)
```

## カスタムアクション（上級）

Probeには強力な組み込みアクションが付属していますが、特殊なニーズのためにカスタムアクションで拡張できます。

### カスタムアクション インターフェース

カスタムアクションはActionsインターフェースを実装する必要があります：

```go
type Actions interface {
    Run(with map[string]any) (map[string]any, error)
}
```

### アクションプラグイン構造

プラグインは上記のインターフェースを実装し、gRPC経由で提供されます。

```go
// カスタムアクションプラグインの例
package main

import (
    "github.com/linyows/probe"
    "github.com/hashicorp/go-plugin"
)

type CustomAction struct{}

func (c *CustomAction) Run(with map[string]any) (map[string]any, error) {
    // ここにカスタムアクションロジック
    return map[string]any{
        "status": "success",
        "result": "custom action completed",
    }, nil
}

func main() {
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: probe.Handshake,
        Plugins: map[string]plugin.Plugin{
            "actions": &probe.ActionsPlugin{Impl: &CustomAction{}},
        },
        GRPCServer: plugin.DefaultGRPCServer,
    })
}
```

## 次のステップ

アクションシステムを理解したら、以下を探索してください：

1. **[式とテンプレート](/ja/guide/concepts/expressions-and-templates)** - 動的設定とテストを学ぶ
2. **[データフロー](/ja/guide/concepts/data-flow)** - アクション間でのデータの流れを理解する
3. **[ハウツー](/ja/guide/how-tos/api-testing)** - 実用的なアクション使用パターンを見る

アクションはProbeの働き手です。組み込みアクションをマスターし、プラグインアーキテクチャを理解して、強力で拡張可能な自動化ワークフローを構築しましょう。
