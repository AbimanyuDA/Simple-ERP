package domain

const (
	RoleStaff      = "STAFF"
	RoleSupervisor = "SUPERVISOR"
	RoleSuperAdmin = "SUPER_ADMIN"
)

func ValidRole(role string) bool {
	return role == RoleStaff || role == RoleSupervisor || role == RoleSuperAdmin
}

func CanCreate(role string) bool {
	return role == RoleStaff || role == RoleSuperAdmin
}

func CanApprove(role string) bool {
	return role == RoleSupervisor || role == RoleSuperAdmin
}

func CanManageUsers(role string) bool { return role == RoleSuperAdmin }
