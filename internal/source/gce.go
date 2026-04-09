package source

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const (
	gceMaxConcurrency = 3
	gceCommandTimeout = 15 * time.Second
)

type GCESource struct {
	Projects []string // if set, skip auto-discovery
}

func (g *GCESource) Name() string { return "GCE" }

func (g *GCESource) Fetch() ([]Target, error) {
	projects := g.Projects
	if len(projects) == 0 {
		var err error
		projects, err = gceListProjects()
		if err != nil {
			return nil, fmt.Errorf("list projects: %w", err)
		}
	}

	type result struct {
		targets []Target
	}
	results := make([]result, len(projects))
	var wg sync.WaitGroup
	sem := make(chan struct{}, gceMaxConcurrency)

	for i, p := range projects {
		wg.Add(1)
		go func(idx int, project string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			vms, err := gceListInstances(project)
			if err == nil {
				results[idx] = result{targets: vms}
			}
		}(i, p)
	}
	wg.Wait()

	var targets []Target
	for _, r := range results {
		targets = append(targets, r.targets...)
	}
	return targets, nil
}

func (g *GCESource) Connect(t Target) error {
	gcloudPath, err := findCLI("gcloud")
	if err != nil {
		return fmt.Errorf("gcloud not found: %w", err)
	}

	args := []string{
		"gcloud", "compute", "ssh", t.Alias,
		"--zone", t.Meta["zone"],
		"--project", t.Meta["project"],
	}

	return syscall.Exec(gcloudPath, args, os.Environ())
}

type gceProject struct {
	ProjectID string `json:"projectId"`
}

type gceInstance struct {
	Name              string `json:"name"`
	Zone              string `json:"zone"`
	Status            string `json:"status"`
	NetworkInterfaces []struct {
		NetworkIP     string `json:"networkIP"`
		AccessConfigs []struct {
			NatIP string `json:"natIP"`
		} `json:"accessConfigs"`
	} `json:"networkInterfaces"`
	MachineType string `json:"machineType"`
}

var systemProjectPrefixes = []string{
	"sys-",
	"gen-lang-client-",
}

func isSystemProject(id string) bool {
	for _, prefix := range systemProjectPrefixes {
		if len(id) >= len(prefix) && id[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func gceListProjects() ([]string, error) {
	gcloudPath, err := findCLI("gcloud")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), gceCommandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, gcloudPath, "projects", "list", "--format=json(projectId)", "--quiet").Output()
	if err != nil {
		return nil, err
	}
	var projects []gceProject
	if err := json.Unmarshal(out, &projects); err != nil {
		return nil, err
	}
	var ids []string
	for _, p := range projects {
		if !isSystemProject(p.ProjectID) {
			ids = append(ids, p.ProjectID)
		}
	}
	return ids, nil
}

func gceListInstances(project string) ([]Target, error) {
	gcloudPath, err := findCLI("gcloud")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), gceCommandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, gcloudPath, "compute", "instances", "list",
		"--project", project,
		"--format=json",
		"--quiet",
	).Output()
	if err != nil {
		return nil, err
	}

	var instances []gceInstance
	if err := json.Unmarshal(out, &instances); err != nil {
		return nil, err
	}

	var targets []Target
	for _, inst := range instances {
		if inst.Status != "RUNNING" {
			continue
		}

		zone := inst.Zone
		if idx := lastIndex(zone, '/'); idx != -1 {
			zone = zone[idx+1:]
		}

		machineType := inst.MachineType
		if idx := lastIndex(machineType, '/'); idx != -1 {
			machineType = machineType[idx+1:]
		}

		var internalIP, externalIP string
		if len(inst.NetworkInterfaces) > 0 {
			internalIP = inst.NetworkInterfaces[0].NetworkIP
			if len(inst.NetworkInterfaces[0].AccessConfigs) > 0 {
				externalIP = inst.NetworkInterfaces[0].AccessConfigs[0].NatIP
			}
		}

		ip := externalIP
		if ip == "" {
			ip = internalIP
		}

		detail := fmt.Sprintf("%s  %s  %s", zone, machineType, ip)

		targets = append(targets, Target{
			Source: "gce",
			Alias:  inst.Name,
			Group:  project,
			Detail: detail,
			Meta: map[string]string{
				"zone":        zone,
				"project":     project,
				"machineType": machineType,
				"internalIP":  internalIP,
				"externalIP":  externalIP,
			},
		})
	}
	return targets, nil
}

func lastIndex(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}
