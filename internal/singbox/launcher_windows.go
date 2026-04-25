//go:build windows

package singbox

import "os/exec"

func setProcAttrs(_ *exec.Cmd) {}

func killProc(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
}
