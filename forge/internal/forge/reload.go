package forge

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RunReload sends a POST /reload to the running config server using the
// listen address from DefaultDeployConfig. The server reloads its
// instances.json without restarting, so this is the lightest-weight way to
// pick up changes after a re-export.
func RunReload() error {
	cfg := DefaultDeployConfig()
	url := fmt.Sprintf("http://%s:%s/reload", cfg.ServerIP, cfg.ServerPort)

	client := http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(nil))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("contact server at %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	if len(body) > 0 {
		fmt.Println(string(body))
	} else {
		fmt.Println("reload ok")
	}
	return nil
}
