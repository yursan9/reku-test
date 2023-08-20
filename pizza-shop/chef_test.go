package main

import "testing"

func TestChefPool(t *testing.T) {
	pool := NewLocalChefPool(func(pm PizzaMenu) {
		if pm != PizzaBBQ {
			t.Error("Chef should be called with Pizza BBQ")
		}
	})

	chef := pool.Get()
	chef(PizzaBBQ)
}

func TestPoolExpand(t *testing.T) {
	pool := NewLocalChefPool(func(pm PizzaMenu) {})

	if cap(pool.Pool) != 1 {
		t.Error("Pool capacity should be 1")
	}

	pool.Expand()
	if cap(pool.Pool) != 2 {
		t.Error("Pool capacity should be 2 after expand")
	}
}
