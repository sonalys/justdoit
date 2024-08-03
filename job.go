package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Job struct {
	Env     map[string]string `yaml:"env"`
	EnvFile []string          `yaml:"envFile"`
	Run     string            `yaml:"run"`
	Defer   string            `yaml:"defer"`
	Depends []string          `yaml:"depends"`
}

func (j Job) Prepare(envs []string, args *StringMap) (runStmt, deferStmt string, err error) {
	envs = append(envs, convertEnvFile(j.EnvFile)...)
	envs = append(envs, convertEnv(j.Env)...)
	envs = append(envs, convertEnv(args.data)...)
	if err = validateEnv(envs, j.Env); err != nil {
		return
	}
	if j.Run != "" {
		runStmt = prepareWithEnv(j.Run, envs)
	}
	if j.Defer != "" {
		deferStmt = prepareWithEnv(j.Defer, envs)
	}
	return
}

func prepareWithEnv(cmd string, envs []string) string {
	envCmd := strings.Join(envs, " ")
	return fmt.Sprintf("export %s\n%s", envCmd, cmd)
}

func runCmd(cmd string) error {
	c := exec.Command("bash", "-c", cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to run command: %v", err)
	}
	return nil
}
