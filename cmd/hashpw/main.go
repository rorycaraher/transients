// Command hashpw prints a bcrypt hash for ADMIN_PASSWORD_HASH.
//
// Usage: go run ./cmd/hashpw
package main

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/rorycaraher/transients/internal/auth"
)

func main() {
	fmt.Fprint(os.Stderr, "Password: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read password:", err)
		os.Exit(1)
	}

	hash, err := auth.HashPassword(string(pw))
	if err != nil {
		fmt.Fprintln(os.Stderr, "hash password:", err)
		os.Exit(1)
	}
	fmt.Println(hash)
}
