package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Header struct {
	Builder string            `yaml:"builder"`
	Env     map[string]string `yaml:"env"`
	EnvFile []string          `yaml:"envFile"`
}

type Configuration struct {
	Header Header         `yaml:"header"`
	Jobs   map[string]Job `yaml:"jobs"`
}

type TemplateData struct {
	Args map[string]string
}

func loadRecipe(filepath string, args *StringMap) (Configuration, error) {
	var config Configuration

	buf, err := os.ReadFile(filepath)
	if err != nil {
		return config, err
	}

	idx := bytes.Index(buf, []byte("\n\n\n"))
	headerBuf := buf[:idx]
	buf = buf[idx+3:]

	decoder := yaml.NewDecoder(bytes.NewReader(headerBuf))
	decoder.KnownFields(true)

	if err := decoder.Decode(&config.Header); err != nil {
		return config, fmt.Errorf("failed to decode header: %w", err)
	}

	template, err := template.New("recipe").Parse(string(buf))
	if err != nil {
		return config, err
	}

	r, w, err := os.Pipe()
	if err != nil {
		return config, err
	}

	go func() {
		if err := template.Execute(w, TemplateData{
			Args: args.data,
		}); err != nil {
			log.Fatalf("failed to execute template: %v", err)
		}
		w.Close()
	}()

	// Load configuration from file
	if err := yaml.NewDecoder(r).Decode(&config.Jobs); err != nil {
		return config, err
	}
	return config, nil
}

func main() {
	args := NewStringMap()
	flag.Var(args, "env", "Environment variables set as -env key=value")
	recipe := flag.String("f", ".jdi", "Recipe filepath")
	flag.Parse()
	recipes := flag.Args()

	config, err := loadRecipe(*recipe, args)
	if err != nil {
		log.Fatalf("failed to load recipe: %v", err)
	}

	if config.Header.Builder != "" {
		if err := buildDockerImage(config.Header.Builder, "justdoit"); err != nil {
			log.Fatalf("failed to build docker image: %v", err)
		}
	}

	envs := convertEnvFile(config.Header.EnvFile)
	envs = append(envs, convertEnv(config.Header.Env)...)

	depends := getDepends(config, recipes)

	runStmts, deferStmts := []string{}, []string{}

	for _, recipe := range appendUnique(depends, recipes...) {
		job, ok := config.Jobs[recipe]
		if !ok {
			log.Fatalf("recipe %s not found", recipe)
		}

		runStmt, deferStmt, err := job.Prepare(envs, args)
		if err != nil {
			log.Fatalf("failed to execute job: %v", err)
		}

		runStmts = append(runStmts, runStmt)
		deferStmts = append(deferStmts, deferStmt)
	}

	err = runInteractiveContainer(context.Background(), "justdoit", func(run func(string) error) error {
		for i := range runStmts {
			err := run(runStmts[i])
			run(deferStmts[i])
			if err != nil {
				return fmt.Errorf("failed to run command: %v", err)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("failed to run interactive container: %v", err)
	}
}

func getDepends(config Configuration, recipes []string) []string {
	var depends []string
	for _, recipe := range recipes {
		job := config.Jobs[recipe]
		for _, dep := range job.Depends {
			depends = appendUnique(depends, getDepends(config, []string{dep})...)
			depends = appendUnique(depends, dep)
		}
	}
	return depends
}

func appendUnique(slice []string, values ...string) []string {
outer:
	for _, value := range values {
		for _, v := range slice {
			if v == value {
				continue outer
			}
		}
		slice = append(slice, value)
	}
	return slice
}
