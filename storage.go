package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type Storage interface {
	AddAccount(*Account) error
	DeleteAccount(string) error
	GetAccount(string) (*Account, error)
	GetAllAccounts() ([]*Account, error)
	CheckAccountPresent(string) bool
	AccountAlreadyExists(string) bool
	AddTransaction(*Transaction) error
	DeleteTransaction(string) error
	GetAllTransactions() ([]*Transaction, error)
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	connStr := "user=postgres dbname=postgres password=postgres123 sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &PostgresStore{db: db}, nil
}

func (p *PostgresStore) InitializeDataStore() error {
	err := p.createAccountsTable()
	if err != nil {
		return err
	}
	err = p.createTransactionsTable()
	if err != nil {
		return err
	}

	return nil
}

func (p *PostgresStore) createAccountsTable() error {
	accountTableQuery := `
        CREATE TABLE IF NOT EXISTS accounts (
            id SERIAL PRIMARY KEY,
            first_name VARCHAR(50),
            last_name VARCHAR(50),
            email_id VARCHAR(50),
            account_number TEXT,
            balance INT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `

	_, err := p.db.Exec(accountTableQuery)

	if err != nil {
		return err
	}
	fmt.Println("Successfully created accounts table")
	return nil
}

func (p *PostgresStore) createTransactionsTable() error {
	transactionTableQuery := `
        CREATE TABLE IF NOT EXISTS transactions (
            id SERIAL PRIMARY KEY,
            transaction_id TEXT,
            transaction_type VARCHAR(50),
            holder_account_num TEXT,
            receiver_account_num TEXT,
            amount INT,
            transaction_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            status VARCHAR(20),
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `
	_, err := p.db.Exec(transactionTableQuery)

	if err != nil {
		return err
	}

	fmt.Println("Successfully created transactions table...")
	return nil
}

func (p *PostgresStore) AccountAlreadyExists(userEmail string) bool {
	accountCheckQuery := `SELECT 1 AS account_present FROM accounts WHERE email_id = $1`
	rows, err := p.db.Query(accountCheckQuery, userEmail)

	if err != nil {
		log.Fatal(err)
		return false
	}

	if !rows.Next() {
		fmt.Println("Account not present in table with email:", userEmail)
		return false
	}

	return true
}

func (p *PostgresStore) AddAccount(acc *Account) error {
	addAccountQuery := `INSERT INTO accounts (first_name, last_name, email_id, account_number, balance, created_at) VALUES ($1, $2, $3, $4, $5, $6)`

	res, err := p.db.Exec(addAccountQuery, acc.FirstName, acc.LastName, acc.Email, acc.AccountNumber, acc.Balance, acc.CreatedAt)
	if err != nil {
		return err
	}

	affectedRowsCount, err := res.RowsAffected()
	if err != nil {
		return err
	}

	fmt.Println("Successfully added account to PostgreSQL db with affected rows:", affectedRowsCount)

	return nil
}

