package user

// Known permissions: a fixed vocabulary tied to real endpoints across the
// two services, not per-committee data (unlike roles — see docs/adr/0012).
// Which role gets which of these is the per-committee part (roles.json),
// not the permission names themselves — see docs/adr/0018.
const (
	PermUsersManage = "users:manage"
	PermShiftsRead  = "shifts:read"
	PermShiftsWrite = "shifts:write"
	// PermShiftsConfigure decide giorni/orari dei turni-template (ricorrenti)
	// — distinto da PermShiftsWrite, che resta per approvare/rifiutare
	// prenotazioni e prenotare direttamente per un volontario. Vedi
	// docs/adr/0025-modello-dati-turni.md.
	PermShiftsConfigure = "shifts:configure"
	// PermShiftsRequest permette di richiedere una prenotazione su uno slot
	// libero — assegnato solo al ruolo "volontario di emergenza"
	// (config/pavullo/registry/roles.json). Un permesso dedicato, non un
	// controllo su claims.Roles: stesso meccanismo usato per ogni altra
	// autorizzazione in questo sistema.
	PermShiftsRequest = "shifts:request"
)

// AllPermissions grants everything — used only to seed the bootstrap admin's
// technical role (see Bootstrap), never assigned to an organizational role
// from config.
var AllPermissions = []string{PermUsersManage, PermShiftsRead, PermShiftsWrite, PermShiftsConfigure, PermShiftsRequest}

var knownPermissions = map[string]bool{
	PermUsersManage:     true,
	PermShiftsRead:      true,
	PermShiftsWrite:     true,
	PermShiftsConfigure: true,
	PermShiftsRequest:   true,
}

func isKnownPermission(slug string) bool {
	return knownPermissions[slug]
}
