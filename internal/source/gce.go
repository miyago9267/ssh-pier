package source

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type GCESource struct{}

func (g *GCESource) Name() string { return "GCE" }

func (g *GCESource) Fetch() ([]Target, error) {
	projects, err := gceListProjects()
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	var targets []Target
	for _, p := range projects {
		vms, err := gceListInstances(p)
		if err != nil {
			continue // skip projects we can't access
		}
		targets = append(targets, vms...)
	}
	return targets, nil
}

func (g *GCESource) Connect(t Target) error {
	gcloudPath, err := exec.LookPath("gcloud")
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
	Name   string `json:"name"`
	Zone   string `json:"zone"`
	Status string `json:"status"`
	NetworkInterfaces []struct {
		NetworkIP    string `json:"networkIP"`
		AccessConfigs []struct {
			NatIP string `json:"natIP"`
		} `json:"accessConfigs"`
	} `json:"networkInterfaces"`
	MachineType string `json:"machineType"`
}

func gceListProjects() ([]string, error) {
	out, err := exec.Command("gcloud", "projects", "list", "--format=json(projectId)").Output()
	if err != nil {
		return nil, err
	}
	var projects []gceProject
	if err := json.Unmarshal(out, &projects); err != nil {
		return nil, err
	}
	ids := make([]string, len(projects))
	for i, p := range projects {
		ids[i] = p.ProjectID
	}
	return ids, nil
}

func gceListInstances(project string) ([]Target, error) {
	out, err := exec.Command(
		"gcloud", "compute", "instances", "list",
		"--project", project,
		"--format=json",
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

		// Extract zone short name from full URL
		zone := inst.Zone
		if idx := lastIndex(zone, '/'); idx != -1 {
			zone = zone[idx+1:]
		}

		// Extract machine type short name
		machineType := inst.MachineType
		if idx := lastIndex(machineType, '/'); idx != -1 {
			machineType = machineType[idx+1:]
		}

		// Get IPs
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
