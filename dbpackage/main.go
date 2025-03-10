package main

import (
	"dbpackage/db"
	"fmt"
	"time"
)

func main() {

	err := db.InitDB(db.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "admin", //
		DBName:   "hyperliquid_copy_trading",
	})
	if err != nil {
		fmt.Printf("Error initializing DB: %v\n", err)
		return
	}

	// Create wallet
	uniqueAddress := fmt.Sprintf("0xTestWallet%d", time.Now().UnixNano())
	wallet := &db.Wallet{
		HyperliquidAddress: uniqueAddress,
		HyperliquidAPIKey:  stringPtr("test-api-key"),
		BalanceUSDC:        1000.0,
		IsOwned:            true,
	}
	err = db.InsertWallet(wallet)
	if err != nil {
		fmt.Printf("Error inserting wallet: %v\n", err)
		return
	}
	fmt.Printf("Wallet created with ID: %s\n", wallet.WalletID)

	// Get wallet by ID
	fetchedWallet, err := db.GetWalletByID(wallet.WalletID)
	if err != nil {
		fmt.Printf("Error fetching wallet by ID: %v\n", err)
		return
	}
	fmt.Printf("Fetched wallet by ID: Address=%s, Balance=%.2f\n", fetchedWallet.HyperliquidAddress, fetchedWallet.BalanceUSDC)

	// Get wallet by address
	fetchedByAddress, err := db.GetWalletByAddress(uniqueAddress)
	if err != nil {
		fmt.Printf("Error fetching wallet by address: %v\n", err)
		return
	}
	fmt.Printf("Fetched by address: ID=%s\n", fetchedByAddress.WalletID)

	// Change balance
	err = db.UpdateWalletBalance(wallet.WalletID, 1500.0)
	if err != nil {
		fmt.Printf("Error updating balance: %v\n", err)
		return
	}
	fmt.Println("Wallet balance updated to 1500.0")

	// // Change API key
	// err = db.UpdateWalletAPIKey(wallet.WalletID, "new-api-key")
	// if err != nil {
	// 	fmt.Printf("Error updating API key: %v\n", err)
	// 	return
	// }
	// fmt.Println("Wallet API key updated")

	// // Remove wallet
	// err = db.DeleteWallet(wallet.WalletID)
	// if err != nil {
	// 	fmt.Printf("Error deleting wallet: %v\n", err)
	// 	return
	// }
	// fmt.Println("Wallet deleted")
}

func stringPtr(s string) *string {
	return &s
}
