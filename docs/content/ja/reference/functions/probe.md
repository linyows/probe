# Probe固有関数

Probeがexprに追加している関数です。

## `match_json`

2つのオブジェクトを厳密に比較します。
キーと値が完全に一致する必要があり、片方にだけ存在するキーがあれば一致しません。

**構文:** `match_json(src, target)`
**戻り値:** Boolean

```yaml
test: match_json(res.body, {"status": "ok", "count": 3})
```

## `diff_json`

`match_json`と同じ基準で比較し、差分を文字列で返します。
一致する場合は`No diff`を返します。

**構文:** `diff_json(src, target)`
**戻り値:** String

```yaml
echo: "{{diff_json(res.body, vars.expected)}}"
```

## `parse_json`

JSON文字列をオブジェクトに解析します。

**構文:** `parse_json(string)`
**戻り値:** 任意の型

```yaml
outputs:
  meta: "{{parse_json(res.body.metadata)}}"

test: parse_json(res.body.metadata).version == "1.0"
```

## `parse_int`

文字列または数値を64ビット整数に変換します。
10進整数として解釈できない文字列はエラーになります。

**構文:** `parse_int(value)`
**戻り値:** Integer

```yaml
test: parse_int(res.headers["Content-Length"]) > 0
```

## `parse_float`

文字列を浮動小数点数に変換します。

**構文:** `parse_float(string)`
**戻り値:** Float

```yaml
test: parse_float(res.body.score) >= 8.5
```

## `encode_base64`

文字列を標準Base64でエンコードします。

**構文:** `encode_base64(string)`
**戻り値:** String

```yaml
with:
  headers:
    Authorization: "Basic {{encode_base64(vars.user + ':' + vars.pass)}}"
```

## `decode_base64`

標準Base64の文字列をデコードします。

**構文:** `decode_base64(string)`
**戻り値:** String

```yaml
outputs:
  payload: "{{decode_base64(res.body.data)}}"
```

## `unixtime`

現在時刻をUnixタイムスタンプ（秒）で返します。

**構文:** `unixtime()`
**戻り値:** Integer

```yaml
with:
  headers:
    X-Timestamp: "{{unixtime()}}"
```

## `random_int`

`[0, n)`の範囲の乱数を返します。
`n`は正の整数である必要があります。

**構文:** `random_int(n)`
**戻り値:** Integer

```yaml
with:
  url: "{{vars.base_url}}/test?seed={{random_int(1000)}}"
```

## `random_str`

指定した長さのランダムな英数字（`[a-zA-Z0-9]`）を返します。
長さの上限は1000000文字です。

**構文:** `random_str(length)`
**戻り値:** String

```yaml
vars:
  email: "user-{{random_str(8)}}@example.com"
```
