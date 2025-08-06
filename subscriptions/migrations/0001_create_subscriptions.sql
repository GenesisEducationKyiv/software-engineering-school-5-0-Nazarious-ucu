-- +goose Up
CREATE TABLE subscriptions (
                               id            INTEGER PRIMARY KEY AUTOINCREMENT,
                               email         TEXT NOT NULL,
                               city          TEXT NOT NULL,
                               token         TEXT NOT NULL,
                               confirmed     INTEGER NOT NULL DEFAULT 0,
                               unsubscribed  INTEGER NOT NULL DEFAULT 0,
                               created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose Down
DROP TABLE subscriptions;
