# 文字列関数

文字列操作とフォーマット用の関数群です。

## `upper`

文字列を大文字に変換します。

**構文:** `string | upper` または `upper(string)`  
**戻り値:** String

```yaml
vars:
  SERVICE_NAME: "{{SERVICE | upper}}"
  # SERVICEが"api-service"の場合、結果は"API-SERVICE"

test: res.body.json.status | upper == "SUCCESS"
```

## `lower`

文字列を小文字に変換します。

**構文:** `string | lower` または `lower(string)`  
**戻り値:** String

```yaml
with:
  url: "{{vars.BASE_URL}}/{{vars.ENDPOINT | lower}}"
  # ENDPOINTが"USERS"の場合、結果は"/users"

test: res.body.json.type | lower == "error"
```

## `trim`

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

## `trimPrefix`

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

## `trimSuffix`

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

## `replace`

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

## `split`

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

## `join`

文字列の配列を区切り文字で結合します。

**構文:** `join(array, delimiter)`  
**戻り値:** String

```yaml
vars:
  # 前のステップからの配列があると仮定
  COMBINED: "{{join(outputs.data.items, ', ')}}"
  # ["apple", "banana", "cherry"]が"apple, banana, cherry"になる
```

## `contains`

文字列に部分文字列が含まれているかチェックします。

**構文:** `contains(string, substring)` または `string | contains(substring)`  
**戻り値:** Boolean

```yaml
test: |
  res.body.json.message | contains("success") &&
  res.headers["Content-Type"] | contains("application/json")

if: "{{vars.ENVIRONMENT}}" | contains("prod")
```

## `hasPrefix`

文字列がプレフィックスで始まるかチェックします。

**構文:** `hasPrefix(string, prefix)`  
**戻り値:** Boolean

```yaml
test: hasPrefix(res.headers.Location, "https://")

if: hasPrefix("{{vars.API_URL}}", "https://secure")
```

## `hasSuffix`

文字列がサフィックスで終わるかチェックします。

**構文:** `hasSuffix(string, suffix)`  
**戻り値:** Boolean

```yaml
test: hasSuffix(res.body.json.filename, ".json")

if: hasSuffix("{{vars.IMAGE_NAME}}", ":latest")
```

## `length`

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
