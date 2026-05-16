CREATE TABLE "user" (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    is_superuser BOOLEAN NOT NULL DEFAULT FALSE,
    email VARCHAR(318) UNIQUE,
    password bytea NOT NULL,
    salt bytea NOT NULL,
    password_memory INTEGER NOT NULL DEFAULT 65536 CHECK (password_memory > 0),
    password_iterations INTEGER NOT NULL DEFAULT 3 CHECK (password_iterations > 0),
    password_parallelism INTEGER NOT NULL DEFAULT 0 CHECK (
        password_parallelism >= 0 AND password_parallelism <= 255
    ),
    password_key_length INTEGER NOT NULL DEFAULT 32 CHECK (password_key_length > 0),
    refresh_version INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
