package htpasswd

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"hash"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/mmhk/svn-admin/pkg/model"
)

// cryptAlphabet is the crypt(3) base64 alphabet, also used by APR1 and the
// SHA-crypt family.
const cryptAlphabet = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var errUnsupportedHash = errors.New("htpasswd: unsupported password hash format")

// verifyMD5Crypt verifies an APR1 or md5crypt hash. Both variants follow
// Apache's apr_md5_encode, including the NUL byte in the "weird" bit loop;
// they differ only in the magic prefix ($apr1$ vs $1$).
func verifyMD5Crypt(password, hash, prefix string) error {
	rest := strings.TrimPrefix(hash, prefix)

	salt, _, ok := strings.Cut(rest, "$")
	if !ok || salt == "" {
		return errUnsupportedHash
	}

	if len(salt) > 8 {
		salt = salt[:8]
	}

	computed := md5Crypt(password, salt, prefix)
	if subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) != 1 {
		return model.ErrInvalidCredentials
	}

	return nil
}

func md5Crypt(password, salt, prefix string) string {
	pw := []byte(password)

	alternate := md5.New()
	alternate.Write(pw)
	alternate.Write([]byte(salt))
	alternate.Write(pw)
	digest := alternate.Sum(nil)

	ctx := md5.New()
	ctx.Write(pw)
	ctx.Write([]byte(prefix))
	ctx.Write([]byte(salt))

	for remaining := len(pw); remaining > 0; remaining -= md5.Size {
		count := remaining
		if count > md5.Size {
			count = md5.Size
		}

		ctx.Write(digest[:count])
	}

	for i := len(pw); i != 0; i >>= 1 {
		if i&1 != 0 {
			// apr_md5_encode zeroes the digest before this loop, so a 1 bit
			// contributes a NUL byte rather than a digest byte.
			ctx.Write([]byte{0})
		} else {
			ctx.Write(pw[:1])
		}
	}

	digest = ctx.Sum(nil)

	for i := 0; i < 1000; i++ {
		round := md5.New()

		if i&1 != 0 {
			round.Write(pw)
		} else {
			round.Write(digest)
		}

		if i%3 != 0 {
			round.Write([]byte(salt))
		}

		if i%7 != 0 {
			round.Write(pw)
		}

		if i&1 != 0 {
			round.Write(digest)
		} else {
			round.Write(pw)
		}

		digest = round.Sum(nil)
	}

	var encoded strings.Builder
	encoded.WriteString(prefix)
	encoded.WriteString(salt)
	encoded.WriteByte('$')

	to64(&encoded, uint32(digest[0])<<16|uint32(digest[6])<<8|uint32(digest[12]), 4)
	to64(&encoded, uint32(digest[1])<<16|uint32(digest[7])<<8|uint32(digest[13]), 4)
	to64(&encoded, uint32(digest[2])<<16|uint32(digest[8])<<8|uint32(digest[14]), 4)
	to64(&encoded, uint32(digest[3])<<16|uint32(digest[9])<<8|uint32(digest[15]), 4)
	to64(&encoded, uint32(digest[4])<<16|uint32(digest[10])<<8|uint32(digest[5]), 4)
	to64(&encoded, uint32(digest[11]), 2)

	return encoded.String()
}

func to64(builder *strings.Builder, value uint32, count int) {
	for ; count > 0; count-- {
		builder.WriteByte(cryptAlphabet[value&0x3f])
		value >>= 6
	}
}

func verifySHA1(password, encoded string) error {
	sum := sha1.Sum([]byte(password))
	expected := base64.StdEncoding.EncodeToString(sum[:])

	if subtle.ConstantTimeCompare([]byte(expected), []byte(encoded)) != 1 {
		return model.ErrInvalidCredentials
	}

	return nil
}

func verifyBcrypt(password, hash string) error {
	candidate := hash
	// x/crypto/bcrypt only knows the 2a/2b prefixes; 2y is bit-identical.
	if strings.HasPrefix(hash, "$2y$") {
		candidate = "$2a$" + hash[len("$2y$"):]
	}

	if err := bcrypt.CompareHashAndPassword([]byte(candidate), []byte(password)); err != nil {
		return model.ErrInvalidCredentials
	}

	return nil
}

// shaSpec describes one member of the SHA-crypt family.
type shaSpec struct {
	prefix string
	size   int
	new    func() hash.Hash
	order  []int // flattened encoding triples
	last   []int // digest indices for the final group, right aligned
	lastN  int
}

var sha256Spec = shaSpec{
	prefix: "$5$",
	size:   sha256.Size,
	new:    sha256.New,
	order: []int{
		0, 10, 20, 21, 1, 11, 12, 22, 2, 3, 13, 23, 24, 4, 14,
		15, 25, 5, 6, 16, 26, 27, 7, 17, 18, 28, 8, 9, 19, 29,
	},
	last:  []int{31, 30},
	lastN: 3,
}

