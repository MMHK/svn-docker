package htpasswd

import (
	"crypto/rand"
	"fmt"
)

// apr1SaltLength is the salt length `htpasswd -m` uses.
const apr1SaltLength = 8

// HashAPR1 hashes password with a fresh random salt. APR1 is what Apache's
// mod_authn_file reads on every supported version and what the v2 tree already
// contains, so writes never introduce a format Apache cannot verify (plan D5).
func HashAPR1(password string) (string, error) {
	salt, err := randomSalt()
	if err != nil {
		return "", err
	}

	return hashAPR1WithSalt(password, salt), nil
}

// hashAPR1WithSalt is HashAPR1 with a caller supplied salt, used by the vector
// tests and by the self-verification path.
func hashAPR1WithSalt(password, salt string) string {
	if len(salt) > apr1SaltLength {
		salt = salt[:apr1SaltLength]
	}

	return md5Crypt(password, salt, "$apr1$")
}

func randomSalt() (string, error) {
	buffer := make([]byte, apr1SaltLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("htpasswd: generate salt: %w", err)
	}

	// Modulo bias over 64 symbols is irrelevant for a salt: it only has to be
	// unpredictable, and every symbol of the crypt alphabet is safe in a
	// `$apr1$salt$hash` field.
	for index, value := range buffer {
		buffer[index] = cryptAlphabet[int(value)%len(cryptAlphabet)]
	}

	return string(buffer), nil
}
