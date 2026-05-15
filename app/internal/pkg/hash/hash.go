// Package hash provides bcrypt-based password hashing utilities.
package hash

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

// Hash generates a bcrypt hash of the given password with cost=12.
func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Check compares a plaintext password against a bcrypt hash.
// Returns true if the password matches the hash.
func Check(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
