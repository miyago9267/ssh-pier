package source

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type GKESource struct {
	Shell string // default "/bin/sh"
}

func (g *GKESource) Name() string { return "GKE" }

func (g *GKESource) Fetch() ([]Target, error) {
	kubectlPath, err := findCLI("kubectl")
	if err != nil {
		return nil, fmt.Errorf("kubectl not found: %w", err)
	}
	out, err := exec.Command(kubectlPath, "get", "pods", "-A", "-o", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl get pods: %w", err)
	}

	var podList k8sPodList
	if err := json.Unmarshal(out, &podList); err != nil {
		return nil, err
	}

	var targets []Target
	for _, pod := range podList.Items {
		if pod.Status.Phase != "Running" {
			continue
		}

		var containers []string
		for _, c := range pod.Spec.Containers {
			containers = append(containers, c.Name)
		}

		containerStr := ""
		if len(containers) > 0 {
			containerStr = containers[0]
		}

		detail := fmt.Sprintf("%s  %s", pod.Metadata.Namespace, containerStr)

		targets = append(targets, Target{
			Source: "gke",
			Alias:  pod.Metadata.Name,
			Group:  pod.Metadata.Namespace,
			Detail: detail,
			Meta: map[string]string{
				"namespace":  pod.Metadata.Namespace,
				"node":       pod.Spec.NodeName,
				"container":  containerStr,
				"containers": joinStrings(containers, ","),
			},
		})
	}
	return targets, nil
}

func (g *GKESource) Connect(t Target) error {
	kubectlPath, err := findCLI("kubectl")
	if err != nil {
		return fmt.Errorf("kubectl not found: %w", err)
	}

	shell := g.Shell
	if shell == "" {
		shell = "/bin/sh"
	}

	args := []string{
		"kubectl", "exec", "-it",
		t.Alias,
		"-n", t.Meta["namespace"],
	}

	// If multiple containers, specify the first one
	if t.Meta["container"] != "" {
		args = append(args, "-c", t.Meta["container"])
	}

	args = append(args, "--", shell)

	return syscall.Exec(kubectlPath, args, os.Environ())
}

// SetShell updates the shell for GKE connections.
func (g *GKESource) SetShell(shell string) {
	g.Shell = shell
}

type k8sPodList struct {
	Items []k8sPod `json:"items"`
}

type k8sPod struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		NodeName   string `json:"nodeName"`
		Containers []struct {
			Name string `json:"name"`
		} `json:"containers"`
	} `json:"spec"`
	Status struct {
		Phase string `json:"phase"`
	} `json:"status"`
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}
