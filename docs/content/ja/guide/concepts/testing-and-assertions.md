# テストとアサーション

テストとアサーションは Probe ワークフローの品質ゲートです。アクションが期待される結果を生み出すことを検証し、システムの信頼性を確保します。このガイドではテスト式、アサーションパターン、ワークフローに堅牢な検証を組み込むための戦略について説明します。

## テストの基礎

Probe のすべてのステップには、アクションの結果を検証する `test` 条件を含めることができます。テストはステップの成功または失敗を決定するブール式です。

### 基本テスト構造

```yaml
- name: API Health Check
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/health"
  test: res.code == 200
```

テスト式はレスポンス（`res`）を評価し、成功には true、失敗には false を返します。

### テスト式のコンテキスト

テスト式は包括的なレスポンスデータにアクセスできます：

```yaml
# HTTP レスポンステストコンテキスト
test: |
  res.code == 200 &&           # HTTP ステータスコード
  (rt.sec * 1000) < 1000 &&             # レスポンス時間（ミリ秒）
  res.body_size > 0 &&           # レスポンスボディサイズ（バイト）
  res.headers["Content-Type"] == "application/json" &&  # レスポンスヘッダー
  res.body.status == "healthy" && # 解析された JSON レスポンス
  res.body contains "success"   # テキストとしてのレスポンスボディ
```

## HTTP レスポンステスト

### ステータスコード検証

```yaml
# 正確なステータスコード
test: res.code == 200

# ステータスコード範囲
test: res.code >= 200 && res.code < 300

# 複数の許可されるコード
test: res.code in [200, 201, 202]

# クライアント vs サーバーエラー
test: res.code < 400  # 成功またはリダイレクト
test: res.code >= 400 && res.code < 500  # クライアントエラー
test: res.code >= 500  # サーバーエラー
```

### レスポンス時間テスト

```yaml
# パフォーマンス検証
test: (rt.sec * 1000) < 1000                    # 1秒以内にレスポンス必要
test: (rt.sec * 1000) >= 100 && (rt.sec * 1000) <= 500  # レスポンス時間範囲
test: (rt.sec * 1000) < {{vars.MAX_RESPONSE_TIME || 2000}}  # 設定可能な閾値

# パフォーマンスカテゴリ
test: |
  res.code == 200 && (
    (rt.sec * 1000) < 200 ? "excellent" :
    (rt.sec * 1000) < 500 ? "good" :
    (rt.sec * 1000) < 1000 ? "acceptable" : "poor"
  ) != "poor"
```

### レスポンスサイズ検証

```yaml
# コンテンツ存在
test: res.body_size > 0                    # コンテンツあり
test: res.body_size > 100                  # 最小コンテンツサイズ
test: res.body_size < 1048576             # 最大 1MB レスポンス

# サイズベース検証
test: |
  res.code == 200 &&
  res.body_size > 50 &&                   # 空のエラーメッセージでない
  res.body_size < 100000                  # 予期せず大きくない
```

### ヘッダー検証

```yaml
# コンテンツタイプチェック
test: res.headers["Content-Type"] == "application/json"
test: res.headers["Content-Type"] startsWith "text/"
test: res.headers["Content-Type"] contains "charset=utf-8"

# セキュリティヘッダー
test: |
  "X-Frame-Options" in res.headers &&
  "X-Content-Type-Options" in res.headers &&
  res.headers["X-Frame-Options"] == "DENY"

# キャッシュ制御
test: res.headers["Cache-Control"] contains "no-cache"

# レート制限
test: res.headers["X-Rate-Limit-Remaining"] > "10"

# カスタムヘッダー
test: |
  "X-Request-Id" in res.headers &&
  len(res.headers["X-Request-Id"]) == 36  # UUID形式
```

## JSON レスポンステスト

### 基本 JSON 検証

```yaml
# JSON 構造検証
test: |
  res.code == 200 &&
  res.body != null &&
  res.body.status == "success" &&
  res.body.data != null

# 必須フィールド存在
test: |
  "id" in res.body &&
  "name" in res.body &&
  "email" in res.body &&
  "created_at" in res.body
```

### データタイプ検証

```yaml
# タイプチェック
test: |
  typeof(res.body.id) == "number" &&
  typeof(res.body.name) == "string" &&
  typeof(res.body.active) == "boolean" &&
  typeof(res.body.tags) == "array" &&
  typeof(res.body.metadata) == "object"

# 値制約
test: |
  res.body.id > 0 &&
  len(res.body.name) >= 2 &&
  res.body.score >= 0 && res.body.score <= 100
```

