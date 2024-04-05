package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type APIServer struct {
	portAddress string
	store       Storage
}

type APIError struct {
	Error string `json:"error"`
}

type ApiFunc func(http.ResponseWriter, *http.Request) error

func WriteJSON(w http.ResponseWriter, statusCode int, msg any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	return json.NewEncoder(w).Encode(msg)
}

func makeHTTPHandleFunc(f ApiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		}
	}
}

func NewAPIServer(portAddress string, store Storage) *APIServer {
	apiServer := &APIServer{
		portAddress: portAddress,
		store:       store,
	}

	return apiServer
}

func createJWTToken(accountInfo *Account) (string, error) {
	mySigningKey := []byte(os.Getenv("JWT_SECRET_KEY"))

	// Create the Claims
	claims := &jwt.MapClaims{
		"expiresAt":     jwt.NewNumericDate(time.Unix(1516239022, 0)),
		"accountNumber": accountInfo.AccountNumber,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(mySigningKey)

	return ss, err
}

func denyPermission(w http.ResponseWriter) {
	WriteJSON(w, http.StatusBadRequest, APIError{Error: "Permission Denied"})
}

func authenticateJWTToken(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Calling JWT authentication Token middleware")

		tokenString := r.Header.Get("x-jwt-token")
		accountNum := mux.Vars(r)["id"]

		jwtTokenVal, err := validateJWT(tokenString)
		if err != nil {
			denyPermission(w)
			return
		}

		claims := jwtTokenVal.Claims.(jwt.MapClaims)
		fmt.Println(claims["accountNumber"], accountNum)
		if claims["accountNumber"] != accountNum {
			denyPermission(w)
			return
		}

		handlerFunc(w, r)
	}
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	jwtSecretKey := os.Getenv("JWT_SECRET_KEY")

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(jwtSecretKey), nil
	})
}

func (apiServer *APIServer) RunServer() {
	router := mux.NewRouter()

	fmt.Println("Transaction server listening and serving request on PORT:", apiServer.portAddress)

	router.HandleFunc("/", makeHTTPHandleFunc(apiServer.getTransactionHomePage))
	router.HandleFunc("/account", makeHTTPHandleFunc(apiServer.handleAccountsRequest))
	// router.HandleFunc("/account/{id}", makeHTTPHandleFunc(apiServer.handleSingleAccountRequest))
	router.HandleFunc("/account/{id}", authenticateJWTToken(makeHTTPHandleFunc(apiServer.handleSingleAccountRequest)))
	router.HandleFunc("/transaction", makeHTTPHandleFunc(apiServer.handleTransactionsRequest))

	err := http.ListenAndServe(apiServer.portAddress, router)
	if err != nil {
		log.Fatal(err)
	}
}

func (apiServer *APIServer) getTransactionHomePage(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode("Welcome to Go Transactions")

	return nil
}

func (apiServer *APIServer) handleAccountsRequest(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return apiServer.getTransactionAccounts(w, r)
	} else if r.Method == "POST" {
		return apiServer.createTransactionAccount(w, r)
	}
	return nil
}

func (apiServer *APIServer) handleTransactionsRequest(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return apiServer.getTransactionDetails(w, r)
	} else if r.Method == "POST" {
		return apiServer.createNewTransaction(w, r)
	} else if r.Method == "DELETE" {
		return apiServer.deleteTransaction(w, r)
	}
	return nil
}

func (apiServer *APIServer) handleSingleAccountRequest(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return apiServer.getSingleTransactionAccount(w, r)
	} else if r.Method == "DELETE" {
		return apiServer.deleteTransactionAccount(w, r)
	}
	return nil
}

func (apiServer *APIServer) getTransactionAccounts(w http.ResponseWriter, r *http.Request) error {
	allAccountsInfo, err := apiServer.store.GetAllAccounts()
	if err != nil {
		log.Fatal(err)
	}

	WriteJSON(w, http.StatusOK, allAccountsInfo)
	return nil
}

func (apiServer *APIServer) getSingleTransactionAccount(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	accountNum := vars["id"]
	fmt.Println("AccountNumber: ", accountNum)

	accountInfo, err := apiServer.store.GetAccount(accountNum)
	if err != nil {
		return err
	}

	if accountInfo.Email == "" {
		return fmt.Errorf("account not found")
	}

	WriteJSON(w, http.StatusOK, &accountInfo)

	return nil
}

func (apiServer *APIServer) createTransactionAccount(w http.ResponseWriter, r *http.Request) error {
	var newAccount CreateAccountRequest

	err := json.NewDecoder(r.Body).Decode(&newAccount)
	if err != nil {
		log.Fatal(err)
	}

	if apiServer.store.AccountAlreadyExists(newAccount.Email) {
		WriteJSON(w, http.StatusOK, "User Account already present")
		return nil
	}

	newlyCreatedAccount := CreateNewAccount(&newAccount)

	err = apiServer.store.AddAccount(newlyCreatedAccount)

	if err != nil {
		return err
	}

	// Add JWT Token for this newly created user
	tokenString, err := createJWTToken(newlyCreatedAccount)
	if err != nil {
		return err
	}
	fmt.Println("UserAccount:", newlyCreatedAccount.AccountNumber, "token:", tokenString)

	WriteJSON(w, http.StatusOK, newlyCreatedAccount)

	return nil
}

func (apiServer *APIServer) getTransactionDetails(w http.ResponseWriter, r *http.Request) error {
	allTransactionsInfo, err := apiServer.store.GetAllTransactions()
	if err != nil {
		return err
	}

	WriteJSON(w, http.StatusOK, allTransactionsInfo)
	return nil
}

func (apiServer *APIServer) createNewTransaction(w http.ResponseWriter, r *http.Request) error {
	var newTransactionInfo CreateTransactionRequest

	err := json.NewDecoder(r.Body).Decode(&newTransactionInfo)
	if err != nil {
		return err
	}

	newTransactionObj := CreateNewTransactionObj(&newTransactionInfo)

	err = apiServer.store.AddTransaction(newTransactionObj)
	if err != nil {
		return err
	}

	WriteJSON(w, http.StatusOK, newTransactionObj)
	return nil
}

func (apiServer *APIServer) deleteTransactionAccount(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	accountNum := vars["id"]
	fmt.Println("AccountNumber: ", accountNum)

	err := apiServer.store.DeleteAccount(accountNum)
	if err != nil {
		return err
	}
	fmt.Println("Successfully Deleted account:", accountNum)

	WriteJSON(w, http.StatusOK, "Account Successfully Deleted!")

	return nil
}

func (apiServer *APIServer) deleteTransaction(w http.ResponseWriter, r *http.Request) error {
	fmt.Println("Hey Delete transaction")
	return nil
}
