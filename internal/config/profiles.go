// Package config reads named profiles: several Penpot accounts or instances
// served by one penpot-mcp. The file holds only names of environment
// variables, never secrets.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
)

// Profile is one account. *Env fields name the environment variables that
// hold its secrets.
type Profile struct {
	Name        string `json:"name"`
	BaseURL     string `json:"base_url,omitempty"`
	TokenEnv    string `json:"token_env"`
	EmailEnv    string `json:"email_env,omitempty"`
	PasswordEnv string `json:"password_env,omitempty"`
	MCPURLEnv   string `json:"mcp_url_env,omitempty"`
}

var (
	nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
	envRE  = regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,63}$`)
)

// Load reads a JSON list of profiles from path.
func Load(path string) ([]Profile, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- the user's own config path
	if err != nil {
		return nil, fmt.Errorf("PENPOT_PROFILES: %w", err)
	}
	var ps []Profile
	if err := json.Unmarshal(data, &ps); err != nil {
		return nil, fmt.Errorf("PENPOT_PROFILES is not a JSON list of profiles: %w", err)
	}
	if len(ps) == 0 {
		return nil, errors.New("PENPOT_PROFILES lists no profiles")
	}
	seen := map[string]bool{}
	for i, p := range ps {
		if !nameRE.MatchString(p.Name) {
			return nil, fmt.Errorf("profile %d: name must be lowercase letters, digits, _ or -, up to 32", i+1)
		}
		if seen[p.Name] {
			return nil, fmt.Errorf("profile %q is listed twice", p.Name)
		}
		seen[p.Name] = true
		for label, env := range map[string]string{"token_env": p.TokenEnv, "email_env": p.EmailEnv, "password_env": p.PasswordEnv, "mcp_url_env": p.MCPURLEnv} {
			if env == "" && label != "token_env" {
				continue
			}
			if !envRE.MatchString(env) {
				return nil, fmt.Errorf("profile %q: %s must name an environment variable, such as PENPOT_WORK_TOKEN", p.Name, label)
			}
		}
		if os.Getenv(p.TokenEnv) == "" {
			return nil, fmt.Errorf("profile %q: %s is not set", p.Name, p.TokenEnv)
		}
	}
	return ps, nil
}
