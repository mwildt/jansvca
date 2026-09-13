// Command idp-user adds a user to an IdP config file. The password is stored
// as a bcrypt hash under password_hash; existing entries, comments and
// formatting are preserved.
//
// Usage:
//
//	idp-user -config ./idp-config.yaml -subject alice -name "Alice Liddell" -password-stdin
//	printf 's3cret' | idp-user -config ./idp-config.yaml -subject alice -password-stdin
//	idp-user -config ./idp-config.yaml -subject alice -password s3cret
//
// It refuses to overwrite an existing user. The config file is rewritten in
// place only after a successful, non-duplicate append. Prefer -password-stdin
// over -password so the password does not end up in the shell history.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/mwildt/jansvca/idp/internal/config"
)

func main() {
	configPath := flag.String("config", envOr("IDP_CONFIG", "./idp-config.yaml"), "path to the IdP config file")
	subject := flag.String("subject", "", "user subject (login id), required")
	name := flag.String("name", "", "optional display name")
	passwordStdin := flag.Bool("password-stdin", false, "read the password from stdin instead of -password")
	passwordFlag := flag.String("password", "", "user password (will appear in shell history; prefer -password-stdin)")
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "usage: idp-user -config <path> -subject <id> [-name <display>] [-password-stdin | -password <pw>]\n\n")
		fmt.Fprintf(out, "Adds a user with a bcrypt-hashed password to the IdP config file.\nExisting users, clients, comments and formatting are preserved.\n\n")
		fmt.Fprintf(out, "Prefer -password-stdin (e.g. read from a file or a password manager) so the\npassword is not stored in the shell history.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *subject == "" {
		log.Fatal("idp-user: -subject is required")
	}
	*subject = strings.TrimSpace(*subject)

	data, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("idp-user: read config: %v", err)
	}

	password, err := readPassword(*passwordStdin, *passwordFlag)
	if err != nil {
		log.Fatalf("idp-user: read password: %v", err)
	}
	if password == "" {
		log.Fatal("idp-user: password is required")
	}

	out, err := config.AppendUser(data, *subject, *name, password)
	if err != nil {
		log.Fatalf("idp-user: %v", err)
	}
	if err := os.WriteFile(*configPath, out, 0o600); err != nil {
		log.Fatalf("idp-user: write config: %v", err)
	}
	log.Printf("idp-user: added user %q to %s", *subject, *configPath)
}

func readPassword(stdin bool, flag string) (string, error) {
	if flag != "" {
		if stdin {
			return "", fmt.Errorf("-password and -password-stdin are mutually exclusive")
		}
		return flag, nil
	}
	if stdin {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(b), "\r\n"), nil
	}
	return "", fmt.Errorf("no password provided (use -password-stdin or -password)")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
