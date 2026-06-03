package main

import (
	"os"
	"strings"
	"testing"
)

func TestDockerAssetsExistAndDocumentRunFlow(t *testing.T) {
	dockerfile, err := os.ReadFile("Dockerfile")
	if err != nil {
		t.Fatalf("expected Dockerfile to exist: %v", err)
	}
	dockerText := string(dockerfile)
	for _, required := range []string{
		"FROM golang:",
		"ENTRYPOINT [\"billing\"]",
	} {
		if !strings.Contains(dockerText, required) {
			t.Fatalf("expected Dockerfile to contain %q", required)
		}
	}

	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("expected README.md to exist: %v", err)
	}
	readmeText := string(readme)
	for _, required := range []string{
		"docker build",
		"docker run",
		"BILLING_DSN",
	} {
		if !strings.Contains(readmeText, required) {
			t.Fatalf("expected README to contain %q", required)
		}
	}
}
