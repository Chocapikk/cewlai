package main

import (
	"testing"

	"github.com/alecthomas/kong"
)

func parseCLI(t *testing.T, args []string) CLI {
	t.Helper()
	var cli CLI
	parser, err := kong.New(&cli, kong.Name("cewlai"), kong.Exit(func(int) {}))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	_, err = parser.Parse(args)
	if err != nil {
		t.Fatalf("failed to parse args %v: %v", args, err)
	}
	return cli
}

func TestSecretsFlag(t *testing.T) {
	cli := parseCLI(t, []string{"-u", "https://example.com", "--secrets"})
	if !cli.Secrets {
		t.Error("expected Secrets to be true")
	}
}

func TestSecretsShortFlag(t *testing.T) {
	cli := parseCLI(t, []string{"-u", "https://example.com", "-s"})
	if !cli.Secrets {
		t.Error("expected Secrets to be true with -s")
	}
}

func TestSecretsFileFlag(t *testing.T) {
	cli := parseCLI(t, []string{"-u", "https://example.com", "--secrets", "--secrets-file", "/tmp/out.txt"})
	if !cli.Secrets {
		t.Error("expected Secrets to be true")
	}
	if cli.SecretsFile != "/tmp/out.txt" {
		t.Errorf("expected SecretsFile '/tmp/out.txt', got %q", cli.SecretsFile)
	}
}

func TestSecretsDefaultOff(t *testing.T) {
	cli := parseCLI(t, []string{"-u", "https://example.com"})
	if cli.Secrets {
		t.Error("expected Secrets to be false by default")
	}
	if cli.SecretsFile != "" {
		t.Errorf("expected SecretsFile empty, got %q", cli.SecretsFile)
	}
}

func TestDefaultValues(t *testing.T) {
	cli := parseCLI(t, []string{"-u", "https://example.com"})
	if cli.Depth != 2 {
		t.Errorf("expected Depth 2, got %d", cli.Depth)
	}
	if cli.Threads != 2 {
		t.Errorf("expected Threads 2, got %d", cli.Threads)
	}
	if cli.AIWords != 200 {
		t.Errorf("expected AIWords 200, got %d", cli.AIWords)
	}
	if cli.Mode != "default" {
		t.Errorf("expected Mode 'default', got %q", cli.Mode)
	}
}

func TestEmailFlags(t *testing.T) {
	cli := parseCLI(t, []string{"-u", "https://example.com", "-e", "--email-file", "/tmp/emails.txt"})
	if !cli.Email {
		t.Error("expected Email true")
	}
	if cli.EmailFile != "/tmp/emails.txt" {
		t.Errorf("expected EmailFile '/tmp/emails.txt', got %q", cli.EmailFile)
	}
}

func TestMetaFlags(t *testing.T) {
	cli := parseCLI(t, []string{"-u", "https://example.com", "-a", "--meta-file", "/tmp/meta.txt"})
	if !cli.Meta {
		t.Error("expected Meta true")
	}
	if cli.MetaFile != "/tmp/meta.txt" {
		t.Errorf("expected MetaFile '/tmp/meta.txt', got %q", cli.MetaFile)
	}
}

func TestCountFlag(t *testing.T) {
	cli := parseCLI(t, []string{"-u", "https://example.com", "--count"})
	if !cli.Count {
		t.Error("expected Count true")
	}
}

func TestCompletionSubcommand(t *testing.T) {
	var cli CLI
	parser, err := kong.New(&cli, kong.Name("cewlai"), kong.Exit(func(int) {}))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}
	kongCtx, err := parser.Parse([]string{"completion", "bash"})
	if err != nil {
		t.Fatalf("failed to parse completion command: %v", err)
	}
	if kongCtx.Command() != "completion <shell>" {
		t.Errorf("expected command 'completion <shell>', got %q", kongCtx.Command())
	}
}
