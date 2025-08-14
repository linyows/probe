# 環境変数リファレンス

このページでは、Probeの動作、設定、実行を制御するすべての環境変数の包括的なドキュメントを提供します。

## 概要

Probeは以下のために環境変数を使用します：

- **ランタイム設定** - ログ、タイムアウト、動作の制御
- **認証** - APIキー、トークン、認証情報
- **統合** - CI/CDシステム、監視ツール
- **カスタマイズ** - プラグインディレクトリ、デフォルト設定

環境変数はシステムレベル、CI/CDパイプライン、またはワークフローファイル内の`vars`セクションで定義できます。

## ランタイム設定変数

### `PROBE_LOG_LEVEL`

**型:** String  
**値:** `debug`, `info`, `warn`, `error`  
**デフォルト:** `info`  
**説明:** Probeのログ出力の詳細レベルを制御

```bash
# デバッグログを有効化
export PROBE_LOG_LEVEL=debug
probe workflow.yml

# 警告とエラーのみに減らす
export PROBE_LOG_LEVEL=warn
probe workflow.yml
```

**出力例:**

```bash
# infoレベル（デフォルト）
2023-09-01 12:30:00 [INFO] Starting workflow: API Health Check
2023-09-01 12:30:01 [INFO] Job 'health-check' completed successfully

# debugレベル
2023-09-01 12:30:00 [DEBUG] Loading workflow file: workflow.yml
2023-09-01 12:30:00 [DEBUG] Parsing YAML configuration
2023-09-01 12:30:00 [INFO] Starting workflow: API Health Check
2023-09-01 12:30:00 [DEBUG] Starting job: health-check
2023-09-01 12:30:00 [DEBUG] Executing step: Check API Status
2023-09-01 12:30:01 [DEBUG] HTTP request: GET https://api.example.com/health
2023-09-01 12:30:01 [DEBUG] HTTP response: 200 OK (345ms)
2023-09-01 12:30:01 [INFO] Job 'health-check' completed successfully
```

### `PROBE_NO_COLOR`

**型:** Boolean  
**値:** `true`, `false`, `1`, `0`  
**デフォルト:** `false`  
**説明:** ターミナルでのカラー出力を無効化

```bash
# カラーを無効化（CI/CDログに有用）
export PROBE_NO_COLOR=true
probe workflow.yml

# カラー出力を強制（ターミナル検出を上書き）
export PROBE_NO_COLOR=false
probe workflow.yml
```

### `PROBE_TIMEOUT`

**型:** Duration  
**デフォルト:** `300s`（5分）  
**説明:** ワークフロー全体実行のグローバルタイムアウト

```bash
# 10分タイムアウトを設定
export PROBE_TIMEOUT=600s
probe long-running-workflow.yml

# クイックテスト用30秒タイムアウト
export PROBE_TIMEOUT=30s
probe quick-health-check.yml
```

### `PROBE_CONFIG`

**型:** String（ファイルパス）  
**デフォルト:** なし  
**説明:** すべてのワークフローとマージされるデフォルト設定ファイルのパス

```bash
# グローバルデフォルトを使用
export PROBE_CONFIG=/etc/probe/defaults.yml
probe workflow.yml  # defaults.ymlとマージされる

# ユーザー固有のデフォルト
export PROBE_CONFIG=~/.probe/defaults.yml
probe workflow.yml
```

**デフォルト設定ファイルの例:**

```yaml
# /etc/probe/defaults.yml
vars:
  # varsを通じてアクセスされる環境変数
  user_agent: "{{USER_AGENT ?? 'Probe Monitor v1.0'}}"
  default_timeout: "{{DEFAULT_TIMEOUT ?? '30s'}}"

jobs:
- name: default
  defaults:
    http:
      timeout: "{{vars.default_timeout}}"
      headers:
        User-Agent: "{{vars.user_agent}}"
        Accept: "application/json"
```

### `PROBE_PLUGIN_DIR`

**型:** String（ディレクトリパス）  
**デフォルト:** `~/.probe/plugins`  
**説明:** カスタムアクションプラグインを含むディレクトリ

```bash
# システム全体のプラグインを使用
export PROBE_PLUGIN_DIR=/usr/local/lib/probe/plugins
probe workflow.yml

# プロジェクト固有のプラグインを使用
export PROBE_PLUGIN_DIR=./plugins
probe workflow.yml
```

