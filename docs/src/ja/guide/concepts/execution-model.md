# 実行モデル

Probe の実行モデルを理解することは、効率的なワークフローを設計し、実行の問題をトラブルシューティングするために重要です。このガイドでは、Probe がワークフローコンポーネントを最初から最後までどのようにスケジュール、実行、管理するかについて詳しく説明します。

## 実行概要

Probe は予測可能で決定論的な方法でワークフローを処理する構造化された実行モデルに従います：

1. **ワークフロー解析**: YAML 設定の解析と検証
2. **依存関係解決**: ジョブ依存関係に基づく実行グラフの構築
3. **ジョブスケジューリング**: 依存関係に基づくジョブの実行スケジュール
4. **ステップ実行**: 各ジョブ内でステップを順次実行
5. **状態管理**: 実行状態と結果の追跡
6. **リソースクリーンアップ**: 実行後のリソースクリーンアップ

### 実行階層

```
Workflow
├── Job 1 (独立)
│   ├── Step 1.1 (順次)
│   ├── Step 1.2 (順次)
│   └── Step 1.3 (順次)
├── Job 2 (独立、Job 1 と並列)
│   ├── Step 2.1 (順次)
│   └── Step 2.2 (順次)
└── Job 3 (Job 1 と Job 2 に依存)
    ├── Step 3.1 (順次)
    └── Step 3.2 (順次)
```

## ジョブ実行モデル

### 独立ジョブ実行

依存関係のないジョブは並列実行されます：

```yaml
name: Parallel Service Check
description: Check multiple services simultaneously

jobs:
  database-check:     # 即座に実行
    name: Database Health
    steps:
      - name: Check Database
        uses: http
        with:
          url: "{{vars.DB_URL}}/health"
        test: res.code == 200

  api-check:          # database-check と並列実行
    name: API Health
    steps:
      - name: Check API
        uses: http
        with:
          url: "{{vars.API_URL}}/health"
        test: res.code == 200

  cache-check:        # 他と並列実行
    name: Cache Health
    steps:
      - name: Check Cache
        uses: http
        with:
          url: "{{vars.CACHE_URL}}/health"
        test: res.code == 200
```

**実行タイムライン:**
```
時刻 0: database-check、api-check、cache-check を同時開始
時刻 T: すべてのジョブ完了 (T = 全ジョブの最大実行時間)
```

### 依存ジョブ実行

依存関係を持つジョブは前提ジョブの完了を待ちます：

```yaml
name: Staged Deployment Validation
description: Validate deployment in dependency order

jobs:
  infrastructure:     # 最初に実行
    name: Infrastructure Check
    steps:
      - name: Database Connectivity
        uses: http
        with:
          url: "{{vars.DB_URL}}/ping"
        test: res.code == 200
        outputs:
          db_healthy: res.code == 200

  services:          # infrastructure を待つ
    name: Service Check
    needs: [infrastructure]
    steps:
      - name: API Service
        uses: http
        with:
          url: "{{vars.API_URL}}/health"
        test: res.code == 200

  integration:       # services を待つ
    name: Integration Test
    needs: [services]
    steps:
      - name: End-to-End Test
        uses: http
        with:
          url: "{{vars.API_URL}}/integration-test"
        test: res.code == 200

  notification:      # integration を待つ
    name: Send Notification
    needs: [integration]
    steps:
      - name: Notify Success
        echo: "Deployment validation completed successfully"
```

**実行タイムライン:**
```
時刻 0: infrastructure ジョブ開始
時刻 T1: infrastructure 完了 → services ジョブ開始
時刻 T2: services 完了 → integration ジョブ開始
時刻 T3: integration 完了 → notification ジョブ開始
時刻 T4: notification 完了 → ワークフロー完了
```

### 複雑な依存関係グラフ

ジョブは複数の依存関係を持ち、複雑な実行グラフを形成できます：

