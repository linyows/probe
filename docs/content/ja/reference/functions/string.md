# 文字列関数

文字列操作とフォーマット用の関数群です。

| 関数 | 説明 |
|---|---|
| `upper(s)` | 大文字に変換 |
| `lower(s)` | 小文字に変換 |
| `trim(s)` / `trim(s, chars)` | 両端の空白、または指定文字を削除 |
| `trimPrefix(s, prefix)` | 先頭のプレフィックスを削除 |
| `trimSuffix(s, suffix)` | 末尾のサフィックスを削除 |
| `replace(s, old, new)` | すべての出現箇所を置換 |
| `split(s, sep)` / `split(s, sep, n)` | 区切り文字で分割して配列を返す |
| `splitAfter(s, sep)` | 区切り文字を各要素に残したまま分割 |
| `join(array, sep)` | 配列を区切り文字で結合 |
| `repeat(s, n)` | 文字列をn回繰り返す |
| `indexOf(s, sub)` | 最初に出現する位置。見つからなければ`-1` |
| `lastIndexOf(s, sub)` | 最後に出現する位置。見つからなければ`-1` |
| `hasPrefix(s, prefix)` | プレフィックスの判定 |
| `hasSuffix(s, suffix)` | サフィックスの判定 |
| `len(s)` | バイト長 |

```yaml
vars:
  host: "{{trimPrefix(API_URL, 'https://')}}"
  slug: "{{replace(lower(TITLE), ' ', '-')}}"

test: hasSuffix(res.body.filename, ".json")
```

## 演算子で書く判定

部分一致や正規表現は関数ではなく演算子です。
`contains()`や`matches()`という関数はありません。

```yaml
test: |
  res.headers["Content-Type"] contains "application/json" &&
  res.body.name startsWith "user-" &&
  res.body.id matches "^[0-9a-f]{8}"
```

長さの取得は`len()`を使います。`length()`という関数はありません。

```yaml
test: len(res.body.items) > 0
```
