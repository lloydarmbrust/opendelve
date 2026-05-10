package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lloydarmbrust/opendelve/internal/packs"
)

// TestPacksAvailable verifies all 4 v1 packs are bundled and discoverable.
func TestPacksAvailable(t *testing.T) {
	avail, err := packs.Available()
	if err != nil {
		t.Fatalf("packs.Available: %v", err)
	}
	want := []string{"iso13485-starter", "iso9001", "pci-dss-starter", "soc2-starter"}
	if len(avail) != len(want) {
		t.Fatalf("got %d packs, want %d (%v vs %v)", len(avail), len(want), avail, want)
	}
	for i, w := range want {
		if avail[i] != w {
			t.Errorf("pack[%d]: got %q, want %q", i, avail[i], w)
		}
	}
}

// TestPacksLoadIso9001 verifies the deep ISO 9001 pack manifest loads cleanly.
func TestPacksLoadIso9001(t *testing.T) {
	m, err := packs.Load("iso9001")
	if err != nil {
		t.Fatalf("packs.Load(iso9001): %v", err)
	}
	if m.SchemaVersion != "v1" {
		t.Errorf("schemaVersion: got %q, want v1", m.SchemaVersion)
	}
	if m.ID != "iso9001" {
		t.Errorf("id: got %q, want iso9001", m.ID)
	}
	if m.Depth != "deep" {
		t.Errorf("depth: got %q, want deep", m.Depth)
	}
	if m.Framework.Name != "ISO 9001" {
		t.Errorf("framework.name: got %q", m.Framework.Name)
	}
	if len(m.Files) < 4 {
		t.Errorf("expected at least 4 files (sop+workflow+evidence+control-map), got %d", len(m.Files))
	}
	if !contains(m.Capabilities, "approval-signing") {
		t.Errorf("expected 'approval-signing' capability")
	}
}

// TestPacksLoadStubs verifies all 3 coming-soon stubs declare depth correctly.
func TestPacksLoadStubs(t *testing.T) {
	for _, name := range []string{"soc2-starter", "iso13485-starter", "pci-dss-starter"} {
		m, err := packs.Load(name)
		if err != nil {
			t.Errorf("packs.Load(%s): %v", name, err)
			continue
		}
		if m.Depth != "coming-soon" {
			t.Errorf("%s depth: got %q, want coming-soon", name, m.Depth)
		}
	}
}

// TestMaterializeIso9001 runs the same code path as init, verifies file destinations.
func TestMaterializeIso9001(t *testing.T) {
	target := t.TempDir()
	written, err := packs.Materialize("iso9001", target, func(rel string, content []byte) error {
		full := filepath.Join(target, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		return os.WriteFile(full, content, 0o644)
	})
	if err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	wantPaths := []string{
		"packs/iso9001/pack.yaml",
		"sops/SOP-0001-temperature-sensitive-shipment-release.md",
		"workflows/WF-0001-shipment-release.yml",
		"evidence-schemas/temperature-log.yaml",
		"controls/iso9001/control-map.yaml",
	}
	for _, want := range wantPaths {
		full := filepath.Join(target, want)
		st, err := os.Stat(full)
		if err != nil {
			t.Errorf("missing materialized file %s: %v", want, err)
			continue
		}
		if st.Size() == 0 {
			t.Errorf("materialized file %s is zero bytes", want)
		}
	}
	if len(written) != len(wantPaths) {
		t.Errorf("wrote %d files, expected %d (%v)", len(written), len(wantPaths), written)
	}
}

// TestMaterializeStubGoesUnderPacks verifies stub packs land under packs/<name>/.
func TestMaterializeStubGoesUnderPacks(t *testing.T) {
	target := t.TempDir()
	_, err := packs.Materialize("soc2-starter", target, func(rel string, content []byte) error {
		full := filepath.Join(target, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		return os.WriteFile(full, content, 0o644)
	})
	if err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	// STUB.md (role: doc) should live at packs/soc2-starter/STUB.md per repoDest.
	stub := filepath.Join(target, "packs", "soc2-starter", "STUB.md")
	if _, err := os.Stat(stub); err != nil {
		t.Errorf("expected stub at %s: %v", stub, err)
	}
}

// TestPacksUnknownReturnsError verifies invalid pack name fails cleanly.
func TestPacksUnknownReturnsError(t *testing.T) {
	_, err := packs.Load("nope")
	if err == nil {
		t.Errorf("expected error for unknown pack, got nil")
	}
}
