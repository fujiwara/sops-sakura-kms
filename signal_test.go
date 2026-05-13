package ssk_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	ssk "github.com/fujiwara/sops-sakura-kms"
)

// TestMain lets this test binary double as a wrapper helper. When
// SSK_TEST_HELPER=1 is set, the binary runs RunWrapper instead of the
// test suite so the signal tests can drive it as a real subprocess.
// SSK_TEST_TTY=1 makes RunWrapper see stdin as a terminal regardless
// of how the helper was launched.
func TestMain(m *testing.M) {
	if os.Getenv("SSK_TEST_HELPER") == "1" {
		ttyForced := os.Getenv("SSK_TEST_TTY") == "1"
		ssk.IsStdinTerminal = func() bool { return ttyForced }
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		args := strings.Split(os.Getenv("SSK_TEST_ARGS"), "\x1f")
		exitCode, _ := ssk.RunWrapper(ctx, args)
		os.Exit(exitCode)
	}
	os.Exit(m.Run())
}

func startWrapperHelper(t *testing.T, addr, args string, tty bool) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	env := append(os.Environ(),
		"SSK_TEST_HELPER=1",
		"SSK_TEST_ARGS="+args,
		"SAKURACLOUD_KMS_KEY_ID=test-key-id",
		"SSK_COMMAND=sh",
		"SSK_SERVER_ADDR="+addr,
	)
	if tty {
		env = append(env, "SSK_TEST_TTY=1")
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper: %v", err)
	}
	return cmd
}

func waitWithTimeout(t *testing.T, cmd *exec.Cmd, d time.Duration) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(d):
		_ = cmd.Process.Kill()
		<-done
		t.Fatalf("wrapper did not exit within %s", d)
		return nil
	}
}

// TestRunWrapperExecEnvLetsChildExitOnItsOwn verifies that for the
// `exec-env` subcommand running in a tty, the wrapper does not kill
// the child when the wrapper itself receives SIGINT. Interactive
// children (e.g. mysql cli) need to decide for themselves whether and
// when to exit on Ctrl-C, and may legitimately accept many of them.
// We send SIGINT to the wrapper a few times while the child runs and
// then verify the wrapper propagates the child's own exit code.
func TestRunWrapperExecEnvLetsChildExitOnItsOwn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal tests are unix-only")
	}
	// Args = ["-c", "echo READY; sleep 2; exit 42", "exec-env"]. The
	// trailing "exec-env" is what makes RunWrapper take the
	// no-context path; sh treats it as $0 and ignores it. tty=true
	// satisfies the second gating condition.
	cmd := startWrapperHelper(t, "127.0.0.1:18301",
		"-c\x1fecho READY; sleep 2; exit 42\x1fexec-env", true)

	time.Sleep(1500 * time.Millisecond)
	for range 3 {
		if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
			t.Fatalf("send SIGINT: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	err := waitWithTimeout(t, cmd, 10*time.Second)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError from helper, got %v", err)
	}
	if got := exitErr.ExitCode(); got != 42 {
		t.Errorf("exit code = %d, want 42 (child exited on its own)", got)
	}
}

// TestRunWrapperExecEnvNonTTYForwardsSIGTERM verifies that for
// `exec-env` running outside a tty (typical orchestrator setup —
// systemd, k8s, CI), the wrapper forwards SIGTERM to the child via
// exec.Cmd.Cancel rather than the default SIGKILL. This gives the
// child a chance to shut down cleanly.
func TestRunWrapperExecEnvNonTTYForwardsSIGTERM(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal tests are unix-only")
	}
	// Child traps SIGTERM, kills its background sleep, and exits 17.
	// Without forwarding the child would be SIGKILLed and exit
	// non-zero with a signal-status exit code, not 17.
	cmd := startWrapperHelper(t, "127.0.0.1:18303",
		"-c\x1ftrap 'kill $! 2>/dev/null; exit 17' TERM; echo READY; sleep 30 & wait\x1fexec-env",
		false)

	time.Sleep(1500 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}

	err := waitWithTimeout(t, cmd, 5*time.Second)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError from helper, got %v", err)
	}
	if got := exitErr.ExitCode(); got != 17 {
		t.Errorf("exit code = %d, want 17 (child SIGTERM trap fired)", got)
	}
}

// TestRunWrapperNonExecEnvForwardsSIGTERM verifies that for non
// `exec-env` subcommands (e.g. `sops -i` which spawns an editor),
// SIGTERM is forwarded to the child via exec.Cmd.Cancel so the child
// has a chance to clean up before exiting.
func TestRunWrapperNonExecEnvForwardsSIGTERM(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal tests are unix-only")
	}
	// Child traps SIGTERM, kills its background sleep, and exits 17.
	// If the wrapper SIGKILLed the child instead, the trap could not
	// fire and we'd see a signal-status exit code.
	cmd := startWrapperHelper(t, "127.0.0.1:18302",
		"-c\x1ftrap 'kill $! 2>/dev/null; exit 17' TERM; echo READY; sleep 30 & wait",
		false)

	time.Sleep(1500 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}

	err := waitWithTimeout(t, cmd, 5*time.Second)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError from helper, got %v", err)
	}
	if got := exitErr.ExitCode(); got != 17 {
		t.Errorf("exit code = %d, want 17 (child SIGTERM trap fired)", got)
	}
}
