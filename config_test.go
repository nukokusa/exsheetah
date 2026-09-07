package exsheetah_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nukokusa/exsheetah"
)

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "exsheetah.yaml")
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConfig_Location(t *testing.T) {
	t.Parallel()

	t.Run("default is UTC", func(t *testing.T) {
		t.Parallel()
		path := writeConfig(t, `
sheets:
  - name: item
    columns:
      - name: id
        type: number
`)
		config, err := exsheetah.LoadConfig(path)
		if err != nil {
			t.Fatal(err)
		}
		loc, err := config.Location()
		if err != nil {
			t.Fatal(err)
		}
		if loc != time.UTC {
			t.Errorf("Location() = %v, want UTC", loc)
		}
	})

	t.Run("explicit timezone", func(t *testing.T) {
		t.Parallel()
		path := writeConfig(t, `
timezone: Asia/Tokyo
sheets:
  - name: item
    columns:
      - name: id
        type: number
`)
		config, err := exsheetah.LoadConfig(path)
		if err != nil {
			t.Fatal(err)
		}
		loc, err := config.Location()
		if err != nil {
			t.Fatal(err)
		}
		want, err := time.LoadLocation("Asia/Tokyo")
		if err != nil {
			t.Fatal(err)
		}
		if loc.String() != want.String() {
			t.Errorf("Location() = %v, want %v", loc, want)
		}
	})

	t.Run("invalid timezone is rejected at load time", func(t *testing.T) {
		t.Parallel()
		path := writeConfig(t, `
timezone: Not/A_Real_Zone
sheets:
  - name: item
    columns:
      - name: id
        type: number
`)
		if _, err := exsheetah.LoadConfig(path); err == nil {
			t.Fatal("expected an error for an invalid timezone")
		}
	})
}

func TestConfig_FilterSheets(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `
sheets:
  - name: weapon
    columns:
      - name: id
        type: number
  - name: item
    columns:
      - name: id
        type: number
  - name: armor
    columns:
      - name: id
        type: number
`)
	config, err := exsheetah.LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("no names means no filtering", func(t *testing.T) {
		t.Parallel()
		filtered, err := config.FilterSheets(nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(filtered) != 3 {
			t.Fatalf("expected 3 sheets, got %d", len(filtered))
		}
	})

	t.Run("filters down to the requested names, keeping config order", func(t *testing.T) {
		t.Parallel()
		filtered, err := config.FilterSheets([]string{"armor", "weapon"})
		if err != nil {
			t.Fatal(err)
		}
		if len(filtered) != 2 {
			t.Fatalf("expected 2 sheets, got %d", len(filtered))
		}
		if filtered[0].Name != "weapon" || filtered[1].Name != "armor" {
			t.Errorf("expected [weapon armor] in config order, got [%s %s]", filtered[0].Name, filtered[1].Name)
		}
	})

	t.Run("unknown name is an error", func(t *testing.T) {
		t.Parallel()
		if _, err := config.FilterSheets([]string{"weapon", "does-not-exist"}); err == nil {
			t.Fatal("expected an error for an unknown sheet name")
		}
	})
}