### 配列とコレクションテスト

```yaml
# 配列検証
test: |
  res.body.users != null &&
  len(res.body.users) > 0 &&
  len(res.body.users) <= 100

# 配列コンテンツ検証
test: |
  all(res.body.users, #.id != null && 
    #.email != null)

# 特定要素チェック
test: |
  any(res.body.users, #.role == "admin") &&
  len(filter(res.body.users, #.active == true)) > 0

# 配列ユニーク性
test: |
  len(res.body.user_ids) == len(uniq(res.body.user_ids))
```

### ネストしたデータ検証

```yaml
# 深いオブジェクト検証
test: |
  res.body.user != null &&
  res.body.user.profile != null &&
  res.body.user.profile.preferences != null &&
  res.body.user.profile.preferences.notifications == true

# 複雑なネスト構造
test: |
  all(res.body.data.orders, #.id != null &&
    len(#.items) > 0 &&
    #all(.items, #.product_id != null && 
      #.quantity > 0 && 
      #.price > 0) &&
    #.total == sum(map(#.items, #.quantity * #.price)))
```

## テキストレスポンステスト

### パターンマッチング

```yaml
# シンプルなテキストマッチング
test: res.body contains "success"
test: res.body startsWith "<!DOCTYPE html>"
test: res.body endsWith "</html>"

# 大文字小文字を区別しないマッチング
test: lower(res.body) contains "error"

# 複数パターン
test: |
  res.body contains "status" &&
  res.body contains "healthy" &&
  !res.body contains "error"
```

### 正規表現テスト

```yaml
# レスポンス内のメール検証
test: res.body matches "user-\\d+@example\\.com"

# URL パターン検証
test: res.body matches "https://[a-zA-Z0-9.-]+/api/v\\d+/"

# データフォーマット検証
test: |
  res.body matches "\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z"  # ISO タイムスタンプ

# データ抽出と検証
test: |
  res.body matches "Version: v\\d+\\.\\d+\\.\\d+" &&
  res.body.extract("v(\\d+)\\.(\\d+)\\.(\\d+)")[1] >= "2"  # メジャーバージョン >= 2
```

### コンテンツ長と品質

```yaml
# コンテンツ長検証
test: |
  len(res.body) > 100 &&
  len(res.body) < 10000

# コンテンツ品質チェック
test: |
  len(split(res.body, "\\n")) > 5 &&           # 複数行コンテンツ
  !res.body contains "Lorem ipsum" &&          # プレースホルダーテキストでない
  len(split(res.body, " ")) > 20               # 実質的なコンテンツ
```

## 高度なテストパターン

### 条件付きテスト

```yaml
# 環境固有テスト
test: |
  res.code == 200 &&
  (vars.NODE_ENV == "development" ? 
    (rt.sec * 1000) < 5000 :           # 開発環境では緩い設定
    (rt.sec * 1000) < 1000             # プロダクション用は厳格
  )

# 機能フラグテスト
test: |
  res.code == 200 &&
  (res.body.features.beta_enabled == true ?
    res.body.beta_data != null :    # ベータ機能にはデータが必要
    res.body.beta_data == null      # ベータ機能は存在しないべき
  )
```

### ステップ間検証

```yaml
jobs:
- id: data-consistency-test
  name: data-consistency-test
  steps:
    - name: Get User Count
      id: user-count
      uses: http
      with:
        method: GET
        url: "{{vars.API_URL}}/users/count"
      test: res.code == 200
      outputs:
        total_users: res.body.count

    - name: Get User List
      id: user-list
      uses: http
      with:
        method: GET
        url: "{{vars.API_URL}}/users"
      test: |
        res.code == 200 &&
        len(res.body.users) == outputs['user-count'].total_users  # 整合性チェック
      outputs:
        user_list: res.body.users

    - name: Validate User Data Integrity
      uses: http
      with:
        method: GET
        url: "{{vars.API_URL}}/users/{{outputs['user-list'].user_list[0].id}}"
      test: |
        res.code == 200 &&
        res.body.user.id == outputs['user-list'].user_list[0].id &&
        res.body.user.email == outputs['user-list'].user_list[0].email
```

