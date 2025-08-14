# JSON関数

JSON操作とクエリの関数群です。

## `tojson`

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

## `fromjson`

JSON文字列をオブジェクトに解析します。

**構文:** `fromjson(json_string)` または `json_string | fromjson`  
**戻り値:** Object

```yaml
outputs:
  parsed_data: fromjson(res.body.text)
  # JSON文字列をオブジェクトに解析

test: fromjson(res.body.json.metadata).version == "1.0"
```

## `jsonpath`

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

## `keys`

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

## `values`

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