**プラグインディレクトリ構造:**

```
/usr/local/lib/probe/plugins/
├── custom-http/
│   └── custom-http-plugin
├── database/
│   └── db-plugin
└── notification/
    └── notification-plugin
```

## 認証変数

### API認証

ワークフローでのAPI認証の一般的なパターン：

#### `API_TOKEN` / `API_KEY`

```bash
# Bearerトークン認証
export API_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

```yaml
# workflow.yml
vars:
  api_token: "{{API_TOKEN}}"

jobs:
- name: default
  defaults:
    http:
      headers:
        Authorization: "Bearer {{vars.api_token}}"
```

#### `USERNAME` / `PASSWORD`

```bash
# Basic認証資格情報
export API_USERNAME="admin"
export API_PASSWORD="secret123"
```

```yaml
# workflow.yml
vars:
  username: "{{API_USERNAME}}"
  password: "{{API_PASSWORD}}"

steps:
  - name: "Authenticated Request"
    uses: http
    with:
      url: "https://api.example.com/protected"
      headers:
        Authorization: "Basic {{base64(vars.username + ':' + vars.password)}}"
```

### メール認証

#### SMTP設定

```bash
# Gmailでアプリパスワード使用
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT=587
export SMTP_USERNAME="alerts@example.com"
export SMTP_PASSWORD="app-specific-password"

# Office 365
export SMTP_HOST="smtp.office365.com"
export SMTP_PORT=587
export SMTP_USERNAME="alerts@company.com"
export SMTP_PASSWORD="account-password"
```

```yaml
# workflow.yml
vars:
  smtp_host: "{{SMTP_HOST}}"
  smtp_port: "{{SMTP_PORT}}"
  smtp_username: "{{SMTP_USERNAME}}"
  smtp_password: "{{SMTP_PASSWORD}}"

jobs:
- name: default
  defaults:
    smtp:
      host: "{{vars.smtp_host}}"
      port: "{{vars.smtp_port}}"
      username: "{{vars.smtp_username}}"
      password: "{{vars.smtp_password}}"
      from: "{{vars.smtp_username}}"
```

## アプリケーション設定変数

### サービスURL

```bash
# APIエンドポイント
export API_BASE_URL="https://api.production.com"
export API_VERSION="v2"
export HEALTH_CHECK_URL="${API_BASE_URL}/${API_VERSION}/health"

# データベース接続
export DATABASE_URL="postgresql://user:pass@localhost:5432/db"
export REDIS_URL="redis://localhost:6379/0"

# 外部サービス
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
export MONITORING_URL="https://monitoring.example.com/api/alerts"
```

### 機能フラグ

```bash
# 機能の有効/無効
export ENABLE_MONITORING=true
export ENABLE_SLACK_NOTIFICATIONS=false
export ENABLE_DETAILED_LOGGING=true
export ENABLE_PERFORMANCE_TRACKING=true
```

```yaml
# workflow.yml
vars:
  enable_monitoring: "{{ENABLE_MONITORING}}"
  enable_performance_tracking: "{{ENABLE_PERFORMANCE_TRACKING}}"
  enable_slack_notifications: "{{ENABLE_SLACK_NOTIFICATIONS}}"
  api_base_url: "{{API_BASE_URL}}"
  slack_webhook_url: "{{SLACK_WEBHOOK_URL}}"

jobs:
- name: monitoring
  if: vars.enable_monitoring == "true"
  steps:
    - name: "Performance Check"
      if: vars.enable_performance_tracking == "true"
      uses: http
      with:
        url: "{{vars.api_base_url}}/metrics"

- name: notifications
  if: vars.enable_slack_notifications == "true"
  needs: [monitoring]
  steps:
    - name: "Slack Alert"
      uses: http
      with:
        url: "{{vars.slack_webhook_url}}"
        method: "POST"
        body: |
          {
            "text": "Monitoring completed: {{jobs.monitoring.status}}"
          }
