-- Schema for Strike2 project
CREATE SCHEMA IF NOT EXISTS strike2;

-- Users table
CREATE TABLE IF NOT EXISTS strike2.users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true
);

-- API Tokens table (plaintext version)
CREATE TABLE IF NOT EXISTS strike2.api_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES strike2.users(id),
    token VARCHAR(72) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    tier VARCHAR(50) DEFAULT 'free',
    requests_limit INTEGER,
    requests_used INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_strike2_tokens_token ON strike2.api_tokens(token);
CREATE INDEX idx_strike2_tokens_user_id ON strike2.api_tokens(user_id);
