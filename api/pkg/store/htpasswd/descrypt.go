package htpasswd

import (
	"encoding/binary"
	"errors"
	"strings"
)

// Traditional crypt(3) DES. The algorithm is plain DES (FIPS 46-3) with two
// differences: it encrypts the 64-bit zero block 25 times with the password
// derived key, and the salt swaps twelve bits inside the E expansion, which is
// the historical "salt in the E-box" construction.
//
// The implementation follows the FreeSec/libxcrypt lineage so hashes written by
// glibc, Apache and old BSD tools verify identically.

var errInvalidDESSalt = errors.New("htpasswd: invalid DES salt")

var desIPTables = []int{
	58, 50, 42, 34, 26, 18, 10, 2,
	60, 52, 44, 36, 28, 20, 12, 4,
	62, 54, 46, 38, 30, 22, 14, 6,
	64, 56, 48, 40, 32, 24, 16, 8,
	57, 49, 41, 33, 25, 17, 9, 1,
	59, 51, 43, 35, 27, 19, 11, 3,
	61, 53, 45, 37, 29, 21, 13, 5,
	63, 55, 47, 39, 31, 23, 15, 7,
}

var desFPTables = []int{
	40, 8, 48, 16, 56, 24, 64, 32,
	39, 7, 47, 15, 55, 23, 63, 31,
	38, 6, 46, 14, 54, 22, 62, 30,
	37, 5, 45, 13, 53, 21, 61, 29,
	36, 4, 44, 12, 52, 20, 60, 28,
	35, 3, 43, 11, 51, 19, 59, 27,
	34, 2, 42, 10, 50, 18, 58, 26,
	33, 1, 41, 9, 49, 17, 57, 25,
}

var desETable = []int{
	32, 1, 2, 3, 4, 5,
	4, 5, 6, 7, 8, 9,
	8, 9, 10, 11, 12, 13,
	12, 13, 14, 15, 16, 17,
	16, 17, 18, 19, 20, 21,
	20, 21, 22, 23, 24, 25,
	24, 25, 26, 27, 28, 29,
	28, 29, 30, 31, 32, 1,
}

var desPTable = []int{
	16, 7, 20, 21, 29, 12, 28, 17,
	1, 15, 23, 26, 5, 18, 31, 10,
	2, 8, 24, 14, 32, 27, 3, 9,
	19, 13, 30, 6, 22, 11, 4, 25,
}

var desPC1Table = []int{
	57, 49, 41, 33, 25, 17, 9,
	1, 58, 50, 42, 34, 26, 18,
	10, 2, 59, 51, 43, 35, 27,
	19, 11, 3, 60, 52, 44, 36,
	63, 55, 47, 39, 31, 23, 15,
	7, 62, 54, 46, 38, 30, 22,
	14, 6, 61, 53, 45, 37, 29,
	21, 13, 5, 28, 20, 12, 4,
}

var desPC2Table = []int{
	14, 17, 11, 24, 1, 5,
	3, 28, 15, 6, 21, 10,
	23, 19, 12, 4, 26, 8,
	16, 7, 27, 20, 13, 2,
	41, 52, 31, 37, 47, 55,
	30, 40, 51, 45, 33, 48,
	44, 49, 39, 56, 34, 53,
	46, 42, 50, 36, 29, 32,
}

var desShifts = []int{1, 1, 2, 2, 2, 2, 2, 2, 1, 2, 2, 2, 2, 2, 2, 1}

