package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type Wallet struct {
	ID      string  `json:"id" gorm:"primary_key"`
	Balance float64 `json:"balance"`
}

type Transaction struct {
	ID        uint      `gorm:"primary_key"`
	Time      time.Time `json:"time"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    float64   `json:"amount"`
	WalletID  string    `json:"wallet_id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

func main() {

	router := gin.Default()

	db, err := gorm.Open("postgres", "host=localhost user=postgres dbname=postgres sslmode=disable password=root")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.AutoMigrate(&Wallet{}, &Transaction{})

	router.POST("/api/v1/wallet", func(c *gin.Context) {
		wallet := Wallet{ID: generateID(), Balance: 100.0}
		db.Create(&wallet)
		c.JSON(http.StatusOK, wallet)
	})

	router.POST("/api/v1/wallet/:walletId/send", func(c *gin.Context) {
		senderID := c.Param("walletId")

		var transactionRequest struct {
			To     string  `json:"to"`
			Amount float64 `json:"amount"`
		}

		if err := c.BindJSON(&transactionRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
			return
		}

		var sender Wallet
		var receiver Wallet

		if err := db.Where("id = ?", senderID).First(&sender).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sender wallet not found"})
			return
		}

		if err := db.Where("id = ?", transactionRequest.To).First(&receiver).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Receiver wallet not found"})
			return
		}

		if sender.Balance < transactionRequest.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient funds"})
			return
		}

		db.Transaction(func(tx *gorm.DB) error {
			sender.Balance -= transactionRequest.Amount
			receiver.Balance += transactionRequest.Amount

			tx.Save(&sender)
			tx.Save(&receiver)

			transaction := Transaction{
				Time:     time.Now(),
				From:     senderID,
				To:       transactionRequest.To,
				Amount:   transactionRequest.Amount,
				WalletID: senderID,
			}
			tx.Create(&transaction)

			return nil
		})

		c.JSON(http.StatusOK, gin.H{"message": "Transaction successful"})
	})

	router.GET("/api/v1/wallet/:walletId/history", func(c *gin.Context) {
		walletID := c.Param("walletId")

		var wallet Wallet
		if err := db.Where("id = ?", walletID).First(&wallet).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
			return
		}

		var transactions []Transaction
		db.Where("wallet_id = ?", walletID).Find(&transactions)

		c.JSON(http.StatusOK, transactions)
	})

	router.GET("/api/v1/wallet/:walletId", func(c *gin.Context) {
		walletID := c.Param("walletId")

		var wallet Wallet
		if err := db.Where("id = ?", walletID).First(&wallet).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
			return
		}

		c.JSON(http.StatusOK, wallet)
	})

	router.Run(":8080")
}

func generateID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
