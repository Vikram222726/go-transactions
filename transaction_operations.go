package main

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

func CreateNewTransactionObj(transactionInfo *CreateTransactionRequest) *Transaction {
	return &Transaction{
		TransactionId:      strings.ReplaceAll(uuid.New().String(), "-", ""),
		TransactionType:    transactionInfo.TransactionType,
		HolderAccountNum:   transactionInfo.HolderAccountNum,
		ReceiverAccountNum: transactionInfo.ReceiverAccountNum,
		Amount:             transactionInfo.Amount,
		TransactionDate:    transactionInfo.TransactionDate,
		Status:             "successfull",
		CreatedAt:          time.Now().UTC(),
	}
}
