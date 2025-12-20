-- +goose Up
-- +goose StatementBegin
CREATE TABLE products(
    sku     BIGINT    PRIMARY KEY,
    price   NUMERIC   NOT NULL,
    name    TEXT      NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE products;
-- +goose StatementEnd