-- +goose Up
CREATE SCHEMA outbox;
-- +goose Down
DROP SCHEMA outbox;
