package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

type handler func(run func(string) error) error

func runInteractiveContainer(ctx context.Context, imageName string, run handler) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %v", err)
	}

	args := []string{
		"run",
		"-i",
		"--rm",
		"--privileged",
		"-v", fmt.Sprintf("%s:%s", cwd, cwd),
		"-w", cwd,
		"-u", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		imageName,
	}

	// Run the container
	cmd := exec.CommandContext(ctx, "docker", args...)

	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %v", err)
	}
	defer w.Close()

	cmd.Stdin = r
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		run(func(cmd string) error {
			_, err := fmt.Fprintln(w, cmd)
			return err
		})
		w.WriteString("exit\n")
	}()

	return cmd.Run()
}

func buildDockerImage(dockerfilePath, imageName string) error {
	// Check if Docker is installed
	_, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("docker is not installed")
	}

	output := bytes.NewBuffer(make([]byte, 0, 1024))
	// Build the image
	cmd := exec.Command("docker", "build", "-t", imageName, "-f", dockerfilePath, ".")
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Run(); err != nil {
		os.Stderr.Write(output.Bytes())
		return fmt.Errorf("failed to build docker image: %v", err)
	}
	return nil
}
