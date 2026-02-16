package main

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
)

type NetworkMessage struct {
	Data []byte `json:"data"`
	Crc  uint32 `json:"crc"`
}

func calculateCrc(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

func createNetworkMessage(data interface{}) (NetworkMessage, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return NetworkMessage{}, fmt.Errorf("failed to marshal data: %v", err)
	}
	return NetworkMessage{
		Data: jsonData,
		Crc:  calculateCrc(jsonData),
	}, nil
}

func (nm *NetworkMessage) validateMessage() bool {
	return nm.Crc == calculateCrc(nm.Data)
}

func (nm *NetworkMessage) GetData(v interface{}) error {
	if !nm.validateMessage() {
		return fmt.Errorf("data integrity check failed: CRC does not match")
	}
	return json.Unmarshal(nm.Data, v)
}
