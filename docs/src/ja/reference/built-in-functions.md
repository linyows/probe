# 組み込み関数リファレンス

このページでは、Probe式とテンプレートで利用可能なすべての組み込み関数の包括的なドキュメントを提供します。これらの関数はテンプレート式（`{{}}`）とワークフロー全体のテスト条件で使用できます。

## 概要

組み込み関数は文字列操作、データフォーマット、数学的操作などのユーティリティを提供します。以下の式コンテキストで利用できます：

- 環境変数の値
- HTTPリクエストのURL、ヘッダー、ボディ
- テスト条件とアサーション
- 出力式
- 条件文（`if`式）

### 関数カテゴリ

- **[文字列関数](#文字列関数)** - 文字列操作とフォーマット
- **[日時関数](#日時関数)** - 日付と時刻のユーティリティ
- **[エンコーディング関数](#エンコーディング関数)** - Base64、URLエンコーディングなど
- **[数学関数](#数学関数)** - 数値演算
- **[ユーティリティ関数](#ユーティリティ関数)** - 汎用ユーティリティ
- **[JSON関数](#json関数)** - JSON操作とクエリ

## 関数の構文

関数はパイプ演算子（`|`）を使用してテンプレート式内で呼び出すか、直接的な関数呼び出しとして使用します：

```yaml
# パイプ構文（チェーンに推奨）
vars:
  user_name: "{{USER_NAME}}"
  base_url: "{{BASE_URL}}"
  path: "{{PATH}}"

value: "{{vars.user_name | upper | trim}}"

# 直接関数呼び出し
value: "{{upper(vars.user_name)}}"

# 混合使用
value: "{{vars.base_url}}/{{vars.path | lower | replace(' ', '-')}}"
```

## 文字列関数

### `upper`

文字列を大文字に変換します。

**構文:** `string | upper` または `upper(string)`  
**戻り値:** String

```yaml
vars:
  SERVICE_NAME: "{{SERVICE | upper}}"
  # SERVICEが"api-service"の場合、結果は"API-SERVICE"

test: res.body.json.status | upper == "SUCCESS"
```

### `lower`

文字列を小文字に変換します。

**構文:** `string | lower` または `lower(string)`  
**戻り値:** String

```yaml
with:
  url: "{{vars.BASE_URL}}/{{vars.ENDPOINT | lower}}"
  # ENDPOINTが"USERS"の場合、結果は"/users"

test: res.body.json.type | lower == "error"
```

### `trim`

文字列の両端から空白を削除します。

**構文:** `string | trim` または `trim(string)`  
**戻り値:** String

```yaml
vars:
  API_KEY: "{{RAW_API_KEY | trim}}"
  # 前後の空白を削除

with:
  headers:
    Authorization: "Bearer {{TOKEN | trim}}"
```

### `trimPrefix`

文字列の先頭からプレフィックスを削除します。

**構文:** `trimPrefix(string, prefix)`  
**戻り値:** String

```yaml
outputs:
  clean_url: trimPrefix(res.headers.Location, "https://")
  # "https://api.example.com/users"が"api.example.com/users"になる

vars:
  full_path: "{{FULL_PATH}}"
  clean_path: "{{trimPrefix(vars.full_path, '/api/v1')}}"
```

### `trimSuffix`

文字列の末尾からサフィックスを削除します。

**構文:** `trimSuffix(string, suffix)`  
**戻り値:** String

```yaml
outputs:
  base_name: trimSuffix(res.body.json.filename, ".json")
  # "data.json"が"data"になる

vars:
  container_name: "{{CONTAINER_NAME}}"
  service_name: "{{trimSuffix(vars.container_name, '-container')}}"
```

### `replace`

文字列内のすべての部分文字列を別の文字列で置換します。

**構文:** `replace(string, old, new)` または `string | replace(old, new)`  
**戻り値:** String

```yaml
with:
  url: "{{vars.TEMPLATE_URL | replace('{id}', outputs.user.id)}}"
  # "/users/{id}/profile"が"/users/123/profile"になる

vars:
  SAFE_NAME: "{{USER_INPUT | replace(' ', '_') | replace('-', '_')}}"
```

### `split`

文字列を区切り文字で分割して配列を返します。

**構文:** `split(string, delimiter)`  
**戻り値:** 文字列の配列

```yaml
vars:
  comma_list: "{{COMMA_LIST}}"

outputs:
  url_parts: split(res.headers.Location, "/")
  # "https://api.example.com/v1/users"が["https:", "", "api.example.com", "v1", "users"]になる
  
  first_part: split(vars.comma_list, ",")[0]
  # "apple,banana,cherry" -> 最初の要素は"apple"
```

### `join`

文字列の配列を区切り文字で結合します。

**構文:** `join(array, delimiter)`  
**戻り値:** String

```yaml
vars:
  # 前のステップからの配列があると仮定
  COMBINED: "{{join(outputs.data.items, ', ')}}"
  # ["apple", "banana", "cherry"]が"apple, banana, cherry"になる
```

### `contains`

文字列に部分文字列が含まれているかチェックします。

**構文:** `contains(string, substring)` または `string | contains(substring)`  
**戻り値:** Boolean

```yaml
test: |
  res.body.json.message | contains("success") &&
  res.headers["Content-Type"] | contains("application/json")

if: "{{vars.ENVIRONMENT}}" | contains("prod")
```

### `hasPrefix`

文字列がプレフィックスで始まるかチェックします。

**構文:** `hasPrefix(string, prefix)`  
**戻り値:** Boolean

```yaml
test: hasPrefix(res.headers.Location, "https://")

if: hasPrefix("{{vars.API_URL}}", "https://secure")
```

### `hasSuffix`

文字列がサフィックスで終わるかチェックします。

**構文:** `hasSuffix(string, suffix)`  
**戻り値:** Boolean

```yaml
test: hasSuffix(res.body.json.filename, ".json")

if: hasSuffix("{{vars.IMAGE_NAME}}", ":latest")
```

### `length`

文字列または配列の長さを返します。

**構文:** `length(value)` または `value | length`  
**戻り値:** Integer

```yaml
test: |
  res.body.json.items | length > 0 &&
  res.body.json.message | length > 10

outputs:
  item_count: res.body.json.data | length
  name_length: res.body.json.user.name | length
```

## 日時関数

### `now`

現在のUnixタイムスタンプを返します。

**構文:** `now()`  
**戻り値:** Integer（Unixタイムスタンプ）

```yaml
outputs:
  timestamp: now()
  # 1693574400のような値を返す

vars:
  REQUEST_TIME: "{{now()}}"
```

### `unixtime`

`now()`のエイリアス - 現在のUnixタイムスタンプを返します。

**構文:** `unixtime()`  
**戻り値:** Integer（Unixタイムスタンプ）

```yaml
with:
  headers:
    X-Timestamp: "{{unixtime()}}"
```

### `iso8601`

現在の時刻をISO 8601形式で返します。

**構文:** `iso8601()`  
**戻り値:** String（ISO 8601フォーマット）

```yaml
outputs:
  created_at: iso8601()
  # "2023-09-01T12:30:00Z"のような値を返す

with:
  body: |
    {
      "timestamp": "{{iso8601()}}",
      "event": "test_execution"
    }
```

### `date`

Goの時刻フォーマットレイアウトを使用して現在時刻をフォーマットします。

**構文:** `date(layout)`  
**戻り値:** String（フォーマットされた日付）

**一般的なレイアウト:**
- `2006-01-02` - 日付（YYYY-MM-DD）
- `15:04:05` - 時刻（HH:MM:SS）
- `2006-01-02 15:04:05` - 日時
- `Mon Jan 2 15:04:05 2006` - 完全フォーマット

```yaml
outputs:
  date_only: date("2006-01-02")
  # "2023-09-01"を返す
  
  time_only: date("15:04:05")
  # "12:30:45"を返す
  
  full_datetime: date("2006-01-02 15:04:05")
  # "2023-09-01 12:30:45"を返す

with:
  headers:
    X-Date: "{{date('Mon Jan 2 15:04:05 2006')}}"
    # "Fri Sep 1 12:30:45 2023"を返す
```

## エンコーディング関数

### `base64`

文字列をbase64にエンコードします。

**構文:** `base64(string)` または `string | base64`  
**戻り値:** String（base64エンコード済み）

```yaml
with:
  headers:
    Authorization: "Basic {{base64(\"{{vars.USERNAME}}\" + ':' + \"{{vars.PASSWORD}}\")}}"
    # "user:pass"を"dXNlcjpwYXNz"にエンコード

outputs:
  encoded_data: base64(res.body.text)
```

### `base64decode`

base64文字列をデコードします。

**構文:** `base64decode(string)` または `string | base64decode`  
**戻り値:** String（デコード済み）

```yaml
outputs:
  decoded_token: base64decode(res.body.json.token)
  # base64トークンをプレーンテキストにデコード

test: base64decode(res.body.json.data) | contains("expected_value")
```

### `urlEncode`

文字列をURLエンコード（パーセントエンコーディング）します。

**構文:** `urlEncode(string)` または `string | urlEncode`  
**戻り値:** String（URLエンコード済み）

```yaml
with:
  url: "{{vars.BASE_URL}}/search?q={{vars.SEARCH_TERM | urlEncode}}"
  # "hello world"を"hello%20world"にエンコード

outputs:
  encoded_param: urlEncode(res.body.json.user_input)
```

### `urlDecode`

URLエンコードされた文字列をデコードします。

**構文:** `urlDecode(string)` または `string | urlDecode`  
**戻り値:** String（デコード済み）

```yaml
outputs:
  original_query: urlDecode(res.body.json.encoded_query)
  # "hello%20world"を"hello world"にデコード
```

## 数学関数

### `add`

2つの数値を加算します。

**構文:** `add(a, b)`  
**戻り値:** Number

```yaml
outputs:
  total_time: add(res.time, 100)
  # レスポンス時間に100msを追加

test: add(res.body.json.count, res.body.json.pending) > 50
```

### `sub`

最初の数値から2番目の数値を減算します。

**構文:** `sub(a, b)`  
**戻り値:** Number

```yaml
outputs:
  time_diff: sub(unixtime(), res.body.json.created_at)
  # 秒単位の経過時間を計算

test: sub(res.time, outputs.baseline.time) < 500
```

### `mul`

2つの数値を乗算します。

**構文:** `mul(a, b)`  
**戻り値:** Number

```yaml
outputs:
  time_in_seconds: mul(res.time, 0.001)
  # ミリ秒を秒に変換

test: mul(res.body.json.price, res.body.json.quantity) <= 1000
```

### `div`

最初の数値を2番目の数値で除算します。

**構文:** `div(a, b)`  
**戻り値:** Number

```yaml
outputs:
  average_time: div(res.body.json.total_time, res.body.json.request_count)
  # 平均を計算

test: div(res.body.json.success_count, res.body.json.total_count) > 0.95
```

### `mod`

除算の余りを返します。

**構文:** `mod(a, b)`  
**戻り値:** Number

```yaml
test: mod(res.body.json.id, 2) == 0
# IDが偶数かチェック

if: mod(unixtime(), 3600) < 60
# 毎時の最初の1分間のみ実行
```

### `round`

数値を最も近い整数に四捨五入します。

**構文:** `round(number)`  
**戻り値:** Integer

```yaml
outputs:
  rounded_time: round(div(res.time, 1000))
  # 秒に変換して四捨五入

test: round(res.body.json.score) >= 8
```

### `floor`

数値を最も近い整数に切り下げます。

**構文:** `floor(number)`  
**戻り値:** Integer

```yaml
outputs:
  time_seconds: floor(div(res.time, 1000))
  # ミリ秒を秒に変換（切り下げ）
```

### `ceil`

数値を最も近い整数に切り上げます。

**構文:** `ceil(number)`  
**戻り値:** Integer

```yaml
outputs:
  min_requests: ceil(mul(res.body.json.users, 1.5))
  # 必要な最小リクエスト数を計算（切り上げ）
```

## ユーティリティ関数

### `uuid`

ランダムなUUID（バージョン4）を生成します。

**構文:** `uuid()`  
**戻り値:** String（UUID）

```yaml
with:
  headers:
    X-Request-ID: "{{uuid()}}"
    # "f47ac10b-58cc-4372-a567-0e02b2c3d479"のような値を生成

outputs:
  correlation_id: uuid()
```

### `random`

0から指定された最大値（排他的）までのランダムな整数を生成します。

**構文:** `random(max)`  
**戻り値:** Integer

```yaml
with:
  url: "{{vars.BASE_URL}}/test?seed={{random(1000)}}"
  # 0-999のランダム数値を生成

outputs:
  random_delay: random(5000)
  # 0-4999のランダム数値（遅延用ミリ秒）
```

### `default`

入力が空またはnullの場合にデフォルト値を返します。

**構文:** `default(value, default_value)` または `value ?? default_value`  
**戻り値:** 任意の型

```yaml
vars:
  request_timeout: "{{REQUEST_TIMEOUT ?? '30s'}}"
  custom_timeout: "{{CUSTOM_TIMEOUT}}"

with:
  timeout: "{{default(vars.custom_timeout, '60s')}}"
```

### `coalesce`

リストから最初の空でない値を返します。

**構文:** `coalesce(value1, value2, value3, ...)`  
**戻り値:** 任意の型

```yaml
vars:
  custom_api_url: "{{CUSTOM_API_URL}}"
  default_api_url: "{{DEFAULT_API_URL}}"
  step_timeout: "{{STEP_TIMEOUT}}"
  job_timeout: "{{JOB_TIMEOUT}}"
  api_url: "{{coalesce(vars.custom_api_url, vars.default_api_url, 'https://api.example.com')}}"

with:
  timeout: "{{coalesce(vars.step_timeout, vars.job_timeout, '30s')}}"
```

## JSON関数

### `tojson`

値をJSON文字列に変換します。

**構文:** `tojson(value)` または `value | tojson`  
**戻り値:** String（JSON）

```yaml
with:
  body: "{{outputs.user_data | tojson}}"
  # オブジェクトをJSON文字列に変換

outputs:
  json_response: tojson(res.body.json)
```

### `fromjson`

JSON文字列をオブジェクトに解析します。

**構文:** `fromjson(json_string)` または `json_string | fromjson`  
**戻り値:** Object

```yaml
outputs:
  parsed_data: fromjson(res.body.text)
  # JSON文字列をオブジェクトに解析

test: fromjson(res.body.json.metadata).version == "1.0"
```

### `jsonpath`

JSONPathの式を使用してJSONから値を抽出します。

**構文:** `jsonpath(json_object, path)`  
**戻り値:** 任意の型

```yaml
outputs:
  user_names: jsonpath(res.body.json, "$.users[*].name")
  # 配列からすべてのユーザー名を抽出
  
  first_email: jsonpath(res.body.json, "$.users[0].email")
  # 最初のユーザーのメールを取得

test: jsonpath(res.body.json, "$.status.code") == 200
```

### `keys`

オブジェクトのキーを配列として返します。

**構文:** `keys(object)`  
**戻り値:** 文字列の配列

```yaml
outputs:
  header_names: keys(res.headers)
  # すべてのレスポンスヘッダー名を取得
  
  json_fields: keys(res.body.json)
  # すべてのJSONオブジェクトキーを取得

test: keys(res.body.json) | length > 5
# レスポンスが5つ以上のフィールドを持つことを確認
```

### `values`

オブジェクトの値を配列として返します。

**構文:** `values(object)`  
**戻り値:** Array

```yaml
outputs:
  header_values: values(res.headers)
  # すべてのレスポンスヘッダー値を取得
  
  all_user_names: values(res.body.json.users)
  # usersオブジェクトからすべての値を取得
```

## 高度な関数使用法

### 関数チェーン

パイプ演算子を使用して関数をチェーンできます：

```yaml
vars:
  CLEAN_NAME: "{{RAW_NAME | trim | lower | replace(' ', '-')}}"
  # チェーン: 空白をトリム → 小文字 → スペースをハイフンに置換

with:
  url: "{{vars.BASE_URL | trimSuffix('/') | replace('http://', 'https://')}}/api"
  # チェーン: 末尾スラッシュ削除 → HTTPSに強制 → パス追加
```

### 条件付き関数使用

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

### 複雑なデータ操作

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

## パフォーマンスの考慮事項

### 関数パフォーマンス

- **文字列関数:** 一般的に高速ですが、過度なチェーンは避ける
- **日時関数:** `now()`と`iso8601()`は最小限のオーバーヘッド
- **JSON関数:** `jsonpath()`は大きなオブジェクトで遅くなる可能性
- **数学関数:** 単純な操作では非常に高速

### ベストプラクティス

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