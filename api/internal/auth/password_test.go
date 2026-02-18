package auth

import (
	"testing"
)

func TestHashPassword_NonEmpty(t *testing.T) {
	hash, err := HashPassword("s3cret")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty string")
	}
}

func TestVerifyPassword_Correct(t *testing.T) {
	password := "correct-horse-battery-staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if err := VerifyPassword(hash, password); err != nil {
		t.Fatalf("VerifyPassword failed with correct password: %v", err)
	}
}

func TestVerifyPassword_Wrong(t *testing.T) {
	hash, err := HashPassword("real-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if err := VerifyPassword(hash, "wrong-password"); err == nil {
		t.Fatal("VerifyPassword should fail with wrong password")
	}
}

func TestHashPassword_DifferentSalts(t *testing.T) {
	password := "same-password"
	h1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("first HashPassword returned error: %v", err)
	}
	h2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("second HashPassword returned error: %v", err)
	}

	if h1 == h2 {
		t.Fatal("HashPassword produced identical hashes for the same input - bcrypt salt is not working")
	}
}
