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

### `retry` (オプション)

**型:** Object  
**説明:** コマンドが成功（exit code 0）するまでリトライする設定

```yaml
with:
  cmd: "curl -f http://localhost:8080/health"
  retry:
    max_attempts: 30      # 最大試行回数 (1-100)
    interval: "2s"        # リトライ間隔
    initial_delay: "5s"   # 初回実行前の待機時間（オプション）
```

#### `retry.max_attempts` (必須)

**型:** Integer  
**範囲:** 1-100  
**説明:** リトライする最大試行回数

#### `retry.interval` (オプション)

**型:** String または Duration  
**デフォルト:** `1s`  
**形式:** Go duration形式 (`500ms`, `2s`, `1m`) または数値 (秒)  
**説明:** 各リトライ試行の間隔

#### `retry.initial_delay` (オプション)

**型:** String または Duration  
**デフォルト:** `0s` (遅延なし)  
**形式:** Go duration形式 (`500ms`, `2s`, `1m`) または数値 (秒)  
**説明:** 最初の試行前の待機時間

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

## リトライ機能

shellアクションは、コマンドが成功するまで自動的にリトライする機能を提供します。これは、サービスの起動待機、ネットワーク接続の確立、ファイル作成の監視などのシナリオで特に有用です。

### 基本的な動作

- **成功条件**: コマンドのexit code が `0` の場合に成功とみなされます
- **リトライ条件**: exit code が `0` 以外の場合にリトライが実行されます
- **最大試行回数**: `max_attempts` で指定された回数まで試行されます
- **リトライ間隔**: 各試行の間に `interval` で指定された時間だけ待機します
- **初期遅延**: 最初の試行前に `initial_delay` で指定された時間だけ待機します（オプション）

### 実行フロー

1. `initial_delay` が指定されている場合、その時間だけ待機
2. コマンドを実行
3. exit code が `0` の場合、成功として結果を返す
4. exit code が `0` 以外で、まだ試行回数に余裕がある場合：
   - `interval` の時間だけ待機
   - ステップ2に戻る
5. 最大試行回数に達した場合、最後の実行結果を返す

### 一般的な使用例

#### データベース接続待機
```yaml
- name: "Wait for PostgreSQL"
  uses: shell
  with:
    cmd: "pg_isready -h db -p 5432 -U app"
    retry:
      max_attempts: 30
      interval: "2s"
      initial_delay: "5s"
  test: res.code == 0
```

#### Kubernetes Pod起動待機
```yaml
- name: "Wait for Pod Ready"
  uses: shell
  with:
    cmd: "kubectl get pod myapp -o jsonpath='{.status.phase}' | grep -q Running"
    retry:
      max_attempts: 60
      interval: "5s"
  test: res.code == 0
```

#### ファイル作成監視
```yaml
- name: "Wait for Configuration File"
  uses: shell
  with:
    cmd: "test -f /etc/myapp/config.yaml"
    retry:
      max_attempts: 20
      interval: "1s"
  test: res.code == 0
```

### CI/CDパイプラインでの活用

```yaml
- name: "Deploy and Verify"
  uses: shell
  with:
    cmd: "kubectl rollout status deployment/myapp"
    retry:
      max_attempts: 30
      interval: "10s"
      initial_delay: "30s"
  test: res.code == 0

- name: "Health Check After Deploy"
  uses: shell
  with:
    cmd: "curl -f https://myapp.example.com/health"
    retry:
      max_attempts: 20
      interval: "3s"
  test: res.code == 0 && (res.stdout | contains("healthy"))
```

### 注意事項

- **リソース消費**: 長時間のリトライは実行時間とリソースを消費します
- **タイムアウト**: 各コマンド実行には個別に `timeout` が適用されます
- **ログ出力**: 各試行の結果はデバッグログに記録されます
- **失敗時の結果**: 全ての試行が失敗した場合、最後の試行結果が返されます

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
