package rbac

type Role string

const (
	RoleAdministrator      Role = "administrator"
	RoleProgramCoordinator Role = "program_coordinator"
	RoleInstructor         Role = "instructor"
	RoleLearner            Role = "learner"
)

func AllRoles() []Role {
	return []Role{
		RoleAdministrator,
		RoleProgramCoordinator,
		RoleInstructor,
		RoleLearner,
	}
}

func IsKnownRole(role Role) bool {
	for _, candidate := range AllRoles() {
		if candidate == role {
			return true
		}
	}
	return false
}
