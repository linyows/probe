# データフロー

データフローは、情報が Probe ワークフローを通して移動するメカニズムです。データフローパターンを理解することで、ステップ、ジョブ、さらには外部システム間で情報をやり取りする高度なワークフローを構築できます。このガイドでは Probe の完全なデータフローシステムについて詳しく説明します。

## データフロー概要

Probe は構造化されたデータフローアプローチを使用します：

1. **入力ソース**: 環境変数、設定ファイル、ユーザー入力
2. **処理**: アクションがレスポンスと出力を生成
3. **ストレージ**: 出力が後で使用するために保存
4. **伝播**: データがステップとジョブ間を流れる
5. **消費**: 他のステップが動的設定のためにデータを使用

## データソース

### 環境変数

環境変数は外部設定と実行時コンテキストを提供します。

```yaml
# 環境変数にアクセス
steps:
  - name: Environment-based Configuration
    uses: http
    with:
      url: "{{vars.API_BASE_URL}}/{{vars.API_VERSION}}/users"
      headers:
        Authorization: "Bearer {{vars.API_TOKEN}}"
        X-Environment: "{{vars.DEPLOYMENT_ENV}}"
    test: res.code == 200
```

### 設定マージ

データはマージされた設定ファイルから取得できます：

**base-config.yml:**
```yaml
defaults:
  api:
    timeout: 30s
    retry_count: 3
vars:
  API_BASE_URL: https://api.example.com
```

**production.yml:**
```yaml
vars:
  API_BASE_URL: https://api.production.example.com
  API_TOKEN: ${PROD_API_TOKEN}
defaults:
  api:
    timeout: 10s  # プロダクション用のオーバーライド
```

使用方法:
```bash
probe workflow.yml,base-config.yml,production.yml
```

## ステップ出力

ステップは後続のステップやジョブで消費可能な出力を生成します。

### 基本的な出力定義

```yaml
steps:
  - name: User Authentication
    id: auth
    uses: http
    with:
      url: "{{vars.API_URL}}/auth/login"
      method: POST
      body: |
        {
          "username": "{{vars.USERNAME}}",
          "password": "{{vars.PASSWORD}}"
        }
    test: res.code == 200
    outputs:
      access_token: res.body.json.access_token
      refresh_token: res.body.json.refresh_token
      user_id: res.body.json.user.id
      expires_at: res.body.json.expires_at
      user_roles: res.body.json.user.roles
```

### 出力データタイプ

出力にはさまざまなデータタイプを含むことができます：

```yaml
- name: Comprehensive Data Collection
  id: data-collection
  uses: http
  with:
    url: "{{vars.API_URL}}/comprehensive-data"
  test: res.code == 200
  outputs:
    # シンプルな値
    user_count: res.body.json.stats.user_count
    server_version: res.body.json.version
    is_healthy: res.body.json.health.status == "healthy"
    
    # 複雑なオブジェクト
    user_profile: res.body.json.user
    configuration: res.body.json.config
    metrics: res.body.json.metrics
    
    # 配列
    active_users: res.body.json.users.filter(u -> u.active == true)
    error_codes: res.body.json.errors.map(e -> e.code)
    
    # 計算された値
    success_rate: (res.body.json.successful_requests / res.body.json.total_requests) * 100
    avg_response_time: res.body.json.response_times.sum() / res.body.json.response_times.length
    
    # レスポンスメタデータ
    response_time: res.time
    response_size: res.body_size
    content_type: res.headers["content-type"]
```

### 出力スコープ

出力は含まれるステップにスコープされ、ID で参照できます：

