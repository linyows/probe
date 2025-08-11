---
title: クイックスタート
description: 5分でProbeを使い始める
weight: 20
---

# クイックスタート

このガイドでは、数分でProbeを使い始めることができます。最初のワークフローを作成し、Probeの動作を確認しましょう。

## 前提条件

- システムにProbeが[インストール](./installation/)されていること
- YAML構文の基本的な理解

## 最初のワークフロー

ウェブサイトが正しく応答しているかをチェックする簡単なワークフローを作成しましょう。

### 1. ワークフローファイルの作成

`my-first-workflow.yml`という新しいファイルを作成します：

```yaml
name: My First Website Check
description: Check if my website is responding correctly

jobs:
- name: Website Health Check
  defaults:
    http:
      url: https://httpbin.org

  steps:
  - name: Check Homepage
    id: homepage
    uses: http
    with:
      get: /status/200
    test: res.code == 200
    outputs:
      code: res.code

  - name: Check API Endpoint
    id: api
    uses: http
    with:
      get: /json
    test: |
      res.code == 200 &&
      match_json(res.body, vars.expected)
    vars:
      expected:
        slideshow:
          title: Sample Slide Show
          author: Yours Truly
          date: date of publication
          slides:
          - title: Wake up to WonderWidgets!
            type: all
          - title: Overview
            type: all
            items:
            - "Why <em>WonderWidgets</em> are great"
            - "Who <em>buys</em> WonderWidgets"
    outputs:
      code: res.code

  - name: All Success Message
    skipif: outputs.homepage.code != 200 || outputs.api.code != 200
    uses: hello
    echo: "🎉 All checks passed! Website is healthy."
```

### 2. ワークフローの実行

Probe CLIを使用してワークフローを実行します：

```bash
$ probe my-first-workflow.yml
```

以下のような出力が表示されるはずです：

```
My First Website Check
Check if my website is responding correctly

⏺ Website Health Check (Completed in 5.33s)
  ⎿  0. ✔︎  Check Homepage
     1. ✔︎  Check API Endpoint
     2. ▲ All Success Message
           🎉 All checks passed! Website is healthy.

Total workflow time: 5.33s ✔︎ All jobs succeeded
```

### 3. 実行内容の理解

このワークフローが何をしているかを詳しく見てみましょう：

1. **ワークフローメタデータ**: `name`と`description`でワークフローの内容を説明
2. **ジョブ定義**: `name`と`defaults`でジョブ名とジョブ全体で適応するデフォルト設定をアクションごとに設定する
3. **HTTPアクション**: `uses` の指定により2つのステップでHTTPリクエストを行い、エンドポイントをテスト
4. **テストアサーション**: 各HTTPステップにレスポンスを検証する`test`を含む
5. **式評価**: `match_json`関数を使って変数`vars`で定義したjsonの一致検証
6. **変数代入**: ジョブ間で使用できる変数 `outputs` にアクション結果を代入
7. **ステップスキップ**: `skipif` によって条件にマッチするとステップの実行をスキップする
8. **エコー出力**: `echo` によって成功メッセージを表示

## 詳細出力の追加

何が起こっているかの詳細を見たい場合は、詳細フラグを付けて実行します：

```bash
$ probe my-first-workflow.yml --verbose
```

これにより、各HTTPリクエストとレスポンスの詳細情報が表示されます。

## 次のステップ

最初のワークフローを実行したので、次の内容に進むことができます：

1. **[核となる概念を学ぶ](../understanding-probe/)** - ワークフロー、ジョブ、ステップを理解する
2. **[最初のカスタムワークフローを作成](../your-first-workflow/)** - ニーズに合わせた独自のワークフローを構築する
3. **[CLIオプションを探る](../cli-basics/)** - 利用可能なすべてのコマンドライン・オプションを学ぶ

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

テストが失敗した場合は、詳細モード（`--verbose`）を使用して実際のレスポンスデータを確認し、テスト式をデバッグしてください。

さらに深く学ぶ準備はできましたか？[Probeを理解する](./understanding-probe/)で核となる概念を学びましょう。
