package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func convertEnvFile(filepath []string) []string {
	var current []string
	for _, file := range filepath {
		buf, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to convert env file: %v", err)
		}
		lines := strings.Split(string(buf), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			current = append(current, line)
		}
	}
	return current
}

func convertEnv(from map[string]string) []string {
	current := make([]string, 0, len(from))
	for key, value := range from {
		if value != "" {
			current = append(current, fmt.Sprintf("%s=%v", key, value))
		}
	}
	return current
}

func validateEnv(envs []string, required map[string]string) error {
	for key := range required {
		found := false
		for _, env := range envs {
			if strings.HasPrefix(env, key) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Required environment variable %s not found", key)
		}
	}
	return nil
}

// StringMap defines a custom flag type that holds a map[string]string
type StringMap struct {
	data map[string]string
}

// NewStringMap creates a new StringMap instance
func NewStringMap() *StringMap {
	return &StringMap{data: make(map[string]string)}
}

// String returns the string representation of the map
func (sm *StringMap) String() string {
	return fmt.Sprintf("%v", sm.data)
}

// Set parses and sets the key-value pairs
func (sm *StringMap) Set(value string) error {
	pair := strings.SplitN(value, "=", 2)
	if len(pair) != 2 {
		return fmt.Errorf("invalid format: %s", value)
	}
	sm.data[pair[0]] = pair[1]
	return nil
}

// Type returns the type of the flag as a string
func (sm *StringMap) Type() string {
	return "stringMap"
}
