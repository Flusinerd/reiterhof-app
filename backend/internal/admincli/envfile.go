package admincli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// DefaultEnvFile is the environment file of the API service on the server.
const DefaultEnvFile = "/etc/reiterhof/api.env"

// ParseEnvFile parses the systemd EnvironmentFile syntax:
//
//   - one KEY=value per line, leading and trailing whitespace is ignored;
//   - empty lines and lines starting with '#' or ';' are comments;
//   - a value may be wrapped in double quotes (the inner whitespace is kept, a backslash
//     escapes the next character) or in single quotes (taken literally);
//   - in an unquoted value a backslash escapes the next character too, and a backslash at
//     the end of a line continues the value on the next line;
//   - there is no variable expansion and no "export" prefix (as in systemd).
//
// The later assignment of a key wins.
func ParseEnvFile(content string) (map[string]string, error) {
	out := map[string]string{}
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		start := i + 1
		line := strings.TrimSpace(lines[i])
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		// A trailing backslash continues the assignment on the next line.
		for strings.HasSuffix(line, `\`) && !strings.HasSuffix(line, `\\`) && i+1 < len(lines) {
			i++
			line = line[:len(line)-1] + strings.TrimLeft(lines[i], " \t")
		}
		key, raw, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || !validEnvKey(key) {
			return nil, fmt.Errorf("line %d: expected KEY=value", start)
		}
		val, err := parseEnvValue(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("line %d (%s): %w", start, key, err)
		}
		out[key] = val
	}
	return out, nil
}

func validEnvKey(k string) bool {
	if k == "" {
		return false
	}
	for i, r := range k {
		switch {
		case r == '_', r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}

func parseEnvValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	switch raw[0] {
	case '\'':
		end := strings.IndexByte(raw[1:], '\'')
		if end < 0 {
			return "", errors.New("unterminated single quote")
		}
		if rest := strings.TrimSpace(raw[end+2:]); rest != "" {
			return "", errors.New("unexpected text after closing quote")
		}
		return raw[1 : end+1], nil
	case '"':
		var b strings.Builder
		for i := 1; i < len(raw); i++ {
			switch c := raw[i]; c {
			case '\\':
				if i+1 >= len(raw) {
					return "", errors.New("unterminated double quote")
				}
				i++
				b.WriteByte(raw[i])
			case '"':
				if rest := strings.TrimSpace(raw[i+1:]); rest != "" {
					return "", errors.New("unexpected text after closing quote")
				}
				return b.String(), nil
			default:
				b.WriteByte(c)
			}
		}
		return "", errors.New("unterminated double quote")
	}
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\\' && i+1 < len(raw) {
			i++
		}
		b.WriteByte(raw[i])
	}
	return b.String(), nil
}

// LoadEnvFile reads the file and sets every variable that is not set in the environment
// yet (the environment wins, like a value passed on the command line). A missing file is
// an error unless optional is true; the returned slice lists the variables that were set.
func LoadEnvFile(path string, optional bool) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if optional && errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		if errors.Is(err, fs.ErrPermission) {
			return nil, fmt.Errorf("cannot read %s: permission denied (run it as user reiterhof: sudo stallfunk-admin ...)", path)
		}
		return nil, fmt.Errorf("read env file: %w", err)
	}
	vars, err := ParseEnvFile(string(raw))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var set []string
	for k, v := range vars {
		if _, exists := os.LookupEnv(k); exists {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return nil, fmt.Errorf("set %s: %w", k, err)
		}
		set = append(set, k)
	}
	return set, nil
}
