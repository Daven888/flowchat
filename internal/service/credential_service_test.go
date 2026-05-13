package service

import (
	"testing"
)

func TestValidateProvider(t *testing.T) {
	svc := &CredentialService{
		providerNames: []string{"mock", "deepseek", "siliconflow"},
	}

	tests := []struct {
		name     string
		provider string
		wantErr  error
	}{
		{"valid provider", "deepseek", nil},
		{"valid provider 2", "siliconflow", nil},
		{"mock provider", "mock", ErrMockNoKey},
		{"unknown provider", "unknown", ErrInvalidProvider},
		{"empty provider", "", ErrInvalidProvider},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateProvider(tt.provider)
			if err != tt.wantErr {
				t.Errorf("validateProvider(%q) = %v, want %v", tt.provider, err, tt.wantErr)
			}
		})
	}
}

func TestValidateProviderMockNotInList(t *testing.T) {
	// Even if "mock" is not explicitly in providerNames, it must still be rejected.
	svc := &CredentialService{
		providerNames: []string{"deepseek", "siliconflow"},
	}

	err := svc.validateProvider("mock")
	if err != ErrMockNoKey {
		t.Errorf("validateProvider(mock) = %v, want ErrMockNoKey", err)
	}
}

func TestComputeKeySuffix(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   string
	}{
		{"normal key", "sk-abcdefgh1234", "1234"},
		{"short key", "abc", "abc"},
		{"exact 4 chars", "abcd", "abcd"},
		{"empty key", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeKeySuffix(tt.apiKey)
			if got != tt.want {
				t.Errorf("computeKeySuffix(%q) = %q, want %q", tt.apiKey, got, tt.want)
			}
		})
	}
}

func TestSentinelErrorValues(t *testing.T) {
	_ = ErrCredentialNotFound
	_ = ErrMockNoKey
	_ = ErrInvalidProvider
	_ = ErrAPIKeyEmpty
}
