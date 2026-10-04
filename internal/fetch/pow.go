package fetch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var nonceRe = regexp.MustCompile(`nonce="([0-9a-f]+)"`)

func extractNonce(body string) (string, error) {
	m := nonceRe.FindStringSubmatch(body)
	if m == nil {
		return "", fmt.Errorf("pow: no nonce found in challenge body")
	}
	return m[1], nil
}

func solvePow(nonce, target string) (int, error) {
	if target == "" {
		target = "00"
	}
	buf := append([]byte(nonce), ':')
	for n := 0; n < 100_000_000; n++ {
		sum := sha256.Sum256(append(buf, strconv.Itoa(n)...))
		if strings.HasPrefix(hex.EncodeToString(sum[:]), target) {
			return n, nil
		}
	}
	return 0, fmt.Errorf("pow: no solution found for nonce %s", nonce)
}
