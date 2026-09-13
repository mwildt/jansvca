// Package config loads the IdP users from a YAML file. Each user has a subject,
// an optional display name, and a bcrypt password hash. The file is read once at
// startup and kept in memory.
package config

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// User is a single identity known to the IdP.
type User struct {
	// Subject is the stable user identifier (sub claim).
	Subject string `yaml:"subject"`
	// Name is an optional human-readable display name.
	Name string `yaml:"name"`
	// PasswordHash is a bcrypt hash of the user's password (base64/std or raw
	// bcrypt output, "$2a$...").
	PasswordHash string `yaml:"password_hash"`
	// Password is a plaintext password, accepted for local development
	// convenience. If set, it is hashed on load and PasswordHash takes
	// precedence when both are present.
	Password string `yaml:"password"`
}

// Config is the top-level IdP configuration file.
type Config struct {
	// Clients are the OAuth2 clients allowed to use this IdP. A client is
	// identified by its id and an optional shared secret; the configured
	// redirect_uris are the only callback URLs accepted.
	Clients []Client `yaml:"clients"`
	Users   []User   `yaml:"users"`
}

// Client is an OAuth2 client (the BFF in our setup).
type Client struct {
	ID           string   `yaml:"id"`
	Secret       string   `yaml:"secret"`
	RedirectURIs []string `yaml:"redirect_uris"`
}

// Store holds the loaded config and indexes for fast lookup.
type Store struct {
	users   []User
	clients map[string]Client
}

// Load reads and parses the YAML config file at path. Plaintext Password
// values are hashed once with bcrypt on load for dev convenience.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("idp: read config: %w", err)
	}
	return Parse(data)
}

// Parse decodes a YAML config document. Plaintext Password values are hashed
// once with bcrypt for dev convenience. Used by tests and Load.
func Parse(data []byte) (*Store, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("idp: parse config: %w", err)
	}
	for i := range cfg.Users {
		u := &cfg.Users[i]
		if u.PasswordHash == "" && u.Password != "" {
			h, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
			if err != nil {
				return nil, fmt.Errorf("idp: hash password for %q: %w", u.Subject, err)
			}
			u.PasswordHash = string(h)
			u.Password = ""
		}
	}
	clients := map[string]Client{}
	for _, c := range cfg.Clients {
		clients[c.ID] = c
	}
	return &Store{users: cfg.Users, clients: clients}, nil
}

// VerifyPassword returns the user matching subject/password, or
// ErrInvalidCredentials.
func (s *Store) VerifyPassword(subject, password string) (User, error) {
	for _, u := range s.users {
		if u.Subject != subject {
			continue
		}
		if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err == nil {
			return u, nil
		}
	}
	return User{}, ErrInvalidCredentials
}

// User returns a user by subject, or false.
func (s *Store) User(subject string) (User, bool) {
	for _, u := range s.users {
		if u.Subject == subject {
			return u, true
		}
	}
	return User{}, false
}

// Client returns a client by id, or false.
func (s *Store) Client(id string) (Client, bool) {
	c, ok := s.clients[id]
	return c, ok
}

// Users returns all loaded users (for startup logging).
func (s *Store) Users() []User { return s.users }

// Clients returns all loaded client ids (for startup logging).
func (s *Store) Clients() []string {
	ids := make([]string, 0, len(s.clients))
	for id := range s.clients {
		ids = append(ids, id)
	}
	return ids
}

// LoadString parses a YAML config document from a string. Convenience for tests.
func LoadString(data string) (*Store, error) {
	return Parse([]byte(data))
}

// ErrInvalidCredentials is returned when subject/password don't match.
var ErrInvalidCredentials = errors.New("idp: invalid credentials")
