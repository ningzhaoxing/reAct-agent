package schema

// Role represents the role of a message sender.
// It follows the UML enum: User, String, Assistant, Tool.
// Using iota for stable internal representation.
type Role int

const (
	RoleUser Role = iota
	RoleSystem
	RoleAssistant
	RoleTool
)

func (r Role) String() string {
	switch r {
	case RoleUser:
		return "user"
	case RoleSystem:
		return "system"
	case RoleAssistant:
		return "assistant"
	case RoleTool:
		return "tool"
	default:
		return "user"
	}
}

// Message models a chat message with a role and textual content.
type Message struct {
	Role    Role
	Content string
}
