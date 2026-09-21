# Helloアクション

`hello` アクションは成功するだけで何もしないアクションです。プレースホルダーとして、`echo` だけを目的とするステップとして、あるいは式の動作確認に使います。

## 基本的な構文

```yaml
steps:
  - name: Report
    uses: hello
    echo: "Checked {{outputs.health.endpoint}}"
```

## パラメータ

このアクション固有のパラメータはありません。`with` に渡した内容はそのまま `res` に返るため、計算した値を outputs として公開したいときに便利です。

```yaml
steps:
  - name: Build a summary
    id: summary
    uses: hello
    with:
      run_id: "{{vars.run_id}}"
      checked_at: "{{now().Format('2006-01-02T15:04:05Z07:00')}}"
    outputs:
      run_id: res.run_id
      checked_at: res.checked_at
```

## レスポンスオブジェクト

| プロパティ | 型 | 説明 |
|---|---|---|
| `res.<key>` | 任意 | `with` に渡したキーがそのまま入ります |
| `res.status` | Integer | 常に `0` |
| `status` | Integer | 常に `0` |

## 関連項目

- **[変数](./variables)** - ステップで使える変数
- **[YAML設定](../yaml-configuration)** - ステップのプロパティ
