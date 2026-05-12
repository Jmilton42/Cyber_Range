package forge

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// DefaultTemplatesDir is the canonical on-disk location for forge
// templates on a deployed CEROC host. Every immediate subdirectory of
// this path is treated as a template by `forge new` - no manifest file
// is required.
const DefaultTemplatesDir = "/home/ceroc/InSPIRE/templates"

// blankTemplateID is the synthetic id used for the "empty starter
// main.tf" template. It does not correspond to any directory on disk.
const blankTemplateID = "blank"

// Template is one entry returned by LoadTemplates. Path is the absolute
// filesystem location of the template's source directory. An empty
// Path marks the synthetic "blank" template.
type Template struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

// templatesDir returns the absolute path to the directory forge should
// scan for template subdirectories. Lookup order, first hit wins:
//  1. $FORGE_TEMPLATES if set and pointing at an existing directory.
//  2. DefaultTemplatesDir (the canonical production location).
//  3. Walk up from cwd looking for a `templates` or `range` directory
//     (developer convenience inside the repo).
//  4. "" - no templates dir on this host; only the synthetic "blank"
//     template will be available.
func templatesDir() string {
	if env := os.Getenv("FORGE_TEMPLATES"); env != "" {
		if info, err := os.Stat(env); err == nil && info.IsDir() {
			return env
		}
	}
	if info, err := os.Stat(DefaultTemplatesDir); err == nil && info.IsDir() {
		return DefaultTemplatesDir
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for {
		for _, candidate := range []string{
			filepath.Join(dir, "templates"),
			filepath.Join(dir, "range"),
		} {
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// LoadTemplates discovers templates by enumerating immediate
// subdirectories of templatesDir(). The synthetic "blank" template is
// always appended last so callers can rely on having at least one
// template available, even when no templates dir is on disk.
//
// Hidden directories (names starting with ".") and any directory
// without a main.tf are skipped, so a stray `.git` or doc-only folder
// won't pollute the picker.
func LoadTemplates() ([]Template, error) {
	dir := templatesDir()
	var tpls []Template
	if dir != "" {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("read templates dir %s: %w", dir, err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			full := filepath.Join(dir, name)
			mainTF := filepath.Join(full, "main.tf")
			if _, err := os.Stat(mainTF); err != nil {
				continue
			}
			tpls = append(tpls, Template{
				ID:          name,
				Path:        full,
				Description: parseTemplateDescription(mainTF),
			})
		}
		sort.Slice(tpls, func(i, j int) bool {
			return strings.ToLower(tpls[i].ID) < strings.ToLower(tpls[j].ID)
		})
	}
	tpls = append(tpls, Template{
		ID:          blankTemplateID,
		Path:        "",
		Description: "Empty main.tf with project_name only",
	})
	return tpls, nil
}

// FindTemplate returns the template with the given id, or an error if
// it doesn't exist. Lookup is case-insensitive so users don't have to
// match the directory's exact casing.
func FindTemplate(id string) (Template, error) {
	tpls, err := LoadTemplates()
	if err != nil {
		return Template{}, err
	}
	want := strings.ToLower(id)
	for _, t := range tpls {
		if strings.ToLower(t.ID) == want {
			return t, nil
		}
	}
	return Template{}, fmt.Errorf("template %q not found (run `forge new` with no args to see the list)", id)
}

// NewOptions configures a `forge new` invocation.
type NewOptions struct {
	Template  string
	Name      string
	TargetDir string
	Reserve   bool
	AutoYes   bool
	// Interactive selects the CLI's interactive picker when fields are
	// missing. Set to false for scripted use.
	Interactive bool
}

// projectNameRe mirrors what LXD itself accepts for project names:
// letters (any case), digits, dashes, and underscores. The first
// character must be alphanumeric. Length cap of 41 matches the original
// forge constraint and stays well under LXD's 63-char limit.
var projectNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,40}$`)

// RunNew is the entry point invoked by `forge new`. It prepares the
// target directory, copies (or generates) the template, rewrites
// project_name, and optionally allocates a subnet.
func RunNew(opts NewOptions) error {
	if opts.Name == "" {
		return fmt.Errorf("project name is required (--name)")
	}
	if !projectNameRe.MatchString(opts.Name) {
		return fmt.Errorf("invalid project name %q: must match %s", opts.Name, projectNameRe.String())
	}
	if opts.TargetDir == "" {
		opts.TargetDir = opts.Name
	}
	if _, err := os.Stat(opts.TargetDir); err == nil {
		return fmt.Errorf("target directory %s already exists - aborting to avoid overwriting", opts.TargetDir)
	}
	if opts.Template == "" {
		return fmt.Errorf("template id is required (--template), e.g. blank")
	}

	tpl, err := FindTemplate(opts.Template)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(opts.TargetDir, 0755); err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}

	if tpl.Path == "" {
		if err := os.WriteFile(filepath.Join(opts.TargetDir, "main.tf"), []byte(blankMainTF(opts.Name)), 0644); err != nil {
			return fmt.Errorf("write blank main.tf: %w", err)
		}
	} else {
		if _, err := os.Stat(tpl.Path); err != nil {
			return fmt.Errorf("template %q source missing at %s: %w", tpl.ID, tpl.Path, err)
		}
		if err := copyDir(tpl.Path, opts.TargetDir); err != nil {
			return fmt.Errorf("copy template files: %w", err)
		}
		mainTf := filepath.Join(opts.TargetDir, "main.tf")
		if _, err := os.Stat(mainTf); err == nil {
			if err := rewriteProjectName(mainTf, opts.Name); err != nil {
				return fmt.Errorf("rewrite project_name: %w", err)
			}
		}
	}

	fmt.Printf("[INFO] Created %s\n", opts.TargetDir)
	fmt.Printf("[INFO] Set project_name = %q\n", opts.Name)

	if opts.Reserve {
		octet, err := AllocateSubnet(opts.Name)
		if err != nil {
			fmt.Printf("[WARN] Could not reserve subnet: %v\n", err)
		} else {
			fmt.Printf("[INFO] Reserved subnet octet %d (%s)\n", octet, FormatSubnet(octet))
		}
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", opts.TargetDir)
	fmt.Println("  forge plan")
	fmt.Println("  forge apply")
	return nil
}

// PromptForTemplate prints a numbered list of templates and reads a
// 1-based index from stdin. Returns the selected template id. Users may
// also type the id (or its directory name) directly.
func PromptForTemplate(tpls []Template) (string, error) {
	if len(tpls) == 0 {
		return "", fmt.Errorf("no templates available")
	}
	fmt.Println("Available templates:")
	for i, t := range tpls {
		desc := t.Description
		if desc == "" {
			desc = "(directory: " + filepath.Base(t.Path) + ")"
		}
		fmt.Printf("  %d) %-22s %s\n", i+1, t.ID, desc)
	}
	fmt.Print("\nSelect template [1]: ")
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return tpls[0].ID, nil
	}
	var idx int
	if _, err := fmt.Sscanf(line, "%d", &idx); err != nil {
		want := strings.ToLower(line)
		for _, t := range tpls {
			if strings.ToLower(t.ID) == want {
				return t.ID, nil
			}
		}
		return "", fmt.Errorf("invalid selection %q", line)
	}
	if idx < 1 || idx > len(tpls) {
		return "", fmt.Errorf("selection out of range: %d", idx)
	}
	return tpls[idx-1].ID, nil
}

// PromptForProjectName reads a project name from stdin and validates it.
func PromptForProjectName() (string, error) {
	fmt.Print("Project name (letters, digits, '-' or '_'): ")
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if !projectNameRe.MatchString(line) {
		return "", fmt.Errorf("invalid project name %q (must match %s)", line, projectNameRe.String())
	}
	return line, nil
}

// blankMainTF returns the minimum main.tf content for the "blank"
// template: just the project_name variable forge needs to operate, with
// a placeholder for guac_subnet_octet.
func blankMainTF(name string) string {
	return fmt.Sprintf(`variable "project_name" {
  type    = string
  default = %q
}

variable "guac_subnet_octet" {
  type        = number
  default     = 1
  description = "Third octet for guac subnet (forge populates this at apply time)"
}

# Add your lxd_project / lxd_network / lxd_instance resources below.
`, name)
}

// copyDir recursively copies src into dst. Symlinks are skipped (we
// don't expect any in template directories). Existing files in dst are
// overwritten - but RunNew only calls copyDir into a freshly-created
// dir, so that case isn't reachable in normal flow.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

// parseTemplateDescription reads the description value from the first
// lxd_project resource block in a main.tf file. Returns "" if not found.
func parseTemplateDescription(mainTFPath string) string {
	data, err := os.ReadFile(mainTFPath)
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`(?s)resource\s+"lxd_project"\s+"[^"]*"\s*\{[^}]*?description\s*=\s*"([^"]*)"`)
	if m := re.FindSubmatch(data); len(m) > 1 {
		return strings.TrimSpace(string(m[1]))
	}
	return ""
}

// rewriteProjectName mutates the project_name variable's default value
// in a main.tf file. We rely on the same regex shape `ParseProjectName`
// uses so behavior is symmetric.
func rewriteProjectName(path, name string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	re := regexp.MustCompile(`(variable\s+"project_name"\s*\{[^}]*default\s*=\s*")([^"]+)(")`)
	if !re.Match(data) {
		return fmt.Errorf("could not locate project_name default in %s", path)
	}
	out := re.ReplaceAll(data, []byte("${1}"+name+"${3}"))
	return os.WriteFile(path, out, 0644)
}
