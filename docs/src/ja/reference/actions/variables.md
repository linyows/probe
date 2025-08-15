# 変数

すべてのアクションにおいて、アクション実行後、以下の変数がステップ内のテンプレート式で利用可能になります：

- **`req`** - アクションに送信された元のリクエスト情報
- **`res`** - アクションから返されたレスポンスデータ
- **`rt.dulation`** - レスポンス時間（durationの文字列型、例: 12ms）
- **`rt.sec`** - レスポンス時間（秒のnumber型、例: 0.012456）
- **`status`** - アクションの実行ステータス（0 = 成功、1 = 失敗）

## reqとres

アクションによって変数のreqとresに入れられるデータは異なります。
いくつかの例を見ていきましょう。

### HTTP

HTTPアクションの場合は以下の通りです。

```yaml
echo: |
  Request:
    URL:     {{ req.url }}
    Method:  {{ req.method }}
    Headers: {{ req.headers }}
    Body:    {{ req.body }}
  Response:
    Code:    {{ res.code }}
    Status:  {{ res.status }}
    Headers: {{ res.headers }}
    Body:    {{ res.body }}
  Action Status: {{ status }}    # 0 = 成功, 1 = 失敗
  Response Time: {{ rt.duration }}
```

## status フィールド

`status` フィールドは全てのアクションで共通して利用可能な実行結果ステータスです：

- **`0`** - 成功（アクションが正常に完了）
- **`1`** - 失敗（エラーが発生またはアクション固有の失敗条件）

### アクション別のステータス条件

#### HTTP アクション
```yaml
# HTTP ステータスコード 200-299 の場合: status = 0
# その他のステータスコードの場合: status = 1
test: status == 0 && res.code == 200
```

#### Shell アクション  
```yaml
# 終了コード 0 の場合: status = 0
# 終了コード 0 以外の場合: status = 1
test: status == 0 && res.code == 0
```

#### DB アクション
```yaml
# クエリ成功の場合: status = 0
# エラー発生の場合: status = 1
test: status == 0 && res.code == 0
```

## リトライ機能でのstatus利用

リトライ機能は `status` フィールドを使用して成功/失敗を判定します：

```yaml
- name: "Service Health Check with Retry"
  uses: http
  with:
    url: "http://localhost:8080/health"
  retry:
    max_attempts: 10
    interval: "2s"
  test: status == 0  # status が 0 になるまでリトライ
```

詳細なリトライ設定については[アクションガイド](../../guide/concepts/actions/#リトライ機能)を参照してください。
