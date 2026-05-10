// Package packs loads OpenDelve compliance packs that are embedded into the
// binary at build time via go:embed. v1 ships 4 packs: iso9001 (deep) plus
// 3 "coming soon" stubs (soc2-starter, iso13485-starter, pci-dss-starter).
package packs

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Embedded is the bundle of packs/*.
// The relative path here is from the package directory (internal/packs/) to
// the repo root packs/ directory: ../../packs.
//
//go:embed all:embedded
var Embedded embed.FS

// Manifest matches schemas/pack.schema.json.
type Manifest struct {
	SchemaVersion string       `yaml:"schemaVersion" json:"schemaVersion"`
	ID            string       `yaml:"id"            json:"id"`
	Version       string       `yaml:"version"       json:"version"`
	Framework     Framework    `yaml:"framework"     json:"framework"`
	Name          string       `yaml:"name"          json:"name"`
	Description   string       `yaml:"description"   json:"description"`
	Depth         string       `yaml:"depth"         json:"depth,omitempty"`
	Files         []FileEntry  `yaml:"files"         json:"files"`
	Controls      []Control    `yaml:"controls"      json:"controls,omitempty"`
	Mappings      []Mapping    `yaml:"mappings"      json:"mappings,omitempty"`
	Dependencies  []Dependency `yaml:"dependencies"  json:"dependencies,omitempty"`
	Capabilities  []string     `yaml:"capabilities"  json:"capabilities,omitempty"`
}

// Framework identifies the standard a pack covers.
type Framework struct {
	Name    string `yaml:"name"    json:"name"`
	Version string `yaml:"version" json:"version"`
	URL     string `yaml:"url"     json:"url,omitempty"`
}

// FileEntry is one file in a pack with its role + control mappings.
type FileEntry struct {
	Path       string   `yaml:"path"       json:"path"`
	Role       string   `yaml:"role"       json:"role"`
	ControlIDs []string `yaml:"controlIds" json:"controlIds,omitempty"`
}

// Control is a framework control covered by the pack.
type Control struct {
	ID    string `yaml:"id"    json:"id"`
	Title string `yaml:"title" json:"title"`
	Scope string `yaml:"scope" json:"scope,omitempty"`
}

// Mapping cross-links one framework's control to another's.
type Mapping struct {
	FromControl string `yaml:"fromControl" json:"fromControl"`
	ToFramework string `yaml:"toFramework" json:"toFramework"`
	ToControl   string `yaml:"toControl"   json:"toControl"`
	Alignment   string `yaml:"alignment"   json:"alignment,omitempty"`
}

// Dependency declares another pack required by this one.
type Dependency struct {
	Pack    string `yaml:"pack"    json:"pack"`
	Version string `yaml:"version" json:"version"`
}

// Available returns the names of all bundled packs, sorted.
func Available() ([]string, error) {
	entries, err := fs.ReadDir(Embedded, "embedded")
	if err != nil {
		return nil, fmt.Errorf("read embedded packs: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// A valid pack must contain pack.yaml.
		if _, err := fs.Stat(Embedded, path.Join("embedded", e.Name(), "pack.yaml")); err == nil {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// Load parses the pack.yaml of the named pack.
func Load(name string) (*Manifest, error) {
	if !validName(name) {
		return nil, fmt.Errorf("invalid pack name %q", name)
	}
	manifestPath := path.Join("embedded", name, "pack.yaml")
	data, err := fs.ReadFile(Embedded, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read pack manifest %s: %w", name, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse pack manifest %s: %w", name, err)
	}
	if m.ID != name {
		return nil, fmt.Errorf("pack %s manifest declares id=%q (mismatch)", name, m.ID)
	}
	return &m, nil
}

// Materialize copies the named pack's files into targetDir, preserving relative paths.
// Returns the list of relative paths written.
func Materialize(name, targetDir string, write func(relPath string, content []byte) error) ([]string, error) {
	m, err := Load(name)
	if err != nil {
		return nil, err
	}
	written := make([]string, 0, len(m.Files)+1)

	// Always copy pack.yaml (so the materialized pack stays self-describing).
	pkgYAML, err := fs.ReadFile(Embedded, path.Join("embedded", name, "pack.yaml"))
	if err != nil {
		return nil, fmt.Errorf("read pack.yaml: %w", err)
	}
	relPackYAML := path.Join("packs", name, "pack.yaml")
	if err := write(relPackYAML, pkgYAML); err != nil {
		return nil, fmt.Errorf("write %s: %w", relPackYAML, err)
	}
	written = append(written, relPackYAML)

	// Copy each declared file.
	for _, fe := range m.Files {
		src := path.Join("embedded", name, fe.Path)
		content, err := fs.ReadFile(Embedded, src)
		if err != nil {
			return nil, fmt.Errorf("read embedded %s: %w", src, err)
		}
		// For SOPs / workflows / evidence-schemas / controls we want them at the
		// repo root level (sops/, workflows/, evidence-schemas/, controls/), not
		// nested inside packs/<name>/. The pack.yaml file declares which directory
		// each file's role maps to. v1 mapping:
		dest := repoDest(fe, name)
		if err := write(dest, content); err != nil {
			return nil, fmt.Errorf("write %s: %w", dest, err)
		}
		written = append(written, dest)
	}
	return written, nil
}

// repoDest maps a pack file to its destination within the user's repo.
// "deep" packs surface their content at the repo root (sops/, workflows/, etc.).
// "coming-soon" stub packs put their STUB.md under packs/<name>/.
func repoDest(fe FileEntry, packName string) string {
	switch fe.Role {
	case "sop":
		return path.Join("sops", path.Base(fe.Path))
	case "workflow":
		return path.Join("workflows", path.Base(fe.Path))
	case "evidence-schema":
		return path.Join("evidence-schemas", path.Base(fe.Path))
	case "control-map":
		return path.Join("controls", packName, path.Base(fe.Path))
	case "policy":
		return path.Join("policies", path.Base(fe.Path))
	case "doc":
		return path.Join("packs", packName, path.Base(fe.Path))
	default:
		// Unknown role: keep under packs/<name>/<original-relpath>.
		return path.Join("packs", packName, fe.Path)
	}
}

// validName matches the pack id pattern from pack.schema.json.
func validName(s string) bool {
	if len(s) < 2 {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' && i > 0 && i < len(s)-1:
		default:
			return false
		}
	}
	return true
}

// FormatList returns a human-friendly comma-separated list of available packs.
func FormatList(names []string) string {
	return strings.Join(names, ", ")
}
