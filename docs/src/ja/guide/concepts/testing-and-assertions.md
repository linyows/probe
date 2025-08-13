# テストとアサーション

テストとアサーションは Probe ワークフローの品質ゲートです。アクションが期待される結果を生み出すことを検証し、システムの信頼性を確保します。このガイドではテスト式、アサーションパターン、ワークフローに堅牢な検証を組み込むための戦略について説明します。

## テストの基礎

Probe のすべてのステップには、アクションの結果を検証する `test` 条件を含めることができます。テストはステップの成功または失敗を決定するブール式です。

### 基本テスト構造

```yaml
- name: API Health Check
  uses: http
  with:
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
  res.time < 1000 &&             # レスポンス時間（ミリ秒）
  res.body_size > 0 &&           # レスポンスボディサイズ（バイト）
  res.headers["content-type"] == "application/json" &&  # レスポンスヘッダー
  res.body.json.status == "healthy" && # 解析された JSON レスポンス
  res.body.contains("success")   # テキストとしてのレスポンスボディ
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
test: res.time < 1000                    # 1秒以内にレスポンス必要
test: res.time >= 100 && res.time <= 500  # レスポンス時間範囲
test: res.time < {{vars.MAX_RESPONSE_TIME || 2000}}  # 設定可能な閾値

# パフォーマンスカテゴリ
test: |
  res.code == 200 && (
    res.time < 200 ? "excellent" :
    res.time < 500 ? "good" :
    res.time < 1000 ? "acceptable" : "poor"
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
test: res.headers["content-type"] == "application/json"
test: res.headers["content-type"].startsWith("text/")
test: res.headers["content-type"].contains("charset=utf-8")

# セキュリティヘッダー
test: |
  res.headers.has("x-frame-options") &&
  res.headers.has("x-content-type-options") &&
  res.headers["x-frame-options"] == "DENY"

# キャッシュ制御
test: res.headers["cache-control"].contains("no-cache")

# レート制限
test: res.headers["x-rate-limit-remaining"] > "10"

# カスタムヘッダー
test: |
  res.headers.has("x-request-id") &&
  res.headers["x-request-id"].length == 36  # UUID形式
```

## JSON レスポンステスト

### 基本 JSON 検証

```yaml
# JSON 構造検証
test: |
  res.code == 200 &&
  res.body.json != null &&
  res.body.json.status == "success" &&
  res.body.json.data != null

# 必須フィールド存在
test: |
  res.body.json.has("id") &&
  res.body.json.has("name") &&
  res.body.json.has("email") &&
  res.body.json.has("created_at")
```

### データタイプ検証

```yaml
# タイプチェック
test: |
  typeof(res.body.json.id) == "number" &&
  typeof(res.body.json.name) == "string" &&
  typeof(res.body.json.active) == "boolean" &&
  typeof(res.body.json.tags) == "array" &&
  typeof(res.body.json.metadata) == "object"

# 値制約
test: |
  res.body.json.id > 0 &&
  res.body.json.name.length >= 2 &&
  res.body.json.score >= 0 && res.body.json.score <= 100
```

### 配列とコレクションテスト

```yaml
# 配列検証
test: |
  res.body.json.users != null &&
  res.body.json.users.length > 0 &&
  res.body.json.users.length <= 100

# 配列コンテンツ検証
test: |
  res.body.json.users.all(user -> 
    user.id != null && 
    user.email != null
  )

# 特定要素チェック
test: |
  res.body.json.users.any(user -> user.role == "admin") &&
  res.body.json.users.filter(user -> user.active == true).length > 0

# 配列ユニーク性
test: |
  res.body.json.user_ids.length == res.body.json.user_ids.unique().length
```

### ネストしたデータ検証

```yaml
# 深いオブジェクト検証
test: |
  res.body.json.user != null &&
  res.body.json.user.profile != null &&
  res.body.json.user.profile.preferences != null &&
  res.body.json.user.profile.preferences.notifications == true

# 複雑なネスト構造
test: |
  res.body.json.data.orders.all(order ->
    order.id != null &&
    order.items.length > 0 &&
    order.items.all(item -> 
      item.product_id != null && 
      item.quantity > 0 && 
      item.price > 0
    ) &&
    order.total == order.items.map(item -> item.quantity * item.price).sum()
  )
```

## テキストレスポンステスト

### パターンマッチング

```yaml
# シンプルなテキストマッチング
test: res.body.contains("success")
test: res.body.startsWith("<!DOCTYPE html>")
test: res.body.endsWith("</html>")

# 大文字小文字を区別しないマッチング
test: res.body.lower().contains("error")

# 複数パターン
test: |
  res.body.contains("status") &&
  res.body.contains("healthy") &&
  !res.body.contains("error")
```

