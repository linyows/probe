---
title: 最初のワークフロー
description: 実用的な例で実践的なワークフローの構築を学ぶ
weight: 40
---

# 最初のワークフロー

[核となる概念](../understanding-probe/)を理解したので、実践的なワークフローをゼロから構築しましょう。このガイドでは、ウェブアプリケーション用の包括的な監視ワークフローの作成を説明します。

## シナリオ

完全なウェブアプリケーションスタックを監視するワークフローを作成します：

1. **フロントエンド**: ウェブアプリケーションが正しく読み込まれるかチェック
2. **API**: REST APIが応答しているか確認
3. **データベース**: API経由でデータベース接続をテスト
4. **外部サービス**: サードパーティサービス統合をチェック

## ステップ1: 基本構造

基本的なワークフロー構造から始めましょう：

```yaml
name: Web Application Health Check
description: Comprehensive monitoring for our web application stack

jobs:
  # ここにジョブを追加します
```

## ステップ2: フロントエンド監視

フロントエンドアプリケーションをチェックするジョブを追加：

```yaml
name: Web Application Health Check
description: Comprehensive monitoring for our web application stack

jobs:
  frontend-check:
    name: Frontend Application Check
    steps:
      - name: Check Homepage
        action: http
        with:
          url: https://myapp.example.com
          method: GET
          headers:
            User-Agent: "Probe Health Check"
        test: res.status == 200 && res.time < 3000
        outputs:
          homepage_response_time: res.time

      - name: Check Critical Page
        action: http
        with:
          url: https://myapp.example.com/dashboard
          method: GET
        test: res.status == 200 || res.status == 302

      - name: Report Frontend Status
        echo: "✅ Frontend is healthy ({{outputs.homepage_response_time}}ms)"
```

## ステップ3: API監視

API監視用の別のジョブを追加：

```yaml
  api-check:
    name: API Health Check
    steps:
      - name: Check API Health Endpoint
        id: health-check
        action: http
        with:
          url: https://api.myapp.example.com/health
          method: GET
          headers:
            Accept: "application/json"
        test: res.status == 200 && res.json.status == "healthy"
        outputs:
          api_version: res.json.version
          database_status: res.json.database

      - name: Test User Authentication
        action: http
        with:
          url: https://api.myapp.example.com/auth/login
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "username": "healthcheck",
              "password": "{{env.HEALTH_CHECK_PASSWORD}}"
            }
        test: res.status == 200 && res.json.token != null
        outputs:
          auth_token: res.json.token

      - name: Test Authenticated Endpoint
        action: http
        with:
          url: https://api.myapp.example.com/user/profile
          method: GET
          headers:
            Authorization: "Bearer {{outputs.auth_token}}"
        test: res.status == 200

      - name: Report API Status
        echo: "✅ API v{{outputs.api_version}} is healthy"
```

## ステップ4: 依存関係の追加

APIチェックを成功したフロントエンドチェックに依存させる：

```yaml
  api-check:
    name: API Health Check
    needs: [frontend-check]  # フロントエンドが正常になるまで待機
    steps:
      # ... 既存のステップ
```

## ステップ5: 外部サービスチェック

外部サービス用のチェックを追加：

```yaml
  external-services:
    name: External Services Check
    steps:
      - name: Check Email Service
        action: http
        with:
          url: https://api.sendgrid.com/v3/mail/send
          method: POST
          headers:
            Authorization: "Bearer {{env.SENDGRID_API_KEY}}"
            Content-Type: "application/json"
          body: |
            {
              "from": {"email": "health@myapp.example.com"},
              "subject": "Health Check Test",
              "content": [{"type": "text/plain", "value": "Test"}],
              "personalizations": [{"to": [{"email": "test@myapp.example.com"}]}]
            }
        test: res.status == 202

      - name: Check Payment Gateway
        action: http
        with:
          url: https://api.stripe.com/v1/charges
          method: GET
          headers:
            Authorization: "Bearer {{env.STRIPE_SECRET_KEY}}"
        test: res.status == 200

      - name: Report External Services
        echo: "✅ All external services are responding"
```

## ステップ6: エラーハンドリングと通知

エラーハンドリングと通知ロジックを追加：

```yaml
  notification:
    name: Send Notifications
    needs: [frontend-check, api-check, external-services]
    steps:
      - name: Success Notification
        if: jobs.frontend-check.success && jobs.api-check.success && jobs.external-services.success
        echo: |
          🎉 All systems are healthy!
          
          Frontend: ✅ ({{outputs.frontend-check.homepage_response_time}}ms)
          API: ✅ v{{outputs.api-check.api_version}}
          External Services: ✅
          
          Monitoring completed at {{unixtime()}}

      - name: Failure Notification
        if: jobs.frontend-check.failed || jobs.api-check.failed || jobs.external-services.failed
        echo: |
          🚨 ALERT: System health check failed!
          
          Frontend: {{jobs.frontend-check.success ? "✅" : "❌"}}
          API: {{jobs.api-check.success ? "✅" : "❌"}}
          External Services: {{jobs.external-services.success ? "✅" : "❌"}}
          
          Please investigate immediately.
```

## 完全なワークフロー

完全なワークフローファイル（`health-check.yml`）は以下の通りです：

