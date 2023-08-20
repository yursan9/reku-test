package main

import (
	"fmt"
	"log"
	"time"
)

type PizzaMenu string

const (
	PizzaCheese PizzaMenu = "pizza-cheese"
	PizzaBBQ    PizzaMenu = "pizza-bbq"
)

var (
	menu = map[PizzaMenu]map[string]any{
		PizzaBBQ:    {"name": "Pizza BBQ", "process_time": 5},
		PizzaCheese: {"name": "Pizza Cheese", "process_time": 3},
	}
)

type ChefFunc func(PizzaMenu)

type ChefPool interface {
	Expand()
	Get() ChefFunc
	Add()
}

type LocalChefPool struct {
	Pool        chan ChefFunc
	Chef        ChefFunc
	RestartChan chan struct{}
}

func NewLocalChefPool(chef ChefFunc) *LocalChefPool {
	cp := LocalChefPool{
		Pool:        make(chan ChefFunc, 1),
		RestartChan: make(chan struct{}, 1),
		Chef:        chef,
	}

	// Initialize Pool with 1 worker
	go func() { cp.Pool <- cp.Chef }()
	return &cp
}

// Expand replace Pool channel with new channel with bigger capacity
func (p *LocalChefPool) Expand() {
	n := cap(p.Pool) + 1
	p.Pool = make(chan ChefFunc, n)
	p.RestartChan <- struct{}{}

	fmt.Println("Expand chef pool:", cap(p.Pool))
}

// Get return ChefFunc from channel or block if channel is empty
// Also return ChefFunc if Pool is still blocked and get restart by calling Expand
func (p *LocalChefPool) Get() ChefFunc {
	select {
	case f := <-p.Pool:
		return f
	case <-p.RestartChan:
		return p.Chef
	}
}

// Add insert ChefFunc to pool
func (p *LocalChefPool) Add() {
	go func() { p.Pool <- p.Chef }()
}

// RealChef handle processing Pizza according to menu
func RealChef(code PizzaMenu) {
	if pizza, ok := menu[code]; ok {
		dur := pizza["process_time"].(int)
		fmt.Println("Proccessing:", pizza["name"], "for", dur, "seconds")
		time.Sleep(time.Duration(dur) * time.Second)
	} else {
		log.Println("Unknown menu:", menu)
	}
}
