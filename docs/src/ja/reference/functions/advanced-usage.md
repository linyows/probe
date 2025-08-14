# 高度な関数使用法

このページでは、関数の高度な使い方やパフォーマンスについて記載しています。

## 関数チェーン

パイプ演算子を使用して関数をチェーンできます：

```yaml
vars:
  CLEAN_NAME: "{{RAW_NAME | trim | lower | replace(' ', '-')}}"
  # チェーン: 空白をトリム → 小文字 → スペースをハイフンに置換

with:
  url: "{{vars.BASE_URL | trimSuffix('/') | replace('http://', 'https://')}}/api"
  # チェーン: 末尾スラッシュ削除 → HTTPSに強制 → パス追加
```

## 条件付き関数使用

関数は条件式で使用できます：

```yaml
test: |
  res.code == 200 &&
  res.body.json.items | length > 0 &&
  contains(res.body.json.status | upper, "SUCCESS")

if: |
  "{{vars.ENVIRONMENT}}" == "production" ||
  ("{{vars.ENVIRONMENT}}" == "staging" && contains("{{vars.BRANCH_NAME}}", "release"))
```

## 複雑なデータ操作

```yaml
vars:
  base_url: "{{BASE_URL}}"
  api_version: "{{API_VERSION}}"
  resource: "{{RESOURCE}}"

outputs:
  # ユーザーデータの抽出とフォーマット
  formatted_users: |
    {{range res.body.json.users}}
      {{.name | upper}}: {{.email | lower}}
    {{end}}

  # メトリクス計算
  success_rate: |
    {{div(mul(res.body.json.successful_requests, 100), res.body.json.total_requests)}}%

  # URL生成
  api_endpoints: |
    {{vars.base_url | trimSuffix('/')}}/{{vars.api_version}}/{{vars.resource | lower}}
```

### エラーセーフな関数使用

デフォルト値とnullチェックを使用して関数をより堅牢にします：

```yaml
vars:
  custom_url: "{{CUSTOM_URL}}"
  default_url: "{{DEFAULT_URL}}"

outputs:
  safe_length: "{{res.body.json.items ?? [] | length}}"
  # itemsがnullの場合は空配列を使用

  safe_name: "{{res.body.json.user.name ?? 'Unknown' | upper}}"  
  # 名前が欠けている場合はデフォルトを提供

  safe_url: "{{coalesce(vars.custom_url, vars.default_url, 'https://fallback.com')}}"
  # 複数のフォールバックオプション
```

## 関数パフォーマンス

- **文字列関数:** 一般的に高速ですが、過度なチェーンは避ける
- **日時関数:** `now()`と`iso8601()`は最小限のオーバーヘッド
- **JSON関数:** `jsonpath()`は大きなオブジェクトで遅くなる可能性
- **数学関数:** 単純な操作では非常に高速

## ベストプラクティス

```yaml
# 良い: 一度計算して再利用
vars:
  base_url: "{{BASE_URL}}"
  current_time: "{{iso8601()}}"
  api_url: "{{vars.base_url | trimSuffix('/')}}"

jobs:
- name: test
  steps:
    - name: "Use precomputed values"
      with:
        url: "{{vars.api_url}}/health"
        headers:
          X-Timestamp: "{{vars.current_time}}"

# 避ける: 各ステップで再計算
    - name: "Inefficient"
      with:
        url: "{{vars.base_url | trimSuffix('/')}}/health"  # 再計算
        headers:
          X-Timestamp: "{{iso8601()}}"  # 異なるタイムスタンプ
```

## 関連項目

- **[YAML設定](../yaml-configuration/)** - YAML設定での関数使用
- **[アクションリファレンス](../actions-reference/)** - アクションパラメータでの関数
- **[概念: 式とテンプレート](../../concepts/expressions-and-templates/)** - 式言語ガイド
- **[ハウツー: 動的設定](../../how-tos/environment-management/)** - 実用的な関数使用
