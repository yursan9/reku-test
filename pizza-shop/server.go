package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Handler struct {
	Pool ChefPool
}

// AddChef handles POST /chef request
func (h *Handler) AddChef(w http.ResponseWriter, r *http.Request) {
	if !CheckMethod(r.Method, http.MethodPost) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.Pool.Expand()
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintln(w, "New chef added")
}

// ListMenu handles GET /menus request
func (h *Handler) ListMenu(w http.ResponseWriter, r *http.Request) {
	if !CheckMethod(r.Method, http.MethodGet) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := json.NewEncoder(w).Encode(menu)
	if err != nil {
		log.Println(err)
	}
}

// PlaceOrder handles POST /orders request
func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	if !CheckMethod(r.Method, http.MethodPost) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	order := struct {
		Code string `json:"code"`
	}{}
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at %d)", syntaxError.Offset)
			http.Error(w, msg, http.StatusBadRequest)

		case errors.Is(err, io.ErrUnexpectedEOF):
			http.Error(w, "Request body contains badly-formed JSON", http.StatusBadRequest)

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			http.Error(w, msg, http.StatusBadRequest)

		case errors.Is(err, io.EOF):
			http.Error(w, "Request body must not be empty", http.StatusBadRequest)

		case errors.As(err, &maxBytesError):
			msg := fmt.Sprintf("Request body must not be larger than %.2f MB", float64(maxBytesError.Limit)/float64(1048576))
			http.Error(w, msg, http.StatusRequestEntityTooLarge)

		default:
			log.Print(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	fmt.Println("New orders placed:", order.Code)
	chef := h.Pool.Get()
	chef(PizzaMenu(order.Code))

	defer h.Pool.Add()
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Order Finished")
	fmt.Println("Order Finished:", order.Code)
}

// CheckMethod return True if m inside allowed, else False
func CheckMethod(m string, allowed ...string) bool {
	for _, v := range allowed {
		if m == v {
			return true
		}
	}
	return false
}
