package main

import "time"

type CreateAccountRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email_id"`
}

type CreateTransactionRequest struct {
	HolderAccountNum   string    `json:"holder_account_num"`
	ReceiverAccountNum string    `json:"receiver_account_num"`
	TransactionType    string    `json:"transaction_type"`
	Amount             int64     `json:"amount"`
	TransactionDate    time.Time `json:"transaction_date"`
}

type Account struct {
	ID            int       `json:"id"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Email         string    `json:"email_id"`
	AccountNumber string    `json:"account_number"`
	Balance       int64     `json:"balance"`
	CreatedAt     time.Time `json:"created_at"`
}

type Transaction struct {
	ID                 int       `json:"id"`
	TransactionId      string    `json:"transaction_id"`
	TransactionType    string    `json:"transaction_type"`
	HolderAccountNum   string    `json:"holder_account_num"`
	ReceiverAccountNum string    `json:"receiver_account_num"`
	Amount             int64     `json:"amount"`
	TransactionDate    time.Time `json:"transaction_date"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
}
