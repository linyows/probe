# CLIリファレンス

このページでは、すべてのコマンド、オプション、使用パターンを含む、Probeコマンドラインインターフェイスの完全なドキュメントを提供します。

## 基本的な使用法

```bash
probe [options] <workflow-file>
probe <subcommand> [options] <file>
```

## コマンド構文

### 基本コマンド

単一のワークフローファイルを実行：

```bash
probe workflow.yml
```

### ファイルマージ

設定マージでワークフローを実行：

```bash
probe base.yml,environment.yml,overrides.yml
```

ファイルは左から右に連結され、1つのYAMLドキュメントとして解析されます。複数のファイルで定義されたトップレベルのキーは最後のファイルの値になり、エントリ単位ではなくキーごと置き換わります。

### 位置引数

#### `workflow-path`

**型:** String（必須）  
**説明:** ワークフローYAMLファイルのパス、またはマージ用のカンマ区切りファイルリスト

**例:**
```bash
# 単一ファイル
probe workflow.yml

# 複数ファイル（マージ）
probe base.yml,production.yml

# 相対パス
probe ./workflows/api-test.yml

# 絶対パス
probe /home/user/workflows/monitoring.yml
```

## コマンドラインオプション

### `-v, --verbose`

**型:** ブールフラグ  
**デフォルト:** `false`  
**説明:** 詳細な実行情報を表示する詳細出力を有効化

**例:**
```bash
probe -v workflow.yml
probe --verbose workflow.yml
```

**詳細出力に含まれる内容:**
- ステップバイステップの実行詳細
- HTTPリクエスト/レスポンス情報
- テンプレート評価結果
- タイミング情報
- デバッグメッセージ

### `-h, --help`

**型:** ブールフラグ  
**説明:** コマンド使用法ヘルプを表示して終了

**例:**
```bash
probe -h
probe --help
```

### `--version`

**型:** ブールフラグ
**説明:** バージョン情報を表示して終了

**例:**
```bash
probe --version
```

**出力形式:**
```
Probe Version 1.2.3 (commit: abc1234)
```

### `--timing`

**型:** ブールフラグ  
**デフォルト:** `false`  
**説明:** ステップごとの時刻情報（開始時刻とレスポンスタイム）を表示

**例:**
```bash
probe --timing workflow.yml
```

### `--output`

**型:** String  
**値:** `auto`, `spinner`, `stream`  
**デフォルト:** `auto`  
**説明:** レポートの出力方法を選択します。`auto`は対話的な端末なら`spinner`、それ以外なら`stream`を選びます。`spinner`は進捗をその場で描き換え、`stream`は完了したものから順に書き出すため、CIのログやパイプに向いています。

値は環境変数`PROBE_OUTPUT`でも指定できます。優先順位はフラグ、環境変数、自動判定の順です。

**例:**
```bash
probe --output stream workflow.yml
probe --output=spinner workflow.yml
PROBE_OUTPUT=stream probe workflow.yml
```

## サブコマンド

### `gen`

OpenAPI仕様からprobeワークフローYAMLを生成します。

**使い方:**
```bash
probe gen <openapi-file>
```

**例:**
```bash
probe gen petstore.yml
```

### `dag`

ワークフローを実行せずにジョブ依存関係グラフを表示します。デフォルトではASCIIアートで出力します。`--mermaid`を指定するとMermaidフローチャート形式で出力します。

**使い方:**
```bash
probe dag <workflow-file>
probe dag --mermaid <workflow-file>
```

**オプション:**

| オプション | 説明 |
|--------|-------------|
| `--mermaid` | ASCIIアートの代わりにMermaidフローチャート形式で出力 |

**ASCII出力例:**
```
╭───────────────────────╮
│         Setup         │
├───────────────────────┤
│ ○ Initialize          │
╰───────────┬───────────╯
            │
            │
            ↓
╭───────────────────────╮
│         Build         │
├───────────────────────┤
│ ○ Compile             │
│ ○ Package             │
╰───────────┬───────────╯
            │
            ├──────────────────────────┐
            ↓                          ↓
╭───────────────────────╮  ╭───────────────────────╮
│        Test A         │  │        Test B         │
├───────────────────────┤  ├───────────────────────┤
│ ○ Run tests           │  │ ○ Run tests           │
╰───────────────────────╯  ╰───────────────────────╯
```

