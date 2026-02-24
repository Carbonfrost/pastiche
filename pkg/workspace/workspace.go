package workspace

import (
	"context"
	"os"
	"path/filepath"
)

func LogDir(_ context.Context) string {
	// TODO This directory should be the workspace
	logDir := filepath.Join(".pastiche", "logs")
	os.MkdirAll(logDir, 0755)

	return logDir
}

func ClearLogDir(c context.Context) error {
	err := os.RemoveAll(LogDir(c))
	if err != nil {
		return err
	}

	_ = LogDir(c) // Recreate the directory
	return nil
}