```yaml
jobs:
  # 基盤レイヤー (並列)
  database-setup:
    name: Database Setup
    steps:
      - name: Initialize Database
        outputs:
          db_session_id: "{{random_str(16)}}"

  cache-setup:
    name: Cache Setup
    steps:
      - name: Initialize Cache
        outputs:
          cache_session_id: "{{random_str(16)}}"

  # サービスレイヤー (基盤に依存)
  user-service:
    name: User Service Test
    needs: [database-setup, cache-setup]
    steps:
      - name: Test User Service
        outputs:
          user_service_ready: true

  order-service:
    name: Order Service Test
    needs: [database-setup]  # データベースのみ必要
    steps:
      - name: Test Order Service
        outputs:
          order_service_ready: true

  # 統合レイヤー (サービスに依存)
  integration-test:
    name: Integration Test
    needs: [user-service, order-service]
    steps:
      - name: Test Service Integration
        echo: "Testing integration between user and order services"

  # レポートレイヤー (すべてに依存)
  final-report:
    name: Final Report
    needs: [integration-test]
    steps:
      - name: Generate Report
        echo: |
          Execution Report:
          Database Setup: {{outputs.database-setup ? "✅" : "❌"}}
          Cache Setup: {{outputs.cache-setup ? "✅" : "❌"}}
          User Service: {{outputs.user-service ? "✅" : "❌"}}
          Order Service: {{outputs.order-service ? "✅" : "❌"}}
          Integration Test: {{jobs.integration-test.success ? "✅" : "❌"}}
```

**実行タイムライン:**
```
時刻 0: database-setup、cache-setup 開始 (並列)
時刻 T1: 両基盤ジョブ完了 → user-service、order-service 開始
時刻 T2: 両サービスジョブ完了 → integration-test 開始
時刻 T3: integration-test 完了 → final-report 開始
時刻 T4: final-report 完了 → ワークフロー完了
```

## ステップ実行モデル

### 順次ステップ実行

ジョブ内では、ステップは定義された順序で順次実行されます：

```yaml
jobs:
  user-workflow:
    name: User Management Workflow
    steps:
      - name: Step 1 - Create User
        id: create
        uses: http
        with:
          url: "{{vars.API_URL}}/users"
          method: POST
          body: '{"name": "Test User", "email": "test@example.com"}'
        test: res.code == 201
        outputs:
          user_id: res.body.json.user.id

      - name: Step 2 - Verify User
        id: verify
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.create.user_id}}"
        test: res.code == 200
        outputs:
          user_verified: true

      - name: Step 3 - Update User
        id: update
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.create.user_id}}"
          method: PUT
          body: '{"name": "Updated User"}'
        test: res.code == 200

      - name: Step 4 - Delete User
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.create.user_id}}"
          method: DELETE
        test: res.code == 204

      - name: Step 5 - Confirm Deletion
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.create.user_id}}"
        test: res.code == 404
```

**ステップ実行順序:**
```
Step 1 → Step 2 → Step 3 → Step 4 → Step 5
```

各ステップは前のステップの完了を待ってから開始します。

### 条件付きステップ実行

ステップは条件に基づいてスキップできますが、評価順序は順次のままです：

```yaml
steps:
  - name: Primary Service Check
    id: primary
    uses: http
    with:
      url: "{{vars.PRIMARY_URL}}/health"
    test: res.code == 200
    continue_on_error: true
    outputs:
      primary_healthy: res.code == 200

  - name: Backup Service Check
    if: "!outputs.primary.primary_healthy"  # プライマリが失敗した場合のみ
    id: backup
    uses: http
    with:
      url: "{{vars.BACKUP_URL}}/health"
    test: res.code == 200
    outputs:
      backup_healthy: res.code == 200

  - name: Success Path
    if: outputs.primary.primary_healthy     # プライマリが成功した場合のみ
    echo: "Primary service is healthy"

  - name: Fallback Path
    if: "!outputs.primary.primary_healthy && outputs.backup.backup_healthy"
    echo: "Primary failed, but backup is healthy"

  - name: Failure Path
    if: "!outputs.primary.primary_healthy && (!outputs.backup || !outputs.backup.backup_healthy)"
    echo: "Both primary and backup services failed"

  - name: Always Runs
    echo: "This step always executes"
```

