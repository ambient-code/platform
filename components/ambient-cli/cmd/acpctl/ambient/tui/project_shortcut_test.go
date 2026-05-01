package tui

import "testing"

// TestProjectShortcutHandledViews ensures every view reachable by number-key
// project switching has an explicit case in handleProjectShortcut. If a new
// view is added without handling it, this test fails — preventing silent
// fallthrough to the agents view.
func TestProjectShortcutHandledViews(t *testing.T) {
	// All views that exist in the TUI.
	allViews := []string{
		"projects",
		"agents",
		"sessions",
		"scheduledsessions",
		"inbox",
		"messages",
		"detail",
		"contexts",
		"help",
	}

	for _, v := range allViews {
		if numberKeyExcludedViews[v] {
			continue
		}
		if !projectShortcutHandledViews[v] {
			t.Errorf("view %q is reachable by number-key project switching but has no explicit case in handleProjectShortcut — add it to projectShortcutHandledViews and handle it in the switch", v)
		}
	}
}

// TestNumberKeyExcludedAndHandledAreDisjoint verifies the two sets don't
// overlap, which would indicate a misconfiguration.
func TestNumberKeyExcludedAndHandledAreDisjoint(t *testing.T) {
	for v := range numberKeyExcludedViews {
		if projectShortcutHandledViews[v] {
			t.Errorf("view %q appears in both numberKeyExcludedViews and projectShortcutHandledViews", v)
		}
	}
}
