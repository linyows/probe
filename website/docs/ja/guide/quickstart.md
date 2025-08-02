---
title: クイックスタート
description: 5分でProbeを使い始める
weight: 20
---

# クイックスタート

このガイドでは、数分でProbeを使い始めることができます。最初のワークフローを作成し、Probeの動作を確認しましょう。

## 前提条件

- システムに[Probeがインストール](../installation/)されていること
- YAML構文の基本的な理解

## 最初のワークフロー

ウェブサイトが正しく応答しているかをチェックする簡単なワークフローを作成しましょう。

### 1. ワークフローファイルの作成

`my-first-workflow.yml`という新しいファイルを作成します：

```yaml
name: My First Website Check
description: Check if my website is responding correctly

jobs:
  health-check:
    name: Website Health Check
    steps:
      - name: Check Homepage
        action: http
        with:
          url: https://httpbin.org/status/200
          method: GET
        test: res.status == 200
        
      - name: Check API Endpoint
        action: http
        with:
          url: https://httpbin.org/json
          method: GET
        test: res.status == 200 && res.json.slideshow != null
        
      - name: Success Message
        echo: "✅ All checks passed! Website is healthy."
```

### 2. ワークフローの実行

Probe CLIを使用してワークフローを実行します：

```bash
probe my-first-workflow.yml
```

以下のような出力が表示されるはずです：

```
My First Website Check
Check if my website is responding correctly

⏺ Website Health Check (Completed in 1.23s)
  ⎿ 0. ✔︎  Check Homepage (245ms)
     1. ✔︎  Check API Endpoint (189ms)
     2. ✔︎  Success Message
       ✅ All checks passed! Website is healthy.

Total workflow time: 1.23s ✔︎ All jobs succeeded
```

### 3. 実行内容の理解

このワークフローが何をしているかを詳しく見てみましょう：

1. **ワークフローメタデータ**: `name`と`description`でワークフローの内容を説明
2. **ジョブ定義**: `health-check`ジョブに実行するステップを含む
3. **HTTPアクション**: 2つのステップでHTTPリクエストを行い、エンドポイントをテスト
4. **テストアサーション**: 各HTTPステップにレスポンスを検証する`test`を含む
5. **エコーステップ**: すべてのチェックが成功した時に成功メッセージを表示

## 詳細出力の追加

何が起こっているかの詳細を見たい場合は、詳細フラグを付けて実行します：

```bash
probe -v my-first-workflow.yml
```

これにより、各HTTPリクエストとレスポンスの詳細情報が表示されます。

## 次のステップ

最初のワークフローを実行したので、次の内容に進むことができます：

1. **[核となる概念を学ぶ](../understanding-probe/)** - ワークフロー、ジョブ、ステップを理解する
2. **[最初のカスタムワークフローを作成](../your-first-workflow/)** - ニーズに合わせた独自のワークフローを構築する
3. **[CLIオプションを探る](../cli-basics/)** - 利用可能なすべてのコマンドライン・オプションを学ぶ

## 一般的な次のステップ

### 複数環境のチェック

複数の環境をチェックするようにワークフローを修正：

```yaml
name: Multi-Environment Health Check
description: Check health across multiple environments

jobs:
  production-check:
    name: Production Health Check
    steps:
      - name: Check Production API
        action: http
        with:
          url: https://api.myapp.com/health
          method: GET
        test: res.status == 200

  staging-check:
    name: Staging Health Check
    steps:
      - name: Check Staging API
        action: http
        with:
          url: https://staging-api.myapp.com/health
          method: GET
        test: res.status == 200
```

### エラーハンドリングの追加

失敗を適切に処理するステップを含める：

```yaml
- name: Check Service
  action: http
  with:
    url: https://api.example.com/status
    method: GET
  test: res.status == 200
  
- name: Fallback Check
  if: steps.previous.failed
  action: http
  with:
    url: https://backup-api.example.com/status
    method: GET
  test: res.status == 200
```

## トラブルシューティング

### ワークフローファイルが見つからない

```
[ERROR] workflow is required
```

YAMLファイルへの正しいパスを指定していることを確認してください。

### 無効なYAML構文

```
[ERROR] yaml: line 5: mapping values are not allowed in this context
```

YAML構文を確認してください。よくある問題：
- 不正なインデント（タブではなくスペースを使用）
- キーの後のコロンの欠落
- 特殊文字を含む引用符なしの文字列

### テストの失敗

テストが失敗した場合は、詳細モード（`-v`）を使用して実際のレスポンスデータを確認し、テスト式をデバッグしてください。

さらに深く学ぶ準備はできましたか？[Understanding Probe](../understanding-probe/)で核となる概念を学びましょう。