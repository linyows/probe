# データベースアクション

`db`アクションはMySQL、PostgreSQL、SQLiteデータベースでSQLクエリを実行し、包括的な結果処理とエラーレポートを提供します。

## 基本的な構文

```yaml
steps:
  - name: "Database Query"
    uses: db
    with:
      dsn: "mysql://user:password@localhost:3306/database"
      query: "SELECT * FROM users WHERE active = ?"
      params: [true]
    test: res.code == 0 && res.rows_affected > 0
```

## パラメータ

### `dsn` (必須)

**型:** String  
**説明:** 自動ドライバー検出付きのデータベース接続文字列  
**サポート:** テンプレート式

```yaml
# MySQL
vars:
  db_pass: "{{DB_PASS}}"

with:
  dsn: "mysql://user:password@localhost:3306/database"
  dsn: "mysql://{{vars.db_user}}:{{vars.db_pass}}@{{vars.db_host}}/{{vars.db_name}}"

# PostgreSQL
vars:
  pg_user: "{{PG_USER}}"
  pg_pass: "{{PG_PASS}}"
  pg_host: "{{PG_HOST}}"
  pg_db: "{{PG_DB}}"

with:
  dsn: "postgres://user:password@localhost:5432/database?sslmode=disable"
  dsn: "postgres://{{vars.pg_user}}:{{vars.pg_pass}}@{{vars.pg_host}}/{{vars.pg_db}}"

# SQLite
with:
  dsn: "file:./testdata/sqlite.db"
  dsn: "file:/absolute/path/to/database.db"
  dsn: "file:{{vars.data_dir}}/app.db"
```

### `query` (必須)

**型:** String  
**説明:** 実行するSQLクエリ  
**サポート:** テンプレート式と複数行文字列

```yaml
with:
  query: "SELECT * FROM users"
  query: "INSERT INTO logs (message, timestamp) VALUES (?, NOW())"
  query: |
    SELECT u.name, u.email, p.title 
    FROM users u 
    JOIN profiles p ON u.id = p.user_id 
    WHERE u.active = ? AND u.created_at > ?
```

### `params` (オプション)

**型:** 混合値の配列 (String, Number, Boolean)  
**説明:** プリペアドステートメント用のクエリパラメータ  
**サポート:** テンプレート式

```yaml
with:
  query: "SELECT * FROM users WHERE id = ? AND active = ?"
  params: [123, true, "{{vars.user_email}}"]
```

### `timeout` (オプション)

**型:** Duration  
**デフォルト:** `30s`  
**説明:** クエリ実行タイムアウト

```yaml
with:
  query: "SELECT COUNT(*) FROM large_table"
  timeout: "60s"
```

## レスポンスオブジェクト

データベースアクションは次のプロパティを持つ`res`オブジェクトを提供します：

| プロパティ | 型 | 説明 |
|----------|------|-------------|
| `code` | Integer | 操作結果 (0 = 成功, 1 = エラー) |
| `rows_affected` | Integer | クエリによって影響を受けた行数 |
| `rows` | Array | SELECTステートメントのクエリ結果（オブジェクトとして） |
| `error` | String | 操作が失敗した場合のエラーメッセージ |

## レスポンス例

### SELECTクエリレスポンス

```yaml
steps:
  - name: "Fetch Users"
    id: fetch-users
    uses: db
    with:
      dsn: "mysql://user:pass@localhost/db"
      query: "SELECT id, name, email FROM users WHERE active = ?"
      params: [true]
    test: res.code == 0 && res.rows_affected > 0
    outputs:
      user_count: res.rows_affected
      first_user_id: res.rows[0].id
      first_user_name: res.rows[0].name
```

### INSERT/UPDATEクエリレスポンス

```yaml
steps:
  - name: "Insert User"
    uses: db
    with:
      dsn: "postgres://user:pass@localhost/db"
      query: "INSERT INTO users (name, email) VALUES ($1, $2)"
      params: ["John Doe", "john@example.com"]
    test: res.code == 0 && res.rows_affected == 1
```

## データベース固有の機能

### MySQL例

```yaml
# 接続オプション付きMySQL
- name: "MySQL Query"
  uses: db
  with:
    dsn: "mysql://user:pass@tcp(localhost:3306)/database?charset=utf8mb4&parseTime=true"
    query: "SELECT VERSION() as mysql_version, NOW() as current_time"
  test: res.code == 0

# MySQLストアドプロシージャ
- name: "Call Procedure"
  uses: db
  with:
    dsn: "mysql://user:pass@localhost:3306/database"
    query: "CALL GetUsersByDepartment(?)"
    params: ["Engineering"]
  test: res.code == 0
```