### ビジネスロジックテスト

```yaml
- name: E-commerce Business Logic Test
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/orders/{{vars.TEST_ORDER_ID}}"
  test: |
    res.code == 200 &&
    res.body.order != null &&
    
    # 注文合計が品目の合計と等しい
    res.body.order.total == 
      sum(map(res.body.order.line_items, #.quantity * #.price)) &&
    
    # 税計算が正しい（8%税率を想定）
    res.body.order.tax_amount == 
      round(Math) / 100 &&
    
    # 送料が正しく適用される
    (res.body.order.subtotal >= 100 ? 
      res.body.order.shipping_cost == 0 :     # $100以上は送料無料
      res.body.order.shipping_cost == 9.99    # 標準送料
    ) &&
    
    # 最終合計計算
    res.body.order.total == 
      res.body.order.subtotal + res.body.order.tax_amount + res.body.order.shipping_cost
```

## エラーテストと負のケース

### 期待されるエラーシナリオ

```yaml
- name: Test Invalid Authentication
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/protected"
    headers:
      Authorization: "Bearer invalid-token"
  test: |
    res.code == 401 &&
    res.body.error == "invalid_token" &&
    res.body.message contains "authentication"

- name: Test Rate Limiting
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/rate-limited-endpoint"
  test: |
    res.code in [200, 429] &&  # 成功またはレート制限
    (res.code == 429 ? 
      "Retry-After" in res.headers && 
      res.body.error == "rate_limit_exceeded" :
      res.body.status == "success"
    )

- name: Test Malformed Request
  uses: http
  with:
    url: "{{vars.API_URL}}/users"
    method: POST
    body: '{"invalid": json}'  # 意図的に不正なフォーマット
  test: |
    res.code == 400 &&
    res.body.error contains "json" &&
    res.body.details != null
```

### 境界値テスト

```yaml
- name: Test Input Boundaries
  uses: http
  with:
    url: "{{vars.API_URL}}/users"
    method: POST
    body: |
      {
        "name": "{{random_str(255)}}",  # 最大長
        "age": 150,                     # 上限境界
        "score": 0                      # 下限境界
      }
  test: |
    res.code in [201, 400] &&  # 作成成功またはバリデーションエラー
    (res.code == 400 ? 
      res.body.validation_errors != null :
      res.body.user.id != null
    )
```

## テスト整理パターン

### レイヤードテスト戦略

```yaml
jobs:
- id: smoke-tests
  name: Smoke Tests
  steps:
    - name: Basic Connectivity
      uses: http
      with:
        method: GET
        url: "{{vars.API_URL}}/ping"
      test: res.code == 200

- id: functional-tests
  name: Functional Tests
  needs: [smoke-tests]
  steps:
    - name: User Management
      uses: http
      with:
        method: GET
        url: "{{vars.API_URL}}/users"
      test: |
        res.code == 200 &&
        res.body.users != null &&
        res.body.pagination != null

- id: integration-tests
  name: Integration Tests
  needs: [functional-tests]
  steps:
    - name: Cross-Service Integration
      uses: http
      with:
        method: GET
        url: "{{vars.API_URL}}/integration/full-flow"
      test: |
        res.code == 200 &&
        res.body.all_services_connected == true &&
        res.body.data_consistency_check == true
```

### 包括的テストスイート