```yaml
steps:
  - name: Database Setup
    id: db-setup
    uses: http
    with:
      url: "{{vars.DB_API}}/initialize"
    outputs:
      db_session_id: res.body.json.session_id
      db_host: res.body.json.host
      db_port: res.body.json.port

  - name: Application Test
    id: app-test
    uses: http
    with:
      url: "{{vars.APP_URL}}/test"
      headers:
        X-DB-Session: "{{outputs.db-setup.db_session_id}}"
        X-DB-Host: "{{outputs.db-setup.db_host}}"
    outputs:
      test_result: res.body.json.result
      test_duration: res.time

  - name: Performance Analysis
    echo: |
      Performance Analysis:
      Database: {{outputs.db-setup.db_host}}:{{outputs.db-setup.db_port}}
      Test Result: {{outputs.app-test.test_result}}
      Test Duration: {{outputs.app-test.test_duration}}ms
```

## ジョブ間データフロー

データはジョブレベルの出力と依存関係を通してジョブ間を流れることができます。

### ジョブ依存関係とデータ共有

```yaml
jobs:
  initialization:
    name: System Initialization
    steps:
      - name: Create Test Environment
        id: env-setup
        uses: http
        with:
          url: "{{vars.SETUP_API}}/create-environment"
          method: POST
          body: |
            {
              "environment_name": "test_{{random_str(8)}}",
              "configuration": "standard"
            }
        test: res.code == 201
        outputs:
          environment_id: res.body.json.environment.id
          environment_name: res.body.json.environment.name
          database_url: res.body.json.environment.database_url
          api_endpoint: res.body.json.environment.api_endpoint

  api-tests:
    name: API Testing Suite
    needs: [initialization]  # initialization の完了を待つ
    steps:
      - name: Test User API
        uses: http
        with:
          url: "{{outputs.initialization.api_endpoint}}/users"
          headers:
            X-Environment: "{{outputs.initialization.environment_id}}"
        test: res.code == 200
        outputs:
          user_count: res.body.json.total_users
          api_response_time: res.time

      - name: Test Database Connectivity
        uses: http
        with:
          url: "{{outputs.initialization.database_url}}/ping"
        test: res.code == 200
        outputs:
          db_response_time: res.time

  reporting:
    name: Test Reporting
    needs: [initialization, api-tests]  # 両ジョブを待つ
    steps:
      - name: Generate Test Report
        echo: |
          Test Execution Report
          =====================
          
          Environment: {{outputs.initialization.environment_name}}
          Environment ID: {{outputs.initialization.environment_id}}
          
          API Tests:
          - User Count: {{outputs.api-tests.user_count}}
          - API Response Time: {{outputs.api-tests.api_response_time}}ms
          
          Database Tests:
          - DB Response Time: {{outputs.api-tests.db_response_time}}ms
          
          Overall Status: All tests completed successfully

  cleanup:
    name: Environment Cleanup
    needs: [reporting]  # reporting 完了後に実行
    steps:
      - name: Destroy Test Environment
        uses: http
        with:
          url: "{{vars.SETUP_API}}/environments/{{outputs.initialization.environment_id}}"
          method: DELETE
        test: res.code == 204
```

### ジョブ間出力参照

`outputs.job-name.output-name` 構文を使用して他のジョブの出力にアクセスします：

```yaml
jobs:
  data-collection:
    steps:
      - name: Collect User Data
        outputs:
          total_users: res.body.json.count
          active_users: res.body.json.active_count

  analysis:
    needs: [data-collection]
    steps:
      - name: Analyze User Metrics
        echo: |
          User Analysis:
          Total Users: {{outputs.data-collection.total_users}}
          Active Users: {{outputs.data-collection.active_users}}
          Activity Rate: {{(outputs.data-collection.active_users / outputs.data-collection.total_users) * 100}}%
```

## 高度なデータフローパターン

### データ変換チェーン

複数のステップを通してデータを変換します：

