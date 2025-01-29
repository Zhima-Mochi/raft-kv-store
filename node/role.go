package node

import "context"

type RoleState interface {
	run(ctx context.Context)
	handleEvent(ctx context.Context, e *Event) error
	onExit()
	getRole() Role
}

// roleString is a small helper to safely get the role's name
func roleString(rs RoleState) string {
	if rs == nil {
		return "nil"
	}
	switch rs.getRole() {
	case FollowerRole:
		return "Follower"
	case CandidateRole:
		return "Candidate"
	case LeaderRole:
		return "Leader"
	default:
		return "Unknown"
	}
}
