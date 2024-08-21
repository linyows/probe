package smtp

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"time"
)

func HashID() string {
	rand.Seed(time.Now().UnixNano())
	hash := sha1.New()
	io.WriteString(hash, fmt.Sprintf("%d", rand.Int63()))
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
