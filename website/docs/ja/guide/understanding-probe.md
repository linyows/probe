---
title: Probeを理解する
description: Probeの核となる概念とアーキテクチャを学ぶ
weight: 30
---

# Probeを理解する

ProbeはYAMLベースのワークフロー自動化ツールで、監視、テスト、自動化タスクのために設計されています。このガイドでは、Probeを効果的に使用するために理解すべき核となる概念を説明します。

## 核となる概念

### ワークフロー

**ワークフロー**は、Probeが実行すべき内容を定義するトップレベルのコンテナです。以下から構成されます：

- **メタデータ**: ワークフローの名前と説明
- **ジョブ**: 並行または順次実行される1つ以上のジョブ
- **グローバル設定**: すべてのジョブに適用される共有設定

```yaml
name: My Workflow
description: What this workflow does
# Jobs go here...
```

### ジョブ

**ジョブ**は一緒に実行されるステップの集合です。ジョブは以下が可能です：

- 他のジョブと並行実行
- 他のジョブに対する依存関係を持つ
- 他のジョブと出力を共有
- 独自の設定とコンテキストを持つ

```yaml
jobs:
  job-name:
    name: Human-readable job name
    needs: [other-job]  # Optional: wait for other jobs
    steps:
      # Steps go here...
```

### ステップ

**ステップ**は実行の最小単位です。各ステップは以下が可能です：

- アクション（HTTPリクエスト、メールなど）を実行
- 結果を検証するテストを実行
- 出力にメッセージをエコー
- 他のステップで使用する出力を設定
- 条件付き実行ロジックを持つ

```yaml
steps:
  - name: Step Name
    action: http          # The action to execute
    with:                 # Parameters for the action
      url: https://api.example.com
      method: GET
    test: res.status == 200  # Test condition
    outputs:              # Data to pass to other steps
      response_time: res.time
```

### アクション

**アクション**は実際に作業を行う構築ブロックです。Probeには組み込みアクションが含まれています：

- **`http`**: HTTP/HTTPSリクエストを作成
- **`hello`**: 簡単な挨拶アクション（主にテスト用）
- **`smtp`**: SMTP経由でメールを送信

アクションはプラグインとして実装されているため、カスタムアクションでProbeを拡張できます。

## ワークフロー実行モデル

### 並行実行

デフォルトでは、ジョブは最大効率のために並行実行されます：

```yaml
jobs:
  frontend-check:    # These jobs run
    # ...             # at the same time
  backend-check:     # (in parallel)
    # ...
  database-check:
    # ...
```

### 依存関係による順次実行

`needs`キーワードを使用して依存関係を作成：

```yaml
jobs:
  setup:
    name: Setup Environment
    steps:
      # Setup steps...

  test:
    name: Run Tests
    needs: [setup]     # Wait for 'setup' to complete
    steps:
      # Test steps...

  cleanup:
    name: Clean Up
    needs: [test]      # Wait for 'test' to complete
    steps:
      # Cleanup steps...
```

### データフロー

データは**出力**を使用してワークフロー内を流れます：

```yaml
jobs:
  data-fetch:
    steps:
      - name: Get User Info
        action: http
        with:
          url: https://api.example.com/user/123
        outputs:
          user_id: res.json.id
          user_name: res.json.name

  notification:
    needs: [data-fetch]
    steps:
      - name: Send Welcome Email
        echo: "Welcome {{outputs.data-fetch.user_name}}!"
```

## 式システム

Probeは動的な値とテストのために式を使用します。式は`{{}}`構文を使って記述されます：

### テンプレート式

テンプレート式を使用して動的な値を挿入：

```yaml
- name: Greet User
  echo: "Hello {{outputs.previous-step.username}}!"
```

### テスト式

テスト式を使用して結果を検証：

```yaml
- name: Check API Response
  action: http
  with:
    url: https://api.example.com/status
  test: res.status == 200 && res.json.healthy == true
```

### 利用可能な変数

式では以下にアクセスできます：

- **`res`**: 現在のアクションからのレスポンス
- **`outputs`**: 前のステップ/ジョブからの出力
- **`env`**: 環境変数
- **カスタム関数**: `random_int()`、`random_str()`、`unixtime()`

## ファイルマージ

Probeは複数のYAMLファイルのマージをサポートしており、以下に有用です：

- ワークフローロジックから設定を分離
- ワークフロー間での共通定義の再利用
- 環境固有のオーバーライド

```bash
# ベースワークフローと環境固有の設定をマージ
probe base-workflow.yml,production-config.yml
```

ファイルは順序でマージされ、後のファイルが前のファイルの値を上書きします。

## エラーハンドリング

Probeはエラーを処理するためのいくつかのメカニズムを提供します：

### テストの失敗

テストが失敗すると、ステップは失敗としてマークされます：

```yaml
- name: Critical Check
  action: http
  with:
    url: https://critical-api.example.com
  test: res.status == 200  # If this fails, step fails
```

### 条件付き実行

`if`条件を使用して失敗を処理：

```yaml
- name: Primary Service Check
  id: primary
  action: http
  with:
    url: https://primary-api.example.com
  test: res.status == 200

- name: Fallback Check
  if: steps.primary.failed
  action: http
  with:
    url: https://backup-api.example.com
  test: res.status == 200
```

### 失敗時の継続

デフォルトでは、ジョブ実行は最初の失敗で停止します。この動作を変更できます：

```yaml
- name: Non-Critical Check
  continue_on_error: true
  action: http
  with:
    url: https://optional-service.example.com
  test: res.status == 200
```

## ベストプラクティス

### 1. 説明的な名前を使用

```yaml
# Good
- name: Check Production API Health
  action: http
  # ...

# Not so good  
- name: HTTP Check
  action: http
  # ...
```

### 2. 関連するステップをジョブにグループ化

```yaml
jobs:
  infrastructure-check:
    name: Infrastructure Health Check
    steps:
      - name: Check Database
        # ...
      - name: Check Cache
        # ...
      - name: Check Load Balancer
        # ...
```

### 3. データ共有に出力を使用

```yaml
- name: Fetch Configuration
  id: config
  action: http
  with:
    url: https://config-service.example.com
  outputs:
    database_url: res.json.database_url
    
- name: Test Database Connection
  action: http
  with:
    url: "{{outputs.config.database_url}}/health"
```

### 4. 意味のあるテスト条件を追加

```yaml
# Good - specific test conditions
test: res.status == 200 && res.json.status == "healthy" && res.time < 1000

# Not so good - generic test
test: res.status == 200
```

## 次のステップ

核となる概念を理解したので、次の内容に進むことができます：

1. **[最初のワークフローを作成](../your-first-workflow/)** - 実践的なワークフローを構築
2. **[CLI基本を学ぶ](../cli-basics/)** - コマンドライン・インターフェースをマスター
3. **[リファレンスを探る](../../reference/)** - 利用可能なすべてのオプションを深く理解

Probeをマスターするキーは実践です。シンプルなワークフローから始めて、概念に慣れてきたら徐々により複雑な自動化を構築していきましょう。