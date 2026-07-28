package provider

import (
	"os"
	"testing"
)

// cldprovisioning v0.1.0 cannot accept credentials through WithSecurity: it stores
// the Security value in an interface{}, while the getters its auth hook type-asserts
// against are declared with pointer receivers, so the assertion never succeeds. The
// SDK then falls back to environment variables, which it reads once inside New().
//
// Those variable names carry a PROVISIONING_ infix and are therefore NOT the same as
// this provider's own CLOUDINARY_API_KEY / CLOUDINARY_API_SECRET fallbacks. Exporting
// the provider-named variables has no effect on the SDK.
func TestExportSDKCredentialEnvUsesProvisioningPrefixedNames(t *testing.T) {
	t.Setenv("CLOUDINARY_ACCOUNT_ID", "")
	t.Setenv("CLOUDINARY_PROVISIONING_API_KEY", "")
	t.Setenv("CLOUDINARY_PROVISIONING_API_SECRET", "")

	if err := exportSDKCredentialEnv("acct-1", "key-1", "secret-1"); err != nil {
		t.Fatalf("exportSDKCredentialEnv returned error: %v", err)
	}

	want := map[string]string{
		"CLOUDINARY_ACCOUNT_ID":              "acct-1",
		"CLOUDINARY_PROVISIONING_API_KEY":    "key-1",
		"CLOUDINARY_PROVISIONING_API_SECRET": "secret-1",
	}
	for name, expected := range want {
		if got := os.Getenv(name); got != expected {
			t.Errorf("%s = %q, want %q", name, got, expected)
		}
	}
}

// The exported variables are process-global while ProviderConfigs are per-instance,
// so a second Configure must fully replace the first one's credentials. Leaving a
// stale value behind would authenticate later requests as the wrong principal.
func TestExportSDKCredentialEnvOverwritesStaleValues(t *testing.T) {
	t.Setenv("CLOUDINARY_ACCOUNT_ID", "stale-acct")
	t.Setenv("CLOUDINARY_PROVISIONING_API_KEY", "stale-key")
	t.Setenv("CLOUDINARY_PROVISIONING_API_SECRET", "stale-secret")

	if err := exportSDKCredentialEnv("acct-2", "key-2", "secret-2"); err != nil {
		t.Fatalf("exportSDKCredentialEnv returned error: %v", err)
	}

	want := map[string]string{
		"CLOUDINARY_ACCOUNT_ID":              "acct-2",
		"CLOUDINARY_PROVISIONING_API_KEY":    "key-2",
		"CLOUDINARY_PROVISIONING_API_SECRET": "secret-2",
	}
	for name, expected := range want {
		if got := os.Getenv(name); got != expected {
			t.Errorf("%s = %q, want %q (stale value not replaced)", name, got, expected)
		}
	}
}
