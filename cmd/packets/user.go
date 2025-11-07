package main

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func GetPacketsUID() (int, error) {
	_ = exec.Command("useradd", "-m", "-N", "-r", "-s", "/bin/false", "-d", HomeDir, PacketsUsername).Run()
	cmd := exec.Command("id", "-u", PacketsUsername)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return -1, err
	}

	s := strings.TrimSpace(string(out))
	uid, err := strconv.Atoi(s)
	if err != nil {
		return -1, err
	}
	return uid, nil
}

func ChangeToNoPermission() error {
	uid, err := GetPacketsUID()
	if err != nil {
		return err
	}

	return syscall.Setresuid(0, uid, 0)

}

func ElevatePermission() error { return syscall.Setresuid(0, 0, 0) }
