package utils

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

func Fingerprint(cfg any) (string, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("utils: fingerprint marshal: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}
