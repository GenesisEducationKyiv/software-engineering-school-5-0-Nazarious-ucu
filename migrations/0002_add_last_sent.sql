-- +goose Up
ALTER TABLE subscriptions ADD COLUMN last_sent TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE subscriptions ADD COLUMN frequency TEXT NOT NULL default 'hourly' ;
-- +goose Down
