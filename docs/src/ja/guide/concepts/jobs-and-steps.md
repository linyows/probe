# ジョブとステップ

ジョブとステップは Probe ワークフローの構築要素です。それらの仕組み、実行モデル、相互作用パターンを理解することは、効果的な自動化を構築するために重要です。このガイドでは詳細な動作と高度な使用例について説明します。

## ジョブの基礎

**ジョブ**は、関連するステップをまとめて一つの単位として実行する論理的なグループです。ジョブは以下を提供します：

- **分離**: 各ジョブは独自のコンテキストで実行される
- **並列性**: 依存関係がない限り、ジョブは同時実行可能
- **状態管理**: ジョブは実行状態と結果を追跡
- **出力共有**: ジョブは他のジョブが使用する出力を生成可能

### ジョブ構造

```yaml
jobs:
  job-id:                    # 一意の識別子（英数字、ハイフン、アンダースコア）
    name: Human Readable Name # オプション: 表示名
    needs: [other-job]       # オプション: ジョブ依存関係
    if: condition            # オプション: 条件付き実行
    continue_on_error: true  # オプション: 失敗時もワークフローを継続
    timeout: 300s            # オプション: ジョブタイムアウト
    steps:                   # 必須: ステップの配列
      # ステップ定義...
```

### ジョブライフサイクル

ジョブは実行中に複数の状態を経過します：

1. **Pending**: ジョブが実行待ちキューに入っている
2. **Running**: ジョブがアクティブにステップを実行中
3. **Success**: すべてのステップが正常に完了
4. **Failed**: 一つ以上のステップが失敗
5. **Skipped**: 条件によりジョブがスキップされた
6. **Cancelled**: タイムアウトやエラーによりジョブがキャンセルされた

### ジョブ依存関係

`needs` キーワードを使用して実行依存関係を作成します：

```yaml
jobs:
  setup:
    name: Environment Setup
    steps:
      - name: Initialize Database
        uses: http
        with:
          post: "/init"
        test: res.code == 200
        outputs:
          db_session_id: res.body.json.session_id

  test-suite-a:
    name: API Test Suite A
    needs: [setup]           # setup の完了を待つ
    steps:
      - name: Test User API
        uses: http
        with:
          get: "/users"
          headers:
            X-Session-ID: "{{outputs.setup.db_session_id}}"
        test: res.code == 200

  test-suite-b:
    name: API Test Suite B
    needs: [setup]           # setup にも依存
    steps:
      - name: Test Order API
        uses: http
        with:
          get: "/orders"
          headers:
            X-Session-ID: "{{outputs.setup.db_session_id}}"
        test: res.code == 200

  cleanup:
    name: Environment Cleanup
    needs: [test-suite-a, test-suite-b]  # 両方のテストスイートを待つ
    steps:
      - name: Clean Database
        uses: http
        with:
          post: "/cleanup"
          headers:
            X-Session-ID: "{{outputs.setup.db_session_id}}"
        test: res.code == 200
```

### 条件付きジョブ実行

ジョブは他のジョブの結果に基づいて条件付きで実行できます：

```yaml
jobs:
  health-check:
    name: Basic Health Check
    steps:
      - name: Ping Service
        id: ping
        uses: http
        with:
          get: "/ping"
        test: res.code == 200
        outputs:
          service_responsive: res.code == 200

  detailed-check:
    name: Detailed Health Check
    if: jobs.health-check.success && outputs.health-check.service_responsive
    steps:
      - name: Deep Health Check
        uses: http
        with:
          get: "/health/detailed"
        test: res.code == 200

  recovery:
    name: Service Recovery
    if: jobs.health-check.failed
    steps:
      - name: Restart Service
        uses: http
        with:
          post: "/restart"
        test: res.code == 200

  notification:
    name: Send Notifications
    needs: [health-check]
    if: jobs.health-check.failed || jobs.recovery.executed
    steps:
      - name: Alert Team
        echo: |
          Service Status Alert:
          Health Check: {{jobs.health-check.success ? "✅" : "❌"}}
          Recovery Attempted: {{jobs.recovery.executed ? "Yes" : "No"}}
          Recovery Successful: {{jobs.recovery.success ? "✅" : "❌"}}
```

