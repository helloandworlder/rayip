package grpcapi

import "testing"

func TestValidateEnrollmentTokenAllowsConfiguredToken(t *testing.T) {
	if err := validateEnrollmentToken("secret-token", "secret-token"); err != nil {
		t.Fatalf("validateEnrollmentToken() error = %v", err)
	}
}

func TestValidateEnrollmentTokenRejectsMismatch(t *testing.T) {
	if err := validateEnrollmentToken("secret-token", "wrong-token"); err == nil {
		t.Fatal("validateEnrollmentToken() error = nil, want mismatch error")
	}
}

func TestValidateEnrollmentTokenAllowsDevEmptyExpected(t *testing.T) {
	if err := validateEnrollmentToken("", "anything"); err != nil {
		t.Fatalf("validateEnrollmentToken() with empty expected error = %v", err)
	}
}