```yaml
jobs:
- id: api-test-suite
  name: Comprehensive API Test Suite
  steps:
    # 認証テスト
    - name: Valid Login
      id: login
      uses: http
      with:
        url: "{{vars.API_URL}}/auth/login"
        method: POST
        body: |
          {
            "username": "{{vars.TEST_USERNAME}}",
            "password": "{{vars.TEST_PASSWORD}}"
          }
      test: |
        res.code == 200 &&
        res.body.access_token != null &&
        res.body.refresh_token != null &&
        res.body.expires_in > 0
      outputs:
        access_token: res.body.access_token

    - name: Invalid Login
      uses: http
      with:
        url: "{{vars.API_URL}}/auth/login"
        method: POST
        body: |
          {
            "username": "invalid",
            "password": "wrong"
          }
      test: |
        res.code == 401 &&
        res.body.error == "invalid_credentials"

    # CRUD 操作テスト
    - name: Create User
      id: create-user
      uses: http
      with:
        url: "{{vars.API_URL}}/users"
        method: POST
        headers:
          Authorization: "Bearer {{outputs.login.access_token}}"
        body: |
          {
            "name": "Test User {{random_str(6)}}",
            "email": "test{{random_str(8)}}@example.com",
            "role": "user"
          }
      test: |
        res.code == 201 &&
        res.body.user.id != null &&
        res.body.user.name != null &&
        res.body.user.email != null
      outputs:
        user_id: res.body.user.id
        user_email: res.body.user.email

    - name: Read User
      uses: http
      with:
        method: GET
        url: "{{vars.API_URL}}/users/{{outputs['create-user'].user_id}}"
        headers:
          Authorization: "Bearer {{outputs.login.access_token}}"
      test: |
        res.code == 200 &&
        res.body.user.id == outputs['create-user'].user_id &&
        res.body.user.email == "{{outputs['create-user'].user_email}}"

    - name: Update User
      uses: http
      with:
        url: "{{vars.API_URL}}/users/{{outputs['create-user'].user_id}}"
        method: PUT
        headers:
          Authorization: "Bearer {{outputs.login.access_token}}"
        body: |
          {
            "name": "Updated Test User"
          }
      test: |
        res.code == 200 &&
        res.body.user.name == "Updated Test User"

    - name: Delete User
      uses: http
      with:
        url: "{{vars.API_URL}}/users/{{outputs['create-user'].user_id}}"
        method: DELETE
        headers:
          Authorization: "Bearer {{outputs.login.access_token}}"
      test: res.code == 204

    - name: Verify Deletion
      uses: http
      with:
        method: GET
        url: "{{vars.API_URL}}/users/{{outputs['create-user'].user_id}}"
        headers:
          Authorization: "Bearer {{outputs.login.access_token}}"
      test: res.code == 404
```

## パフォーマンステスト

### レスポンス時間ベンチマーク

```yaml
- name: Performance Benchmark Test
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/performance-test"
  test: |
    res.code == 200 &&
    
    # 段階的パフォーマンス期待値
    (vars.NODE_ENV == "production" ? 
      (rt.sec * 1000) < 500 :              # プロダクション: < 500ms
      (rt.sec * 1000) < 2000               # 非プロダクション: < 2s
    ) &&
    
    # 追加パフォーマンスメトリクス
    res.body.query_time < 100 &&    # データベースクエリ時間
    res.body.render_time < 50       # テンプレートレンダー時間
  outputs:
    response_time: (rt.sec * 1000)
    query_time: res.body.query_time
    render_time: res.body.render_time
```

### 負荷テスト検証

```yaml
- name: Load Test Results Validation
  uses: http
  with:
    method: GET
    url: "{{vars.LOAD_TEST_URL}}/results"
  test: |
    res.code == 200 &&
    res.body.test_completed == true &&
    
    # 成功率要件
    res.body.success_rate >= 0.95 &&
    
    # パフォーマンスパーセンタイル
    res.body.percentiles.p50 < 1000 &&
    res.body.percentiles.p95 < 2000 &&
    res.body.percentiles.p99 < 5000 &&
    
    # エラー率制限
    res.body.error_rate < 0.05 &&
    
    # 重大なエラーがない
    res.body.critical_errors == 0
```

## セキュリティテスト

### 認証と承認

```yaml
- name: Test Unauthorized Access
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/admin/users"
  test: |
    res.code == 401 &&
    res.body.error == "authentication_required"

- name: Test Insufficient Permissions
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/admin/users"
    headers:
      Authorization: "Bearer {{vars.USER_TOKEN}}"  # 通常ユーザートークン
  test: |
    res.code == 403 &&
    res.body.error == "insufficient_permissions"

- name: Test Token Expiration
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/protected"
    headers:
      Authorization: "Bearer {{vars.EXPIRED_TOKEN}}"
  test: |
    res.code == 401 &&
    res.body.error == "token_expired"
```

### 入力検証セキュリティ

```yaml
- name: Test SQL Injection Protection
  uses: http
  with:
    method: GET
    url: "{{vars.API_URL}}/users?search='; DROP TABLE users; --"
  test: |
    res.code in [200, 400] &&  # フィルタされるか拒否される
    !res.body contains "sql" && # SQL エラーメッセージなし
    !res.body contains "syntax" &&
    res.body.error != "internal_server_error"  # サーバーエラーを起こさない

- name: Test XSS Protection
  uses: http
  with:
    url: "{{vars.API_URL}}/comments"
    method: POST
    body: |
      {
        "content": "<script>alert('xss')</script>"
      }
  test: |
    res.code in [201, 400] &&
    (res.code == 201 ? 
      !res.body.comment.content contains "<script>" :  # サニタイズされるべき
      res.body.validation_errors != null               # または拒否される
    )
```

