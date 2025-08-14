# 日時関数

日付と時刻のユーティリティ関数群です。

## `now`

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

## `unixtime`

`now()`のエイリアス - 現在のUnixタイムスタンプを返します。

**構文:** `unixtime()`  
**戻り値:** Integer（Unixタイムスタンプ）

```yaml
with:
  headers:
    X-Timestamp: "{{unixtime()}}"
```

## `iso8601`

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

## `date`

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
