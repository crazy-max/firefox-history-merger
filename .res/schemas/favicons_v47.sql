BEGIN TRANSACTION;
CREATE TABLE moz_pages_w_icons ( id INTEGER PRIMARY KEY, page_url TEXT NOT NULL, page_url_hash INTEGER NOT NULL );
CREATE TABLE moz_icons_to_pages ( page_id INTEGER NOT NULL, icon_id INTEGER NOT NULL, PRIMARY KEY (page_id, icon_id), FOREIGN KEY (page_id) REFERENCES moz_pages_w_icons ON DELETE CASCADE, FOREIGN KEY (icon_id) REFERENCES moz_icons ON DELETE CASCADE ) WITHOUT ROWID ;
CREATE TABLE moz_icons ( id INTEGER PRIMARY KEY, icon_url TEXT NOT NULL, fixed_icon_url_hash INTEGER NOT NULL, width INTEGER NOT NULL DEFAULT 0, root INTEGER NOT NULL DEFAULT 0, color INTEGER, expire_ms INTEGER NOT NULL DEFAULT 0, data BLOB );
CREATE INDEX moz_pages_w_icons_urlhashindex ON moz_pages_w_icons (page_url_hash);
CREATE INDEX moz_icons_iconurlhashindex ON moz_icons (fixed_icon_url_hash);
COMMIT;
