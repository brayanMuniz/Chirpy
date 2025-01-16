-- in a "real" init file I would make this a template and to render it out with the env variables

-- Create user if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'brayanmuniz') THEN
        CREATE USER brayanmuniz WITH PASSWORD 'password';
    END IF;
END $$;

-- Create database if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'chirpy') THEN
        CREATE DATABASE chirpy;
    END IF;
END $$;

GRANT ALL PRIVILEGES ON DATABASE chirpy TO brayanmuniz;
\connect chirpy;

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    email TEXT NOT NULL UNIQUE,
    hashed_password TEXT NOT NULL DEFAULT 'unset',
    is_chirpy_red BOOL NOT NULL DEFAULT FALSE
);

-- Create chirps table
CREATE TABLE IF NOT EXISTS chirps (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    body VARCHAR(140) NOT NULL
);

-- Create refresh_tokens table
CREATE TABLE IF NOT EXISTS refresh_tokens (
    token VARCHAR(64) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP
);
