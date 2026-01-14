package main

import (
	. "fmt"
)

var i int = 0

func incrementing() {
	for j := 0; j < 100000; j++ {
		i++
	}
}

func decrementing() {
	for j := 0; j < 100001; j++ {
		i--
	}
}

func runTask1() {
	i = 0
	incrementing()
	decrementing()

	Println("Final value:", i)
}
