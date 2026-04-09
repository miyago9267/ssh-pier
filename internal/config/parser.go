package config

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/miyago9267/ssh-pier/internal/model"
)

type Host = model.Host

const defaultPort = "22"

// Parse reads an SSH config from r and returns all non-wildcard hosts.
func Parse(r io.Reader) ([]Host, error) {
	scanner := bufio.NewScanner(r)
	var hosts []Host
	var current *Host
	currentGroup := "ungrouped"

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Group annotation: # @group: <name>
		if strings.HasPrefix(trimmed, "# @group:") {
			g := strings.TrimSpace(strings.TrimPrefix(trimmed, "# @group:"))
			if g != "" {
				currentGroup = g
			}
			continue
		}

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Host directive
		if strings.HasPrefix(trimmed, "Host ") || strings.HasPrefix(trimmed, "Host\t") {
			// Flush previous host
			if current != nil && current.Alias != "*" {
				if current.Port == "" {
					current.Port = defaultPort
				}
				hosts = append(hosts, *current)
			}
			alias := strings.TrimSpace(strings.TrimPrefix(trimmed, "Host"))
			current = &Host{
				Alias: alias,
				Group: currentGroup,
			}
			continue
		}

		// Key-value inside a Host block
		if current == nil {
			continue
		}
		key, val := parseKeyValue(trimmed)
		switch strings.ToLower(key) {
		case "hostname":
			current.Hostname = val
		case "user":
			current.User = val
		case "port":
			current.Port = val
		case "identityfile":
			current.IdentityFile = val
		}
	}

	// Flush last host
	if current != nil && current.Alias != "*" {
		if current.Port == "" {
			current.Port = defaultPort
		}
		hosts = append(hosts, *current)
	}

	return hosts, scanner.Err()
}

// ParseFile reads and parses ~/.ssh/config (or a given path).
func ParseFile(path string) ([]Host, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func parseKeyValue(line string) (string, string) {
	// SSH config supports both "Key Value" and "Key=Value"
	if idx := strings.IndexByte(line, '='); idx != -1 {
		return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:])
	}
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		parts = strings.SplitN(line, "\t", 2)
	}
	if len(parts) < 2 {
		return line, ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}
