# 変数

すべてのアクションにおいて、アクション実行後、以下の変数がステップ内のテンプレート式で利用可能になります：

- **`req`** - アクションに送信された元のリクエスト情報
- **`res`** - アクションから返されたレスポンスデータ
- **`rt.dulation`** - レスポンス時間（durationの文字列型、例: 12ms）
- **`rt.sec`** - レスポンス時間（秒のnumber型、例: 0.012456）

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
```
