package main

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

func CreateNewAccount(newAcc *CreateAccountRequest) *Account {
	return &Account{
		FirstName:     newAcc.FirstName,
		LastName:      newAcc.LastName,
		Email:         newAcc.Email,
		AccountNumber: strings.ReplaceAll(uuid.New().String(), "-", ""),
		Balance:       0,
		CreatedAt:     time.Now().UTC(),
	}
}
