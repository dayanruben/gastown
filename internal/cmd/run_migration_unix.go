//go:build !windows

package cmd

import (
	"os/exec"
	"syscall"
)

// setMigrationProcAttr puts the process in its own process group so we can
// kill all children on timeout, not just the bash process.
func setMigrationProcAttr(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		return syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
	}
}
