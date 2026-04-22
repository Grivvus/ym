-- +goose Up
-- +goose StatementBegin
CREATE TABLE playlist (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    is_public BOOLEAN NOT NULL,
    owner_id INTEGER NOT NULL,
    FOREIGN KEY (owner_id) REFERENCES "user" (id) ON DELETE RESTRICT,
    UNIQUE(owner_id, name)
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "playlist";
-- +goose StatementEnd
