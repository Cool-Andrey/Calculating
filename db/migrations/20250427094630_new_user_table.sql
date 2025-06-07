-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS expressions(
    id SERIAL PRIMARY KEY,
    status VARCHAR(9),
    result VARCHAR(100),
    expression TEXT,
    ast_data JSONB,
    main_task_id INTEGER,
    FOREIGN KEY (main_task_id) REFERENCES tasks(id) ON DELETE SET NULL
);
CREATE TABLE IF NOT EXISTS users(
    login TEXT,
    password_hash TEXT
);
CREATE TABLE IF NOT EXISTS tasks(
    id SERIAL PRIMARY KEY,
    left_id INTEGER,
    right_id INTEGER,
    operation VARCHAR(1),
    arg1 DOUBLE PRECISION,
    arg2 DOUBLE PRECISION,
    result DOUBLE PRECISION,
    ready BOOLEAN,
    FOREIGN KEY (left_id) REFERENCES tasks(id) ON DELETE SET NULL,
    FOREIGN KEY (right_id) REFERENCES tasks(id) ON DELETE SET NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS expressions;
-- +goose StatementEnd
