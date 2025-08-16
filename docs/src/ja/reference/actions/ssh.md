# SSH Action

SSH Action を使用すると、リモートサーバーに SSH 接続してコマンドを実行できます。デプロイメント、リモート監視、サーバー管理などの用途に最適です。

## 基本的な使用方法

```yaml
- name: サーバー状況を確認
  uses: ssh
  with:
    host: "example.com"
    user: "deploy"
    password: "your_password"
    cmd: "uptime && df -h"
  test: res.code == 0
```

## パラメータ

### 必須パラメータ

| パラメータ | 型 | 説明 |
|-----------|---|------|
| `host` | string | 接続先ホスト名またはIPアドレス |
| `user` | string | SSH ユーザー名 |
| `cmd` | string | 実行するコマンド |

### 認証パラメータ（いずれか必須）

| パラメータ | 型 | 説明 |
|-----------|---|------|
| `password` | string | パスワード認証 |
| `key_file` | string | 秘密鍵ファイルのパス（`~` 展開対応） |
| `key_passphrase` | string | 秘密鍵のパスフレーズ（暗号化された鍵の場合） |

### オプションパラメータ

| パラメータ | 型 | デフォルト | 説明 |
|-----------|---|-----------|------|
| `port` | int | 22 | SSH ポート番号 |
| `timeout` | string | "30s" | コマンド実行のタイムアウト |
| `workdir` | string | - | リモートでの作業ディレクトリ |
| `strict_host_check` | bool | true | Host Key の厳密検証 |
| `known_hosts` | string | `~/.ssh/known_hosts` | Known Hosts ファイルのパス |

### 環境変数

`env__` プレフィックスで環境変数を設定できます：

```yaml
with:
  env__NODE_ENV: production
  env__APP_VERSION: v1.2.3
```

## 戻り値

SSH Action は以下の値を返します：

| フィールド | 型 | 説明 |
|-----------|---|------|
| `res.code` | int | 終了コード（0 = 成功） |
| `res.stdout` | string | 標準出力 |
| `res.stderr` | string | 標準エラー出力 |
| `res.status` | int | 実行ステータス（0 = 成功、1 = 失敗） |

## 使用例

### パスワード認証

```yaml
- name: パスワード認証でログイン
  uses: ssh
  with:
    host: "server.example.com"
    port: 22
    user: "admin"
    password: "secure_password"
    cmd: "systemctl status nginx"
    timeout: "60s"
  test: res.code == 0
```

### 公開鍵認証

```yaml
- name: SSH鍵でログイン
  uses: ssh
  with:
    host: "prod.example.com"
    user: "deploy"
    key_file: "~/.ssh/deploy_key"
    cmd: "git pull origin main && systemctl restart app"
    workdir: "/opt/myapp"
    timeout: "300s"
  test: res.code == 0
```

### 暗号化された秘密鍵

```yaml
- name: パスフレーズ付き鍵でログイン
  uses: ssh
  with:
    host: "secure.example.com"
    user: "admin"
    key_file: "~/.ssh/id_rsa"
    key_passphrase: "key_passphrase"
    cmd: "docker ps"
  test: res.code == 0
```

### 環境変数を使用

```yaml
- name: 環境変数付きでデプロイ
  uses: ssh
  with:
    host: "app.example.com"
    user: "deploy"
    key_file: "~/.ssh/deploy_key"
    cmd: |
      echo "Deploying version: $APP_VERSION"
      echo "Environment: $DEPLOY_ENV"
      ./deploy.sh
    env__APP_VERSION: "v2.1.0"
    env__DEPLOY_ENV: "production"
    workdir: "/opt/myapp"
    timeout: "600s"
  test: res.code == 0 && contains(res.stdout, "Deploy completed")
```

### 複数行コマンド

```yaml
- name: システム情報収集
  uses: ssh
  with:
    host: "monitor.example.com"
    user: "admin"
    password: "admin_pass"
    cmd: |
      echo "=== System Information ==="
      uname -a
      echo "=== Disk Usage ==="
      df -h
      echo "=== Memory Usage ==="
      free -h
      echo "=== Process List ==="
      ps aux | head -10
    timeout: "120s"
  test: res.code == 0
  echo: "{{res.stdout}}"
```

