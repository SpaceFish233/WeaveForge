//go:build !windows

package vectordb

import "os/exec"

func hideConsoleWindow(cmd *exec.Cmd) {}
