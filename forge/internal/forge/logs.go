package forge

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// RunLogs prints the contents of server.log for the current project. When
// follow is true, the function blocks and streams new bytes as they are
// appended (similar to `tail -f`), polling on a 500ms interval so we don't
// pull in inotify dependencies. Returns once the file is fully read in
// non-follow mode, or when stdin closes / SIGINT in follow mode.
func RunLogs(workDir string, follow bool) error {
	logPath := filepath.Join(workDir, "server.log")

	f, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", logPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(os.Stdout, f); err != nil {
		return fmt.Errorf("read %s: %w", logPath, err)
	}

	if !follow {
		return nil
	}

	for {
		time.Sleep(500 * time.Millisecond)
		if _, err := io.Copy(os.Stdout, f); err != nil {
			return fmt.Errorf("read %s: %w", logPath, err)
		}
	}
}
