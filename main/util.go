package main
import "os/exec"
func RunCmd(cmd string, args ...string) (string, error) {
	cmdPath, err := exec.LookPath(cmd)
	if err != nil {
		return "", err
	}

	out, err := exec.Command(cmdPath, args...).CombinedOutput()
	if err != nil {
		return string(out), err
	}

	return string(out), nil
}