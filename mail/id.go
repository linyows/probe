package mail

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"time"
)

func HashID() string {
	hash := sha1.New()
	timestamp := time.Now().UnixNano()
	if _, err := io.WriteString(hash, fmt.Sprintf("%d", timestamp)); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func OptimisticUID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