## ステップの基礎

**ステップ**は Probe の最小実行単位です。各ステップは特定のアクションを実行し、以下が可能です：

- アクションの実行（HTTP リクエスト、コマンドなど）
- アサーションによる結果テスト
- 他のステップで使用する出力の生成
- コンソールへのメッセージ出力
- 条件付き実行

### ステップ構造

```yaml
steps:
  - name: Step Name          # 必須: 説明的な名前
    id: step-id             # オプション: 参照用の一意識別子
    action: http            # オプション: 実行するアクション
    with:                   # オプション: アクションパラメータ
      url: https://api.example.com
      method: GET
    test: res.code == 200 # オプション: テスト条件
    outputs:                # オプション: 他のステップに渡すデータ
      response_time: res.time
      user_count: res.body.json.total_users
    echo: "Message"         # オプション: メッセージ表示
    if: condition           # オプション: 条件付き実行
    continue_on_error: false # オプション: ステップ失敗時に継続
    timeout: 30s            # オプション: ステップタイムアウト
```

### ステップタイプ

#### 1. アクションステップ

HTTP リクエストなどの特定のアクションを実行：

```yaml
- name: Check User API
  action: http
  with:
    url: "{{vars.API_URL}}/users/{{vars.TEST_USER_ID}}"
    method: GET
    headers:
      Authorization: "Bearer {{vars.API_TOKEN}}"
      Accept: "application/json"
  test: res.code == 200 && res.body.json.user.active == true
  outputs:
    user_id: res.body.json.user.id
    user_email: res.body.json.user.email
    last_login: res.body.json.user.last_login
```

#### 2. Echo ステップ

メッセージや計算された値を表示：

```yaml
- name: Display Results
  echo: |
    Test Results Summary:
    
    User ID: {{outputs.previous-step.user_id}}
    Email: {{outputs.previous-step.user_email}}
    Last Login: {{outputs.previous-step.last_login}}
    
    Response Time: {{outputs.previous-step.response_time}}ms
    Test Completed: {{unixtime()}}
```

#### 3. ハイブリッドステップ

アクションと echo メッセージを組み合わせ：

```yaml
- name: Test and Report
  action: http
  with:
    url: "{{vars.API_URL}}/status"
  test: res.code == 200
  echo: |
    API Status Check:
    Status Code: {{res.status}}
    Response Time: {{res.time}}ms
    API Version: {{res.body.json.version}}
```

### ステップ実行フロー

ジョブ内のステップはデフォルトで順次実行されます：

```yaml
jobs:
  sequential-test:
    name: Sequential Step Execution
    steps:
      - name: Step 1 - Setup
        id: setup
        uses: http
        with:
          url: "{{vars.API_URL}}/setup"
        test: res.code == 200
        outputs:
          session_id: res.body.json.session_id

      - name: Step 2 - Execute Test
        id: test
        uses: http
        with:
          url: "{{vars.API_URL}}/test"
          headers:
            X-Session-ID: "{{outputs.setup.session_id}}"
        test: res.code == 200
        outputs:
          test_result: res.body.json.result

      - name: Step 3 - Cleanup
        uses: http
        with:
          url: "{{vars.API_URL}}/cleanup"
          headers:
            X-Session-ID: "{{outputs.setup.session_id}}"
        test: res.code == 200

      - name: Step 4 - Report
        echo: "Test completed with result: {{outputs.test.test_result}}"
```

### 条件付きステップ実行

ステップは条件に基づいて実行できます：