```

## CI/CD統合変数

### GitHub Actions

```yaml
# .github/workflows/probe.yml
name: API Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Run Probe Tests
        env:
          API_TOKEN: ${{ secrets.API_TOKEN }}
          ENVIRONMENT: ${{ github.ref == 'refs/heads/main' && 'production' || 'staging' }}
          BRANCH_NAME: ${{ github.ref_name }}
          COMMIT_SHA: ${{ github.sha }}
          RUN_ID: ${{ github.run_id }}
        run: probe workflow.yml
```

**ワークフローで利用可能:**

```bash
export GITHUB_REPOSITORY="owner/repo"
export GITHUB_REF="refs/heads/main"
export GITHUB_SHA="abc123def456"
export GITHUB_RUN_ID="123456789"
export GITHUB_ACTOR="username"
```

### GitLab CI

```yaml
# .gitlab-ci.yml
probe-test:
  stage: test
  script:
    - probe workflow.yml
  variables:
    API_TOKEN: $API_TOKEN
    ENVIRONMENT: $CI_ENVIRONMENT_NAME
    BRANCH_NAME: $CI_COMMIT_REF_NAME
    COMMIT_SHA: $CI_COMMIT_SHA
    PIPELINE_ID: $CI_PIPELINE_ID
```

**ワークフローで利用可能:**

```bash
export CI_PROJECT_NAME="project-name"
export CI_COMMIT_REF_NAME="main"
export CI_COMMIT_SHA="abc123"
export CI_PIPELINE_ID="123456"
export CI_JOB_ID="789012"
```

### Jenkins

```groovy
// Jenkinsfile
pipeline {
    agent any
    environment {
        API_TOKEN = credentials('api-token')
        ENVIRONMENT = "${BRANCH_NAME == 'main' ? 'production' : 'staging'}"
        BUILD_NUMBER = "${BUILD_NUMBER}"
        JOB_NAME = "${JOB_NAME}"
    }
    stages {
        stage('Test') {
            steps {
                sh 'probe workflow.yml'
            }
        }
    }
}
```

**ワークフローで利用可能:**

```bash
export BUILD_NUMBER="123"
export JOB_NAME="api-tests"
export WORKSPACE="/var/jenkins_home/workspace/api-tests"
export JENKINS_URL="https://jenkins.example.com"
```

## 環境固有設定

### マルチ環境セットアップ

```bash
# ベース設定（常に設定）
export API_VERSION="v1"
export DEFAULT_TIMEOUT="30s"

# 環境固有（環境ごとに設定）
case $ENVIRONMENT in
  "production")
    export API_BASE_URL="https://api.prod.com"
    export DATABASE_URL="postgresql://prod-db:5432/app"
    export LOG_LEVEL="warn"
    export ENABLE_MONITORING=true
    ;;
  "staging")  
    export API_BASE_URL="https://api.staging.com"
    export DATABASE_URL="postgresql://staging-db:5432/app"
    export LOG_LEVEL="info"
    export ENABLE_MONITORING=true
    ;;
  "development")
    export API_BASE_URL="http://localhost:3000"
    export DATABASE_URL="postgresql://localhost:5432/app_dev"
    export LOG_LEVEL="debug"
    export ENABLE_MONITORING=false
    ;;
esac
```

### Docker設定

```dockerfile
# Dockerfile
FROM alpine:latest

# Probeをインストール
RUN curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o /usr/local/bin/probe && \
    chmod +x /usr/local/bin/probe

# デフォルト環境変数を設定
ENV PROBE_LOG_LEVEL=info
ENV PROBE_NO_COLOR=true
ENV PROBE_TIMEOUT=300s

COPY workflows/ /workflows/
WORKDIR /workflows