### PostgreSQL例

```yaml
# JSON操作付きPostgreSQL
- name: "JSON Query"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost:5432/database?sslmode=disable"
    query: |
      SELECT name, data->>'role' as role, data->'preferences' as prefs
      FROM users 
      WHERE data ? 'role' AND data->>'role' = $1
    params: ["admin"]
  test: res.code == 0

# PostgreSQL配列操作
- name: "Array Query"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost:5432/database"
    query: "SELECT name FROM users WHERE tags && $1"
    params: ['{"admin","moderator"}']
  test: res.code == 0
```

### SQLite例

```yaml
# ファイル作成付きSQLite
- name: "SQLite Query"
  uses: db
  with:
    dsn: "file:./testdata/sqlite.db"
    query: |
      CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        email TEXT UNIQUE,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
      )
  test: res.code == 0

# インメモリデータベースSQLite
- name: "Memory Database"
  uses: db
  with:
    dsn: "file::memory:"
    query: "CREATE TABLE temp_data (id INTEGER, value TEXT)"
  test: res.code == 0
```

## 一般的なクエリパターン

### データ検証クエリ

```yaml
- name: "Check Data Integrity"
  uses: db
  with:
    dsn: "mysql://user:pass@localhost/db"
    query: |
      SELECT 
        COUNT(*) as total_users,
        COUNT(CASE WHEN active = 1 THEN 1 END) as active_users,
        COUNT(CASE WHEN email IS NULL THEN 1 END) as missing_emails
      FROM users
  test: |
    res.code == 0 && 
    res.rows[0].total_users > 0 && 
    res.rows[0].missing_emails == 0
```

### パフォーマンス監視

```yaml
- name: "Database Performance Check"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: |
      SELECT 
        schemaname, 
        tablename, 
        seq_scan, 
        seq_tup_read, 
        idx_scan, 
        idx_tup_fetch
      FROM pg_stat_user_tables 
      WHERE seq_scan > 1000
    timeout: "10s"
  test: res.code == 0
  outputs:
    high_seq_scan_tables: res.rows_affected
```

### バッチ操作

```yaml
- name: "Batch Insert"
  uses: db
  with:
    dsn: "mysql://user:pass@localhost/db"
    query: |
      INSERT INTO audit_log (action, table_name, record_id, timestamp) VALUES
      ('CREATE', 'users', 123, NOW()),
      ('UPDATE', 'profiles', 456, NOW()),
      ('DELETE', 'sessions', 789, NOW())
  test: res.code == 0 && res.rows_affected == 3
```

## セキュリティ機能

データベースアクションはいくつかのセキュリティ対策を実装しています：

- **プリペアドステートメント**: すべてのパラメータ化クエリでプリペアドステートメントを使用してSQLインジェクションを防止
- **接続文字列マスキング**: ログと出力でパスワードをマスク
- **タイムアウト保護**: 長時間実行されるクエリのハングを防止
- **ドライバー検証**: 承認されたデータベースドライバーのみをサポート
- **DSN検証**: 実行前に接続文字列形式を検証

## エラーハンドリング

一般的なエラーシナリオと処理パターン：

```yaml
- name: "Database with Error Handling"
  uses: db
  with:
    dsn: "mysql://user:pass@localhost/db"
    query: "SELECT * FROM users WHERE id = ?"
    params: [999999]
  test: |
    res.code == 0 ? true :
    res.error | contains("connection") ? false :
    res.error | contains("not found") ? true :
    false
  outputs:
    query_success: res.code == 0
    error_type: |
      {{res.code == 0 ? "none" :
        res.error | contains("connection") ? "connection" :
        res.error | contains("syntax") ? "syntax" :
        "unknown"}}
```

## トランザクション例

アクションは直接トランザクションをサポートしませんが、データベース固有のトランザクション構文を使用できます：

```yaml
# PostgreSQLトランザクション
- name: "Begin Transaction"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: "BEGIN"
  test: res.code == 0

- name: "Insert Data"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: "INSERT INTO users (name) VALUES ($1)"
    params: ["Test User"]
  test: res.code == 0

- name: "Commit Transaction"
  uses: db
  with:
    dsn: "postgres://user:pass@localhost/db"
    query: "COMMIT"
  test: res.code == 0
```
