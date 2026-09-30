package otp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
)

func Generate() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func Hash(secret []byte, userID int64, purpose, code string) []byte {
	m := hmac.New(sha256.New, secret)
	fmt.Fprintf(m, "%d:%s:%s", userID, purpose, code)
	return m.Sum(nil)
}

func Equal(a, b []byte) bool { return hmac.Equal(a, b) }
