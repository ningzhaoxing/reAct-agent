package schema

// Role represents the role of a message sender.
// It follows the UML enum: User, String, Assistant, Tool.
// Using iota for stable internal representation.
type Role int

const (
	RoleUser Role = iota
	RoleString
	RoleAssistant
	RoleTool
)

func (r Role) String() string {
	return [...]string{"User", "String", "Assistant", "Tool"}[r]
}

// Message models a chat message with a role and textual content.
type Message struct {
	Role    Role
	Content string
}
