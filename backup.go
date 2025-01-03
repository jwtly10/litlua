package litlua

import (
	"fmt"
	"io"
	"os"
	"time"
)

// BackupManager is a helper struct to manage backups of output files
//
// This is a short term solution to ensuring that the output file is not overwritten
// by accident.
type BackupManager struct {
}

func NewBackupManager() *BackupManager {
	return &BackupManager{}
}

// CreateBackupOf creates a backup of an absolute file path if it already exists
//
// Returns the abs path to the backup file, or an empty string if no backup was created
func (bm *BackupManager) CreateBackupOf(absFilePath string) (absBackedUpFile string, err error) {
	if _, err := os.Stat(absFilePath); os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("checking file existence: %w", err)
	}

	absBackedUpFile = fmt.Sprintf("%s.%s.bak", absFilePath, time.Now().Format("20060102_150405"))

	if err := bm.copyFile(absFilePath, absBackedUpFile); err != nil {
		return "", fmt.Errorf("creating backup: %w", err)
	}

	return absBackedUpFile, nil
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