var desSBoxes = [8][64]uint32{
	{
		14, 4, 13, 1, 2, 15, 11, 8, 3, 10, 6, 12, 5, 9, 0, 7,
		0, 15, 7, 4, 14, 2, 13, 1, 10, 6, 12, 11, 9, 5, 3, 8,
		4, 1, 14, 8, 13, 6, 2, 11, 15, 12, 9, 7, 3, 10, 5, 0,
		15, 12, 8, 2, 4, 9, 1, 7, 5, 11, 3, 14, 10, 0, 6, 13,
	},
	{
		15, 1, 8, 14, 6, 11, 3, 4, 9, 7, 2, 13, 12, 0, 5, 10,
		3, 13, 4, 7, 15, 2, 8, 14, 12, 0, 1, 10, 6, 9, 11, 5,
		0, 14, 7, 11, 10, 4, 13, 1, 5, 8, 12, 6, 9, 3, 2, 15,
		13, 8, 10, 1, 3, 15, 4, 2, 11, 6, 7, 12, 0, 5, 14, 9,
	},
	{
		10, 0, 9, 14, 6, 3, 15, 5, 1, 13, 12, 7, 11, 4, 2, 8,
		13, 7, 0, 9, 3, 4, 6, 10, 2, 8, 5, 14, 12, 11, 15, 1,
		13, 6, 4, 9, 8, 15, 3, 0, 11, 1, 2, 12, 5, 10, 14, 7,
		1, 10, 13, 0, 6, 9, 8, 7, 4, 15, 14, 3, 11, 5, 2, 12,
	},
	{
		7, 13, 14, 3, 0, 6, 9, 10, 1, 2, 8, 5, 11, 12, 4, 15,
		13, 8, 11, 5, 6, 15, 0, 3, 4, 7, 2, 12, 1, 10, 14, 9,
		10, 6, 9, 0, 12, 11, 7, 13, 15, 1, 3, 14, 5, 2, 8, 4,
		3, 15, 0, 6, 10, 1, 13, 8, 9, 4, 5, 11, 12, 7, 2, 14,
	},
	{
		2, 12, 4, 1, 7, 10, 11, 6, 8, 5, 3, 15, 13, 0, 14, 9,
		14, 11, 2, 12, 4, 7, 13, 1, 5, 0, 15, 10, 3, 9, 8, 6,
		4, 2, 1, 11, 10, 13, 7, 8, 15, 9, 12, 5, 6, 3, 0, 14,
		11, 8, 12, 7, 1, 14, 2, 13, 6, 15, 0, 9, 10, 4, 5, 3,
	},
	{
		12, 1, 10, 15, 9, 2, 6, 8, 0, 13, 3, 4, 14, 7, 5, 11,
		10, 15, 4, 2, 7, 12, 9, 5, 6, 1, 13, 14, 0, 11, 3, 8,
		9, 14, 15, 5, 2, 8, 12, 3, 7, 0, 4, 10, 1, 13, 11, 6,
		4, 3, 2, 12, 9, 5, 15, 10, 11, 14, 1, 7, 6, 0, 8, 13,
	},
	{
		4, 11, 2, 14, 15, 0, 8, 13, 3, 12, 9, 7, 5, 10, 6, 1,
		13, 0, 11, 7, 4, 9, 1, 10, 14, 3, 5, 12, 2, 15, 8, 6,
		1, 4, 11, 13, 12, 3, 7, 14, 10, 15, 6, 8, 0, 5, 9, 2,
		6, 11, 13, 8, 1, 4, 10, 7, 9, 5, 0, 15, 14, 2, 3, 12,
	},
	{
		13, 2, 8, 4, 6, 15, 11, 1, 10, 9, 3, 14, 5, 0, 12, 7,
		1, 15, 13, 8, 10, 3, 7, 4, 12, 5, 6, 11, 0, 14, 9, 2,
		7, 11, 4, 1, 9, 12, 14, 2, 0, 6, 10, 13, 15, 3, 5, 8,
		2, 1, 14, 7, 4, 10, 8, 13, 15, 12, 9, 0, 3, 5, 6, 11,
	},
}

// desCrypt returns the 13-character traditional crypt(3) hash.
func desCrypt(password, setting string) (string, error) {
	if len(setting) < 2 {
		return "", errInvalidDESSalt
	}

	salt, ok := desSaltValue(setting[0], setting[1])
	if !ok {
		return "", errInvalidDESSalt
	}

	key := desKey(password)
	subkeys := desSubkeys(key)

	swaps := desSaltSwaps(salt)

	var block uint64

	for i := 0; i < 25; i++ {
		block = desEncryptBlock(block, subkeys, swaps)
	}

	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], block)

	return setting[:2] + encodeCrypt(encoded[:]), nil
}

func desSaltValue(first, second byte) (uint32, bool) {
	a := strings.IndexByte(cryptAlphabet, first)
	b := strings.IndexByte(cryptAlphabet, second)
	if a < 0 || b < 0 {
		return 0, false
	}

	return uint32(a) | uint32(b)<<6, true
}