```yaml
steps:
  - name: Primary Health Check
    id: primary
    action: http
    with:
      url: "{{vars.PRIMARY_SERVICE_URL}}/health"
    test: res.code == 200
    continue_on_error: true
    outputs:
      primary_healthy: res.status == 200

  - name: Backup Service Check
    if: "!outputs.primary.primary_healthy"
    action: http
    with:
      url: "{{vars.BACKUP_SERVICE_URL}}/health"
    test: res.code == 200
    outputs:
      backup_healthy: res.status == 200

  - name: Success Report
    if: outputs.primary.primary_healthy || outputs.backup.backup_healthy
    echo: |
      Service Status: ✅ Healthy
      Primary: {{outputs.primary.primary_healthy ? "Online" : "Offline"}}
      Backup: {{outputs.backup.backup_healthy ? "Online" : "N/A"}}

  - name: Failure Report
    if: "!outputs.primary.primary_healthy && !outputs.backup.backup_healthy"
    echo: "🚨 CRITICAL: Both primary and backup services are down!"
```

## 高度なパターン

### 1. エラー回復パターン

回復ステップによる堅牢なエラーハンドリングを実装：

```yaml
jobs:
  resilient-check:
    name: Resilient Service Check
    steps:
      - name: Attempt Primary Connection
        id: primary-attempt
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/api/v1/health"
          timeout: 10s
        test: res.code == 200
        continue_on_error: true
        outputs:
          primary_success: res.status == 200

      - name: Try Alternative Endpoint
        if: "!outputs.primary-attempt.primary_success"
        id: alt-attempt
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/api/v2/health"
          timeout: 15s
        test: res.code == 200
        continue_on_error: true
        outputs:
          alt_success: res.status == 200

      - name: Fallback to Legacy Endpoint
        if: "!outputs.primary-attempt.primary_success && !outputs.alt-attempt.alt_success"
        id: legacy-attempt
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/health"
          timeout: 20s
        test: res.code == 200
        continue_on_error: true
        outputs:
          legacy_success: res.status == 200

      - name: Final Status Report
        echo: |
          Service Health Check Results:
          
          Primary API (v1): {{outputs.primary-attempt.primary_success ? "✅" : "❌"}}
          Alternative API (v2): {{outputs.alt-attempt.alt_success ? "✅" : "❌"}}
          Legacy API: {{outputs.legacy-attempt.legacy_success ? "✅" : "❌"}}
          
          Overall Status: {{
            outputs.primary-attempt.primary_success || 
            outputs.alt-attempt.alt_success || 
            outputs.legacy-attempt.legacy_success ? "HEALTHY" : "DOWN"
          }}
```

### 2. データ収集と集約

分析のため複数のステップでデータを収集：

```yaml
jobs:
  performance-analysis:
    name: Performance Analysis
    steps:
      - name: Test Homepage
        id: homepage
        uses: http
        with:
          url: "{{vars.BASE_URL}}/"
        test: res.code == 200
        outputs:
          homepage_time: res.time
          homepage_size: res.body_size

      - name: Test API Endpoint
        id: api
        uses: http
        with:
          url: "{{vars.BASE_URL}}/api/users"
        test: res.code == 200
        outputs:
          api_time: res.time
          api_size: res.body_size

      - name: Test Search Function
        id: search
        uses: http
        with:
          url: "{{vars.BASE_URL}}/search?q=test"
        test: res.code == 200
        outputs:
          search_time: res.time
          search_size: res.body_size

      - name: Performance Summary
        echo: |
          Performance Analysis Results:
          
          Homepage:
            Response Time: {{outputs.homepage.homepage_time}}ms
            Size: {{outputs.homepage.homepage_size}} bytes
            
          API Endpoint:
            Response Time: {{outputs.api.api_time}}ms
            Size: {{outputs.api.api_size}} bytes
            
          Search Function:
            Response Time: {{outputs.search.search_time}}ms
            Size: {{outputs.search.search_size}} bytes
            
          Average Response Time: {{
            (outputs.homepage.homepage_time + 
             outputs.api.api_time + 
             outputs.search.search_time) / 3
          }}ms
          
          Total Data Transfer: {{
            outputs.homepage.homepage_size + 
            outputs.api.api_size + 
            outputs.search.search_size
          }} bytes
```

