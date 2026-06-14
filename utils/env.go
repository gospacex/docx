package utils

import (
	"fmt"
	"os"
	"regexp"
)

var envRe = regexp.MustCompile(`\$\{env:([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

func ExpandEnvVars(s string) (string, error) {
	var firstErr error
	out := envRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := envRe.FindStringSubmatch(m)
		name, def := sub[1], sub[2]
		if v, ok := os.LookupEnv(name); ok {
			return v
		}
		if def == "" && firstErr == nil {
			firstErr = fmt.Errorf("utils: env var %q not set and no default", name)
		}
		return def
	})
	if firstErr != nil {
		return out, firstErr
	}
	return out, nil
}
