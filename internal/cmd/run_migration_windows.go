//go:build windows

package cmd

import "os/exec"

// setMigrationProcAttr is a no-op on Windows — process group kill
// is not supported via the same mechanism.
func setMigrationProcAttr(c *exec.Cmd) {
	// No-op on Windows
}
