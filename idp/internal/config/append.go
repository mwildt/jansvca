package config

import (
	"bytes"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// AppendUser parses an IdP YAML config document (preserving comments and
// formatting via yaml.Node), appends a new user with the given subject, name
// and bcrypt-hashed password, and returns the updated document. It returns an
// error if a user with the same subject already exists.
func AppendUser(data []byte, subject, name, password string) ([]byte, error) {
	if subject == "" {
		return nil, fmt.Errorf("idp: subject is required")
	}
	if password == "" {
		return nil, fmt.Errorf("idp: password is required")
	}

	var existing Config
	if err := yaml.Unmarshal(data, &existing); err != nil {
		return nil, fmt.Errorf("idp: parse config: %w", err)
	}
	for _, u := range existing.Users {
		if u.Subject == subject {
			return nil, fmt.Errorf("idp: user %q already exists", subject)
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("idp: hash password: %w", err)
	}

	var doc yaml.Node
	if len(bytes.TrimSpace(data)) > 0 {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("idp: parse config: %w", err)
		}
	}
	root := docMapping(&doc)

	i, usersNode := findMappingEntry(root, "users")
	if usersNode == nil || usersNode.Kind != yaml.SequenceNode {
		usersNode = &yaml.Node{Kind: yaml.SequenceNode}
		if i >= 0 {
			root.Content[i+1] = usersNode
		} else {
			root.Content = append(root.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "users"},
				usersNode,
			)
		}
	}
	usersNode.Content = append(usersNode.Content, userNode(subject, name, string(hash)))

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, fmt.Errorf("idp: encode config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("idp: close config: %w", err)
	}
	return out.Bytes(), nil
}

// docMapping returns the root mapping node of a YAML document, creating a fresh
// document/mapping when the input was empty.
func docMapping(doc *yaml.Node) *yaml.Node {
	if doc.Kind == 0 || len(doc.Content) == 0 {
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	}
	root := doc.Content[0]
	if root.Kind == 0 {
		root.Kind = yaml.MappingNode
	}
	return root
}

// findMappingEntry returns the index and value node for a top-level mapping key,
// or -1 and nil if the key is absent.
func findMappingEntry(root *yaml.Node, key string) (int, *yaml.Node) {
	for i := 0; i+1 < len(root.Content); i += 2 {
		k := root.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			return i, root.Content[i+1]
		}
	}
	return -1, nil
}

// userNode builds a mapping node for a single user entry. The password is
// stored as a bcrypt hash under password_hash; name is omitted when empty to
// keep the config minimal.
func userNode(subject, name, hash string) *yaml.Node {
	m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	add := func(k, v string) {
		m.Content = append(m.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v},
		)
	}
	add("subject", subject)
	if name != "" {
		add("name", name)
	}
	add("password_hash", hash)
	return m
}