```yaml
jobs:
  data-processing-pipeline:
    name: Data Processing Pipeline
    steps:
      - name: Fetch Raw Data
        id: raw-data
        uses: http
        with:
          url: "{{vars.DATA_API}}/raw-data"
        outputs:
          raw_records: res.body.json.records
          total_count: res.body.json.total
          fetch_time: res.time

      - name: Filter Data
        id: filtered-data
        echo: "Filtering data..."
        outputs:
          # アクティブなレコードをフィルタ
          active_records: "{{outputs.raw-data.raw_records.filter(r -> r.status == 'active')}}"
          active_count: "{{outputs.raw-data.raw_records.filter(r -> r.status == 'active').length}}"
          
      - name: Aggregate Data
        id: aggregated-data
        echo: "Aggregating data..."
        outputs:
          # カテゴリでグループ化してメトリクスを計算
          categories: "{{outputs.filtered-data.active_records.groupBy(r -> r.category)}}"
          avg_score: "{{outputs.filtered-data.active_records.map(r -> r.score).sum() / outputs.filtered-data.active_count}}"
          
      - name: Generate Summary
        echo: |
          Data Processing Summary:
          
          Raw Records: {{outputs.raw-data.total_count}}
          Active Records: {{outputs.filtered-data.active_count}}
          Processing Rate: {{(outputs.filtered-data.active_count / outputs.raw-data.total_count) * 100}}%
          Average Score: {{outputs.aggregated-data.avg_score}}
          Fetch Time: {{outputs.raw-data.fetch_time}}ms
```

### 条件付きデータフロー

条件に基づいてデータフローを制御します：

```yaml
jobs:
  adaptive-processing:
    steps:
      - name: Assess Data Quality
        id: quality-check
        uses: http
        with:
          url: "{{vars.API_URL}}/data-quality"
        outputs:
          quality_score: res.body.json.quality_score
          has_errors: res.body.json.error_count > 0
          record_count: res.body.json.record_count
          
      - name: Standard Processing
        if: outputs.quality-check.quality_score >= 0.8 && !outputs.quality-check.has_errors
        id: standard-processing
        uses: http
        with:
          url: "{{vars.PROCESSING_API}}/standard"
          method: POST
          body: |
            {
              "record_count": {{outputs.quality-check.record_count}},
              "quality_mode": "standard"
            }
        outputs:
          processing_result: res.body.json.result
          processing_time: res.time
          
      - name: Enhanced Processing
        if: outputs.quality-check.quality_score < 0.8 || outputs.quality-check.has_errors
        id: enhanced-processing
        uses: http
        with:
          url: "{{vars.PROCESSING_API}}/enhanced"
          method: POST
          body: |
            {
              "record_count": {{outputs.quality-check.record_count}},
              "quality_mode": "enhanced",
              "error_correction": true
            }
        outputs:
          processing_result: res.body.json.result
          processing_time: res.time
          corrections_applied: res.body.json.corrections
          
      - name: Processing Summary
        echo: |
          Data Processing Complete:
          
          Quality Score: {{outputs.quality-check.quality_score}}
          Processing Mode: {{outputs.quality-check.quality_score >= 0.8 ? "Standard" : "Enhanced"}}
          
          {{outputs.standard-processing ? "Standard Processing Time: " + outputs.standard-processing.processing_time + "ms" : ""}}
          {{outputs.enhanced-processing ? "Enhanced Processing Time: " + outputs.enhanced-processing.processing_time + "ms" : ""}}
          {{outputs.enhanced-processing ? "Corrections Applied: " + outputs.enhanced-processing.corrections_applied : ""}}
```

### データ蓄積パターン

複数のソースからデータを収集します：

