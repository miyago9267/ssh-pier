package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Connect replaces the current process with ssh to the given alias.
// This uses exec (syscall.Exec) so the TUI hands off completely to ssh.
func Connect(alias string) error {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found: %w", err)
	}

	args := []string{"ssh", alias}

	// exec replaces the current process
	return syscall.Exec(sshPath, args, os.Environ())
}
