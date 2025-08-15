# シェルアクション

`shell`アクションはシェルコマンドとスクリプトを安全に実行し、包括的な出力キャプチャとエラーハンドリングを提供します。

## 基本的な構文

```yaml
steps:
  - name: "Execute Build Script"
    uses: shell
    with:
      cmd: "npm run build"
    test: res.code == 0
```

## パラメータ

### `cmd` (必須)

**型:** String  
**説明:** 実行するシェルコマンド  
**サポート:** テンプレート式

```yaml
vars:
  api_url: "{{API_URL}}"

with:
  cmd: "echo 'Hello World'"
  cmd: "npm run {{vars.build_script}}"
  cmd: "curl -f {{vars.api_url}}/health"
```

### `shell` (オプション)

**型:** String  
**デフォルト:** `/bin/sh`  
**許可値:** `/bin/sh`, `/bin/bash`, `/bin/zsh`, `/bin/dash`, `/usr/bin/sh`, `/usr/bin/bash`, `/usr/bin/zsh`, `/usr/bin/dash`

```yaml
with:
  cmd: "echo $0"
  shell: "/bin/bash"
```

### `workdir` (オプション)

**型:** String  
**説明:** コマンド実行用の作業ディレクトリ（絶対パス必須）  
**サポート:** テンプレート式

```yaml
with:
  cmd: "pwd && ls -la"
  workdir: "/app/src"
  workdir: "{{vars.project_path}}"
```

### `timeout` (オプション)

**型:** String または Duration  
**デフォルト:** `30s`  
**形式:** Go duration形式 (`30s`, `5m`, `1h`) または数値 (秒)

```yaml
with:
  cmd: "npm test"
  timeout: "10m"
  timeout: "300"  # 300秒
```

### `env` (オプション)

**型:** Object  
**説明:** コマンドに設定する環境変数  
**サポート:** 値でのテンプレート式

```yaml
vars:
  production_api_url: "{{PRODUCTION_API_URL}}"

with:
  cmd: "npm run build"
  env:
    NODE_ENV: "production"
    API_URL: "{{vars.production_api_url}}"
    BUILD_VERSION: "{{vars.version}}"
```

## リトライ機能

shellアクションは統一されたステップレベルのリトライ機能をサポートしています。これにより、一時的な障害やサービス起動時間に対してコマンドを自動的に再実行できます。

```yaml
- name: "Wait for Service Startup"
  uses: shell
  with:
    cmd: "curl -f http://localhost:8080/health"
  retry:
    max_attempts: 30      # 最大試行回数 (1-100)
    interval: "2s"        # リトライ間隔
    initial_delay: "5s"   # 初回実行前の待機時間（オプション）
  test: res.code == 0
```

shellアクションでは、**終了コード 0** が成功条件となります。リトライ機能の詳細については[アクションガイド](../../guide/concepts/actions/#リトライ機能)を参照してください。

## レスポンス形式

```yaml
res:
  code: 0                    # 終了コード (0 = 成功)
  stdout: "Build successful" # 標準出力
  stderr: ""                 # 標準エラー出力

req:
  cmd: "npm run build"       # 元のコマンド
  shell: "/bin/sh"          # 使用されたシェル
  workdir: "/app"           # 作業ディレクトリ
  timeout: "30s"            # タイムアウト設定
  env:                      # 環境変数
    NODE_ENV: "production"
```

## 使用例

### 基本的なコマンド実行

```yaml
- name: "System Information"
  uses: shell
  with:
    cmd: "uname -a"
  test: res.code == 0
```

### ビルドとテストパイプライン

```yaml
- name: "Install Dependencies"
  uses: shell
  with:
    cmd: "npm ci"
    workdir: "/app"
    timeout: "5m"
  test: res.code == 0

- name: "Run Tests"
  uses: shell
  with:
    cmd: "npm test"
    workdir: "/app"
    env:
      NODE_ENV: "test"
      CI: "true"
  test: res.code == 0 && (res.stdout | contains("All tests passed"))
```

### 環境固有のデプロイ

```yaml
vars:
  target_env: "{{TARGET_ENV}}"
  deploy_key: "{{DEPLOY_KEY}}"

- name: "Deploy to Environment"
  uses: shell
  with:
    cmd: "./deploy.sh {{vars.target_env}}"
    workdir: "/deploy"
    shell: "/bin/bash"
    timeout: "15m"
    env:
      DEPLOY_KEY: "{{vars.deploy_key}}"
      TARGET_ENV: "{{vars.target_env}}"
  test: res.code == 0
```

### エラーハンドリングとデバッグ

```yaml
- name: "Service Health Check"
  uses: shell
  with:
    cmd: "curl -f http://localhost:8080/health || echo 'Service down'"
  test: res.code == 0 || (res.stderr | contains("Service down"))

- name: "Debug Failed Build"
  uses: shell
  with:
    cmd: "npm run build:debug"
  # デバッグ出力をキャプチャするために失敗を許可
  outputs:
    debug_info: res.stderr
```

### サービス起動とリトライ

```yaml
- name: "Wait for Database Startup"
  uses: shell
  with:
    cmd: "pg_isready -h postgres -p 5432"
  retry:
    max_attempts: 30
    interval: "2s"
    initial_delay: "10s"
  test: res.code == 0

- name: "Verify API Health"
  uses: shell
  with:
    cmd: |
      # APIエンドポイントの確認
      curl -f -H "Accept: application/json" \
           http://api:8080/health
  retry:
    max_attempts: 60
    interval: "1s"
  test: res.code == 0

- name: "Wait for Build Artifact"
  uses: shell
  with:
    cmd: |
      # ビルド成果物の確認
      test -f ./dist/app.js && \
      test -s ./dist/app.js
  retry:
    max_attempts: 20
    interval: "500ms"
  test: res.code == 0
```


## セキュリティ機能

シェルアクションはいくつかのセキュリティ対策を実装しています：

- **シェルパス制限**: 承認されたシェル実行ファイルのみを許可
- **作業ディレクトリ検証**: 絶対パスとディレクトリの存在を確保
- **タイムアウト保護**: 無限実行を防止
- **環境変数フィルタリング**: 環境変数の受け渡しを安全に処理
- **出力サニタイゼーション**: コマンド出力を安全にキャプチャして返す

## エラーハンドリング

一般的な終了コードとその意味：

- **0**: 成功
- **1**: 一般的なエラー
- **2**: シェル組み込みコマンドの誤用
- **126**: コマンドを実行できない（権限拒否）
- **127**: コマンドが見つからない
- **130**: Ctrl+Cでスクリプトが終了
- **255**: 終了ステータスが範囲外

```yaml
- name: "Handle Different Exit Codes"
  uses: shell
  with:
    cmd: "some_command_that_might_fail"
  test: |
    res.code == 0 ? true :
    res.code == 127 ? (res.stderr | contains("not found")) :
    res.code < 128
```
