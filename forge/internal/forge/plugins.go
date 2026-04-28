package forge

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// pluginPrefix is what every external plugin binary's filename must start
// with for forge to discover it via $PATH.
const pluginPrefix = "forge-"

// Plugin describes one discovered external command. Path is the absolute
// filesystem location LookPath returned for it.
type Plugin struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// PluginEnv carries the parts of forge's runtime context that plugins
// would otherwise have to re-derive: the resolved working directory,
// whether globals like --json or --yes were set, and which config files
// were in play. It's serialized into env vars by RunPlugin.
type PluginEnv struct {
	WorkDir     string
	Project     string
	SubnetsFile string
	ConfigPath  string
	AutoYes     bool
	JSON        bool
	Version     string
}

// FindPlugin returns the absolute path to forge-<name> if it is on $PATH
// and executable, or "", false otherwise. exec.LookPath already enforces
// the executable bit on POSIX; on Windows it also handles PATHEXT.
func FindPlugin(name string) (string, bool) {
	if name == "" || strings.ContainsRune(name, os.PathSeparator) {
		return "", false
	}
	path, err := exec.LookPath(pluginPrefix + name)
	if err != nil {
		return "", false
	}
	return path, true
}

// ListPlugins walks every directory in $PATH looking for executables
// whose name starts with `forge-`. Names are deduplicated by basename so
// shadowing (earlier $PATH entries win) matches what the user actually
// gets when typing `forge <name>`.
func ListPlugins() ([]Plugin, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return nil, nil
	}
	sep := string(os.PathListSeparator)
	dirs := strings.Split(pathEnv, sep)

	seen := map[string]bool{}
	var plugins []Plugin
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasPrefix(name, pluginPrefix) || name == pluginPrefix {
				continue
			}
			if e.IsDir() {
				continue
			}
			pluginName := strings.TrimSuffix(name, ".exe")
			pluginName = strings.TrimPrefix(pluginName, pluginPrefix)
			if seen[pluginName] {
				continue
			}
			full := filepath.Join(dir, name)
			seen[pluginName] = true
			plugins = append(plugins, Plugin{Name: pluginName, Path: full})
		}
	}
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].Name < plugins[j].Name })
	return plugins, nil
}

// RunPlugin executes `path args...` with stdin/stdout/stderr inherited
// from forge. Returns the plugin's exit code (0 on clean exit). Any
// startup error returns 1 and a non-nil error.
func RunPlugin(path string, args []string, env PluginEnv) (int, error) {
	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = pluginEnviron(env)
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), nil
		}
		return 1, fmt.Errorf("run plugin %s: %w", path, err)
	}
	return 0, nil
}

// pluginEnviron constructs the child env: existing vars plus the FORGE_*
// keys plugins can rely on. If a key is set in os.Environ() already we
// override it (this is forge's own context, not the user's leftover).
func pluginEnviron(env PluginEnv) []string {
	out := os.Environ()
	overrides := map[string]string{
		"FORGE_VERSION":      env.Version,
		"FORGE_WORK_DIR":     env.WorkDir,
		"FORGE_PROJECT":      env.Project,
		"FORGE_SUBNETS_FILE": env.SubnetsFile,
		"FORGE_CONFIG_PATH":  env.ConfigPath,
	}
	if env.AutoYes {
		overrides["FORGE_AUTO_YES"] = "1"
	}
	if env.JSON {
		overrides["FORGE_JSON"] = "1"
	}

	dropPrefixes := []string{}
	for k := range overrides {
		dropPrefixes = append(dropPrefixes, k+"=")
	}

	cleaned := out[:0]
	for _, kv := range out {
		drop := false
		for _, p := range dropPrefixes {
			if strings.HasPrefix(kv, p) {
				drop = true
				break
			}
		}
		if !drop {
			cleaned = append(cleaned, kv)
		}
	}
	for k, v := range overrides {
		if v == "" {
			continue
		}
		cleaned = append(cleaned, k+"="+v)
	}
	return cleaned
}

// expectedPlugins is the set of plugins that ship with forge. `forge
// doctor` checks that each is reachable on $PATH and warns (does not
// fail) if any are missing.
var expectedPlugins = []string{
	"snapshot", "start", "stop", "migrate", "networks-prune",
	"cost",
}

// ExpectedPlugins returns the plugins that ship with forge so other
// subsystems (doctor, help, the legacy-name error shim) can stay in
// sync with one source of truth.
func ExpectedPlugins() []string {
	out := make([]string, len(expectedPlugins))
	copy(out, expectedPlugins)
	return out
}
