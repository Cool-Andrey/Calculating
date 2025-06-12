-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tasks
(
    id            UUID PRIMARY KEY,
    expression_id INTEGER,
    left_id       UUID,
    right_id      UUID,
    operation     VARCHAR(1),
    arg1          DOUBLE PRECISION,
    arg2          DOUBLE PRECISION,
    result        DOUBLE PRECISION,
    FOREIGN KEY (left_id) REFERENCES tasks (id) ON DELETE SET NULL,
    FOREIGN KEY (right_id) REFERENCES tasks (id) ON DELETE SET NULL
);
CREATE TABLE IF NOT EXISTS expressions
(
    id           SERIAL PRIMARY KEY,
    status       VARCHAR(9),
    result       VARCHAR(100),
    expression   TEXT,
    main_task_id UUID,
    FOREIGN KEY (main_task_id) REFERENCES tasks (id) ON DELETE SET NULL
);
CREATE TABLE IF NOT EXISTS users
(
    login         TEXT,
    password_hash TEXT
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS expressions;
DROP TABLE IF EXISTS tasks;
DROP TABLE If EXISTS users;
-- +goose StatementEnd
