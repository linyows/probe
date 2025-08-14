# エンコーディング関数

Base64、URLエンコーディングなどの関数群です。

## `base64`

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

## `base64decode`

base64文字列をデコードします。

**構文:** `base64decode(string)` または `string | base64decode`  
**戻り値:** String（デコード済み）

```yaml
outputs:
  decoded_token: base64decode(res.body.json.token)
  # base64トークンをプレーンテキストにデコード

test: base64decode(res.body.json.data) | contains("expected_value")
```

## `urlEncode`

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

## `urlDecode`

URLエンコードされた文字列をデコードします。

**構文:** `urlDecode(string)` または `string | urlDecode`  
**戻り値:** String（デコード済み）

```yaml
outputs:
  original_query: urlDecode(res.body.json.encoded_query)
  # "hello%20world"を"hello world"にデコード
```
