package main

import (
	. "fmt"
)

func numberServer(
	incCh <-chan struct{},
	decCh <-chan struct{},
	getCh <-chan chan int,
) {
	i := 0
	for {
		select {
		case <-incCh:
			i++
		case <-decCh:
			i--
		case reply := <-getCh:
			reply <- i
		}
	}

}

func incrementWorker(incCh chan<- struct{}, done chan<- struct{}) {
	for j := 0; j < 100000; j++ {
		incCh <- struct{}{}
	}
	done <- struct{}{}
}

func decrementWorker(decCh chan<- struct{}, done chan<- struct{}) {
	for j := 0; j < 100001; j++ {
		decCh <- struct{}{}
	}
	done <- struct{}{}
}

func runTask3() { //Task 3 runner for ease of testing
	incCh := make(chan struct{})
	decCh := make(chan struct{})
	getCh := make(chan chan int)
	done := make(chan struct{})

	go numberServer(incCh, decCh, getCh)

	go incrementWorker(incCh, done)
	go decrementWorker(decCh, done)

	<-done
	<-done

	reply := make(chan int)
	getCh <- reply
	result := <-reply

	Println("Final value:", result)
}