## テストドキュメントとレポート

### 自己文書化テスト

```yaml
- name: User Registration Flow Test
  uses: http
  with:
    url: "{{vars.API_URL}}/auth/register"
    method: POST
    body: |
      {
        "email": "{{random_str(8)}}@example.com",
        "password": "TestPass123!",
        "confirm_password": "TestPass123!"
      }
  # 明確な検証ポイント付き包括的テスト
  test: |
    res.code == 201 &&                                    # 1. 作成成功
    res.body.user.id != null &&                            # 2. ユーザーID割り当て
    res.body.user.email != null &&                         # 3. メール保存
    res.body.user.password == null &&                      # 4. パスワード未返却
    res.body.user.created_at != null &&                    # 5. タイムスタンプ記録
    res.body.user.email_verified == false &&               # 6. 初期未認証メール
    res.body.verification_email_sent == true &&            # 7. 認証トリガー
    "Location" in res.headers &&                         # 8. Location ヘッダー存在
    res.headers["Location"] contains "/users/"             # 9. 正しいリダイレクトパス
  outputs:
    user_id: res.body.user.id
    user_email: res.body.user.email
    test_summary: |
      Registration test completed:
      - User ID: {{res.body.user.id}}
      - Email: {{res.body.user.email}}
      - Verification: {{res.body.verification_email_sent ? "Sent" : "Failed"}}
      - Response time: {{rt.duration}}
```

### テスト結果集約

```yaml
jobs:
- id: test-summary
  name: Test Results Summary
  needs: [smoke-tests, functional-tests, security-tests]
  steps:
    - name: Generate Test Report
      uses: hello
      echo: |
        Test Execution Summary
        =====================
          
          
        Overall Result: {{
        }}
          
        Execution Time: {{unixtime()}}
        Test Environment: {{vars.NODE_ENV || "development"}}
```

## ベストプラクティス

### 1. 明確なテスト意図

```yaml
# 良い例: 具体的でテスト可能な条件
test: |
  res.code == 200 &&
  len(res.body.users) >= 1 &&
  res.body.users[0].id != null

# 避ける: 曖昧または不完全なテスト
test: res.code == 200  # レスポンス内容はどうなのか？
```

### 2. 包括的エラーカバレッジ

```yaml
# 良い例: 成功と失敗の両方のパスをテスト
- name: Valid Request Test
  test: res.code == 200 && res.body.success == true

- name: Invalid Request Test  
  test: res.code == 400 && res.body.error != null
```

### 3. パフォーマンスを意識したテスト

```yaml
# 良い例: パフォーマンス検証を含む
test: |
  res.code == 200 &&
  (rt.sec * 1000) < 1000 &&
  res.body.data != null

# 良い例: 環境固有のパフォーマンス閾値
test: |
  res.code == 200 &&
  (rt.sec * 1000) < {{vars.MAX_RESPONSE_TIME || 2000}}
```

### 4. 保守しやすいテスト式

```yaml
# 良い例: 読みやすく、よく構造化されたテスト
test: |
  res.code == 200 &&
  res.body.user != null &&
  res.body.user.id > 0 &&
  res.body.user.email contains "@"

# 避ける: 複雑で読みにくいテスト
test: res.code == 200 && res.body.user != null && res.body.user.id > 0 && res.body.user.email contains "@" && res.body.user.active == true && (rt.sec * 1000) < 1000
```

## 次のステップ

テストとアサーションを理解したら、以下を探索してください：

1. **[エラーハンドリング](/ja/guide/concepts/error-handling)** - 失敗を適切に処理する方法を学ぶ
2. **[実行モデル](/ja/guide/concepts/execution-model)** - ワークフロー実行フローを理解する
3. **[ハウツー](/ja/guide/how-tos/api-testing)** - 実用的なテストパターンの実例を見る

テストとアサーションはあなたの品質ゲートです。これらの概念をマスターして、問題がシステムに影響を与える前に捉える信頼性の高い堅牢な自動化を構築しましょう。
