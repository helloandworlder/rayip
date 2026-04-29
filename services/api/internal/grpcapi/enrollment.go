package grpcapi

import "errors"

func validateEnrollmentToken(expected string, actual string) error {
	if expected == "" {
		return nil
	}
	if actual == "" {
		return errors.New("enrollment token is required")
	}
	if actual != expected {
		return errors.New("invalid enrollment token")
	}
	return nil
}
