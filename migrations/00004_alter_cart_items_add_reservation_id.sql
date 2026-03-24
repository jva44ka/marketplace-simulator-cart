-- +goose Up
-- +goose StatementBegin
ALTER TABLE cart_items ADD COLUMN reservation_id BIGINT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE cart_items DROP COLUMN reservation_id;
-- +goose StatementEnd
