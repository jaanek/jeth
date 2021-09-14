package eth

import (
	"encoding/hex"
	"os"
)

func ReadHexFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	bin, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	return bin, nil
}
