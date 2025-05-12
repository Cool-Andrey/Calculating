-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS expressions(
    id SERIAL PRIMARY KEY,
    status VARCHAR(9),
    result VARCHAR(100),
    expression TEXT,
    ast_data JSONB
);
CREATE TABLE IF NOT EXISTS users(
    login TEXT,
    password_hash TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS expressions;
-- +goose StatementEnd
