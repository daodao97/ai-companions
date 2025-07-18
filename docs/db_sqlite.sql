CREATE TABLE companion_conversation (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid INTEGER NOT NULL DEFAULT 0,
  conversation_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  is_deleted INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT (datetime('now', 'localtime')),
  updated_at DATETIME NOT NULL DEFAULT (datetime('now', 'localtime'))
);

CREATE TABLE companion_message (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  conversation_id TEXT NOT NULL DEFAULT '',
  message_id TEXT NOT NULL DEFAULT '',
  content TEXT DEFAULT NULL,
  created_at DATETIME NOT NULL DEFAULT (datetime('now', 'localtime'))
);


-- 一行命令构建 sqlite.db: sqlite3 companion.db < docs/db_sqlite.sql
