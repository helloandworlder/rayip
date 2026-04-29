package runtime

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type XrayProcessConfig struct {
	BinaryPath string
	ConfigPath string
	GRPCAddr   string
}

type XrayProcess struct {
	cmd *exec.Cmd
}

func StartXrayProcess(ctx context.Context, cfg XrayProcessConfig) (*XrayProcess, error) {
	if cfg.BinaryPath == "" {
		return nil, errors.New("xray binary path is required")
	}
	if cfg.ConfigPath == "" {
		return nil, errors.New("xray config path is required")
	}
	if err := os.MkdirAll(filepath.Dir(cfg.ConfigPath), 0o755); err != nil {
		return nil, err
	}
	payload, err := BuildXrayRuntimeConfig(cfg.GRPCAddr)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(cfg.ConfigPath, payload, 0o600); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, cfg.BinaryPath, "run", "-config", cfg.ConfigPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &XrayProcess{cmd: cmd}, nil
}

func (p *XrayProcess) Stop(ctx context.Context) error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()
	_ = p.cmd.Process.Signal(os.Interrupt)
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, os.ErrProcessDone) {
			return err
		}
		return nil
	case <-ctx.Done():
		_ = p.cmd.Process.Kill()
		return ctx.Err()
	case <-time.After(5 * time.Second):
		_ = p.cmd.Process.Kill()
		return nil
	}
}
