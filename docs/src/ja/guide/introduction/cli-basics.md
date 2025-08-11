# CLI基本

Probeコマンドライン・インターフェース（CLI）は、Probeと対話する主要な方法です。このガイドでは、マスターする必要があるすべての基本的なコマンド、オプション、技術をカバーします。

## 基本的な使用法

Probeを実行する最も基本的な方法は：

```bash
$ probe workflow.yml
```

これは`workflow.yml`で定義されたワークフローを実行し、結果を表示します。

### コマンド構文

```bash
$ probe [options] <workflow-file>
```

- **`workflow-file`**: YAMLワークフローファイルへのパス（必須）
- **`options`**: 動作を変更するさまざまなフラグ（オプション）

## オプション

Probe実行時のオプションを説明します。

### ヘルプと情報

利用可能なオプションについてのヘルプを取得：

```bash
$ probe --help
# あるいは -h
```

インストールされたバージョンを確認：

```bash
$ probe --version
```

### 詳細出力

詳細なログを有効にして、内部で何が起こっているかを確認：

```bash
$ probe workflow.yml --verbose
# あるいは -v
```

詳細モードでは以下が表示されます：
- 詳細なHTTPリクエスト/レスポンス情報
- ステップ実行タイミング
- 変数解決の詳細
- プラグイン通信ログ

**詳細出力の例：**
```
[DEBUG] Starting workflow: My Health Check
[DEBUG] --- Step 1: Check Homepage
[DEBUG] Request:
[DEBUG]   method: "GET"
[DEBUG]   url: "https://example.com"
[DEBUG] Response:
[DEBUG]   status: 200
[DEBUG]   body: "<!DOCTYPE html>..."
[DEBUG] RT: 245ms
```

### レスポンス時間表示

HTTPリクエストのレスポンス時間を表示：

```bash
$ probe workflow.yml --rt
```

これにより、`--verbose`の完全な詳細さなしにタイミング情報が出力に追加されます。

### オプションの組み合わせ

複数のオプションを組み合わせることができます：

```bash
$ probe -v --rt workflow.yml
```

## 複数ファイルマージ

Probeの強力な機能の一つは、複数のYAMLファイルをマージできることです：

```bash
$ probe base.yml,overrides.yml
```

### ファイルマージの使用例

**1. 環境固有の設定:**

**base-workflow.yml:**
```yaml
name: API Health Check
jobs:
- name: API
  defaults:
    http:
      url: *endpoint
  steps:
  - name: Check API
    action: http
    with:
      get: /foo/bar
    test: res.code == 200
```

**production.yml:**
```yaml
# 本番環境固有の設定
x_alias:
  x_endpoint: &endpoint "https://api.production.example.com"
```

**staging.yml:**
```yaml
# ステージング環境固有の設定
x_alias:
  x_endpoint: &endpoint "https://api.staging.example.com"
```

異なる環境で実行：
```bash
# 本番環境
$ probe base-workflow.yml,production.yml

# ステージング環境
$ probe base-workflow.yml,staging.yml
```

**2. 共有設定:**

**common-config.yml:**
```yaml
# 共有HTTP設定
defaults:
  http:
    timeout: 30s
    headers:
      User-Agent: "Probe Health Check v1.0"
```

**api-check.yml:**
```yaml
name: API Monitoring
# 共通HTTP設定を継承
jobs:
  # ... ジョブ定義
```

```bash
$ probe common-config.yml,api-check.yml
```

**3. 特定の値をオーバーライド:**

```bash
$ probe workflow.yml,local-overrides.yml
```

マージ順序が重要です - 後のファイルが前のファイルの値を上書きします。

## 環境変数の使用

Probeはworkflow wideの`vars`オブジェクトを使用して環境変数を変数に設定することができます：

```yaml
name: Database Test

vars:
  db_host: "{{DB_HOST ?? 'localhost'}}"
  db_pass: "{{DB_PASS ?? 'secret!!!'}}"

jobs:
- name: DB Operations
  defaults:
    db:
      dsn: mysql://foobar:{{vars.db_pass}}@{{vars.db_host}}:3306/probetest
  steps:
  - name: Test MySQL Connection
    uses: db
    with:
      query: SELECT 1 as connection_test, NOW() as server_time
    test: res.code == 0
```

実行前に環境変数を設定：

```bash
export DB_HOST="https://db.example.com"
export DB_PASS="****************"
$ probe workflow.yml
```

## 終了コード

Probeは結果を示すために標準的な終了コードを使用します：

- **`0`**: 成功 - すべてのジョブが正常に完了
- **`1`**: 失敗 - 1つ以上のジョブが失敗またはエラーが発生

これにより、ProbeはスクリプトやCI/CDパイプラインでの使用に最適です：

