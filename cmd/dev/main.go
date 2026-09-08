package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alumasinde/tuma254-api/internal/platform/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "tuma254 dev engine:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := requireCommand("docker"); err != nil {
		return err
	}

	_, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	if err := runCommand(context.Background(), "docker", "compose", "up", "-d"); err != nil {
		return fmt.Errorf("start postgres: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := waitForPostgres(ctx); err != nil {
		return err
	}

	if err := runCommand(context.Background(), "go", "run", "./cmd/migrate", "up"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	sinkToken := strings.TrimSpace(os.Getenv("SMS_WEBHOOK_TOKEN"))
	if sinkToken == "" {
		var tokenBytes [32]byte
		if _, err := rand.Read(tokenBytes[:]); err != nil {
			return fmt.Errorf("generate local SMS sink token: %w", err)
		}
		sinkToken = hex.EncodeToString(tokenBytes[:])
	}

	if err := writeBrunoEnvironment(sinkToken); err != nil {
		return err
	}

	apiEnv := append(os.Environ(),
		"SMS_PROVIDER=webhook",
		"SMS_WEBHOOK_URL=http://127.0.0.1:8090/messages",
		"SMS_WEBHOOK_TOKEN="+sinkToken,
	)
	env := append(os.Environ(), "SMS_SINK_TOKEN="+sinkToken)
	sink := exec.Command("go", "run", "./cmd/sms-sink")
	sink.Env = env
	sink.Stdout = os.Stdout
	sink.Stderr = os.Stderr
	if err := sink.Start(); err != nil {
		return fmt.Errorf("start sms sink: %w", err)
	}

	api := exec.Command("go", "run", "./cmd/api")
	api.Env = apiEnv
	api.Stdout = os.Stdout
	api.Stderr = os.Stderr
	if err := api.Start(); err != nil {
		_ = sink.Process.Kill()
		return fmt.Errorf("start api: %w", err)
	}

	fmt.Println()
	fmt.Println("Tuma254 development engine is running")
	fmt.Println("API:        http://localhost:8080")
	fmt.Println("Health:     http://localhost:8080/health")
	fmt.Println("SMS sink:   http://127.0.0.1:8090")
	fmt.Println("Bruno env:  api-tests/environments/local.bru (generated automatically)")
	fmt.Println("Database:   managed by Docker Compose")
	fmt.Println("Press Ctrl+C to stop the API and SMS sink.")
	fmt.Println()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	done := make(chan error, 2)
	go func() { done <- api.Wait() }()
	go func() { done <- sink.Wait() }()

	select {
	case <-stop:
		_ = api.Process.Signal(os.Interrupt)
		_ = sink.Process.Signal(os.Interrupt)
		time.Sleep(300 * time.Millisecond)
		_ = api.Process.Kill()
		_ = sink.Process.Kill()
		return nil
	case err := <-done:
		_ = api.Process.Kill()
		_ = sink.Process.Kill()
		if err != nil {
			return err
		}
		return nil
	}
}

func requireCommand(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s is required but was not found in PATH", name)
	}
	return nil
}

func waitForPostgres(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		cmd := exec.CommandContext(ctx, "docker", "compose", "exec", "-T", "postgres", "pg_isready", "-U", os.Getenv("POSTGRES_USER"), "-d", os.Getenv("POSTGRES_DB"))
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("postgres did not become ready before timeout")
		case <-ticker.C:
		}
	}
}

func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}


func writeBrunoEnvironment(sinkToken string) error {
	timestamp := time.Now().Format("20060102150405")
	content := fmt.Sprintf(`vars {
  baseUrl: http://localhost:8080
  smsSinkUrl: http://127.0.0.1:8090
  smsSinkToken: %s
  testEmail: tuma254-smoke-%s@example.test
  testPhone: +254700000000
  testFirstName: Test
  testLastName: User
  testPassword: ChangeThisLocalTestPassword123
}
`, sinkToken, timestamp)
	if err := os.MkdirAll("api-tests/environments", 0o755); err != nil {
		return fmt.Errorf("create Bruno environment directory: %w", err)
	}
	if err := os.WriteFile("api-tests/environments/local.bru", []byte(content), 0o600); err != nil {
		return fmt.Errorf("write Bruno local environment: %w", err)
	}
	return nil
}
