
-- +migrate Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50),
    email VARCHAR(100) UNIQUE
);
-- +migrate Down