**条件付き実行フロー:**
```
1. Primary Service Check (常に実行)
2. Backup Service Check (条件付き - プライマリ失敗時のみ)
3. Success Path (条件付き - プライマリ成功時のみ)
4. Fallback Path (条件付き - プライマリ失敗かつバックアップ成功時のみ)
5. Failure Path (条件付き - 両方失敗時のみ)
6. Always Runs (常に実行)
```

## 状態管理

### ジョブ状態追跡

Probe は各ジョブについて包括的な状態情報を追跡します：

```yaml
# 参照可能なジョブ状態:
jobs:
  example-job:
    steps:
      - name: Example Step
        echo: "Job states can be referenced from other jobs"

  dependent-job:
    needs: [example-job]
    steps:
      - name: Check Job States
        echo: |
          Job State Information:
          
          Job executed: {{jobs.example-job.executed}}      # ジョブが実行された場合 true
          Job success: {{jobs.example-job.success}}        # 全ステップが合格した場合 true
          Job failed: {{jobs.example-job.failed}}          # いずれかのステップが失敗した場合 true
          Job skipped: {{jobs.example-job.skipped}}         # ジョブがスキップされた場合 true
          
          Step count: {{jobs.example-job.steps.length}}    # ステップ数
          Passed steps: {{jobs.example-job.passed_steps}}  # 合格ステップ数
          Failed steps: {{jobs.example-job.failed_steps}}  # 失敗ステップ数
```

### ステップ状態と出力管理

各ステップは状態と出力情報を生成します：

```yaml
steps:
  - name: API Test Step
    id: api-test
    uses: http
    with:
      url: "{{vars.API_URL}}/test"
    test: res.code == 200 && res.time < 1000
    outputs:
      response_time: res.time
      status_code: res.code
      api_healthy: res.code == 200

  - name: Reference Previous Step
    echo: |
      Previous Step Information:
      
      Step executed: {{steps.api-test.executed}}           # ステップが実行された場合 true
      Step success: {{steps.api-test.success}}             # テストが合格した場合 true
      Step failed: {{steps.api-test.failed}}               # テストが失敗した場合 true
      Step skipped: {{steps.api-test.skipped}}             # ステップがスキップされた場合 true
      
      Step outputs:
      Response time: {{outputs.api-test.response_time}}ms
      Status code: {{outputs.api-test.status_code}}
      API healthy: {{outputs.api-test.api_healthy}}
```

### ジョブ間状態参照

ジョブは他のジョブの状態を参照できます：

```yaml
jobs:
  health-check:
    name: Health Check
    steps:
      - name: Check Service
        outputs:
          service_healthy: true

  performance-test:
    name: Performance Test
    needs: [health-check]
    if: jobs.health-check.success  # ヘルスチェックが成功した場合のみ実行
    steps:
      - name: Load Test
        outputs:
          avg_response_time: 250

  reporting:
    name: Generate Report
    needs: [health-check, performance-test]
    steps:
      - name: Status Report
        echo: |
          System Status Report:
          
          Health Check: {{jobs.health-check.success ? "✅ Passed" : "❌ Failed"}}
          Performance Test: {{
            jobs.performance-test.executed ? 
              (jobs.performance-test.success ? "✅ Passed" : "❌ Failed") : 
              "⏸️ Skipped"
          }}
          
          {{jobs.health-check.success && jobs.performance-test.success ? 
            "Average Response Time: " + outputs.performance-test.avg_response_time + "ms" : 
            "Performance data not available"}}
```

## タイミングとパフォーマンス

### 実行タイミング

Probe は複数のレベルでタイミング情報を追跡します：

```yaml
jobs:
  timing-example:
    name: Timing Example
    steps:
      - name: Quick Operation
        id: quick
        uses: http
        with:
          url: "{{vars.API_URL}}/ping"
        test: res.code == 200
        outputs:
          ping_time: res.time

      - name: Slow Operation
        id: slow
        uses: http
        with:
          url: "{{vars.API_URL}}/complex-query"
        test: res.code == 200
        outputs:
          query_time: res.time

      - name: Timing Summary
        echo: |
          Operation Timing:
          
          Quick operation: {{outputs.quick.ping_time}}ms
          Slow operation: {{outputs.slow.query_time}}ms
          Total step time: {{outputs.quick.ping_time + outputs.slow.query_time}}ms
          
          Performance classification:
          Quick: {{outputs.quick.ping_time < 100 ? "Excellent" : (outputs.quick.ping_time < 500 ? "Good" : "Slow")}}
          Slow: {{outputs.slow.query_time < 1000 ? "Fast" : (outputs.slow.query_time < 5000 ? "Acceptable" : "Too Slow")}}
```

