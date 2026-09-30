-- Initialize primary database with test schema and data

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title VARCHAR(200) NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(64) PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    data JSONB,
    expires_at TIMESTAMP NOT NULL
);

-- Insert test data
INSERT INTO users (username, email) VALUES
    ('alice', 'alice@example.com'),
    ('bob', 'bob@example.com'),
    ('charlie', 'charlie@example.com')
ON CONFLICT DO NOTHING;

INSERT INTO posts (user_id, title, content) VALUES
    (1, 'First Post', 'Hello world!'),
    (1, 'Second Post', 'Another day, another post'),
    (2, 'Bob''s Post', 'Thoughts from Bob'),
    (3, 'Charlie''s Adventure', 'A long story...')
ON CONFLICT DO NOTHING;

-- Create test functions for advisory locks
CREATE OR REPLACE FUNCTION test_advisory_lock(lock_id BIGINT, sleep_seconds INT DEFAULT 5)
RETURNS TEXT AS $$
BEGIN
    PERFORM pg_advisory_lock(lock_id);
    PERFORM pg_sleep(sleep_seconds);
    PERFORM pg_advisory_unlock(lock_id);
    RETURN 'Lock acquired and released';
END;
$$ LANGUAGE plpgsql;