ENTRYPOINT ["probe"]
CMD ["workflow.yml"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  probe:
    build: .
    environment:
      - API_TOKEN=${API_TOKEN}
      - API_BASE_URL=https://api.example.com
      - ENVIRONMENT=production
      - PROBE_LOG_LEVEL=info
    volumes:
      - ./workflows:/workflows
      - ./reports:/reports
    command: monitoring.yml
```

## セキュリティの考慮事項

### 機密変数

**機密値をバージョン管理にコミットしないでください:**

```bash
# ❌ 悪い例: ワークフローファイルにハードコード
vars:
  API_TOKEN: "secret-token-here"

# ✅ 良い例: 環境変数を参照
vars:
  api_token: "{{API_TOKEN}}"
```

### 変数管理

```bash
# セキュアな秘密管理を使用
export API_TOKEN=$(aws ssm get-parameter --name "/app/api-token" --with-decryption --query 'Parameter.Value' --output text)
export DB_PASSWORD=$(vault kv get -field=password secret/database)

# 複雑な秘密には一時ファイルを使用
echo "$GOOGLE_CREDENTIALS_JSON" > /tmp/gcp-key.json
export GOOGLE_APPLICATION_CREDENTIALS=/tmp/gcp-key.json
```

### 環境隔離

```bash
# 競合を避けるために環境でプレフィックス変数
export PROD_API_TOKEN="prod-token"
export STAGING_API_TOKEN="staging-token"  
export DEV_API_TOKEN="dev-token"

# 環境固有の選択を使用
export API_TOKEN="${ENVIRONMENT}_API_TOKEN"
export API_TOKEN="${!API_TOKEN}"  # 間接変数展開
```

## 環境変数のデバッグ

### 利用可能な変数の表示

```bash
# すべての環境変数をリスト
env | grep -E '^(PROBE_|API_|SMTP_)' | sort

# 特定の変数をチェック
echo "API_TOKEN: $API_TOKEN"
echo "PROBE_LOG_LEVEL: $PROBE_LOG_LEVEL"

# ワークフローでデバッグ（機密データに注意）
probe -v workflow.yml 2>&1 | grep -i "environment"
```

### テンプレートデバッグ

```yaml
# workflow.yml - 環境変数展開のデバッグ
vars:
  api_base_url: "{{API_BASE_URL}}"
  environment: "{{ENVIRONMENT}}"
  default_timeout: "{{DEFAULT_TIMEOUT ?? 'not set'}}"

steps:
  - name: "Debug Environment"
    uses: hello
    with:
      message: |
        Environment Debug:
        API_URL: "{{vars.api_base_url}}"
        Environment: "{{vars.environment}}"
        Timeout: "{{vars.default_timeout}}"
        
  - name: "Test Variable Access"
    uses: echo
    with:
      message: |
        Available variables:
        {{range $key, $value := vars}}
          {{$key}}: {{$value}}
        {{end}}
```

## 一般的なパターン

### 設定カスケード

```bash
# システムデフォルト
export PROBE_CONFIG=/etc/probe/system.yml

# チームデフォルト  
export TEAM_CONFIG=/opt/team/defaults.yml

# プロジェクト固有
export PROJECT_CONFIG=./probe-defaults.yml

# カスケードでランタイム実行
probe ${PROBE_CONFIG},${TEAM_CONFIG},${PROJECT_CONFIG},workflow.yml
```

### 動的設定

```bash
# 環境に基づいて設定を生成
WORKFLOW_FILE="workflow-${ENVIRONMENT}.yml"
if [[ ! -f "$WORKFLOW_FILE" ]]; then
  WORKFLOW_FILE="workflow.yml"
fi

probe "$WORKFLOW_FILE"
```

### 条件付き実行

```bash
# 環境に基づいて特定のジョブをスキップ
export SKIP_PERFORMANCE_TESTS=$([[ "$ENVIRONMENT" == "development" ]] && echo "true" || echo "false")
export ENABLE_SLACK_ALERTS=$([[ "$ENVIRONMENT" == "production" ]] && echo "true" || echo "false")
```

```yaml
# workflow.yml
vars:
  skip_performance_tests: "{{SKIP_PERFORMANCE_TESTS}}"
  enable_slack_alerts: "{{ENABLE_SLACK_ALERTS}}"

jobs:
- name: performance-tests
  if: "{{vars.skip_performance_tests}}" != "true"
  # パフォーマンステストステップ
    
- name: alerts
  if: "{{vars.enable_slack_alerts}}" == "true"
  needs: [performance-tests]
  # アラートステップ
```

## 関連項目

- **[CLIリファレンス](../cli-reference/)** - コマンドライン環境変数の使用
- **[YAML設定](../yaml-configuration/)** - ワークフローでの環境変数の使用
- **[ハウツー: 環境管理](../../how-tos/environment-management/)** - 実用的な環境管理戦略
- **[概念: ファイルマージ](../../concepts/file-merging/)** - 設定合成パターン