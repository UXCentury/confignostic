package identity

import (
	"strings"
	"testing"
)

func TestPasswordRoundTripAndIndependentSalts(t *testing.T) {
	password := "a long password with spaces 🦊"
	first, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || strings.Contains(first, password) {
		t.Fatal("password leaked or salt reused")
	}
	for _, hash := range []string{first, second} {
		if err := VerifyPassword(hash, password); err != nil {
			t.Fatal(err)
		}
		if err := VerifyPassword(hash, password+" "); err != ErrCredential {
			t.Fatal("wrong password accepted", err)
		}
	}
	for _, corrupt := range []string{"", strings.Repeat("x", 10000), strings.Replace(first, "600000", "999999", 1), strings.Replace(first, "v1", "v2", 1), first[:len(first)-1] + "!"} {
		if err := VerifyPassword(corrupt, password); err != ErrCredential {
			t.Fatal("corrupt hash accepted", err)
		}
	}
}

func TestPasswordBoundsAndNoTruncation(t *testing.T) {
	for _, password := range []string{"", strings.Repeat("a", 14), strings.Repeat("a", 1025), strings.Repeat("a", 15) + string([]byte{255})} {
		if _, err := HashPassword(password); err != ErrPassword {
			t.Fatal("invalid password accepted", err)
		}
	}
	for _, password := range []string{strings.Repeat("a", 15), strings.Repeat("a", 1024)} {
		hash, err := HashPassword(password)
		if err != nil {
			t.Fatal(err)
		}
		if err := VerifyPassword(hash, password); err != nil {
			t.Fatal(err)
		}
		if err := VerifyPassword(hash, password[:len(password)-1]+"b"); err != ErrCredential {
			t.Fatal("password truncated")
		}
	}
}

func TestTokenSecretsAndCanonicalDigests(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		token, err := NewToken()
		if err != nil {
			t.Fatal(err)
		}
		if len(token) != 43 || seen[token] {
			t.Fatal("invalid or repeated token")
		}
		seen[token] = true
		digest, err := TokenDigest(token)
		if err != nil || len(digest) != 64 || digest == token {
			t.Fatal("invalid digest", err)
		}
		again, err := TokenDigest(token)
		if err != nil || again != digest {
			t.Fatal("unstable digest")
		}
	}
	for _, token := range []string{"", strings.Repeat("a", 42), strings.Repeat("a", 44), strings.Repeat("!", 43), strings.Repeat("A", 42) + "B"} {
		if _, err := TokenDigest(token); err != ErrCredential {
			t.Fatal("malformed or noncanonical token accepted", err)
		}
	}
}

func FuzzCredentialParsing(f *testing.F) {
	f.Add("bad", "password", "token")
	f.Fuzz(func(t *testing.T, encoded, password, token string) {
		if len(encoded) > 10000 || len(password) > 2000 || len(token) > 10000 {
			return
		}
		_ = VerifyPassword(encoded, password)
		_, _ = TokenDigest(token)
	})
}
