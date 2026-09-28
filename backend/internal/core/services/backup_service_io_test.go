package services

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

type triggerBackupRepoStub struct {
	lastUpdated *domain.BackupExecution
}

func (r *triggerBackupRepoStub) GetConfig(context.Context, string) (*domain.BackupConfig, error) {
	return nil, errors.New("unexpected config lookup")
}
func (*triggerBackupRepoStub) UpsertConfig(context.Context, *domain.BackupConfig) error { return nil }
func (*triggerBackupRepoStub) CreateExecution(context.Context, *domain.BackupExecution) error {
	return nil
}
func (r *triggerBackupRepoStub) UpdateExecution(_ context.Context, execution *domain.BackupExecution) error {
	r.lastUpdated = execution
	return nil
}
func (*triggerBackupRepoStub) UpdateExecutionVerifyColumns(context.Context, *domain.BackupExecution) error {
	return nil
}
func (*triggerBackupRepoStub) GetExecutionByID(context.Context, string, string) (*domain.BackupExecution, error) {
	return nil, errors.New("unexpected execution lookup")
}
func (*triggerBackupRepoStub) ListExecutions(context.Context, string, int, int) ([]*domain.BackupExecution, int64, error) {
	return nil, 0, nil
}
func (*triggerBackupRepoStub) GetExpiredExecutions(context.Context, string, int) ([]*domain.BackupExecution, error) {
	return nil, nil
}
func (*triggerBackupRepoStub) DeleteExecution(context.Context, string) error { return nil }

type switchableBackupWriter struct {
	destination io.Writer
	failWrites  bool
}

func (w *switchableBackupWriter) Write(payload []byte) (int, error) {
	if w.failWrites {
		return 0, errInjectedBackupWrite
	}
	return w.destination.Write(payload)
}

var errInjectedBackupWrite = errors.New("injected backup destination write failure")

type backupBytesSource struct {
	data       []byte
	afterWrite func()
}

func (s backupBytesSource) WriteBackup(_ context.Context, _ string, _ time.Time, dst io.Writer) error {
	if _, err := dst.Write(s.data); err != nil {
		return err
	}
	if s.afterWrite != nil {
		s.afterWrite()
	}
	return nil
}

func incompressibleBackupFixture(size int) []byte {
	data := make([]byte, size)
	seed := uint32(0x13579bdf)
	for i := range data {
		seed = seed*1664525 + 1013904223
		data[i] = byte(seed >> 24)
	}
	return data
}

func TestBackupService_TriggerBackupFailsClosedOnGzipIOErrors(t *testing.T) {
	tests := []struct {
		name             string
		payloadSize      int
		failDuringWrite  bool
		failAfterContent bool
	}{
		{name: "gzip write error", payloadSize: 64 * 1024, failDuringWrite: true},
		{name: "gzip close error", payloadSize: 256, failAfterContent: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &triggerBackupRepoStub{}
			backupDir := t.TempDir()
			var output *switchableBackupWriter
			source := backupBytesSource{data: incompressibleBackupFixture(test.payloadSize)}
			if test.failAfterContent {
				source.afterWrite = func() { output.failWrites = true }
			}
			svc := NewBackupServiceWithContentSource(repo, nil, backupDir, source)
			svc.newGzipWriter = func(dst io.Writer) io.WriteCloser {
				output = &switchableBackupWriter{destination: dst, failWrites: test.failDuringWrite}
				return gzip.NewWriter(output)
			}

			_, err := svc.TriggerBackup(context.Background(), "tenant-1", "")
			if !errors.Is(err, errInjectedBackupWrite) {
				t.Fatalf("expected injected gzip I/O error, got %v", err)
			}
			if repo.lastUpdated == nil || repo.lastUpdated.Status != domain.BackupStatusFailed {
				t.Fatalf("expected failed execution persisted, got %#v", repo.lastUpdated)
			}
			if repo.lastUpdated.SHA256Checksum != "" {
				t.Fatalf("failed execution must not store a checksum, got %q", repo.lastUpdated.SHA256Checksum)
			}
			if _, statErr := os.Stat(filepath.Join(backupDir, repo.lastUpdated.FileName)); !os.IsNotExist(statErr) {
				t.Fatalf("partial backup should be removed, stat error = %v", statErr)
			}
		})
	}
}

func TestBackupService_TriggerBackupChecksumsReReadBytes(t *testing.T) {
	tests := []struct {
		name     string
		readback []byte
	}{
		{name: "database bytes", readback: []byte(strings.Repeat("persisted-file-bytes-a", 20))},
		{name: "different database bytes", readback: []byte(strings.Repeat("persisted-file-bytes-b", 30))},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &triggerBackupRepoStub{}
			backupDir := t.TempDir()
			svc := NewBackupServiceWithContentSource(repo, nil, backupDir, backupBytesSource{
				data: incompressibleBackupFixture(4096),
			})
			readCount := 0
			svc.readBackupFile = func(path string) ([]byte, error) {
				readCount++
				if _, err := os.Stat(path); err != nil {
					return nil, err
				}
				return bytes.Clone(test.readback), nil
			}

			_, err := svc.TriggerBackup(context.Background(), "tenant-1", "")
			if err != nil {
				t.Fatalf("TriggerBackup() error = %v", err)
			}
			if readCount != 1 {
				t.Fatalf("expected one completed-file read, got %d", readCount)
			}
			if repo.lastUpdated == nil || repo.lastUpdated.Status != domain.BackupStatusSuccess {
				t.Fatalf("expected successful execution, got %#v", repo.lastUpdated)
			}
			wantSum := sha256.Sum256(test.readback)
			if got, want := repo.lastUpdated.SHA256Checksum, hex.EncodeToString(wantSum[:]); got != want {
				t.Fatalf("checksum did not use re-read bytes: got %q, want %q", got, want)
			}
		})
	}
}
