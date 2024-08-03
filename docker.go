package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

type handler func(run func(string) error) error

func runInteractiveContainer(ctx context.Context, imageName string, run handler) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %v", err)
	}

	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %v", err)
	}

	// Create container configuration
	containerConfig := &container.Config{
		Image:      imageName,
		WorkingDir: cwd,
		OpenStdin:  true,
	}

	// Create host configuration
	hostConfig := &container.HostConfig{
		Privileged: true,
		AutoRemove: true,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: cwd,
				Target: cwd,
			},
		},
	}

	// Create the container
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	// Attach to the container
	hijackedResp, err := cli.ContainerAttach(context.Background(), resp.ID, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Logs:   false,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to container: %v", err)
	}
	defer hijackedResp.Close()
	// go io.Copy(os.Stdout, hijackedResp.Reader)

	// Start the container
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start Docker container: %v", err)
	}

	err = run(func(cmd string) error {
		if cmd == "" {
			return nil
		}
		if _, err := hijackedResp.Conn.Write([]byte(cmd + "\n")); err != nil {
			return fmt.Errorf("failed to write command to container: %v", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		Follow:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to get container logs: %v", err)
	}
	defer out.Close()
	go io.Copy(os.Stdout, out)

	hijackedResp.Conn.Write([]byte("exit\n"))

	timeout := 2
	cli.ContainerStop(ctx, resp.ID, container.StopOptions{
		Timeout: &timeout,
	})

	wr, errch := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case <-wr:
	case <-errch:
	case <-ctx.Done():
	}
	return nil
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