// desSaltSwaps returns the pairs of E-expansion bit positions the salt swaps.
// FreeSec keeps the salt in the top twelve bits of both 24-bit expansion
// halves, so salt bit i exchanges expansion bit 23-i with expansion bit 47-i
// (the S1/S5 and S2/S6 inputs).
func desSaltSwaps(salt uint32) [][2]int {
	swaps := make([][2]int, 0, 12)

	for bit := 0; bit < 12; bit++ {
		if salt&(1<<uint(bit)) == 0 {
			continue
		}

		swaps = append(swaps, [2]int{23 - bit, 47 - bit})
	}

	return swaps
}

// desKey mirrors FreeSec's key preparation: the first eight characters, each
// shifted left by one bit to drop the DES parity bit.
func desKey(password string) uint64 {
	var key uint64

	index := 0
	for i := 0; i < 8; i++ {
		var ch byte
		if index < len(password) {
			ch = password[index]
			if ch != 0 {
				index++
			}
		}

		key = key<<8 | uint64(ch)<<1
	}

	return key
}

func desSubkeys(key uint64) [16]uint64 {
	var subkeys [16]uint64

	permuted := desPermute(key, 64, desPC1Table)

	c := uint32(permuted>>28) & 0x0fffffff
	d := uint32(permuted) & 0x0fffffff

	for round := 0; round < 16; round++ {
		c = desRotate28(c, desShifts[round])
		d = desRotate28(d, desShifts[round])

		subkeys[round] = desPermute(uint64(c)<<28|uint64(d), 56, desPC2Table)
	}

	return subkeys
}

func desRotate28(value uint32, shift int) uint32 {
	value &= 0x0fffffff

	return ((value << uint(shift)) | (value >> uint(28-shift))) & 0x0fffffff
}

func desEncryptBlock(block uint64, subkeys [16]uint64, swaps [][2]int) uint64 {
	permuted := desPermute(block, 64, desIPTables)

	left := uint32(permuted >> 32)
	right := uint32(permuted)

	for round := 0; round < 16; round++ {
		left, right = right, left^desFeistel(right, subkeys[round], swaps)
	}

	return desPermute(uint64(right)<<32|uint64(left), 64, desFPTables)
}

func desFeistel(right uint32, subkey uint64, swaps [][2]int) uint32 {
	expanded := desPermute(uint64(right), 32, desETable)

	for _, pair := range swaps {
		left := (expanded >> uint(pair[0])) & 1
		other := (expanded >> uint(pair[1])) & 1

		if left != other {
			expanded ^= 1<<uint(pair[0]) | 1<<uint(pair[1])
		}
	}

	expanded ^= subkey

	var sBoxed uint32

	for box := 0; box < 8; box++ {
		chunk := uint32(expanded>>uint(42-6*box)) & 0x3f
		row := ((chunk >> 4) & 0x2) | (chunk & 0x1)
		column := (chunk >> 1) & 0x0f

		sBoxed = sBoxed<<4 | desSBoxes[box][row*16+column]
	}

	return uint32(desPermute(uint64(sBoxed), 32, desPTable))
}

func desPermute(value uint64, bits int, table []int) uint64 {
	var out uint64

	for _, position := range table {
		out <<= 1
		out |= (value >> uint(bits-position)) & 1
	}

	return out
}

func encodeCrypt(block []byte) string {
	var builder strings.Builder

	index := 0
	for {
		first := uint32(block[index])
		index++
		builder.WriteByte(cryptAlphabet[first>>2])
		first = (first & 0x03) << 4

		if index >= len(block) {
			builder.WriteByte(cryptAlphabet[first])

			break
		}

		second := uint32(block[index])
		index++
		first |= second >> 4
		builder.WriteByte(cryptAlphabet[first])
		first = (second & 0x0f) << 2

		if index >= len(block) {
			builder.WriteByte(cryptAlphabet[first])

			break
		}

		third := uint32(block[index])
		index++
		first |= third >> 6
		builder.WriteByte(cryptAlphabet[first])
		builder.WriteByte(cryptAlphabet[third&0x3f])

		if index >= len(block) {
			break
		}
	}

	return builder.String()
}