```yaml
jobs:
  multi-source-data-collection:
    steps:
      - name: Source A Data
        id: source-a
        uses: http
        with:
          url: "{{vars.SOURCE_A_URL}}/data"
        outputs:
          source_a_count: res.body.json.count
          source_a_data: res.body.json.data
          source_a_time: res.time

      - name: Source B Data
        id: source-b
        uses: http
        with:
          url: "{{vars.SOURCE_B_URL}}/data"
        outputs:
          source_b_count: res.body.json.count
          source_b_data: res.body.json.data
          source_b_time: res.time

      - name: Source C Data
        id: source-c
        uses: http
        with:
          url: "{{vars.SOURCE_C_URL}}/data"
        outputs:
          source_c_count: res.body.json.count
          source_c_data: res.body.json.data
          source_c_time: res.time

      - name: Aggregate All Sources
        echo: |
          Multi-Source Data Summary:
          
          Source A: {{outputs.source-a.source_a_count}} records ({{outputs.source-a.source_a_time}}ms)
          Source B: {{outputs.source-b.source_b_count}} records ({{outputs.source-b.source_b_time}}ms)
          Source C: {{outputs.source-c.source_c_count}} records ({{outputs.source-c.source_c_time}}ms)
          
          Total Records: {{outputs.source-a.source_a_count + outputs.source-b.source_b_count + outputs.source-c.source_c_count}}
          Average Response Time: {{(outputs.source-a.source_a_time + outputs.source-b.source_b_time + outputs.source-c.source_c_time) / 3}}ms
          
          Fastest Source: {{
            outputs.source-a.source_a_time <= outputs.source-b.source_b_time && outputs.source-a.source_a_time <= outputs.source-c.source_c_time ? "Source A" :
            outputs.source-b.source_b_time <= outputs.source-c.source_c_time ? "Source B" : "Source C"
          }}
```

## データ検証と品質

### 出力検証

出力のデータ品質を確保します：

```yaml
- name: Data Collection with Validation
  id: validated-data
  uses: http
  with:
    url: "{{vars.API_URL}}/user-data"
  test: |
    res.code == 200 &&
    res.body.json.users != null &&
    res.body.json.users.length > 0 &&
    res.body.json.users.all(u -> u.id != null && u.email != null)
  outputs:
    # 検証済み出力
    user_count: res.body.json.users.length
    valid_users: res.body.json.users.filter(u -> u.id != null && u.email != null)
    admin_users: res.body.json.users.filter(u -> u.role == "admin")
    
    # データ品質メトリクス
    data_completeness: res.body.json.users.filter(u -> u.id != null && u.email != null).length / res.body.json.users.length
    has_admin_users: res.body.json.users.any(u -> u.role == "admin")
    
    # レスポンスメタデータ
    data_freshness: res.headers["last-modified"]
    cache_status: res.headers["x-cache-status"]
```

### データサニタイズ

使用前にデータをクリーンアップ・サニタイズします：

```yaml
- name: Sanitize User Input
  id: sanitized-input
  uses: http
  with:
    url: "{{vars.API_URL}}/user-input"
  outputs:
    # 生データ
    raw_input: res.body.json.input
    
    # サニタイズ済みデータ
    clean_email: res.body.json.input.email.lower().trim()
    clean_name: res.body.json.input.name.trim()
    safe_description: res.body.json.input.description.substring(0, 500)  # 長さを制限
    
    # バリデーションフラグ
    email_valid: res.body.json.input.email.matches("[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}")
    name_valid: res.body.json.input.name.length >= 2 && res.body.json.input.name.length <= 50
```

## パフォーマンス考慮事項

### 効率的なデータアクセス

データアクセスパターンを最適化します：

```yaml
# 良い例: 直接的なプロパティアクセス
outputs:
  user_id: res.body.json.user.id
  user_name: res.body.json.user.name

# 良い例: 再利用を伴う単一計算
outputs:
  active_users: res.body.json.users.filter(u -> u.active == true)
  active_user_count: res.body.json.users.filter(u -> u.active == true).length

# 避ける: 繰り返しの高コスト計算
# outputs:
#   user_count: res.body.json.users.filter(u -> expensive_validation(u)).length
#   user_list: res.body.json.users.filter(u -> expensive_validation(u))
```

### メモリ管理

大きなデータセットに注意します：