### 3. 動的ステップ設定

実行時条件に基づいてステップを設定：

```yaml
jobs:
  adaptive-monitoring:
    name: Adaptive Monitoring
    steps:
      - name: Determine Environment
        id: env-detect
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/config"
        test: res.code == 200
        outputs:
          environment: res.body.json.environment
          feature_flags: res.body.json.features
          monitoring_level: res.body.json.monitoring.level

      - name: Basic Health Check
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/health"
        test: res.code == 200

      - name: Detailed Monitoring
        if: outputs.env-detect.monitoring_level == "detailed"
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/metrics"
        test: res.code == 200
        outputs:
          cpu_usage: res.body.json.system.cpu_percent
          memory_usage: res.body.json.system.memory_percent

      - name: Feature-Specific Tests
        if: outputs.env-detect.feature_flags.beta_features == true
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/beta/features"
        test: res.code == 200

      - name: Production Alerts
        if: outputs.env-detect.environment == "production" && (outputs.detailed.cpu_usage > 80 || outputs.detailed.memory_usage > 90)
        echo: |
          🚨 PRODUCTION ALERT: High resource usage detected!
          CPU: {{outputs.detailed.cpu_usage}}%
          Memory: {{outputs.detailed.memory_usage}}%
```

## ステップとジョブの識別

### ステップ ID

`id` を使用してワークフローの他の部分からステップを参照します：

```yaml
steps:
  - name: User Authentication Test
    id: auth-test                    # 参照用IDを定義
    action: http
    with:
      url: "{{vars.API_URL}}/auth/login"
      method: POST
      body: |
        {
          "username": "testuser",
          "password": "{{vars.TEST_PASSWORD}}"
        }
    test: res.code == 200
    outputs:
      auth_token: res.body.json.token
      user_id: res.body.json.user.id

  - name: User Profile Test
    action: http
    with:
      url: "{{vars.API_URL}}/users/{{outputs.auth-test.user_id}}"  # IDで参照
      headers:
        Authorization: "Bearer {{outputs.auth-test.auth_token}}"   # IDで参照
    test: res.code == 200
```

### ジョブ参照

他のジョブからジョブ結果を参照：

```yaml
jobs:
  database-check:
    name: Database Connectivity
    steps:
      - name: Test Database
        uses: http
        with:
          url: "{{vars.DB_API}}/ping"
        test: res.code == 200

  api-check:
    name: API Functionality
    needs: [database-check]
    steps:
      - name: Test API
        if: jobs.database-check.success    # ジョブ成功を参照
        uses: http
        with:
          url: "{{vars.API_URL}}/health"
        test: res.code == 200

      - name: Skip Message
        if: jobs.database-check.failed     # ジョブ失敗を参照
        echo: "Skipping API test due to database connectivity issues"
```

## パフォーマンス最適化

### 1. 並列ジョブ実行

可能な限りジョブが並列実行されるよう構造化：

```yaml
jobs:
  # これらのジョブは並列実行可能（依存関係なし）
  frontend-test:
    name: Frontend Tests
    steps:
      - name: Test UI Components
        uses: http
        with:
          url: "{{vars.FRONTEND_URL}}"
        test: res.code == 200

  backend-test:
    name: Backend Tests
    steps:
      - name: Test API Endpoints
        uses: http
        with:
          url: "{{vars.BACKEND_URL}}/api"
        test: res.code == 200

  database-test:
    name: Database Tests
    steps:
      - name: Test Database Connection
        uses: http
        with:
          url: "{{vars.DB_URL}}/health"
        test: res.code == 200

  # このジョブはすべての並列ジョブの完了を待つ
  integration-test:
    name: Integration Tests
    needs: [frontend-test, backend-test, database-test]
    steps:
      - name: End-to-End Test
        uses: http
        with:
          url: "{{vars.APP_URL}}/integration-test"
        test: res.code == 200
```

