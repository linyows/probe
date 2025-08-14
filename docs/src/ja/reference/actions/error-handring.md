# アクションエラーハンドリング

## 一般的なエラーシナリオ

すべてのアクションはさまざまな理由で失敗する可能性があります。一般的な失敗モードを理解することで、堅牢なワークフローの作成に役立ちます。

### HTTPアクションエラー

```yaml
steps:
  - name: "HTTP with Error Handling"
    uses: http
    with:
      url: "https://api.example.com/endpoint"
    test: |
      res.code >= 200 && res.code < 300
    continue_on_error: false
    outputs:
      success: res.code >= 200 && res.code < 300
      error_message: |
        {{res.code >= 400 ? "Client error: " + res.code : 
          res.code >= 500 ? "Server error: " + res.code : ""}}
```

### SMTPアクションエラー

```yaml
vars:
  smtp_user: "{{SMTP_USER}}"
  smtp_pass: "{{SMTP_PASS}}"

steps:
  - name: "SMTP with Error Handling"
    uses: smtp
    with:
      host: "smtp.example.com"
      username: "{{vars.smtp_user}}"
      password: "{{vars.smtp_pass}}"
      from: "test@example.com"
      to: ["admin@example.com"]
      subject: "Test"
      body: "Test message"
    test: res.success == true
    continue_on_error: true
    outputs:
      email_sent: res.success
      send_time: res.time
```

## パフォーマンスの考慮事項

### HTTPアクションパフォーマンス

- **コネクションプーリング:** HTTPアクションは可能な場合接続を再利用
- **タイムアウト:** ハングを防ぐために適切なタイムアウトを設定
- **レスポンスサイズ:** 大きなレスポンスはより多くのメモリを消費
- **同時リクエスト:** 複数のHTTPアクションは並列実行可能

```yaml
# パフォーマンス最適化されたHTTP設定
vars:
  api_url: "{{API_URL}}"

jobs:
- name: performance-test
  defaults:
    http:
      timeout: "10s"
      follow_redirects: true
      max_redirects: 3
  steps:
    - name: "Quick Health Check"
      uses: http
      with:
        url: "{{vars.api_url}}/ping"
        timeout: "2s"
      test: res.code == 200 && res.time < 500
```

### SMTPアクションパフォーマンス

- **接続再利用:** SMTP接続はアクションごとに確立
- **バッチメール:** 接続を減らすために受信者をグループ化することを検討
- **TLSオーバーヘッド:** TLSネゴシエーションは遅延を追加

```yaml
# 効率的なメール通知
vars:
  smtp_user: "{{SMTP_USER}}"
  smtp_pass: "{{SMTP_PASS}}"

steps:
  - name: "Batch Notification"
    uses: smtp
    with:
      host: "smtp.example.com"
      username: "{{vars.smtp_user}}"
      password: "{{vars.smtp_pass}}"
      from: "alerts@example.com"
      to: ["admin1@example.com", "admin2@example.com", "admin3@example.com"]
      subject: "Batch Alert"
      body: "Single email to multiple recipients"
```

## 関連項目

- **[YAML設定](../yaml-configuration/)** - 完全なYAML構文リファレンス
- **[組み込み関数](../built-in-functions/)** - アクションで使用する式関数
- **[概念: アクション](../../concepts/actions/)** - アクションシステムアーキテクチャ
- **[ハウツー: APIテスト](../../how-tos/api-testing/)** - 実用的なHTTPアクション例
- **[ハウツー: エラーハンドリング](../../how-tos/error-handling-strategies/)** - エラーハンドリングパターン