### エラーハンドリング

```yaml
- name: サービス再起動（エラー処理付き）
  uses: ssh
  with:
    host: "app.example.com"
    user: "deploy"
    key_file: "~/.ssh/deploy_key"
    cmd: |
      if systemctl is-active --quiet myapp; then
        echo "Stopping service..."
        systemctl stop myapp
      fi
      echo "Starting service..."
      systemctl start myapp
      systemctl is-active myapp
    timeout: "180s"
  test: |
    res.code == 0 && 
    contains(res.stdout, "active") &&
    !contains(res.stderr, "failed")
```

## セキュリティ設定

### Host Key 検証

```yaml
- name: 厳密なHost Key検証
  uses: ssh
  with:
    host: "secure.example.com"
    user: "admin"
    key_file: "~/.ssh/id_rsa"
    cmd: "whoami"
    strict_host_check: true
    known_hosts: "~/.ssh/known_hosts"
  test: res.code == 0
```

### Host Key 検証を無効化（開発環境のみ推奨）

```yaml
- name: 開発環境での接続
  uses: ssh
  with:
    host: "dev.example.com"
    user: "developer"
    password: "dev_password"
    cmd: "npm test"
    strict_host_check: false  # 開発環境でのみ使用
  test: res.code == 0
```

## ベストプラクティス

### 1. 認証方法の選択

**公開鍵認証を推奨**：
```yaml
# 推奨
key_file: "~/.ssh/deploy_key"

# 可能であれば避ける
password: "plain_password"
```

### 2. タイムアウトの設定

適切なタイムアウト値を設定：
```yaml
# 短時間のコマンド
timeout: "30s"

# デプロイメントなど長時間の処理
timeout: "600s"
```

### 3. エラーハンドリング

```yaml
test: |
  res.code == 0 && 
  !contains(res.stderr, "error") &&
  !contains(res.stderr, "failed")
```

### 4. ログの安全性

パスワードや秘密鍵の情報は自動的にログから除外されます。

### 5. 環境変数の活用

機密情報は環境変数を使用：
```yaml
vars:
  ssh_password: "{{env.SSH_PASSWORD}}"
  deploy_key: "{{env.DEPLOY_KEY_PATH}}"

steps:
- uses: ssh
  with:
    password: "{{vars.ssh_password}}"
    key_file: "{{vars.deploy_key}}"
```

## 制限事項

### SSH Config ファイル非対応

移植性と再現性を重視するため、SSH Config ファイル（`~/.ssh/config`）はサポートしていません。すべての接続パラメータはワークフロー内で明示的に定義する必要があります。

### 対話的コマンド

対話的な入力が必要なコマンドはサポートしていません。すべてのコマンドは非対話的に実行される必要があります。

## トラブルシューティング

### 接続エラー

**Host Key 検証エラー**:
```yaml
# known_hosts を確認
known_hosts: "~/.ssh/known_hosts"

# または開発環境では無効化
strict_host_check: false
```

**認証エラー**:
```yaml
# 鍵ファイルのパスとパーミッションを確認
key_file: "/home/user/.ssh/id_rsa"  # 絶対パス
# または
key_file: "~/.ssh/id_rsa"          # チルダ展開

# パスフレーズが必要な場合
key_passphrase: "your_passphrase"
```

**タイムアウトエラー**:
```yaml
# より長いタイムアウトを設定
timeout: "300s"
```

### 連続実行時の問題

同じホストへの連続接続で問題が発生する場合は、適切な間隔を空ける：

```yaml
- name: 最初の接続
  uses: ssh
  with:
    cmd: "command1"
  
- name: 少し待機
  uses: hello
  with:
    name: "wait"
  wait: 1s
  
- name: 次の接続
  uses: ssh
  with:
    cmd: "command2"
```

## 関連項目

- [Shell Action](./shell.md) - ローカルでのコマンド実行
- [HTTP Action](./http.md) - REST API との通信
- [環境変数の管理](../environment-variables.md)