### 2. 効率的なリソース使用

ベターなリソース利用のためステップ実行を最適化：

```yaml
jobs:
  efficient-monitoring:
    name: Efficient Resource Monitoring
    steps:
      # タイムアウトを使用してハングを防止
      - name: Quick Health Check
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/ping"
          timeout: 5s                    # ping 用の短いタイムアウト
        test: res.code == 200

      # 条件付きの高コスト操作
      - name: Detailed Analysis
        if: outputs.previous.response_time > 1000  # レスポンスが遅い場合のみ処理
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/detailed-metrics"
          timeout: 30s                   # 詳細分析用の長いタイムアウト
        test: res.code == 200

      # 関連操作をバッチ化
      - name: Batch Status Check
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/batch-status"
          method: POST
          body: |
            {
              "checks": [
                {"type": "health", "endpoint": "/health"},
                {"type": "metrics", "endpoint": "/metrics"},
                {"type": "version", "endpoint": "/version"}
              ]
            }
        test: res.code == 200 && res.body.json.all_passed == true
```

## ベストプラクティス

### 1. ジョブ粒度

ジョブサイズの適切なバランスを保つ：

```yaml
# 良い例: 焦点を絞った、一貫性のあるジョブ
jobs:
  authentication-tests:
    name: Authentication System Tests
    steps:
      - name: Test Login
      - name: Test Logout
      - name: Test Token Refresh
      - name: Test Password Reset

  user-management-tests:
    name: User Management Tests
    steps:
      - name: Test User Creation
      - name: Test User Update
      - name: Test User Deletion

# 避ける: 過度に細かいジョブ
jobs:
  test-login:           # 細かすぎる
    steps:
      - name: Test Login
  test-logout:          # それぞれがステップであるべき、ジョブではない
    steps:
      - name: Test Logout

# 避ける: モノリシックなジョブ
jobs:
  all-tests:            # 広すぎる
    steps:
      - name: Test Login
      - name: Test Database
      - name: Test Email
      - name: Test Files
      # ... 50個以上の無関係なステップ
```

### 2. 明確なステップ名

説明的でアクション指向のステップ名を使用：

```yaml
steps:
  # 良い例: 明確で具体的な名前
  - name: Verify User Registration API Returns 201
  - name: Test Database Connection Pool Health
  - name: Validate JWT Token Expiration Logic
  - name: Check Email Service Rate Limiting

  # 避ける: 曖昧または汎用的な名前
  - name: Test API           # 曖昧すぎる
  - name: Check Thing        # 説明的でない
  - name: Step 1             # コンテキストなし
```

### 3. 適切なエラーハンドリング

適切なエラーハンドリング戦略を実装：

```yaml
steps:
  # 重要なステップ - 高速失敗
  - name: Verify Database Connectivity
    action: http
    with:
      url: "{{vars.DB_URL}}/ping"
    test: res.code == 200
    continue_on_error: false        # デフォルト: ジョブを失敗させる

  # 非重要ステップ - 失敗時も継続
  - name: Update Usage Analytics
    action: http
    with:
      url: "{{vars.ANALYTICS_URL}}/update"
    test: res.code == 200
    continue_on_error: true         # 失敗しても継続

  # 回復ステップ
  - name: Log Failure Details
    if: steps.previous.failed
    echo: "Analytics update failed, but continuing with main workflow"
```

## 次のステップ

ジョブとステップを詳しく理解したら、以下を探索してください：

1. **[アクション](../actions/)** - アクションシステムと利用可能なプラグインについて学ぶ
2. **[式とテンプレート](../expressions-and-templates/)** - 動的設定とテストをマスターする
3. **[データフロー](../data-flow/)** - ワークフローを通してデータがどう移動するかを理解する

ジョブとステップは Probe の実行エンジンです。これらの概念をマスターして、効率的で信頼性が高く、保守しやすい自動化ワークフローを構築しましょう。