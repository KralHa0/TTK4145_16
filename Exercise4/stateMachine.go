package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func runStateMachine() {
	if isBackup() {
		fmt.Println("[BACKUP] started")
		waitUntilPrimaryDies()
		fmt.Println("[BACKUP] primary dead, taking over")

		state := becomePrimary()
		spawnBackup()
		runPrimaryLoop(state)
	} else {
		fmt.Println("[PRIMARY] started")
		runInitialPrimary()
	}
}

func runPrimaryLoop(state int) {
	for {
		newstate := increntementState(state)
		writeState(newstate)
		fmt.Println("[PRIMARY]:", newstate)
		state = newstate
		time.Sleep(time.Millisecond * 250)
		simulateCrash(state, 7)
	}
}
func simulateCrash(state, n int) {
	if state == n {
		fmt.Println("[System] Simulating crash!")
		os.Exit(1)
	}
}
func runInitialPrimary() {
	state, err := readState()
	if err != nil {
		panic(err)
	}

	fmt.Println("[PRIMARY] initial start with state:", state)
	spawnBackup()
	runPrimaryLoop(state)
}

func becomePrimary() int {
	state, _ := readState()
	fmt.Println("[BACKUP] Becoming Primary with state:", state)
	return state
}

func increntementState(state int) int {
	return state + 1
}

func waitUntilPrimaryDies() {
	lastMod, _ := LastModified()
	timeout := time.Second * 2
	checkInterval := time.Millisecond * 500
	for {
		newLastMod, _ := LastModified()
		if newLastMod.After(lastMod) {
			lastMod = newLastMod
		} else {
			time.Sleep(checkInterval)
		}

		if time.Since(lastMod) > timeout {
			return
		}
	}
}

func spawnBackup() {
	cmd := exec.Command("./processpair")
	cmd.Env = append(os.Environ(), "ROLE=backup")

	// Redirect output for visibility
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		panic(err)
	}
}

func isBackup() bool {
	role := os.Getenv("ROLE")
	return role == "backup"
}
