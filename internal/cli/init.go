package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lloydarmbrust/opendelve/internal/jsonout"
	"github.com/lloydarmbrust/opendelve/internal/packs"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	initPackList string
	initForce    bool
	initJSON     bool
)

var initCmd = &cobra.Command{
	Use:   "init <repo-name>",
	Short: "Bootstrap a new OpenDelve-managed repo",
	Long: `Materialize a new OpenDelve-managed repo into <repo-name>/. The repo
gets sops/, workflows/, evidence-schemas/, controls/, opendelve.yaml,
AGENTS.md, and a .gitignore template that keeps raw evidence out of git.

Refuses if the target directory already contains files. Pass --force to
overwrite. To extend an existing OpenDelve repo with another pack, use
'opendelve pack add' instead.

When invoked with no --pack flag in an interactive terminal, drops into
a wizard. In non-interactive (CI/agent) contexts, returns an error
listing available packs.`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initPackList, "pack", "",
		"comma-separated list of packs to install (e.g. iso9001 or iso9001,soc2-starter)")
	initCmd.Flags().BoolVar(&initForce, "force", false,
		"overwrite the target directory if it exists with content")
	initCmd.Flags().BoolVar(&initJSON, "json", false,
		"emit machine-readable JSON envelope instead of human output")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	repoName := args[0]
	target, err := filepath.Abs(repoName)
	if err != nil {
		emit(jsonout.NewError("io_failed", "could not resolve target path").
			WithHint(err.Error()))
		os.Exit(1)
	}

	// Resolve which packs to install: --pack flag if set, otherwise wizard or error.
	available, err := packs.Available()
	if err != nil {
		emit(jsonout.NewError("internal_error", "could not list bundled packs").
			WithHint(err.Error()))
		os.Exit(1)
	}

	var selected []string
	switch {
	case initPackList != "":
		for _, p := range strings.Split(initPackList, ",") {
			p = strings.TrimSpace(p)
			if !contains(available, p) {
				emit(jsonout.NewError("pack_unknown",
					fmt.Sprintf("unknown pack %q", p)).
					WithHint(fmt.Sprintf("available: %s", packs.FormatList(available))))
				os.Exit(1)
			}
			selected = append(selected, p)
		}
	case jsonout.IsTTY(os.Stdin):
		// Interactive wizard. v1: simple prompt; can grow to questionary-style later.
		selected, err = runWizard(available)
		if err != nil {
			emit(jsonout.NewError("io_failed", err.Error()))
			os.Exit(1)
		}
	default:
		emit(jsonout.NewError("pack_unknown",
			"--pack flag required in non-interactive mode").
			WithHint(fmt.Sprintf("try --pack iso9001 (available: %s)", packs.FormatList(available))))
		os.Exit(1)
	}

	if len(selected) == 0 {
		emit(jsonout.NewError("pack_unknown", "no packs selected").
			WithHint(fmt.Sprintf("available: %s", packs.FormatList(available))))
		os.Exit(1)
	}

	// Refuse on existing non-empty dir unless --force.
	if entries, err := os.ReadDir(target); err == nil && len(entries) > 0 && !initForce {
		emit(jsonout.NewError("config_invalid",
			fmt.Sprintf("target directory %s already exists and is not empty", target)).
			WithHint("pass --force to overwrite, or pick a fresh directory; for adding a pack to an existing repo use 'opendelve pack add'"))
		os.Exit(1)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		emit(jsonout.NewError("io_failed", "could not create target directory").
			WithHint(err.Error()))
		os.Exit(1)
	}

	// Materialize each pack.
	written := []string{}
	for _, name := range selected {
		paths, err := packs.Materialize(name, target, func(rel string, content []byte) error {
			full := filepath.Join(target, rel)
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return err
			}
			return os.WriteFile(full, content, 0o644)
		})
		if err != nil {
			emit(jsonout.NewError("io_failed",
				fmt.Sprintf("materialize pack %s failed", name)).
				WithHint(err.Error()))
			os.Exit(1)
		}
		written = append(written, paths...)
	}

	// Write opendelve.yaml at repo root.
	cfg := repoConfig{
		SchemaVersion: "v1",
		Name:          repoName,
		DefaultBranch: "main",
		Packs:         []packEntry{},
		Runtime: runtimeConfig{
			EvidenceMode:        "local",
			PacketMode:          "zip-plus-pdf",
			ApprovalMode:        "git-bound",
			RawEvidenceInPacket: false,
		},
	}
	for _, name := range selected {
		m, _ := packs.Load(name)
		cfg.Packs = append(cfg.Packs, packEntry{Name: name, Version: m.Version})
	}
	cfgBytes, err := yaml.Marshal(cfg)
	if err != nil {
		emit(jsonout.NewError("internal_error", "could not marshal opendelve.yaml").
			WithHint(err.Error()))
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(target, "opendelve.yaml"), cfgBytes, 0o644); err != nil {
		emit(jsonout.NewError("io_failed", "could not write opendelve.yaml").
			WithHint(err.Error()))
		os.Exit(1)
	}
	written = append(written, "opendelve.yaml")

	// Write a .gitignore that keeps raw evidence out of git (PHI safety per eng A5).
	gitignore := `# OpenDelve: raw evidence files stay local; only manifests go in git.
evidence/raw/**
*.tmp
.DS_Store
`
	if err := os.WriteFile(filepath.Join(target, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		emit(jsonout.NewError("io_failed", "could not write .gitignore").
			WithHint(err.Error()))
		os.Exit(1)
	}
	written = append(written, ".gitignore")

	// Make the empty raw evidence dir so users see the structure.
	_ = os.MkdirAll(filepath.Join(target, "evidence", "raw"), 0o755)
	_ = os.MkdirAll(filepath.Join(target, "evidence", "manifests"), 0o755)
	_ = os.MkdirAll(filepath.Join(target, "approvals"), 0o755)
	_ = os.MkdirAll(filepath.Join(target, "audit-packets"), 0o755)

	// Success envelope.
	dec := jsonout.NewDecision(jsonout.StatusOK, repoName)
	dec.SubjectType = "subject"
	dec.Data = map[string]any{
		"target":      target,
		"packs":       selected,
		"filesWritten": len(written),
		"createdAt":   time.Now().UTC().Format(time.RFC3339),
	}
	dec.AllowedNext = []string{
		"cd " + repoName,
		"opendelve sop add <slug>",
		"opendelve evidence add <file> --schema temperature-log",
	}

	mode := jsonout.ModeAuto
	if initJSON {
		mode = jsonout.ModeJSON
	}
	exit := jsonout.Emit(os.Stdout, os.Stderr, jsonout.IsTTY(os.Stdout), mode, dec)
	// In human mode, print a friendly post-init nudge with skill install hint (devex F5).
	if !initJSON && jsonout.IsTTY(os.Stdout) {
		fmt.Fprintf(os.Stdout, "\n  created %d files in %s\n", len(written), target)
		fmt.Fprintf(os.Stdout, "  packs: %s\n", strings.Join(selected, ", "))
		fmt.Fprintf(os.Stdout, "  next:  cd %s && opendelve sop add <slug>\n", repoName)
		fmt.Fprintf(os.Stdout, "\n  Tip: Claude Code users can drop opendelve.skill.md into ~/.claude/skills/\n")
		fmt.Fprintf(os.Stdout, "  to enable agent integration. See AGENTS.md in this repo for the contract.\n")
	}
	os.Exit(exit)
	return nil
}