**Mermaid出力例 (`--mermaid`):**
```mermaid
flowchart LR
    subgraph build["Build"]
        build_step0["Compile"]
    end
    subgraph unit_test["Unit Test"]
        unit_test_step0["Run unit"]
    end
    subgraph lint["Lint"]
        lint_step0["Run lint"]
    end
    subgraph deploy["Deploy"]
        deploy_step0["Deploy app"]
    end

    build --> unit_test
    build --> lint
    unit_test --> deploy
    lint --> deploy
```

以下の用途に便利です：
- 実行前にワークフロー構造を可視化
- ジョブ依存関係とそのステップの理解
- ジョブ依存関係の設定をデバッグ
- ドキュメントやダイアグラムの生成
- Markdownファイルへの埋め込み

## 環境変数

以下の環境変数がProbeの動作に影響します。

### `PROBE_OUTPUT`

**型:** String  
**値:** `auto`, `spinner`, `stream`  
**デフォルト:** `auto`  
**説明:** レポートの出力方法。`--output`と同じ値を取り、フラグが指定された場合はそちらが優先されます。

```bash
export PROBE_OUTPUT=stream
probe workflow.yml
```

### `PROBE_MAX_REPEAT_COUNT`

**型:** Integer  
**デフォルト:** `10000`  
**説明:** ステップの`repeat.count`の上限。これを超える指定をしたワークフローはエラーになります。

```bash
export PROBE_MAX_REPEAT_COUNT=50000
probe load-test.yml
```

### `PROBE_MAX_ATTEMPTS`

**型:** Integer  
**デフォルト:** `10000`  
**説明:** ステップのリトライ`max_attempts`の上限。

```bash
export PROBE_MAX_ATTEMPTS=100
probe workflow.yml
```

### `FORCE_COLOR`

**型:** String  
**値:** `1`  
**説明:** 標準出力が端末でない場合でも色付き出力を強制します。CIのログで色を残したいときに使います。

```bash
FORCE_COLOR=1 probe workflow.yml
```

これら以外の環境変数は、ワークフローの`vars`から名前そのままで参照できます。詳しくは[環境変数](/ja/reference/environment-variables)を参照してください。

## 使用例

### 基本的なワークフロー実行

```bash
# 簡単なヘルスチェックを実行
probe health-check.yml

# 詳細出力で実行
probe -v health-check.yml
```

### 環境固有の実行

```bash
# 開発環境
probe workflow.yml,dev.yml

# ステージング環境
probe workflow.yml,staging.yml

# プロダクション環境
probe workflow.yml,prod.yml
```

### 複雑な設定マージ

```bash
# 複数の設定をレイヤー化
probe base.yml,region-us.yml,environment-prod.yml,team-overrides.yml
```

### CI/CD統合

```bash
#!/bin/bash
# deployment-test.sh

set -e

echo "Running deployment validation..."
probe deployment-validation.yml,${ENVIRONMENT}.yml

echo "Running smoke tests..."
probe smoke-tests.yml,${ENVIRONMENT}.yml

echo "All tests passed!"
```

### Docker統合

```bash
# ProbeをDockerコンテナで実行
docker run --rm -v $(pwd):/workspace \
  -e API_TOKEN=$API_TOKEN \
  probe:latest workflow.yml

# Docker Composeサービス
version: '3.8'
services:
  probe:
    image: probe:latest
    volumes:
      - ./workflows:/workflows
    environment:
      - API_TOKEN
      - ENVIRONMENT=production
    command: /workflows/monitoring.yml,/workflows/production.yml
```

### スケジュール実行

```bash
# 定期監視用のcrontabエントリ
# 5分毎に実行
*/5 * * * * /usr/local/bin/probe /opt/workflows/monitoring.yml >> /var/log/probe.log 2>&1

# systemdタイマーユニット
[Unit]
Description=Probe Monitoring
Requires=probe-monitoring.timer

[Service]
Type=oneshot
ExecStart=/usr/local/bin/probe /opt/workflows/monitoring.yml
User=probe
Group=probe

[Install]
WantedBy=multi-user.target
```

## 終了コード

Probeが返す終了コードは2つです。

| 終了コード | 意味 | 説明 |
|-----------|---------|-------------|
| `0` | 成功 | すべてのジョブが完了し、すべてのテストが成功 |
| `1` | 失敗 | テストの失敗、アクションのエラー、またはワークフローを読み込めなかった場合（ファイルが無い、YAMLが不正、未知のフラグ など） |

### 終了コードの例

```bash
# スクリプトで終了コードをチェック
probe workflow.yml
if [ $? -eq 0 ]; then
  echo "Workflow succeeded"
else
  echo "Workflow failed with exit code $?"
fi

# CI/CDパイプラインで使用
probe integration-tests.yml || exit 1
```

## パフォーマンスとリソース使用量

