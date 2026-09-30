-- Initialize replica database with same schema
-- In real setup, this would be a streaming replica, but for testing we create a separate DB

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert same test data (simulating replication)
INSERT INTO users (id, username, email) VALUES
    (1, 'alice', 'alice@example.com'),
    (2, 'bob', 'bob@example.com'),
    (3, 'charlie', 'charlie@example.com')
ON CONFLICT DO NOTHING;

INSERT INTO posts (id, user_id, title, content) VALUES
    (1, 1, 'First Post', 'Hello world!'),
    (2, 1, 'Second Post', 'Another day, another post'),
    (3, 2, 'Bob''s Post', 'Thoughts from Bob'),
    (4, 3, 'Charlie''s Adventure', 'A long story...')
ON CONFLICT DO NOTHING;

-- Reset sequences to match primary
SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));
SELECT setval('posts_id_seq', (SELECT MAX(id) FROM posts));
