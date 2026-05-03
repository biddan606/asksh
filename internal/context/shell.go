package shellctx

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type ShellContext struct {
	CWD   string
	OS    string
	Shell string
	Lang  string
}

func Collect() (ShellContext, error) {
	return CollectWith(swVersVersion)
}

func CollectWith(osVersionFn func() (string, error)) (ShellContext, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return ShellContext{}, err
	}

	osVersion, err := osVersionFn()
	if err != nil {
		osVersion = runtime.GOOS
	}

	shell := ""
	if shellEnv := os.Getenv("SHELL"); shellEnv != "" {
		shell = filepath.Base(shellEnv)
	}

	return ShellContext{
		CWD:   cwd,
		OS:    osVersion,
		Shell: shell,
	}, nil
}

func swVersVersion() (string, error) {
	if runtime.GOOS != "darwin" {
		return runtime.GOOS, nil
	}
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return runtime.GOOS, err
	}
	return strings.TrimSpace(string(out)), nil
}