### メモリ使用量

- **ベースメモリ:** Probeランタイムで約10MB
- **ワークフローあたり:** 複雑さに応じて約1-5MB
- **アクションあたり:** レスポンスサイズに応じて約0.1-1MB

### 実行タイミング

```bash
# ワークフロー実行時間を計測
time probe workflow.yml

# 詳細モードで詳細なタイミング
probe -v workflow.yml 2>&1 | grep "Execution time"
```

### 並行実行

Probeは可能な場合ジョブを並列実行します：

```bash
# 依存関係のないジョブは同時実行される
# 最大同時実行数は通常システムリソースにより制限される
# 詳細モードで実行パターンを確認
probe -v parallel-workflow.yml
```

## トラブルシューティングコマンド

### デバッグ情報

```bash
# 最大限の情報を出す
probe -v --timing workflow.yml

# バージョンとコミットを確認
probe --version

# 実行せずにジョブの依存関係を確認
probe dag workflow.yml
```

### よくある問題

**ファイルが見つからない:**
```bash
probe: error: workflow file 'missing.yml' not found
# ファイルパスと権限をチェック
ls -la missing.yml
```

**権限拒否:**
```bash
probe: error: permission denied reading 'workflow.yml'
# ファイル権限を修正
chmod 644 workflow.yml
```

**YAML構文エラー:**
```bash
probe: error: YAML syntax error at line 15
# YAML構文を検証
yaml-validator workflow.yml
```

## 統合例

### GitHub Actions

```yaml
name: Probe Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Probe
        run: |
          curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o probe
          chmod +x probe
          sudo mv probe /usr/local/bin/
      
      - name: Run Tests
        env:
          API_TOKEN: ${{ secrets.API_TOKEN }}
        run: probe workflow.yml,${GITHUB_REF##*/}.yml
```

### GitLab CI

```yaml
stages:
  - test

probe-test:
  stage: test
  image: alpine:latest
  before_script:
    - apk add --no-cache curl
    - curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o /usr/local/bin/probe
    - chmod +x /usr/local/bin/probe
  script:
    - probe workflow.yml,$CI_ENVIRONMENT_NAME.yml
  variables:
    API_TOKEN: $API_TOKEN
```

### Jenkinsパイプライン

```groovy
pipeline {
    agent any
    
    environment {
        API_TOKEN = credentials('api-token')
        PROBE_OUTPUT = 'stream'
    }
    
    stages {
        stage('Install Probe') {
            steps {
                sh '''
                    curl -L https://github.com/linyows/probe/releases/latest/download/probe-linux-amd64 -o probe
                    chmod +x probe
                    sudo mv probe /usr/local/bin/
                '''
            }
        }
        
        stage('Run Tests') {
            steps {
                sh 'probe workflow.yml,${BRANCH_NAME}.yml'
            }
        }
    }
    
    post {
        always {
            archiveArtifacts artifacts: '*.log', allowEmptyArchive: true
        }
    }
}
```

## 高度な使用パターン

### 設定テンプレート

```bash
# ファイルパスで環境変数を使用
export ENV=production
probe workflow.yml,configs/${ENV}.yml

# 動的ファイル選択
WORKFLOW_FILE=$([ "$ENV" = "prod" ] && echo "prod-workflow.yml" || echo "dev-workflow.yml")
probe $WORKFLOW_FILE
```

### バッチ実行

```bash
# 複数のワークフローを実行
for workflow in workflows/*.yml; do
  echo "Running $workflow..."
  probe "$workflow" || echo "Failed: $workflow"
done

# 並列実行
find workflows/ -name "*.yml" | xargs -P 4 -I {} probe {}
```

### 監視統合

```bash
# 監視システムとの統合
probe monitoring.yml
RESULT=$?

if [ $RESULT -ne 0 ]; then
  # 監視システムにアラートを送信
  curl -X POST https://monitoring.example.com/alert \
    -H "Content-Type: application/json" \
    -d '{"message": "Probe workflow failed", "exit_code": '$RESULT'}'
fi
```

## 関連項目

- **[YAML設定](/ja/reference/yaml-configuration)** - 完全なYAML構文リファレンス
- **[アクションリファレンス](/ja/reference/actions/variables)** - 組み込みアクションとパラメータ
- **[環境変数](/ja/reference/environment-variables)** - サポートされているすべての環境変数
- **[ハウツー](/ja/guide/how-tos/api-testing)** - 実用的な使用例
- **[エラーハンドリング戦略](/ja/guide/how-tos/error-handling-strategies)** - よくある問題と解決策
