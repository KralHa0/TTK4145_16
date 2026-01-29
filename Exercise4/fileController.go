package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)
const stateFile = "state.txt"
const tempFile = "temp.txt"

func readState() (int, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err){
			return 0, nil
		}
		return 0, err
	}

	state := strings.TrimSpace(string(data))
	if state == ""{
		return 0, nil
	}

	stateInt, err := strconv.Atoi(state)
	if err != nil {
		return 0, err
	}

	return stateInt, nil
}

func writeState(state int) error {
	data := []byte(strconv.Itoa(state))

	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return err
	}
	return os.Rename(tempFile, stateFile)
}

func LastModified() (time.Time, error){
	info, err := os.Stat(stateFile)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}