func (p *PostgresStore) DeleteAccount(accountId string) error {
	deleteAccountQuery := `DELETE FROM accounts WHERE account_number = $1`

	res, err := p.db.Exec(deleteAccountQuery, accountId)
	if err != nil {
		return err
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	fmt.Println("Deleted Rows: ", affectedRows)
	return nil
}

func (p *PostgresStore) GetAccount(accountId string) (*Account, error) {
	getAccountQuery := `SELECT * FROM accounts WHERE account_number = $1`

	rows, err := p.db.Query(getAccountQuery, accountId)
	if err != nil {
		return nil, err
	}

	var accountInfo Account
	for rows.Next() {
		err := rows.Scan(&accountInfo.ID, &accountInfo.FirstName, &accountInfo.LastName, &accountInfo.Email, &accountInfo.AccountNumber, &accountInfo.Balance, &accountInfo.CreatedAt)
		if err != nil {
			return nil, err
		}
	}
	return &accountInfo, nil
}

func (p *PostgresStore) GetAllAccounts() ([]*Account, error) {
	getAllAccountsQuery := `SELECT * FROM accounts`

	rows, err := p.db.Query(getAllAccountsQuery)
	if err != nil {
		return nil, err
	}

	accountsResult := []*Account{}
	for rows.Next() {
		var singleAccountInfo Account
		err := rows.Scan(&singleAccountInfo.ID, &singleAccountInfo.FirstName, &singleAccountInfo.LastName, &singleAccountInfo.Email, &singleAccountInfo.AccountNumber, &singleAccountInfo.Balance, &singleAccountInfo.CreatedAt)
		if err != nil {
			return nil, err
		}
		accountsResult = append(accountsResult, &singleAccountInfo)
	}

	return accountsResult, nil
}

func (p *PostgresStore) CheckAccountPresent(accountNum string) bool {
	checkAccountQuery := `SELECT 1 AS account_present FROM accounts WHERE account_number = $1`
	rows, err := p.db.Query(checkAccountQuery, accountNum)
	if err != nil {
		log.Fatal(err)
	}

	if !rows.Next() {
		return false
	}

	return true
}

func (p *PostgresStore) AddTransaction(tns *Transaction) error {
	if tns.TransactionType == "self" {
		accountPresent := p.CheckAccountPresent(tns.HolderAccountNum)
		if !accountPresent {
			return fmt.Errorf("account not found")
		}

		ctx := context.Background()
		tx, err := p.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		fmt.Println(tns.Amount, tns.HolderAccountNum)

		updateAmountQuery := `UPDATE accounts SET balance = balance + $1 WHERE account_number = $2`
		_, err = tx.ExecContext(ctx, updateAmountQuery, tns.Amount, tns.HolderAccountNum)
		if err != nil {
			tx.Rollback()
			return err
		}

		fmt.Println("Successfully added:", tns.Amount, "in account:", tns.HolderAccountNum)

		auditTransactionQuery := `INSERT INTO transactions (transaction_id, transaction_type, holder_account_num, receiver_account_num, amount, status, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err = tx.ExecContext(ctx, auditTransactionQuery, tns.TransactionId, tns.TransactionType, tns.HolderAccountNum, "-", tns.Amount, tns.Status, tns.CreatedAt)
		if err != nil {
			tx.Rollback()
			return err
		}

		fmt.Println("Successfully audit the transaction with id:", tns.TransactionId)

		err = tx.Commit()
		if err != nil {
			return err
		}
	} else {
		holderAccountPresent := p.CheckAccountPresent(tns.HolderAccountNum)
		receiverAccountPresent := p.CheckAccountPresent(tns.ReceiverAccountNum)

		if !holderAccountPresent || !receiverAccountPresent {
			var errMsg string
			if !holderAccountPresent {
				errMsg = "holder account not found"
			} else {
				errMsg = "receiver account not found"
			}
			return fmt.Errorf(errMsg)
		}

		ctx := context.Background()
		tx, err := p.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		checkHolderAmountQuery := `SELECT balance FROM accounts WHERE account_number = $1`
		rows, err := tx.QueryContext(ctx, checkHolderAmountQuery, tns.HolderAccountNum)
		if err != nil {
			tx.Rollback()
			return err
		}

		var balance int64
		for rows.Next() {
			err = rows.Scan(&balance)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

		if balance < tns.Amount {
			tx.Rollback()
			return fmt.Errorf("insufficient balance")
		}

		deductAmountHolderQry := `UPDATE accounts SET balance = balance - $1 WHERE account_number = $2`
		_, err = tx.ExecContext(ctx, deductAmountHolderQry, tns.Amount, tns.HolderAccountNum)
		if err != nil {
			tx.Rollback()
			return err
		}

		addAmountReceiverQry := `UPDATE accounts SET balance = balance + $1 WHERE account_number = $2`
		_, err = tx.ExecContext(ctx, addAmountReceiverQry, tns.Amount, tns.ReceiverAccountNum)
		if err != nil {
			tx.Rollback()
			return err
		}

		fmt.Println("Successfully transferred amount:", tns.Amount, "from holder account:", tns.HolderAccountNum, "to receiver account:", tns.ReceiverAccountNum)

		auditTransactionQry := `INSERT INTO transactions (transaction_id, transaction_type, holder_account_num, receiver_account_num, amount, status, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err = tx.ExecContext(ctx, auditTransactionQry, tns.TransactionId, tns.TransactionType, tns.HolderAccountNum, tns.ReceiverAccountNum, tns.Amount, tns.Status, tns.CreatedAt)
		if err != nil {
			tx.Rollback()
			return err
		}

		tx.Commit()
	}

	fmt.Println("Transaction successfully commited...")

	return nil
}

func (p *PostgresStore) DeleteTransaction(tnsId string) error {
	return nil
}

func (p *PostgresStore) GetAllTransactions() ([]*Transaction, error) {
	rows, err := p.db.Query(`SELECT * FROM transactions`)
	if err != nil {
		return nil, err
	}

	allTransactionList := []*Transaction{}
	for rows.Next() {
		var singleTransInfo Transaction
		err = rows.Scan(&singleTransInfo.ID, &singleTransInfo.TransactionId, &singleTransInfo.TransactionType, &singleTransInfo.HolderAccountNum, &singleTransInfo.ReceiverAccountNum, &singleTransInfo.Amount, &singleTransInfo.TransactionDate, &singleTransInfo.Status, &singleTransInfo.CreatedAt)
		if err != nil {
			return nil, err
		}

		allTransactionList = append(allTransactionList, &singleTransInfo)
	}
	return allTransactionList, nil
}