### 正規表現テスト

```yaml
# レスポンス内のメール検証
test: res.body.matches("user-\\d+@example\\.com")

# URL パターン検証
test: res.body.matches("https://[a-zA-Z0-9.-]+/api/v\\d+/")

# データフォーマット検証
test: |
  res.body.matches("\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z")  # ISO タイムスタンプ

# データ抽出と検証
test: |
  res.body.matches("Version: v\\d+\\.\\d+\\.\\d+") &&
  res.body.extract("v(\\d+)\\.(\\d+)\\.(\\d+)")[1] >= "2"  # メジャーバージョン >= 2
```

### コンテンツ長と品質

```yaml
# コンテンツ長検証
test: |
  res.body.length > 100 &&
  res.body.length < 10000

# コンテンツ品質チェック
test: |
  res.body.split("\\n").length > 5 &&           # 複数行コンテンツ
  !res.body.contains("Lorem ipsum") &&          # プレースホルダーテキストでない
  res.body.split(" ").length > 20               # 実質的なコンテンツ
```

## 高度なテストパターン

### 条件付きテスト

```yaml
# 環境固有テスト
test: |
  res.code == 200 &&
  (vars.NODE_ENV == "development" ? 
    res.time < 5000 :           # 開発環境では緩い設定
    res.time < 1000             # プロダクション用は厳格
  )

# 機能フラグテスト
test: |
  res.code == 200 &&
  (res.body.json.features.beta_enabled == true ?
    res.body.json.beta_data != null :    # ベータ機能にはデータが必要
    res.body.json.beta_data == null      # ベータ機能は存在しないべき
  )
```

### ステップ間検証

```yaml
jobs:
  data-consistency-test:
    steps:
      - name: Get User Count
        id: user-count
        uses: http
        with:
          url: "{{vars.API_URL}}/users/count"
        test: res.code == 200
        outputs:
          total_users: res.body.json.count

      - name: Get User List
        id: user-list
        uses: http
        with:
          url: "{{vars.API_URL}}/users"
        test: |
          res.code == 200 &&
          res.body.json.users.length == outputs.user-count.total_users  # 整合性チェック
        outputs:
          user_list: res.body.json.users

      - name: Validate User Data Integrity
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.user-list.user_list[0].id}}"
        test: |
          res.code == 200 &&
          res.body.json.user.id == outputs.user-list.user_list[0].id &&
          res.body.json.user.email == outputs.user-list.user_list[0].email
```

### ビジネスロジックテスト

```yaml
- name: E-commerce Business Logic Test
  uses: http
  with:
    url: "{{vars.API_URL}}/orders/{{vars.TEST_ORDER_ID}}"
  test: |
    res.code == 200 &&
    res.body.json.order != null &&
    
    # 注文合計が品目の合計と等しい
    res.body.json.order.total == 
      res.body.json.order.line_items.map(item -> item.quantity * item.price).sum() &&
    
    # 税計算が正しい（8%税率を想定）
    res.body.json.order.tax_amount == 
      Math.round(res.body.json.order.subtotal * 0.08 * 100) / 100 &&
    
    # 送料が正しく適用される
    (res.body.json.order.subtotal >= 100 ? 
      res.body.json.order.shipping_cost == 0 :     # $100以上は送料無料
      res.body.json.order.shipping_cost == 9.99    # 標準送料
    ) &&
    
    # 最終合計計算
    res.body.json.order.total == 
      res.body.json.order.subtotal + res.body.json.order.tax_amount + res.body.json.order.shipping_cost
```

## エラーテストと負のケース

### 期待されるエラーシナリオ

```yaml
- name: Test Invalid Authentication
  uses: http
  with:
    url: "{{vars.API_URL}}/protected"
    headers:
      Authorization: "Bearer invalid-token"
  test: |
    res.code == 401 &&
    res.body.json.error == "invalid_token" &&
    res.body.json.message.contains("authentication")

- name: Test Rate Limiting
  uses: http
  with:
    url: "{{vars.API_URL}}/rate-limited-endpoint"
  test: |
    res.code in [200, 429] &&  # 成功またはレート制限
    (res.code == 429 ? 
      res.headers.has("retry-after") && 
      res.body.json.error == "rate_limit_exceeded" :
      res.body.json.status == "success"
    )

- name: Test Malformed Request
  uses: http
  with:
    url: "{{vars.API_URL}}/users"
    method: POST
    body: '{"invalid": json}'  # 意図的に不正なフォーマット
  test: |
    res.code == 400 &&
    res.body.json.error.contains("json") &&
    res.body.json.details != null
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
      res.body.json.validation_errors != null :
      res.body.json.user.id != null
    )
```

