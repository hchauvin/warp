package proc

import (
	"fmt"
	"os/exec"
	"runtime"
)

// KillPort kills the process that is listening on a given port.
func KillPort(port int) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		command := fmt.Sprintf(killPortPsScript, port)
		cmd = exec.Command("powershell", command)
	} else {
		command := fmt.Sprintf("lsof -i tcp:%d | grep LISTEN | awk '{print $2}' | xargs kill -9", port)
		cmd = exec.Command("bash", "-c", command)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kill port: %v: %s", err, out)
	}
	return nil
}

const killPortPsScript = `
$con = (Get-NetTCPConnection -LocalPort %d)
if ($con -ne $null) {
  Stop-Process -Id $($con.OwningProcess)
}
`
