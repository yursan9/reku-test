package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	cp := NewLocalChefPool(RealChef)

	pizzaHandler := &Handler{
		Pool: cp,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/chef", pizzaHandler.AddChef)
	mux.HandleFunc("/menus", pizzaHandler.ListMenu)
	mux.HandleFunc("/orders", pizzaHandler.PlaceOrder)

	srv := http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	fmt.Println("Starting server at :8080")
	srv.ListenAndServe()
}