## テスト整理パターン

### レイヤードテスト戦略

```yaml
jobs:
  smoke-tests:
    name: Smoke Tests
    steps:
      - name: Basic Connectivity
        uses: http
        with:
          url: "{{vars.API_URL}}/ping"
        test: res.code == 200

  functional-tests:
    name: Functional Tests
    needs: [smoke-tests]
    steps:
      - name: User Management
        uses: http
        with:
          url: "{{vars.API_URL}}/users"
        test: |
          res.code == 200 &&
          res.body.json.users != null &&
          res.body.json.pagination != null

  integration-tests:
    name: Integration Tests
    needs: [functional-tests]
    steps:
      - name: Cross-Service Integration
        uses: http
        with:
          url: "{{vars.API_URL}}/integration/full-flow"
        test: |
          res.code == 200 &&
          res.body.json.all_services_connected == true &&
          res.body.json.data_consistency_check == true
```

### 包括的テストスイート

```yaml
jobs:
  api-test-suite:
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
          res.body.json.access_token != null &&
          res.body.json.refresh_token != null &&
          res.body.json.expires_in > 0
        outputs:
          access_token: res.body.json.access_token

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
          res.body.json.error == "invalid_credentials"

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
          res.body.json.user.id != null &&
          res.body.json.user.name != null &&
          res.body.json.user.email != null
        outputs:
          user_id: res.body.json.user.id
          user_email: res.body.json.user.email

      - name: Read User
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.create-user.user_id}}"
          headers:
            Authorization: "Bearer {{outputs.login.access_token}}"
        test: |
          res.code == 200 &&
          res.body.json.user.id == outputs.create-user.user_id &&
          res.body.json.user.email == "{{outputs.create-user.user_email}}"

      - name: Update User
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.create-user.user_id}}"
          method: PUT
          headers:
            Authorization: "Bearer {{outputs.login.access_token}}"
          body: |
            {
              "name": "Updated Test User"
            }
        test: |
          res.code == 200 &&
          res.body.json.user.name == "Updated Test User"

      - name: Delete User
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.create-user.user_id}}"
          method: DELETE
          headers:
            Authorization: "Bearer {{outputs.login.access_token}}"
        test: res.code == 204

      - name: Verify Deletion
        uses: http
        with:
          url: "{{vars.API_URL}}/users/{{outputs.create-user.user_id}}"
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
    url: "{{vars.API_URL}}/performance-test"
  test: |
    res.code == 200 &&
    
    # 段階的パフォーマンス期待値
    (vars.NODE_ENV == "production" ? 
      res.time < 500 :              # プロダクション: < 500ms
      res.time < 2000               # 非プロダクション: < 2s
    ) &&
    
    # 追加パフォーマンスメトリクス
    res.body.json.query_time < 100 &&    # データベースクエリ時間
    res.body.json.render_time < 50       # テンプレートレンダー時間
  outputs:
    response_time: res.time
    query_time: res.body.json.query_time
    render_time: res.body.json.render_time
```

### 負荷テスト検証

```yaml
- name: Load Test Results Validation
  uses: http
  with:
    url: "{{vars.LOAD_TEST_URL}}/results"
  test: |
    res.code == 200 &&
    res.body.json.test_completed == true &&
    
    # 成功率要件
    res.body.json.success_rate >= 0.95 &&
    
    # パフォーマンスパーセンタイル
    res.body.json.percentiles.p50 < 1000 &&
    res.body.json.percentiles.p95 < 2000 &&
    res.body.json.percentiles.p99 < 5000 &&
    
    # エラー率制限
    res.body.json.error_rate < 0.05 &&
    
    # 重大なエラーがない
    res.body.json.critical_errors == 0
```

## セキュリティテスト

### 認証と承認

```yaml
- name: Test Unauthorized Access
  uses: http
  with:
    url: "{{vars.API_URL}}/admin/users"
  test: |
    res.code == 401 &&
    res.body.json.error == "authentication_required"

- name: Test Insufficient Permissions
  uses: http
  with:
    url: "{{vars.API_URL}}/admin/users"
    headers:
      Authorization: "Bearer {{vars.USER_TOKEN}}"  # 通常ユーザートークン
  test: |
    res.code == 403 &&
    res.body.json.error == "insufficient_permissions"

- name: Test Token Expiration
  uses: http
  with:
    url: "{{vars.API_URL}}/protected"
    headers:
      Authorization: "Bearer {{vars.EXPIRED_TOKEN}}"
  test: |
    res.code == 401 &&
    res.body.json.error == "token_expired"
```

