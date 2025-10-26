-- +goose Up
-- +goose StatementBegin
CREATE TABLE playlist (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    owner_id INTEGER,
    FOREIGN KEY (owner_id) REFERENCES "user" (id) ON DELETE SET NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "playlist"
-- +goose StatementEnd
