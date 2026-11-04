package httpapi

// P3 write-action names for the stderr operation log (plan T2.5). As with the
// P2 names these carry only the actor, the target and the result.
const (
	actionGroupCreate       = "group.create"
	actionGroupDelete       = "group.delete"
	actionGroupMembers      = "group.members"
	actionAccessPathCreate  = "access-path.create"
	actionAccessPathDelete  = "access-path.delete"
	actionAccessPathACL     = "access-path.acl"
	actionAccessPathManager = "access-path.managers"
)
