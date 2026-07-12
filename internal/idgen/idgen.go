// Package idgen generates short random URL-safe IDs used as both R2 object
// keys and share-link slugs.
package idgen

import (
	"crypto/rand"
	"math/big"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// New returns a random 10-character ID (~59 bits of entropy).
func New() string {
	return NewN(10)
}

func NewN(n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err) // crypto/rand failure is unrecoverable
		}
		b[i] = alphabet[idx.Int64()]
	}
	return string(b)
}
