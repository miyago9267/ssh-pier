package config

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// WriteFile writes hosts to the given path, creating a .bak backup if the file exists.
func WriteFile(path string, hosts []Host) error {
	// Backup existing file
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read for backup: %w", err)
		}
		if err := os.WriteFile(path+".bak", data, 0644); err != nil {
			return fmt.Errorf("write backup: %w", err)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return Write(f, hosts)
}

// Write writes hosts in SSH config format to w, grouped by their Group field.
func Write(w io.Writer, hosts []Host) error {
	// Collect groups in order of first appearance
	var groupOrder []string
	grouped := make(map[string][]Host)
	for _, h := range hosts {
		g := h.Group
		if g == "" {
			g = "ungrouped"
		}
		if _, exists := grouped[g]; !exists {
			groupOrder = append(groupOrder, g)
		}
		grouped[g] = append(grouped[g], h)
	}

	first := true
	for _, g := range groupOrder {
		if !first {
			fmt.Fprintln(w)
		}
		first = false

		fmt.Fprintf(w, "# @group: %s\n", g)
		for _, h := range grouped[g] {
			fmt.Fprintf(w, "Host %s\n", h.Alias)
			if h.Hostname != "" {
				fmt.Fprintf(w, "    Hostname %s\n", h.Hostname)
			}
			if h.User != "" {
				fmt.Fprintf(w, "    User %s\n", h.User)
			}
			if h.Port != "" && h.Port != "22" {
				fmt.Fprintf(w, "    Port %s\n", h.Port)
			}
			if h.IdentityFile != "" {
				fmt.Fprintf(w, "    IdentityFile %s\n", h.IdentityFile)
			}
			fmt.Fprintln(w)
		}
	}

	return nil
}

// UpdateHost replaces or appends a host in the list.
func UpdateHost(hosts []Host, h Host) []Host {
	for i, existing := range hosts {
		if existing.Alias == h.Alias {
			hosts[i] = h
			return hosts
		}
	}
	return append(hosts, h)
}

// DeleteHost removes a host by alias.
func DeleteHost(hosts []Host, alias string) []Host {
	result := make([]Host, 0, len(hosts))
	for _, h := range hosts {
		if h.Alias != alias {
			result = append(result, h)
		}
	}
	return result
}

// GroupHosts returns hosts organized by group, preserving order.
func GroupHosts(hosts []Host) []struct {
	Name  string
	Hosts []Host
} {
	var order []string
	grouped := make(map[string][]Host)
	for _, h := range hosts {
		g := h.Group
		if g == "" {
			g = "ungrouped"
		}
		if _, exists := grouped[g]; !exists {
			order = append(order, g)
		}
		grouped[g] = append(grouped[g], h)
	}

	result := make([]struct {
		Name  string
		Hosts []Host
	}, len(order))
	for i, g := range order {
		result[i] = struct {
			Name  string
			Hosts []Host
		}{Name: g, Hosts: grouped[g]}
	}
	return result
}

// FilterHosts returns hosts matching the query (fuzzy match on alias, hostname, user).
func FilterHosts(hosts []Host, query string) []Host {
	if query == "" {
		return hosts
	}
	q := strings.ToLower(query)
	var result []Host
	for _, h := range hosts {
		if strings.Contains(strings.ToLower(h.Alias), q) ||
			strings.Contains(strings.ToLower(h.Hostname), q) ||
			strings.Contains(strings.ToLower(h.User), q) {
			result = append(result, h)
		}
	}
	return result
}