```yaml
name: Web Application Health Check
description: Comprehensive monitoring for our web application stack

jobs:
  frontend-check:
    name: Frontend Application Check
    steps:
      - name: Check Homepage
        action: http
        with:
          url: https://myapp.example.com
          method: GET
          headers:
            User-Agent: "Probe Health Check"
        test: res.status == 200 && res.time < 3000
        outputs:
          homepage_response_time: res.time

      - name: Check Critical Page
        action: http
        with:
          url: https://myapp.example.com/dashboard
          method: GET
        test: res.status == 200 || res.status == 302

      - name: Report Frontend Status
        echo: "✅ Frontend is healthy ({{outputs.homepage_response_time}}ms)"

  api-check:
    name: API Health Check
    needs: [frontend-check]
    steps:
      - name: Check API Health Endpoint
        id: health-check
        action: http
        with:
          url: https://api.myapp.example.com/health
          method: GET
          headers:
            Accept: "application/json"
        test: res.status == 200 && res.json.status == "healthy"
        outputs:
          api_version: res.json.version
          database_status: res.json.database

      - name: Test User Authentication
        action: http
        with:
          url: https://api.myapp.example.com/auth/login
          method: POST
          headers:
            Content-Type: "application/json"
          body: |
            {
              "username": "healthcheck",
              "password": "{{env.HEALTH_CHECK_PASSWORD}}"
            }
        test: res.status == 200 && res.json.token != null
        outputs:
          auth_token: res.json.token

      - name: Test Authenticated Endpoint
        action: http
        with:
          url: https://api.myapp.example.com/user/profile
          method: GET
          headers:
            Authorization: "Bearer {{outputs.auth_token}}"
        test: res.status == 200

      - name: Report API Status
        echo: "✅ API v{{outputs.api_version}} is healthy"

  external-services:
    name: External Services Check
    steps:
      - name: Check Email Service
        action: http
        with:
          url: https://api.sendgrid.com/v3/mail/send
          method: POST
          headers:
            Authorization: "Bearer {{env.SENDGRID_API_KEY}}"
            Content-Type: "application/json"
          body: |
            {
              "from": {"email": "health@myapp.example.com"},
              "subject": "Health Check Test",
              "content": [{"type": "text/plain", "value": "Test"}],
              "personalizations": [{"to": [{"email": "test@myapp.example.com"}]}]
            }
        test: res.status == 202

      - name: Check Payment Gateway
        action: http
        with:
          url: https://api.stripe.com/v1/charges
          method: GET
          headers:
            Authorization: "Bearer {{env.STRIPE_SECRET_KEY}}"
        test: res.status == 200

      - name: Report External Services
        echo: "✅ All external services are responding"

  notification:
    name: Send Notifications
    needs: [frontend-check, api-check, external-services]
    steps:
      - name: Success Notification
        if: jobs.frontend-check.success && jobs.api-check.success && jobs.external-services.success
        echo: |
          🎉 All systems are healthy!
          
          Frontend: ✅ ({{outputs.frontend-check.homepage_response_time}}ms)
          API: ✅ v{{outputs.api-check.api_version}}
          External Services: ✅
          
          Monitoring completed at {{unixtime()}}

      - name: Failure Notification
        if: jobs.frontend-check.failed || jobs.api-check.failed || jobs.external-services.failed
        echo: |
          🚨 ALERT: System health check failed!
          
          Frontend: {{jobs.frontend-check.success ? "✅" : "❌"}}
          API: {{jobs.api-check.success ? "✅" : "❌"}}
          External Services: {{jobs.external-services.success ? "✅" : "❌"}}
          
          Please investigate immediately.
```

## ワークフローの実行

### 環境変数の設定

まず、環境変数を設定します：

```bash
export HEALTH_CHECK_PASSWORD="your-test-password"
export SENDGRID_API_KEY="your-sendgrid-key"
export STRIPE_SECRET_KEY="your-stripe-key"
```

### ワークフローの実行

ワークフローを実行：

```bash
probe health-check.yml
```

### デバッグ用の詳細モードを使用

開発中の詳細出力には：

```bash
probe -v health-check.yml
```

## 本番環境対応にする

### 1. 環境固有の設定

環境固有の設定ファイルを作成：

**production.yml:**
```yaml
# 本番環境用のURLをオーバーライド
variables:
  frontend_url: https://app.mycompany.com
  api_url: https://api.mycompany.com
```

**staging.yml:**
```yaml
# ステージング環境用のURLをオーバーライド
variables:
  frontend_url: https://staging.mycompany.com
  api_url: https://api-staging.mycompany.com
```

環境固有の設定で実行：
```bash
probe health-check.yml,production.yml
```

### 2. リトライロジックの追加

```yaml
- name: Check Critical Service
  action: http
  with:
    url: https://critical-service.example.com
    method: GET
    retry_count: 3
    retry_delay: 5s
  test: res.status == 200
```

### 3. 監視スケジュールの設定

定期的に実行するためにcronを使用：
```bash
# crontabに追加 - 5分ごとに実行
*/5 * * * * /usr/local/bin/probe /path/to/health-check.yml
```

## 学んだこと

このガイドでは、以下の方法を学びました：

- ✅ マルチジョブワークフローの構造化
- ✅ `needs`でジョブ依存関係を使用
- ✅ `outputs`を使用してステップ間でデータを渡す
- ✅ API呼び出しで認証を処理
- ✅ `if`で条件ロジックを実装
- ✅ 設定のために環境変数を使用
- ✅ 包括的なエラーハンドリングを作成
- ✅ 異なる環境用に設定ファイルをマージ

## 次のステップ

さらに深く学ぶ準備はできましたか？次のステップは：

1. **[CLIをマスター](../cli-basics/)** - すべてのコマンドライン・オプションを学ぶ
2. **[How-tosを探る](../../how-tos/)** - 具体的な使用例とパターンを見る
3. **[リファレンスを参照](../../reference/)** - 利用可能なすべての機能を深く理解

構築したワークフローは堅実な基盤です。より多くのチェックを追加したり、監視システムと統合したり、特定のアプリケーションスタック用にカスタマイズしたりして拡張できます。