var sha512Spec = shaSpec{
	prefix: "$6$",
	size:   sha512.Size,
	new:    sha512.New,
	order: []int{
		0, 21, 42, 22, 43, 1, 44, 2, 23, 3, 24, 45,
		25, 46, 4, 47, 5, 26, 6, 27, 48, 28, 49, 7,
		50, 8, 29, 9, 30, 51, 31, 52, 10, 53, 11, 32,
		12, 33, 54, 34, 55, 13, 56, 14, 35, 15, 36, 57,
		37, 58, 16, 59, 17, 38, 18, 39, 60, 40, 61, 19,
		62, 20, 41, 63,
	},
	last:  []int{63},
	lastN: 2,
}

const (
	shaCryptDefaultRounds = 5000
	shaCryptMinRounds     = 1000
	shaCryptMaxRounds     = 999999999
	shaCryptMaxSalt       = 16
)

func verifySHACrypt(password, hash string, spec shaSpec) error {
	rest := strings.TrimPrefix(hash, spec.prefix)

	rounds := shaCryptDefaultRounds
	roundsTag := ""

	if strings.HasPrefix(rest, "rounds=") {
		number, tail, ok := strings.Cut(rest[len("rounds="):], "$")
		if !ok {
			return errUnsupportedHash
		}

		parsed, err := strconv.Atoi(number)
		if err != nil || parsed < shaCryptMinRounds || parsed > shaCryptMaxRounds {
			return errUnsupportedHash
		}

		rounds = parsed
		roundsTag = "rounds=" + number + "$"
		rest = tail
	}

	salt, _, ok := strings.Cut(rest, "$")
	if !ok || salt == "" {
		return errUnsupportedHash
	}

	if len(salt) > shaCryptMaxSalt {
		salt = salt[:shaCryptMaxSalt]
	}

	computed := shaCrypt(password, salt, rounds, spec)
	canonical := spec.prefix + roundsTag + salt + "$" + computed

	if subtle.ConstantTimeCompare([]byte(canonical), []byte(hash)) != 1 {
		return model.ErrInvalidCredentials
	}

	return nil
}

func shaCrypt(password, salt string, rounds int, spec shaSpec) string {
	pw := []byte(password)
	saltBytes := []byte(salt)

	alternate := spec.new()
	alternate.Write(pw)
	alternate.Write(saltBytes)
	alternate.Write(pw)
	digest := alternate.Sum(nil)

	ctx := spec.new()
	ctx.Write(pw)
	ctx.Write(saltBytes)

	remaining := len(pw)
	for remaining > spec.size {
		ctx.Write(digest)
		remaining -= spec.size
	}

	ctx.Write(digest[:remaining])

	for i := len(pw); i != 0; i >>= 1 {
		if i&1 != 0 {
			ctx.Write(digest)
		} else {
			ctx.Write(pw)
		}
	}

	digest = ctx.Sum(nil)

	pBytes := spec.new()
	for i := 0; i < len(pw); i++ {
		pBytes.Write(pw)
	}

	pSequence := pBytes.Sum(nil)

	sBytes := spec.new()
	for i := 0; i < 16+int(digest[0]); i++ {
		sBytes.Write(saltBytes)
	}

	sSequence := sBytes.Sum(nil)

	for i := 0; i < rounds; i++ {
		round := spec.new()

		if i&1 != 0 {
			writeRecycled(round, pSequence, len(pw))
		} else {
			round.Write(digest)
		}

		if i%3 != 0 {
			writeRecycled(round, sSequence, len(saltBytes))
		}

		if i%7 != 0 {
			writeRecycled(round, pSequence, len(pw))
		}

		if i&1 != 0 {
			round.Write(digest)
		} else {
			writeRecycled(round, pSequence, len(pw))
		}

		digest = round.Sum(nil)
	}

	var encoded strings.Builder

	for i := 0; i+2 < len(spec.order); i += 3 {
		writeCrypt24(&encoded, digest[spec.order[i]], digest[spec.order[i+1]], digest[spec.order[i+2]], 4)
	}

	var final [3]byte

	for i, index := range spec.last {
		final[3-len(spec.last)+i] = digest[index]
	}

	writeCrypt24(&encoded, final[0], final[1], final[2], spec.lastN)

	return encoded.String()
}

// writeRecycled feeds count bytes of a virtual infinite repetition of block.
func writeRecycled(ctx hash.Hash, block []byte, count int) {
	if len(block) == 0 {
		return
	}

	for ; count >= len(block); count -= len(block) {
		ctx.Write(block)
	}

	ctx.Write(block[:count])
}

func writeCrypt24(builder *strings.Builder, b2, b1, b0 byte, count int) {
	value := uint32(b2)<<16 | uint32(b1)<<8 | uint32(b0)

	for ; count > 0; count-- {
		builder.WriteByte(cryptAlphabet[value&0x3f])
		value >>= 6
	}
}

// verifyDES verifies a traditional 13-character crypt(3) DES hash.
func verifyDES(password, hash string) error {
	computed, err := desCrypt(password, hash[:2])
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) != 1 {
		return model.ErrInvalidCredentials
	}

	return nil
}