### タイムアウト管理

異なるレベルでタイムアウトを設定します：

```yaml
jobs:
  timeout-management:
    name: Timeout Management Example
    timeout: 300s  # ジョブレベルタイムアウト（5分）
    steps:
      - name: Quick API Call
        uses: http
        with:
          url: "{{vars.API_URL}}/quick"
          timeout: 10s  # ステップレベルタイムアウト
        test: res.code == 200

      - name: Database Query
        uses: http
        with:
          url: "{{vars.DB_API}}/complex-query"
          timeout: 60s  # 複雑な操作にはより長いタイムアウト
        test: res.code == 200

      - name: External Service Call
        uses: http
        with:
          url: "{{vars.EXTERNAL_API}}/data"
          timeout: 30s  # 外部サービスは遅い場合がある
        test: res.code == 200
        continue_on_error: true  # 外部サービスが遅くてもワークフローは失敗させない
```

### 並列実行最適化

効果的な並列化によりワークフロー実行を最適化します：

```yaml
name: Optimized Parallel Execution
description: Efficiently organize jobs for maximum parallelism

jobs:
  # ティア1: 独立基盤チェック (すべて並列)
  database-check:
    name: Database Health
    steps:
      - name: DB Connection Test
        uses: http
        with:
          url: "{{vars.DB_URL}}/ping"
        test: res.code == 200

  cache-check:
    name: Cache Health
    steps:
      - name: Cache Connection Test
        uses: http
        with:
          url: "{{vars.CACHE_URL}}/ping"
        test: res.code == 200

  network-check:
    name: Network Connectivity
    steps:
      - name: External API Test
        uses: http
        with:
          url: "{{vars.EXTERNAL_API}}/ping"
        test: res.code == 200

  # ティア2: サービスレベルチェック (並列、インフラに依存)
  user-service-test:
    name: User Service Test
    needs: [database-check, cache-check]
    steps:
      - name: User API Test
        uses: http
        with:
          url: "{{vars.USER_API}}/health"
        test: res.code == 200

  order-service-test:
    name: Order Service Test
    needs: [database-check]
    steps:
      - name: Order API Test
        uses: http
        with:
          url: "{{vars.ORDER_API}}/health"
        test: res.code == 200

  notification-service-test:
    name: Notification Service Test
    needs: [network-check]
    steps:
      - name: Notification API Test
        uses: http
        with:
          url: "{{vars.NOTIFICATION_API}}/health"
        test: res.code == 200

  # ティア3: 統合テスト (サービスに依存)
  user-order-integration:
    name: User-Order Integration
    needs: [user-service-test, order-service-test]
    steps:
      - name: Integration Test
        uses: http
        with:
          url: "{{vars.API_URL}}/integration/user-order"
        test: res.code == 200

  # ティア4: 最終検証 (統合に依存)
  end-to-end-test:
    name: End-to-End Test
    needs: [user-order-integration, notification-service-test]
    steps:
      - name: Complete Workflow Test
        uses: http
        with:
          url: "{{vars.API_URL}}/e2e/complete-workflow"
        test: res.code == 200
```

**実行視覚化:**
```
時刻 0-T1: database-check、cache-check、network-check (並列)
時刻 T1-T2: user-service-test、order-service-test、notification-service-test (並列)
時刻 T2-T3: user-order-integration
時刻 T3-T4: end-to-end-test
```

## エラー伝播と回復

### エラー伝播モデル

実行モデルを通してエラーがどのように伝播するかを理解します：

