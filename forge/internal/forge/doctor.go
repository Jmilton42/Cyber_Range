package forge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// CheckStatus is the outcome of a single doctor check.
type CheckStatus string

const (
	CheckOK   CheckStatus = "OK"
	CheckWarn CheckStatus = "WARN"
	CheckFail CheckStatus = "FAIL"
)

// CheckResult is one row in the doctor report.
type CheckResult struct {
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Message string      `json:"message"`
}

// DoctorReport is the full doctor output, suitable for JSON serialization.
type DoctorReport struct {
	Checks  []CheckResult `json:"checks"`
	OK      int           `json:"ok"`
	Warn    int           `json:"warn"`
	Fail    int           `json:"fail"`
	Healthy bool          `json:"healthy"`
}

// RunDoctor executes preflight checks against the operator environment so
// problems show up before they break a `forge apply`. Returns a non-nil
// report regardless of individual check failures; the caller should look at
// report.Healthy to decide the exit code.
func RunDoctor() DoctorReport {
	cfg := DefaultDeployConfig()
	report := DoctorReport{}

	add := func(name string, status CheckStatus, msg string) {
		report.Checks = append(report.Checks, CheckResult{Name: name, Status: status, Message: msg})
	}

	if path, err := exec.LookPath("tofu"); err == nil {
		add("tofu binary", CheckOK, path)
	} else {
		add("tofu binary", CheckFail, "tofu not found on PATH")
	}

	if path, err := exec.LookPath("lxc"); err == nil {
		add("lxc binary", CheckOK, path)
	} else {
		add("lxc binary", CheckFail, "lxc not found on PATH")
	}

	if path, err := exec.LookPath("jq"); err == nil {
		add("jq binary", CheckOK, path)
	} else {
		add("jq binary", CheckWarn, "jq not found on PATH (required by lxd_scripts wrappers)")
	}

	if data, err := os.ReadFile(SubnetsFile); err == nil {
		var sd SubnetsData
		if jerr := json.Unmarshal(data, &sd); jerr != nil {
			add("subnets.json", CheckFail, fmt.Sprintf("%s exists but is not valid JSON: %v", SubnetsFile, jerr))
		} else {
			add("subnets.json", CheckOK, fmt.Sprintf("%s (%d allocations)", SubnetsFile, len(sd.Allocations)))
		}
	} else if os.IsNotExist(err) {
		add("subnets.json", CheckWarn, fmt.Sprintf("%s missing - run 'forge init' to create it", SubnetsFile))
	} else {
		add("subnets.json", CheckFail, err.Error())
	}

	if _, err := LoadForgeConfig(); err == nil {
		add("config.yaml", CheckOK, "loaded successfully")
	} else {
		add("config.yaml", CheckWarn, fmt.Sprintf("not found - using built-in defaults (%s)", err.Error()))
	}

	if _, err := os.Stat(cfg.StartWinScript); err == nil {
		add("start_win.sh", CheckOK, cfg.StartWinScript)
	} else {
		add("start_win.sh", CheckWarn, fmt.Sprintf("%s missing - 'forge apply' will skip Windows VM start step", cfg.StartWinScript))
	}

	scriptsDir := filepath.Dir(cfg.StartWinScript)
	requiredScripts := []string{"snapshot.sh", "start_vms.sh", "stop_vms.sh", "move_vms.sh", "move_vms_nodes.sh", "remove_networks.sh"}
	missing := []string{}
	for _, s := range requiredScripts {
		if _, err := os.Stat(filepath.Join(scriptsDir, s)); err != nil {
			missing = append(missing, s)
		}
	}
	if len(missing) == 0 {
		add("lxd_scripts", CheckOK, fmt.Sprintf("all wrappers present in %s", scriptsDir))
	} else {
		add("lxd_scripts", CheckWarn, fmt.Sprintf("missing in %s: %v", scriptsDir, missing))
	}

	if out, err := exec.Command("lxc", "cluster", "list", "--format", "json").Output(); err == nil {
		var members []map[string]interface{}
		if jerr := json.Unmarshal(out, &members); jerr == nil {
			add("lxc cluster", CheckOK, fmt.Sprintf("%d members", len(members)))
		} else {
			add("lxc cluster", CheckWarn, "could not parse cluster list output")
		}
	} else {
		add("lxc cluster", CheckWarn, fmt.Sprintf("lxc cluster list failed: %v", err))
	}

	if tpls, err := LoadTemplates(); err == nil {
		dir := templatesDir()
		switch {
		case dir == "" && len(tpls) <= 1:
			add("templates", CheckWarn, fmt.Sprintf("no templates dir found - create %s and drop template directories in it", DefaultTemplatesDir))
		case dir == "":
			add("templates", CheckOK, fmt.Sprintf("%d template(s) (synthetic only - no templates dir on disk)", len(tpls)))
		default:
			add("templates", CheckOK, fmt.Sprintf("%d template(s) discovered in %s", len(tpls), dir))
		}
	} else {
		add("templates", CheckWarn, fmt.Sprintf("templates dir unreadable: %v", err))
	}

	missingPlugins := []string{}
	for _, name := range ExpectedPlugins() {
		if _, ok := FindPlugin(name); !ok {
			missingPlugins = append(missingPlugins, "forge-"+name)
		}
	}
	if len(missingPlugins) == 0 {
		add("plugins", CheckOK, fmt.Sprintf("all bundled plugins reachable on $PATH (%v)", ExpectedPlugins()))
	} else {
		add("plugins", CheckWarn, fmt.Sprintf("missing on $PATH: %v - run forge/scripts/build_all.sh", missingPlugins))
	}

	listenAddr := cfg.ServerIP + ":" + cfg.ServerPort
	statusURL := fmt.Sprintf("http://%s/status", listenAddr)
	client := http.Client{Timeout: 2 * time.Second}
	if resp, err := client.Get(statusURL); err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			add("config server", CheckOK, fmt.Sprintf("%s responded 200", statusURL))
		} else {
			add("config server", CheckWarn, fmt.Sprintf("%s responded %d", statusURL, resp.StatusCode))
		}
	} else {
		add("config server", CheckWarn, fmt.Sprintf("%s unreachable (likely not running yet)", statusURL))
	}

	for _, c := range report.Checks {
		switch c.Status {
		case CheckOK:
			report.OK++
		case CheckWarn:
			report.Warn++
		case CheckFail:
			report.Fail++
		}
	}
	report.Healthy = report.Fail == 0

	return report
}

// PrintDoctorText renders a DoctorReport as colorized text suitable for a
// human looking at a terminal.
func PrintDoctorText(r DoctorReport) {
	for _, c := range r.Checks {
		var tag string
		switch c.Status {
		case CheckOK:
			tag = "\033[32m[OK]\033[0m  "
		case CheckWarn:
			tag = "\033[33m[WARN]\033[0m"
		case CheckFail:
			tag = "\033[31m[FAIL]\033[0m"
		}
		fmt.Printf("%s %-20s  %s\n", tag, c.Name, c.Message)
	}
	fmt.Println()
	fmt.Printf("Summary: %d ok, %d warn, %d fail\n", r.OK, r.Warn, r.Fail)
}
