package fetch

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

func powFor(t *testing.T, nonce, target string) int {
	t.Helper()
	n, err := solvePow(nonce, target)
	if err != nil {
		t.Fatalf("solvePow(%s): %v", nonce, err)
	}
	sum := sha256.Sum256([]byte(nonce + ":" + strconv.Itoa(n)))
	if got := hex.EncodeToString(sum[:]); !strings.HasPrefix(got, target) {
		t.Fatalf("returned n=%d but hash %s does not start with %s", n, got, target)
	}
	return n
}

func TestSolvePowRealChallenge(t *testing.T) {
	nonce := extractNonceFromFixture(t)
	n := powFor(t, nonce, "00")
	t.Logf("nonce %s -> n = %d", nonce, n)
}

func TestSolvePowMultipleNonces(t *testing.T) {
	for _, nonce := range []string{
		"281beec843a6a211",
		"3fda1aec728a6914",
		"0000000000000000",
		"ffffffffffffffff",
	} {
		powFor(t, nonce, "00")
	}
}

func TestSolvePowHarderTarget(t *testing.T) {
	powFor(t, "deadbeef", "0000")
}

func TestSolvePowDefaultTarget(t *testing.T) {
	if _, err := solvePow("abc123", ""); err != nil {
		t.Fatalf("empty target should default to 00: %v", err)
	}
}

func TestExtractNonce(t *testing.T) {
	body := `(function(){var nonce="281beec843a6a211",target='00';})();`
	got, err := extractNonce(body)
	if err != nil {
		t.Fatalf("extractNonce: %v", err)
	}
	if got != "281beec843a6a211" {
		t.Fatalf("got %q, want 281beec843a6a211", got)
	}
}

func TestExtractNonceMissing(t *testing.T) {
	if _, err := extractNonce("<html>no nonce here</html>"); err == nil {
		t.Fatal("expected error when nonce is absent")
	}
}

func TestIsChallenge(t *testing.T) {
	tests := []struct {
		body string
		want bool
	}{
		{`<p>Checking your browser.</p>`, true},
		{`var nonce="abc123"`, true},
		{`<html><body>UFC 332: Silva vs. Wang</body></html>`, false},
		{"", false},
	}
	for _, tc := range tests {
		if got := isChallenge(tc.body); got != tc.want {
			t.Errorf("isChallenge(%q) = %v, want %v", tc.body, got, tc.want)
		}
	}
}

func extractNonceFromFixture(t *testing.T) string {
	t.Helper()
	n, err := extractNonce(fixture(t, "challenge.html"))
	if err != nil {
		t.Skipf("fixture is not a challenge page: %v", err)
	}
	return n
}
