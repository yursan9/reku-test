package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListMenu(t *testing.T) {
	w := httptest.NewRecorder()
	pool := NewLocalChefPool(func(pm PizzaMenu) {})
	h := Handler{
		Pool: pool,
	}
	req := httptest.NewRequest(http.MethodGet, "https://localhost/menus", nil)
	h.ListMenu(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	menuJSON, _ := json.Marshal(menu)

	// Delete line feed from handler's response before comparing with expected result
	if !bytes.Equal(body[:len(body)-1], menuJSON) {
		t.Error("Response is not equal:", body, menuJSON)
	}
}

func TestAddChef(t *testing.T) {
	w := httptest.NewRecorder()
	pool := NewLocalChefPool(func(pm PizzaMenu) {})
	if cap(pool.Pool) != 1 {
		t.Error("Initial chef's pool capacity is not 1")
	}

	h := Handler{
		Pool: pool,
	}
	req := httptest.NewRequest(http.MethodPost, "https://localhost/chef", nil)
	h.AddChef(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expecting status code %d got %d", http.StatusAccepted, resp.StatusCode)
	}

	if cap(pool.Pool) != 2 {
		t.Error("Chef's pool capacity should be 2, got:", cap(pool.Pool))
	}
}

func TestPlaceOrder(t *testing.T) {
	w := httptest.NewRecorder()
	pool := NewLocalChefPool(func(pm PizzaMenu) {})
	if cap(pool.Pool) != 1 {
		t.Error("Initial chef's pool capacity is not 1")
	}

	h := Handler{
		Pool: pool,
	}
	data := map[string]any{
		"code": PizzaBBQ,
	}
	var buff bytes.Buffer
	err := json.NewEncoder(&buff).Encode(data)
	if err != nil {
		t.Error(err)
	}
	req := httptest.NewRequest(http.MethodPost, "https://localhost/orders", &buff)
	h.PlaceOrder(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expecting status code %d got %d", http.StatusOK, resp.StatusCode)
	}
}