```yaml
jobs:
  critical-foundation:
    name: Critical Foundation
    steps:
      - name: Critical Check
        uses: http
        with:
          url: "{{vars.CRITICAL_SERVICE}}/health"
        test: res.code == 200
        # デフォルト: continue_on_error: false (失敗時にジョブを停止)

  dependent-service:
    name: Dependent Service
    needs: [critical-foundation]  # 基盤が失敗した場合は実行されない
    steps:
      - name: Service Test
        uses: http
        with:
          url: "{{vars.SERVICE_URL}}/test"
        test: res.code == 200

  resilient-check:
    name: Resilient Check
    # 依存関係なし - 常に実行
    steps:
      - name: Independent Check
        uses: http
        with:
          url: "{{vars.INDEPENDENT_SERVICE}}/health"
        test: res.code == 200
        continue_on_error: true  # ステップが失敗してもジョブは継続

  conditional-cleanup:
    name: Conditional Cleanup
    needs: [critical-foundation, dependent-service, resilient-check]
    if: jobs.critical-foundation.failed || jobs.dependent-service.failed
    steps:
      - name: Cleanup Failed State
        echo: |
          Cleaning up after failures:
          Critical Foundation: {{jobs.critical-foundation.success ? "✅" : "❌"}}
          Dependent Service: {{jobs.dependent-service.executed ? (jobs.dependent-service.success ? "✅" : "❌") : "⏸️"}}
          Resilient Check: {{jobs.resilient-check.success ? "✅" : "❌"}}
```

### 回復実行モデル

失敗パターンに基づいて実行される回復ワークフローを実装します：

```yaml
jobs:
  primary-workflow:
    name: Primary Workflow
    steps:
      - name: Main Process
        id: main
        uses: http
        with:
          url: "{{vars.API_URL}}/main-process"
        test: res.code == 200
        continue_on_error: true
        outputs:
          process_successful: res.code == 200

  recovery-workflow:
    name: Recovery Workflow
    if: jobs.primary-workflow.failed
    steps:
      - name: Diagnose Failure
        id: diagnose
        uses: http
        with:
          url: "{{vars.API_URL}}/diagnostics"
        test: res.code == 200
        outputs:
          diagnosis: res.body.json.issue_type

      - name: Automated Recovery
        if: outputs.diagnose.diagnosis == "temporary_failure"
        uses: http
        with:
          url: "{{vars.API_URL}}/recovery/auto"
          method: POST
        test: res.code == 200

      - name: Manual Recovery Alert
        if: outputs.diagnose.diagnosis == "critical_failure"
        echo: "🚨 Critical failure detected - manual intervention required"

  validation-workflow:
    name: Validation Workflow
    needs: [primary-workflow, recovery-workflow]
    if: jobs.primary-workflow.success || jobs.recovery-workflow.success
    steps:
      - name: Validate Final State
        uses: http
        with:
          url: "{{vars.API_URL}}/validate"
        test: res.code == 200
        outputs:
          system_healthy: res.code == 200

  final-report:
    name: Final Report
    needs: [validation-workflow]
    steps:
      - name: Execution Summary
        echo: |
          Workflow Execution Summary:
          
          Primary workflow: {{jobs.primary-workflow.success ? "✅ Successful" : "❌ Failed"}}
          Recovery executed: {{jobs.recovery-workflow.executed ? "Yes" : "No"}}
          Recovery successful: {{jobs.recovery-workflow.executed ? (jobs.recovery-workflow.success ? "✅ Yes" : "❌ No") : "N/A"}}
          Final validation: {{jobs.validation-workflow ? (jobs.validation-workflow.success ? "✅ Passed" : "❌ Failed") : "⏸️ Skipped"}}
          
          Overall result: {{
            jobs.validation-workflow.success ? "✅ System operational" :
            jobs.recovery-workflow.executed ? "⚠️ System recovered with issues" :
            "❌ System failed"
          }}
```

## リソース管理

### プラグインライフサイクル管理

Probe はワークフロー実行を通してアクションプラグインを管理します：

