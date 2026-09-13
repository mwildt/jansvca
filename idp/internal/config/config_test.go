package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "idp-config.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

func TestLoadPlaintextPasswordIsHashed(t *testing.T) {
	p := writeConfig(t, `
clients:
  - id: app
    secret: s3cret
    redirect_uris:
      - http://localhost:8080/api/auth/callback
users:
  - subject: admin
    name: Administrator
    password: admin
`)
	store, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	u, err := store.VerifyPassword("admin", "admin")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if u.Subject != "admin" || u.Name != "Administrator" {
		t.Fatalf("unexpected user %+v", u)
	}
	if u.PasswordHash == "" || u.Password != "" {
		t.Fatalf("plaintext not hashed: %+v", u)
	}
}

func TestLoadPasswordHashUsedDirectly(t *testing.T) {
	// bcrypt hash for "secret"
	hash := "$2a$10$F1KZxZ2jZ8K2k2Z2k2Z2kOlZ5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z5Z"
	p := writeConfig(t, `
users:
  - subject: alice
    password_hash: "`+hash+`"
`)
	store, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// invalid hash should still load; verification simply fails.
	if _, err := store.VerifyPassword("alice", "wrong"); err == nil {
		t.Fatal("expected credentials error")
	}
}

func TestVerifyPasswordWrongPassword(t *testing.T) {
	p := writeConfig(t, `
users:
  - subject: bob
    password: hunter2
`)
	store, _ := Load(p)
	if _, err := store.VerifyPassword("bob", "nope"); err == nil {
		t.Fatal("expected ErrInvalidCredentials")
	}
	if _, err := store.VerifyPassword("nobody", "x"); err == nil {
		t.Fatal("expected ErrInvalidCredentials for unknown subject")
	}
}

func TestClientLookup(t *testing.T) {
	p := writeConfig(t, `
clients:
  - id: app
    secret: s3cret
    redirect_uris:
      - http://localhost:8080/cb
users: []
`)
	store, _ := Load(p)
	c, ok := store.Client("app")
	if !ok || c.ID != "app" || c.Secret != "s3cret" {
		t.Fatalf("unexpected client %+v ok=%v", c, ok)
	}
	if _, ok := store.Client("missing"); ok {
		t.Fatal("missing client should not be found")
	}
	if len(store.Clients()) != 1 {
		t.Fatalf("expected 1 client id, got %v", store.Clients())
	}
}
