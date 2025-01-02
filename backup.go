package litlua

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

// BackupManager is a helper struct to manage backups of output files
//
// This is a short term solution to ensuring that the output file is not overwritten
// by accident.
type BackupManager struct {
	path string
}

func NewBackupManager(path string) *BackupManager {
	return &BackupManager{
		path: path,
	}
}

// CreateBackup creates a backup of the output file if it already exists
//
// Returns the path to the backup file, or an empty string if no backup was created
func (bm *BackupManager) CreateBackup() (backupPath string, err error) {
	if _, err := os.Stat(bm.path); os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("checking file existence: %w", err)
	}

	backupPath = fmt.Sprintf("%s.%s.bak", bm.path, time.Now().Format("20060102_150405"))

	if err := bm.copyFile(bm.path, backupPath); err != nil {
		return "", fmt.Errorf("creating backup: %w", err)
	}

	slog.Info("output file already existed. Created a backup.", "backup", backupPath, "output", bm.path)
	return backupPath, nil
}

func (bm *BackupManager) copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	return nil
}
