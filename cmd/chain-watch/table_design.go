package main

const (
	tb_sync_sql = `
CREATE TABLE IF NOT EXISTS sync_proc (
	height INTEGER NOT NULL PRIMARY KEY,
	created_at DATETIME NOT NULL DEFAULT (datetime('now', 'localtime')),
	updated_at DATETIME NOT NULL DEFAULT (datetime('now', 'localtime'))
);
`
)