### 入力検証セキュリティ

```yaml
- name: Test SQL Injection Protection
  uses: http
  with:
    url: "{{vars.API_URL}}/users?search='; DROP TABLE users; --"
  test: |
    res.code in [200, 400] &&  # フィルタされるか拒否される
    !res.body.contains("sql") && # SQL エラーメッセージなし
    !res.body.contains("syntax") &&
    res.body.json.error != "internal_server_error"  # サーバーエラーを起こさない

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
      !res.body.json.comment.content.contains("<script>") :  # サニタイズされるべき
      res.body.json.validation_errors != null               # または拒否される
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
    res.body.json.user.id != null &&                            # 2. ユーザーID割り当て
    res.body.json.user.email != null &&                         # 3. メール保存
    res.body.json.user.password == null &&                      # 4. パスワード未返却
    res.body.json.user.created_at != null &&                    # 5. タイムスタンプ記録
    res.body.json.user.email_verified == false &&               # 6. 初期未認証メール
    res.body.json.verification_email_sent == true &&            # 7. 認証トリガー
    res.headers.has("location") &&                         # 8. Location ヘッダー存在
    res.headers["location"].contains("/users/")             # 9. 正しいリダイレクトパス
  outputs:
    user_id: res.body.json.user.id
    user_email: res.body.json.user.email
    test_summary: |
      Registration test completed:
      - User ID: {{res.body.json.user.id}}
      - Email: {{res.body.json.user.email}}
      - Verification: {{res.body.json.verification_email_sent ? "Sent" : "Failed"}}
      - Response time: {{res.time}}ms
```

### テスト結果集約

```yaml
jobs:
  test-summary:
    name: Test Results Summary
    needs: [smoke-tests, functional-tests, security-tests]
    steps:
      - name: Generate Test Report
        echo: |
          Test Execution Summary
          =====================
          
          Smoke Tests: {{jobs.smoke-tests.success ? "✅ PASSED" : "❌ FAILED"}}
          Functional Tests: {{jobs.functional-tests.success ? "✅ PASSED" : "❌ FAILED"}}
          Security Tests: {{jobs.security-tests.success ? "✅ PASSED" : "❌ FAILED"}}
          
          Overall Result: {{
            jobs.smoke-tests.success && 
            jobs.functional-tests.success && 
            jobs.security-tests.success ? "✅ ALL TESTS PASSED" : "❌ SOME TESTS FAILED"
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
  res.body.json.users.length >= 1 &&
  res.body.json.users[0].id != null

# 避ける: 曖昧または不完全なテスト
test: res.code == 200  # レスポンス内容はどうなのか？
```

### 2. 包括的エラーカバレッジ

```yaml
# 良い例: 成功と失敗の両方のパスをテスト
- name: Valid Request Test
  test: res.code == 200 && res.body.json.success == true

- name: Invalid Request Test  
  test: res.code == 400 && res.body.json.error != null
```

### 3. パフォーマンスを意識したテスト

```yaml
# 良い例: パフォーマンス検証を含む
test: |
  res.code == 200 &&
  res.time < 1000 &&
  res.body.json.data != null

# 良い例: 環境固有のパフォーマンス閾値
test: |
  res.code == 200 &&
  res.time < {{vars.MAX_RESPONSE_TIME || 2000}}
```

### 4. 保守しやすいテスト式

```yaml
# 良い例: 読みやすく、よく構造化されたテスト
test: |
  res.code == 200 &&
  res.body.json.user != null &&
  res.body.json.user.id > 0 &&
  res.body.json.user.email.contains("@")

# 避ける: 複雑で読みにくいテスト
test: res.code == 200 && res.body.json.user != null && res.body.json.user.id > 0 && res.body.json.user.email.contains("@") && res.body.json.user.active == true && res.time < 1000
```

## 次のステップ

テストとアサーションを理解したら、以下を探索してください：

1. **[エラーハンドリング](../error-handling/)** - 失敗を適切に処理する方法を学ぶ
2. **[実行モデル](../execution-model/)** - ワークフロー実行フローを理解する
3. **[ハウツー](../../how-tos/)** - 実用的なテストパターンの実例を見る

テストとアサーションはあなたの品質ゲートです。これらの概念をマスターして、問題がシステムに影響を与える前に捉える信頼性の高い堅牢な自動化を構築しましょう。