```yaml
jobs:
  plugin-intensive-workflow:
    name: Plugin Intensive Workflow
    steps:
      # このステップのために HTTP プラグインが読み込まれる
      - name: API Test
        uses: http
        with:
          url: "{{vars.API_URL}}/test"
        test: res.code == 200

      # このステップのために SMTP プラグインが読み込まれる
      - name: Send Notification
        action: smtp
        with:
          host: "{{vars.SMTP_HOST}}"
          to: ["admin@company.com"]
          subject: "Test Completed"
          body: "API test completed successfully"

      # このステップのために Hello プラグインが読み込まれる
      - name: Debug Message
        action: hello
        with:
          message: "Debug checkpoint reached"

      # HTTP プラグインが再利用される（すでに読み込み済み）
      - name: Follow-up API Test
        uses: http
        with:
          url: "{{vars.API_URL}}/follow-up"
        test: res.code == 200
```

プラグインライフサイクル:
1. 最初のアクションが遭遇したときにプラグインが読み込まれる
2. 同じタイプの後続アクションでプラグインが再利用される
3. ジョブ完了後にプラグインがクリーンアップされる

### メモリとパフォーマンス最適化

Probe は実行をパフォーマンスとリソース使用量に最適化します：

```yaml
jobs:
  optimized-workflow:
    name: Performance Optimized Workflow
    steps:
      # 効率的: 直接的なプロパティアクセス
      - name: User Data Collection
        id: user-data
        uses: http
        with:
          url: "{{vars.API_URL}}/users"
        test: res.code == 200
        outputs:
          user_count: res.body.json.total_users    # 特定の値を抽出
          first_user_id: res.body.json.users[0].id # 直接配列アクセス
          # 避ける: large_user_list: res.body.json.users (配列全体を保存)

      # 効率的: 条件付き処理
      - name: Process Large Dataset
        if: outputs.user-data.user_count < 1000  # 管理可能なサイズの場合のみ処理
        uses: http
        with:
          url: "{{vars.API_URL}}/users/batch-process"
        test: res.code == 200

      # 効率的: スコープされた出力
      - name: Summary Generation
        echo: |
          Processing Summary:
          Total users: {{outputs.user-data.user_count}}
          First user: {{outputs.user-data.first_user_id}}
          Batch processed: {{steps.process-large-dataset.executed ? "Yes" : "No"}}
          # 不要なデータを保存せずに効率的な出力
```

## ベストプラクティス

### 1. 依存関係設計

```yaml
# 良い例: 論理的な依存関係グループ化
jobs:
  infrastructure:    # 基盤レイヤー
  application:       # インフラに依存
    needs: [infrastructure]
  integration:       # アプリケーションに依存
    needs: [application]

# 避ける: 不要な依存関係
jobs:
  independent-check-1:
  independent-check-2:
    needs: [independent-check-1]  # 真に独立なら不要
```

### 2. エラーハンドリング戦略

```yaml
# 良い例: 戦略的エラーハンドリング
- name: Critical Operation
  test: res.code == 200
  continue_on_error: false      # 重要な操作では高速失敗

- name: Optional Operation
  test: res.code == 200
  continue_on_error: true       # オプション操作では継続
```

### 3. 出力効率

```yaml
# 良い例: 効率的な出力
outputs:
  essential_data: res.body.json.id
  computed_value: res.body.json.items.length
  status_flag: res.code == 200

# 避ける: 大きなオブジェクトの保存
outputs:
  # entire_response: res.body.json  # 非常に大きくなる可能性
```

### 4. 実行フロードキュメント

```yaml
name: Well-Documented Workflow
description: |
  実行フロー:
  1. インフラ検証 (並列)
  2. サービスヘルスチェック (並列、インフラに依存)
  3. 統合テスト (順次、サービスに依存)
  4. レポート (全ての前段階に依存)
  
  予想実行時間: 2-5分
  クリティカルパス: infrastructure → services → integration → reporting
```

## 次のステップ

実行モデルを理解したら、以下を探索してください：

1. **[ファイルマージ](../file-merging/)** - 設定構成技術を学ぶ
2. **[ハウツー](../../how-tos/)** - 実用的な実行パターンの実例を見る
3. **[リファレンス](../../reference/)** - 詳細な構文と設定リファレンス

実行モデルを理解することで、並列処理を最適に活用し、失敗を適切に処理する効率的で予測可能なワークフローを設計できるようになります。