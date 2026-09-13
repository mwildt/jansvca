package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

const existingConfig = `clients:
  - id: app
    secret: s3cret
    redirect_uris:
      - http://localhost:8080/cb
users:
  - subject: admin
    name: Administrator
    password: admin
`

func TestAppendUserAddsHashedUser(t *testing.T) {
	out, err := AppendUser([]byte(existingConfig), "alice", "Alice Liddell", "wonderland")
	if err != nil {
		t.Fatalf("AppendUser: %v", err)
	}

	store, err := Parse(out)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	u, err := store.VerifyPassword("alice", "wonderland")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if u.Name != "Alice Liddell" {
		t.Fatalf("unexpected name %q", u.Name)
	}
	if u.PasswordHash == "" || u.Password != "" {
		t.Fatalf("password not hashed: %+v", u)
	}
	if _, err := store.VerifyPassword("alice", "wrong"); err == nil {
		t.Fatal("expected credentials error for wrong password")
	}
	// existing admin must still validate
	if _, err := store.VerifyPassword("admin", "admin"); err != nil {
		t.Fatalf("admin no longer validates: %v", err)
	}
}

func TestAppendUserPreservesClients(t *testing.T) {
	out, err := AppendUser([]byte(existingConfig), "bob", "", "secret")
	if err != nil {
		t.Fatalf("AppendUser: %v", err)
	}
	store, err := Parse(out)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c, ok := store.Client("app")
	if !ok || c.Secret != "s3cret" {
		t.Fatalf("client not preserved: %+v ok=%v", c, ok)
	}
}

func TestAppendUserRejectsDuplicate(t *testing.T) {
	if _, err := AppendUser([]byte(existingConfig), "admin", "x", "y"); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestAppendUserValidatesInput(t *testing.T) {
	if _, err := AppendUser([]byte(existingConfig), "", "n", "p"); err == nil {
		t.Fatal("expected empty subject error")
	}
	if _, err := AppendUser([]byte(existingConfig), "x", "n", ""); err == nil {
		t.Fatal("expected empty password error")
	}
}

func TestAppendUserHashIsBcrypt(t *testing.T) {
	out, err := AppendUser(nil, "carol", "Carol", "password123")
	if err != nil {
		t.Fatalf("AppendUser: %v", err)
	}
	store, err := Parse(out)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	u, _ := store.User("carol")
	// bcrypt.CompareHashAndPassword verifies a real bcrypt hash against input.
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("password123")); err != nil {
		t.Fatalf("hash is not bcrypt-comparable: %v", err)
	}
}

func TestAppendUserRoundTripsOnDisk(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "idp-config.yaml")
	if err := os.WriteFile(p, []byte(existingConfig), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	out, err := AppendUser(data, "dave", "Dave", "hunter2")
	if err != nil {
		t.Fatalf("AppendUser: %v", err)
	}
	if err := os.WriteFile(p, out, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	reloaded, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := reloaded.VerifyPassword("dave", "hunter2"); err != nil {
		t.Fatalf("VerifyPassword after reload: %v", err)
	}
	if !bytes.Contains(out, []byte("password_hash:")) {
		t.Fatalf("expected password_hash in output:\n%s", out)
	}
}
