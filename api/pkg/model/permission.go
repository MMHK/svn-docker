package model

// Permission is an authz access level for a principal on an access path.
type Permission string

const (
	PermissionNone      Permission = "none"
	PermissionRead      Permission = "r"
	PermissionReadWrite Permission = "rw"
)

// Valid reports whether p is a supported access level.
func (p Permission) Valid() bool {
	switch p {
	case PermissionNone, PermissionRead, PermissionReadWrite:
		return true
	default:
		return false
	}
}

// ParsePermission validates a user supplied access level.
func ParsePermission(raw string) (Permission, error) {
	p := Permission(raw)
	if !p.Valid() {
		return "", &ValidationError{
			Field:  "permission",
			Reason: "must be one of none, r, rw",
		}
	}

	return p, nil
}