```yaml
# 良い例: 必要不可欠なデータのみを抽出
outputs:
  user_ids: res.body.json.users.map(u -> u.id)
  user_count: res.body.json.users.length
  first_user: res.body.json.users[0]

# 避ける: 大きなオブジェクトを不要に保存
# outputs:
#   all_user_data: res.body.json.users  # 非常に大きくなる可能性
#   complete_response: res.body.json     # レスポンス全体
```

### 選択的データ抽出

必要なデータのみを抽出します：

```yaml
- name: Efficient Data Extraction
  uses: http
  with:
    url: "{{vars.API_URL}}/large-dataset"
  outputs:
    # 概要情報のみを抽出
    record_count: res.body.json.metadata.total_records
    last_updated: res.body.json.metadata.last_updated
    status: res.body.json.metadata.status
    
    # 基準による特定レコードを抽出
    critical_items: res.body.json.data.filter(item -> item.priority == "critical")
    error_items: res.body.json.data.filter(item -> item.status == "error")
    
    # 集計を計算
    avg_score: res.body.json.data.map(item -> item.score).sum() / res.body.json.data.length
    max_score: res.body.json.data.map(item -> item.score).max()
    
    # データセット全体は保存しない
    # full_dataset: res.body.json.data  # 大きなデータセットでは避ける
```

## ベストプラクティス

### 1. 明確な出力命名

出力には説明的な名前を使用します：

```yaml
# 良い例: 説明的な名前
outputs:
  user_authentication_token: res.body.json.access_token
  session_expiry_timestamp: res.body.json.expires_at
  user_permission_level: res.body.json.user.role

# 避ける: 汎用的な名前
outputs:
  token: res.body.json.access_token
  time: res.body.json.expires_at
  level: res.body.json.user.role
```

### 2. 型整合性のある出力

一貫したデータタイプを維持します：

```yaml
# 良い例: 一貫したタイプ
outputs:
  user_count: res.body.json.users.length          # 常に数値
  is_admin: res.body.json.user.role == "admin"    # 常にブール値
  user_email: res.body.json.user.email || ""      # 常に文字列（デフォルト値付き）

# 避ける: 不整合なタイプ
outputs:
  user_count: res.body.json.users.length || "unknown"  # 数値または文字列
```

### 3. エラーセーフなデータアクセス

潜在的な null/undefined 値を処理します：

```yaml
# 良い例: 安全なデータアクセス
outputs:
  user_id: res.body.json.user && res.body.json.user.id ? res.body.json.user.id : null
  email_verified: res.body.json.user && res.body.json.user.email_verified == true
  profile_complete: res.body.json.user && res.body.json.user.profile && res.body.json.user.profile.complete == true

# 良い例: 安全なナビゲーションを使用
test: res.body.json.user?.id != null && res.body.json.user?.email != null
```

### 4. データ依存関係の文書化

データがどこからどこに流れるかを文書化します：

```yaml
jobs:
  user-setup:
    name: User Account Setup
    steps:
      - name: Create User Account
        # 生成: user_id, username, email
        outputs:
          user_id: res.body.json.user.id
          username: res.body.json.user.username
          email: res.body.json.user.email

  user-verification:
    name: User Account Verification
    needs: [user-setup]
    steps:
      - name: Send Verification Email
        # 使用: user-setup からの user_id, email
        action: smtp
        with:
          to: ["{{outputs.user-setup.email}}"]
          subject: "Verify your account"
          body: "Click here to verify user {{outputs.user-setup.user_id}}"
```

## 次のステップ

データフローを理解したら、以下を探索してください：

1. **[テストとアサーション](../testing-and-assertions/)** - 検証技術を学ぶ
2. **[エラーハンドリング](../error-handling/)** - データフローの失敗を適切に処理する
3. **[ハウツー](../../how-tos/)** - 実用的なデータフローパターンを見る

データフローは Probe ワークフローの循環システムです。これらのパターンをマスターして、高度でデータ駆動の自動化プロセスを構築しましょう。