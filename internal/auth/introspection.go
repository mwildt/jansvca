package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// IntrospectionVerifier verifies OAuth2 bearer tokens against an OAuth2 token
// introspection endpoint (RFC 7662). The provider URL is configurable, making
// it provider-agnostic.
type IntrospectionVerifier struct {
	Endpoint     string
	ClientID     string
	ClientSecret string
	httpClient   *http.Client
}

// NewIntrospectionVerifier creates a verifier for the given introspection
// endpoint.
func NewIntrospectionVerifier(endpoint, clientID, clientSecret string) *IntrospectionVerifier {
	return &IntrospectionVerifier{
		Endpoint:     endpoint,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

type introspectionResponse struct {
	Active bool   `json:"active"`
	Sub    string `json:"sub"`
	SubAlt string `json:"username"`
	Scope  string `json:"scope"`
	Exp    int64  `json:"exp"`
}

// Verify calls the introspection endpoint and returns the principal.
func (v *IntrospectionVerifier) Verify(ctx context.Context, token string) (User, error) {
	form := url.Values{}
	form.Set("token", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.Endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return User{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if v.ClientID != "" || v.ClientSecret != "" {
		req.SetBasicAuth(v.ClientID, v.ClientSecret)
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return User{}, ErrUnauthorized
	}
	var ir introspectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&ir); err != nil {
		return User{}, err
	}
	if !ir.Active {
		return User{}, ErrUnauthorized
	}
	if ir.Exp > 0 && time.Now().Unix() > ir.Exp {
		return User{}, ErrUnauthorized
	}
	subject := ir.Sub
	if subject == "" {
		subject = ir.SubAlt
	}
	if subject == "" {
		return User{}, errors.New("auth: introspection returned no subject")
	}
	return User{
		Subject: subject,
		Name:    ir.SubAlt,
		Roles:   []Role{RoleAdmin},
	}, nil
}
