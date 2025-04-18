CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    telegram_id VARCHAR(64) UNIQUE NOT NULL,
    current_discount INT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS servers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(64),
    ip VARCHAR(64),
    price1 INT,
    price3 INT,
    price6 INT,
    price12 INT,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS vless_keys (
    id SERIAL PRIMARY KEY,
    server_id INT REFERENCES servers(id),
    key VARCHAR(255),
    is_used BOOLEAN DEFAULT FALSE,
    reserved_until BIGINT,
    user_id INT REFERENCES users(id),
    assigned_at BIGINT,
    notified_expiring BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS payments (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    yoo_kassa_id VARCHAR(128),
    amount FLOAT,
    status VARCHAR(32),
    months INT,
    key_id INT REFERENCES vless_keys(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
