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
