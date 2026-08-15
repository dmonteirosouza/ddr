package app

import "testing"

func TestSelectedCleanTargetsSafe(t *testing.T) {
	flags := map[string]bool{"--safe": true}

	got := cleanTargetFlags(selectedCleanTargets(flags))
	want := []string{"--docker-build", "--npm", "--gradle"}

	assertStringSlice(t, got, want)
}

func TestSelectedCleanTargetsAllSafe(t *testing.T) {
	flags := map[string]bool{"--all-safe": true}

	got := cleanTargetFlags(selectedCleanTargets(flags))
	want := []string{"--docker-build", "--docker-system", "--npm", "--gradle"}

	assertStringSlice(t, got, want)
}

func TestSelectedCleanTargetsIndividualFlags(t *testing.T) {
	flags := map[string]bool{
		"--npm":      true,
		"--gradle":   true,
		"--go-build": true,
	}

	got := cleanTargetFlags(selectedCleanTargets(flags))
	want := []string{"--npm", "--gradle", "--go-build"}

	assertStringSlice(t, got, want)
}

func TestSelectedCleanTargetsAllCaches(t *testing.T) {
	flags := map[string]bool{"--all-caches": true}

	got := cleanTargetFlags(selectedCleanTargets(flags))
	want := []string{
		"--docker-build",
		"--npm",
		"--yarn",
		"--pnpm",
		"--gradle",
		"--go-build",
		"--go-mod",
		"--pip",
		"--pub",
		"--cocoapods",
		"--xcode-derived-data",
	}

	assertStringSlice(t, got, want)
}

func cleanTargetFlags(targets []cleanTarget) []string {
	flags := make([]string, 0, len(targets))
	for _, target := range targets {
		flags = append(flags, target.flag)
	}
	return flags
}

func assertStringSlice(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
