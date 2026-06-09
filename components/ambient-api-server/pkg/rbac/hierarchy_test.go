package rbac

import "testing"

func TestCanGrant(t *testing.T) {
	tests := []struct {
		name        string
		callerLevel int
		targetRole  string
		want        bool
	}{
		{"admin can grant admin", 0, RolePlatformAdmin, true},
		{"admin can grant project:owner", 0, RoleProjectOwner, true},
		{"admin can grant project:viewer", 0, RoleProjectViewer, true},
		{"project:owner can grant editor", 1, RoleProjectEditor, true},
		{"project:owner can grant viewer", 1, RoleProjectViewer, true},
		{"project:owner cannot grant project:owner", 1, RoleProjectOwner, false},
		{"project:editor cannot grant project:owner", 2, RoleProjectOwner, false},
		{"project:editor cannot grant project:editor", 2, RoleProjectEditor, false},
		{"project:editor can grant viewer", 2, RoleProjectViewer, true},
		{"project:viewer cannot grant anything", 3, RoleProjectViewer, false},
		{"unknown role rejected", 0, "nonexistent:role", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanGrant(tt.callerLevel, tt.targetRole)
			if got != tt.want {
				t.Errorf("CanGrant(%d, %q) = %v, want %v", tt.callerLevel, tt.targetRole, got, tt.want)
			}
		})
	}
}

func TestHighestLevel(t *testing.T) {
	tests := []struct {
		name  string
		roles []string
		want  int
	}{
		{"admin", []string{RolePlatformAdmin}, 0},
		{"project:owner", []string{RoleProjectOwner}, 1},
		{"multiple roles uses best", []string{RoleProjectViewer, RoleProjectOwner}, 1},
		{"no roles", nil, 999},
		{"unknown roles", []string{"nonexistent"}, 999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HighestLevel(tt.roles)
			if got != tt.want {
				t.Errorf("HighestLevel(%v) = %d, want %d", tt.roles, got, tt.want)
			}
		})
	}
}

func TestInternalRoles(t *testing.T) {
	if !InternalRoles[RoleAgentRunner] {
		t.Error("agent:runner should be internal")
	}
	if !InternalRoles[RoleCredentialTokenReader] {
		t.Error("credential:token-reader should be internal")
	}
	if InternalRoles[RoleProjectOwner] {
		t.Error("project:owner should not be internal")
	}
}
