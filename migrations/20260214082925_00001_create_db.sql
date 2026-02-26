-- +goose Up
-- +goose StatementBegin
    CREATE TABLE IF NOT EXISTS users ( 
        login VARCHAR(100) UNIQUE NOT NULL, 
        password_hash VARCHAR(255) NOT NULL,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()     
    );

    CREATE TABLE IF NOT EXISTS orders (
        order_num VARCHAR(50) UNIQUE NOT NULL,
        login VARCHAR(100) NOT NULL, 
        uploaded TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Внешний ключ на login 
    CONSTRAINT fk_orders_user
        FOREIGN KEY (login)
        REFERENCES users(login)
        ON DELETE RESTRICT            
    );

    CREATE TABLE IF NOT EXISTS withdraw (
        login VARCHAR(100) NOT NULL,
        order_num VARCHAR(50) NOT NULL,
        withdraw DOUBLE PRECISION NOT NULL,
        processed TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        PRIMARY KEY (login, order_num)
    );

    CREATE INDEX IF NOT EXISTS idx_orders_login ON orders (login);
    CREATE INDEX IF NOT EXISTS idx_orders_uploaded ON orders (uploaded);    
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
    DROP TABLE IF EXISTS withdraw;
    DROP TABLE IF EXISTS orders;

    DROP INDEX IF EXISTS idx_orders_login;
    DROP INDEX IF EXISTS idx_orders_uploaded;
    DROP TABLE IF EXISTS users;
-- +goose StatementEnd
