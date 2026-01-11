package main

import (
	. "fmt"
	"runtime"
	"time"
)

func main() {
	runtime.GOMAXPROCS(2)

	go incrementing()
	go decrementing()

	time.Sleep(500 * time.Millisecond)
	Println("The magic number is:", i)
}