```bash
#!/bin/bash
if probe health-check.yml; then
    echo "✅ Health check passed"
    deploy_application
else
    echo "❌ Health check failed"
    exit 1
fi
```

## 実用的な例

### 1. CI/CD統合

```bash
# CI/CDパイプラインで
probe smoke-tests.yml
if [ $? -eq 0 ]; then
    echo "Smoke tests passed, proceeding with deployment"
else
    echo "Smoke tests failed, aborting deployment"
    exit 1
fi
```

### 2. Cronジョブ監視

```bash
# crontabで - 5分ごとに実行
*/5 * * * * /usr/local/bin/probe /opt/monitoring/health-check.yml >> /var/log/probe.log 2>&1
```

### 3. 開発テスト

```bash
# 開発中のクイックテスト
$ probe -v api-tests.yml,local-config.yml
```

## 出力の解釈

Probeの出力を理解することで、問題を迅速に特定できます：

### 成功した実行

```
My Health Check
Monitoring application health

⏺ Frontend Check (Completed in 0.45s)
  ⎿ 0. ✔︎  Check Homepage (234ms)
     1. ✔︎  Check API Health (156ms)

Total workflow time: 0.45s ✔︎ All jobs succeeded
```

**主要な指標:**
- **⏺**: ジョブ完了
- **✔︎**: ステップ成功
- **緑色のテキスト**: 成功ステータス
- **合計時間**: 全体の実行時間

### 失敗した実行

```
My Health Check
Monitoring application health

⏺ Frontend Check (Failed in 1.23s)
  ⎿ 0. ✘  Check Homepage (1.23s)
              request: map[string]interface {}{"method":"GET", "url":"https://example.com"}
              response: map[string]interface {}{"status":500, "body":"Internal Server Error"}
     1. ⏭  Check API Health (skipped)

Total workflow time: 1.23s ✘ 1 job(s) failed
```

**主要な指標:**
- **✘**: ステップ失敗
- **⏭**: ステップスキップ（前の失敗による）
- **赤色のテキスト**: 失敗ステータス
- **Request/Response**: 失敗したHTTPリクエストのデバッグ情報

### 部分的成功

```
Multi-Service Check
Checking multiple services

⏺ Critical Services (Completed in 0.67s)
  ⎿ 0. ✔︎  Database Check (234ms)
     1. ✔︎  API Check (156ms)

⏺ Optional Services (Failed in 2.34s)
  ⎿ 0. ✘  External API Check (2.34s)

Total workflow time: 2.34s ✘ 1 job(s) failed
```

## よくある問題のトラブルシューティング

### ファイルが見つからない

```
[ERROR] workflow is required
```

**解決策:** 有効なファイルパスを提供していることを確認：
```bash
$ probe ./workflows/health-check.yml
```

### YAML構文エラー

```
[ERROR] yaml: line 5: mapping values are not allowed in this context
```

**解決策:** YAML構文を確認：
- インデントにはタブではなくスペースを使用
- コロンを使った適切なキー・バリュー構文を確認
- 特殊文字を含む文字列はクォートで囲む

### 権限エラー

```
[ERROR] permission denied
```

**解決策:** ワークフローファイルが読み取り可能であることを確認：
```bash
chmod +r workflow.yml
```

### ネットワークタイムアウト

ステップがハングまたはタイムアウトする場合：
- `--verbose`を使用して詳細なネットワーク情報を確認
- ターゲットURLへのネットワーク接続を確認
- HTTPアクションでタイムアウト値の追加を検討

## ベストプラクティス

### 1. 説明的なファイル名を使用

```bash
# 良い
$ probe api-health-check.yml
$ probe database-migration-test.yml

# そうでもない
$ probe test.yml
$ probe workflow.yml
```

### 2. ディレクトリで整理

```bash
# 目的別にワークフローを整理
$ probe monitoring/health-check.yml
$ probe deployment/smoke-tests.yml
$ probe maintenance/cleanup.yml
```

### 3. 設定ファイルを使用

シークレットと環境固有の値は別ファイルに保管：

```bash
$ probe workflow.yml,configs/production.yml
```

### 4. 本番環境前に検証

常に詳細モードで最初にテスト：

```bash
# 徹底的にテスト
$ probe -v new-workflow.yml,test-config.yml

# 本番環境にデプロイ
$ probe new-workflow.yml,production-config.yml
```

## 次のステップ

CLI基本をマスターしたので、次の内容に進むことができます：

1. **[クイックスタート](./quickstart/)** - 5分で最初のワークフローを作成・実行
2. **[最初のワークフロー](./your-first-workflow/)** - 簡単な例から始める
3. **[リファレンスを確認](../reference/)** - 利用可能なすべてのオプションを深く理解

CLIはProbeの力への入り口です。これらの基本をマスターしたので、高度な監視と自動化ワークフローを構築する準備が整いました。