// runWizard presents the simplest possible interactive picker for v1.
// A future patch can swap this for charmbracelet/huh or similar.
func runWizard(available []string) ([]string, error) {
	fmt.Fprintln(os.Stderr, "OpenDelve init wizard")
	fmt.Fprintln(os.Stderr, "Available packs:")
	for i, name := range available {
		fmt.Fprintf(os.Stderr, "  [%d] %s\n", i+1, name)
	}
	fmt.Fprint(os.Stderr, "Enter pack name (or comma-separated list): ")
	var line string
	if _, err := fmt.Scanln(&line); err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	var picked []string
	for _, p := range strings.Split(line, ",") {
		p = strings.TrimSpace(p)
		if !contains(available, p) {
			return nil, fmt.Errorf("unknown pack %q", p)
		}
		picked = append(picked, p)
	}
	return picked, nil
}

// repoConfig matches schemas/opendelve.schema.json.
type repoConfig struct {
	SchemaVersion string        `yaml:"schemaVersion"`
	Name          string        `yaml:"name"`
	DefaultBranch string        `yaml:"defaultBranch"`
	Packs         []packEntry   `yaml:"packs"`
	Runtime       runtimeConfig `yaml:"runtime"`
}

type packEntry struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type runtimeConfig struct {
	EvidenceMode        string `yaml:"evidenceMode"`
	PacketMode          string `yaml:"packetMode"`
	ApprovalMode        string `yaml:"approvalMode"`
	RawEvidenceInPacket bool   `yaml:"rawEvidenceInPacket"`
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// emit is a thin convenience that emits an error envelope. (Decision envelopes
// are emitted directly via jsonout.Emit so the right exit code propagates.)
func emit(err *jsonout.Error) {
	mode := jsonout.ModeAuto
	if initJSON {
		mode = jsonout.ModeJSON
	}
	jsonout.Emit(os.Stdout, os.Stderr, jsonout.IsTTY(os.Stdout), mode, err)
}
