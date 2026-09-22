# 配列・マップ関数

配列やマップを操作する関数群です。

| 関数 | 説明 |
|---|---|
| `len(v)` | 要素数 |
| `first(array)` / `last(array)` | 先頭／末尾の要素 |
| `take(array, n)` | 先頭n要素 |
| `reverse(array)` | 逆順のコピー |
| `uniq(array)` | 重複を除去 |
| `concat(a, b, ...)` | 配列を連結 |
| `flatten(array)` | ネストした配列を平坦化 |
| `sort(array)` / `sort(array, "desc")` | ソート |
| `sortBy(array, expr)` | 計算したキーでソート |
| `groupBy(array, expr)` | 計算したキーでマップにグループ化 |
| `filter(array, predicate)` | 条件に合う要素を抽出 |
| `map(array, expr)` | 各要素を変換 |
| `find(array, predicate)` / `findLast(array, predicate)` | 条件に合う最初／最後の要素 |
| `findIndex(array, predicate)` / `findLastIndex(array, predicate)` | 条件に合う要素の位置 |
| `count(array, predicate)` | 条件に合う要素数 |
| `all` / `any` / `one` / `none` | 述語に対する量化 |
| `reduce(array, expr, initial)` | 配列を1つの値に畳み込む |
| `keys(map)` / `values(map)` | マップのキー／値 |
| `get(map, key)` | キーに対応する値。無ければ`nil` |
| `toPairs(map)` / `fromPairs(array)` | マップとキー／値ペアの相互変換 |

述語の中では`#`が現在の要素を、`reduce`では`#acc`が累積値を表します。

```yaml
test: |
  len(res.body.items) > 0 &&
  all(res.body.items, #.status == "active")

outputs:
  names: "{{join(map(res.body.users, #.name), ', ')}}"
  total: "{{reduce(res.body.items, #acc + #.price, 0)}}"
```

要素の存在確認には`in`演算子も使えます。

```yaml
test: "admin" in map(res.body.users, #.role)
```
