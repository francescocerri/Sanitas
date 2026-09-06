package user

// Known permissions: a fixed vocabulary tied to real endpoints across the
// two services, not per-committee data (unlike roles — see docs/adr/0012).
// Which role gets which of these is the per-committee part (roles.json),
// not the permission names themselves — see docs/adr/0018.
const (
	PermUsersManage = "users:manage"
	PermShiftsRead  = "shifts:read"
	PermShiftsWrite = "shifts:write"
)

// AllPermissions grants everything — used only to seed the bootstrap admin's
// technical role (see Bootstrap), never assigned to an organizational role
// from config.
var AllPermissions = []string{PermUsersManage, PermShiftsRead, PermShiftsWrite}

var knownPermissions = map[string]bool{
	PermUsersManage: true,
	PermShiftsRead:  true,
	PermShiftsWrite: true,
}

func isKnownPermission(slug string) bool {
	return knownPermissions[slug]
}
