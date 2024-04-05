package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("Welcome to Go Transactions app")

	store, err := NewPostgresStore()
	if err != nil {
		log.Fatal(err)
	}

	err = store.InitializeDataStore()
	if err != nil {
		log.Fatal(err)
	}

	server := NewAPIServer(":7070", store)
	server.RunServer()
}
