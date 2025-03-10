package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

var DB *sql.DB

func InitDB(cfg Config) error {

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password)
	tempDB, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return fmt.Errorf("error opening postgres database: %v", err)
	}
	defer tempDB.Close()

	var exists bool
	err = tempDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", cfg.DBName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("error checking database existence: %v", err)
	}

	if !exists {
		_, err = tempDB.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.DBName))
		if err != nil {
			return fmt.Errorf("error creating database: %v", err)
		}
		fmt.Println("Database created successfully")
	} else {
		fmt.Println("Database already exists, proceeding...")
	}

	// Connect to the target database
	psqlInfo = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)
	DB, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return fmt.Errorf("error opening database: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		return fmt.Errorf("error pinging database: %v", err)
	}

	_, err = DB.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	if err != nil {
		return fmt.Errorf("error enabling uuid-ossp extension: %v", err)
	}

	err = createSchema()
	if err != nil {
		return fmt.Errorf("error creating schema: %v", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

func createSchema() error {
	schema := `
        DO $$ BEGIN
            IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_type') THEN
                CREATE TYPE order_type AS ENUM ('Market', 'Limit', 'StopMarket', 'StopLimit');
            END IF;
            IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_side') THEN
                CREATE TYPE order_side AS ENUM ('Buy', 'Sell');
            END IF;
            IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_status') THEN
                CREATE TYPE order_status AS ENUM ('Pending', 'Filled', 'Cancelled');
            END IF;
        END $$;

        CREATE TABLE IF NOT EXISTS users (
            user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            username VARCHAR(50) NOT NULL UNIQUE,
            email VARCHAR(255) NOT NULL UNIQUE,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS wallets (
            wallet_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            user_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
            hyperliquid_address VARCHAR(66) NOT NULL UNIQUE,
            hyperliquid_api_key VARCHAR(255),
            balance_usdc DECIMAL(18, 6) DEFAULT 0.0,
            is_owned BOOLEAN NOT NULL DEFAULT FALSE,
            updated_at TIMESTAMP WITH TIME ZONE,
            CONSTRAINT chk_api_key_owned CHECK (is_owned = TRUE OR hyperliquid_api_key IS NULL)
        );

        CREATE TABLE IF NOT EXISTS assets (
            asset_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            symbol VARCHAR(20) NOT NULL UNIQUE,
            base_currency VARCHAR(10) NOT NULL,
            quote_currency VARCHAR(10) NOT NULL,
            is_perpetual BOOLEAN NOT NULL DEFAULT TRUE
        );

        CREATE TABLE IF NOT EXISTS orders (
            order_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            wallet_id UUID NOT NULL REFERENCES wallets(wallet_id) ON DELETE CASCADE,
            asset_id UUID NOT NULL REFERENCES assets(asset_id) ON DELETE RESTRICT,
            order_type order_type NOT NULL,
            side order_side NOT NULL,
            quantity DECIMAL(18, 8) NOT NULL,
            price DECIMAL(18, 6),
            leverage DECIMAL(5, 2) DEFAULT 1.0 CHECK (leverage >= 1.0 AND leverage <= 50.0),
            status order_status NOT NULL DEFAULT 'Pending',
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            hyperliquid_order_id VARCHAR(64),
            is_copied BOOLEAN NOT NULL DEFAULT FALSE,
            CONSTRAINT chk_price_order_type CHECK (order_type != 'Market' OR price IS NULL)
        );

        CREATE TABLE IF NOT EXISTS trades (
            trade_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            order_id UUID NOT NULL REFERENCES orders(order_id) ON DELETE CASCADE,
            asset_id UUID NOT NULL REFERENCES assets(asset_id) ON DELETE RESTRICT,
            executed_price DECIMAL(18, 6) NOT NULL,
            executed_quantity DECIMAL(18, 8) NOT NULL,
            fee DECIMAL(18, 8) NOT NULL DEFAULT 0.0,
            executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        CREATE TABLE IF NOT EXISTS copy_trading_relationships (
            relationship_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            lead_wallet_id UUID NOT NULL REFERENCES wallets(wallet_id) ON DELETE CASCADE,
            follower_wallet_id UUID NOT NULL REFERENCES wallets(wallet_id) ON DELETE CASCADE,
            start_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            end_date TIMESTAMP WITH TIME ZONE,
            profit_share_percentage DECIMAL(5, 2) DEFAULT 0.0 CHECK (profit_share_percentage >= 0.0 AND profit_share_percentage <= 100.0),
            CONSTRAINT unique_relationship UNIQUE (lead_wallet_id, follower_wallet_id),
            CONSTRAINT chk_different_wallets CHECK (lead_wallet_id != follower_wallet_id)
        );

        CREATE INDEX IF NOT EXISTS idx_orders_wallet_id ON orders(wallet_id);
        CREATE INDEX IF NOT EXISTS idx_orders_asset_id ON orders(asset_id);
        CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
        CREATE INDEX IF NOT EXISTS idx_trades_order_id ON trades(order_id);
        CREATE INDEX IF NOT EXISTS idx_trades_executed_at ON trades(executed_at);
        CREATE INDEX IF NOT EXISTS idx_copy_trading_lead_wallet_id ON copy_trading_relationships(lead_wallet_id);
        CREATE INDEX IF NOT EXISTS idx_copy_trading_follower_wallet_id ON copy_trading_relationships(follower_wallet_id);
    `
	_, err := DB.Exec(schema)
	return err
}
