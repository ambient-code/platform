package rbac

var RoleLevel = map[string]int{
	RolePlatformAdmin: 0,

	RoleProjectOwner:    1,
	RoleCredentialOwner: 1,

	RolePlatformViewer:        2,
	RoleProjectEditor:         2,
	RoleAgentOperator:         2,
	"agent:editor":            2,
	RoleCredentialReader:      2,
	RoleCredentialTokenReader: 2,
	"credential:viewer":       2,

	RoleProjectViewer: 3,
	RoleAgentObserver: 3,
}

var InternalRoles = map[string]bool{
	RoleAgentRunner:           true,
	RoleCredentialTokenReader: true,
}

// CanGrant returns true if callerLevel can grant targetRole.
// platform:admin (level 0) can grant at own level (sole exception).
// All others must grant strictly below.
func CanGrant(callerLevel int, targetRoleName string) bool {
	targetLevel, ok := RoleLevel[targetRoleName]
	if !ok {
		return false
	}
	if callerLevel == 0 {
		return true
	}
	return callerLevel < targetLevel
}

// HighestLevel returns the most privileged level across all role names.
// Lower number = higher privilege. Returns 999 if no roles match.
func HighestLevel(roleNames []string) int {
	best := 999
	for _, name := range roleNames {
		if level, ok := RoleLevel[name]; ok && level < best {
			best = level
		}
	}
	return best
}
