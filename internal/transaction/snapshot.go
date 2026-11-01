package transaction

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Snapshot struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Packages  []string  `json:"packages"`
	BackupDir string    `json:"backup_dir"`
}

func CreateSnapshot(packages []string) (*Snapshot, error) {
	id := fmt.Sprintf("tx_%d", time.Now().UnixNano())
	backupDir := filepath.Join("/tmp/lucinexus/snapshots", id)
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create snapshot dir: %w", err)
	}

	s := &Snapshot{
		ID:        id,
		CreatedAt: time.Now(),
		Packages:  packages,
		BackupDir: backupDir,
	}

	for _, pkg := range packages {
		src := filepath.Join("/etc/config", pkg)
		dst := filepath.Join(backupDir, pkg)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		_ = copyFile(src, dst)
	}

	return s, nil
}

func (s *Snapshot) Rollback(ctx context.Context) error {
	for _, pkg := range s.Packages {
		backup := filepath.Join(s.BackupDir, pkg)
		target := filepath.Join("/etc/config", pkg)
		if _, err := os.Stat(backup); err == nil {
			_ = copyFile(backup, target)
			_ = exec.CommandContext(ctx, "uci", "commit", pkg).Run()
		} else {
			_ = exec.CommandContext(ctx, "uci", "revert", pkg).Run()
		}
	}
	_ = exec.CommandContext(ctx, "/etc/init.d/network", "reload").Run()
	_ = exec.CommandContext(ctx, "/etc/init.d/firewall", "reload").Run()
	return nil
}

func (s *Snapshot) HealthCheck(ctx context.Context) error {
	for _, pkg := range s.Packages {
		cmd := exec.CommandContext(ctx, "uci", "show", pkg)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("corrupted UCI package syntax detected for %s: %w", pkg, err)
		}
	}
	return nil
}

func (s *Snapshot) Discard() error {
	return os.RemoveAll(s.BackupDir)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
