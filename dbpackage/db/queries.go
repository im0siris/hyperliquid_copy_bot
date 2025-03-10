package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound  = errors.New("record not found")
	ErrDuplicate = errors.New("duplicate record")
)

type Wallet struct {
	WalletID           uuid.UUID
	UserID             *uuid.UUID // Nullable
	HyperliquidAddress string
	HyperliquidAPIKey  *string // Nullable
	BalanceUSDC        float64
	IsOwned            bool
}

type Asset struct {
	AssetID       uuid.UUID
	Symbol        string
	BaseCurrency  string
	QuoteCurrency string
	IsPerpetual   bool
}

type Order struct {
	OrderID            uuid.UUID
	WalletID           uuid.UUID
	AssetID            uuid.UUID
	OrderType          string
	Side               string
	Quantity           float64
	Price              *float64 // Pointer for nullable field
	Leverage           float64
	Status             string
	CreatedAt          time.Time
	HyperliquidOrderID *string // Pointer for nullable field
	IsCopied           bool
}

// Create: InsertWallet
func InsertWallet(wallet *Wallet) error {
	query := `
        INSERT INTO wallets (
            user_id, hyperliquid_address, hyperliquid_api_key, balance_usdc, is_owned
        ) VALUES ($1, $2, $3, $4, $5)
        RETURNING wallet_id
    `
	err := DB.QueryRow(query,
		wallet.UserID, wallet.HyperliquidAddress, wallet.HyperliquidAPIKey, wallet.BalanceUSDC, wallet.IsOwned,
	).Scan(&wallet.WalletID)
	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "wallets_hyperliquid_address_key"` {
			return ErrDuplicate
		}
		return fmt.Errorf("failed to insert wallet: %v", err)
	}
	return nil
}

// Read: GetWalletByID
func GetWalletByID(id uuid.UUID) (*Wallet, error) {
	wallet := &Wallet{}
	query := `
        SELECT wallet_id, user_id, hyperliquid_address, hyperliquid_api_key, balance_usdc, is_owned
        FROM wallets
        WHERE wallet_id = $1
    `
	err := DB.QueryRow(query, id).Scan(
		&wallet.WalletID, &wallet.UserID, &wallet.HyperliquidAddress, &wallet.HyperliquidAPIKey,
		&wallet.BalanceUSDC, &wallet.IsOwned,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get wallet by ID: %v", err)
	}
	return wallet, nil
}

// Read: GetWalletByAddress
func GetWalletByAddress(address string) (*Wallet, error) {
	wallet := &Wallet{}
	query := `
        SELECT wallet_id, user_id, hyperliquid_address, hyperliquid_api_key, balance_usdc, is_owned
        FROM wallets
        WHERE hyperliquid_address = $1
    `
	err := DB.QueryRow(query, address).Scan(
		&wallet.WalletID, &wallet.UserID, &wallet.HyperliquidAddress, &wallet.HyperliquidAPIKey,
		&wallet.BalanceUSDC, &wallet.IsOwned,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get wallet by address: %v", err)
	}
	return wallet, nil
}

// Update: UpdateWalletBalance
func UpdateWalletBalance(walletID uuid.UUID, balance float64) error {
	query := `
        UPDATE wallets
        SET balance_usdc = $1, updated_at = NOW()
        WHERE wallet_id = $2
    `
	result, err := DB.Exec(query, balance, walletID)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Update: UpdateWalletAPIKey
func UpdateWalletAPIKey(walletID uuid.UUID, apiKey string) error {
	query := `
        UPDATE wallets
        SET hyperliquid_api_key = $1, updated_at = NOW()
        WHERE wallet_id = $2
    `
	result, err := DB.Exec(query, apiKey, walletID)
	if err != nil {
		return fmt.Errorf("failed to update wallet API key: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete: DeleteWallet
func DeleteWallet(walletID uuid.UUID) error {
	query := `
        DELETE FROM wallets
        WHERE wallet_id = $1
    `
	result, err := DB.Exec(query, walletID)
	if err != nil {
		return fmt.Errorf("failed to delete wallet: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// InsertAsset inserts a new asse
func InsertAsset(asset *Asset) error {
	query := `
        INSERT INTO assets (
            symbol, base_currency, quote_currency, is_perpetual
        ) VALUES ($1, $2, $3, $4)
        RETURNING asset_id
    `
	err := DB.QueryRow(query,
		asset.Symbol, asset.BaseCurrency, asset.QuoteCurrency, asset.IsPerpetual,
	).Scan(&asset.AssetID)
	if err != nil {
		return fmt.Errorf("failed to insert asset: %v", err)
	}
	return nil
}

// InsertOrder inserts a new order into the database
func InsertOrder(order *Order) error {
	query := `
        INSERT INTO orders (
            wallet_id, asset_id, order_type, side, quantity, price, leverage,
            status, hyperliquid_order_id, is_copied
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        RETURNING order_id, created_at
    `
	err := DB.QueryRow(query,
		order.WalletID, order.AssetID, order.OrderType, order.Side, order.Quantity,
		order.Price, order.Leverage, order.Status, order.HyperliquidOrderID, order.IsCopied,
	).Scan(&order.OrderID, &order.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert order: %v", err)
	}
	return nil
}
