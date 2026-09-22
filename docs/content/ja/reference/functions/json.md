# JSON・エンコーディング関数

JSON と Base64 を扱う関数群です。

| 関数 | 説明 |
|---|---|
| `toJSON(v)` | 値をインデント付き JSON 文字列に変換 |
| `fromJSON(s)` | JSON 文字列を解析 |
| `toBase64(s)` | Base64 にエンコード |
| `fromBase64(s)` | Base64 をデコード |

関数名はキャメルケースです。
`tojson` や `base64` という名前の関数はありません。

```yaml
with:
  body: "{{toJSON(outputs.setup.user)}}"

outputs:
  version: "{{fromJSON(res.body).version}}"
```

Probe 固有の [`parse_json`](/ja/reference/functions/probe#parse-json)、[`encode_base64`](/ja/reference/functions/probe#encode-base64)、[`decode_base64`](/ja/reference/functions/probe#decode-base64) は同じ用途に使えます。
こちらは引数の型と長さを検証し、Probe のエラーメッセージを返します。

JSON の比較には [`match_json`](/ja/reference/functions/probe#match-json) と [`diff_json`](/ja/reference/functions/probe#diff-json) があります。

```yaml
test: match_json(res.body, vars.expected)
echo: "{{diff_json(res.body, vars.expected)}}"
```

JSONPath を扱う関数はありません。
値の取り出しはフィールドアクセスと配列・マップ関数を組み合わせます。

```yaml
outputs:
  first_email: "{{res.body.users[0].email}}"
  active_names: "{{map(filter(res.body.users, #.active), #.name)}}"
```
