package command

// Role represents the access level for a command
type Role string

const (
	RolePublic Role = "public"
	RoleAdmin  Role = "admin"
)

// Command represents a Telegram bot slash command with its metadata
type Command struct {
	Name        string
	Description string
	Role        Role
}

// CommandSlice is a collection of commands with filtering capabilities
type CommandSlice []Command

// ByRole filters commands by the specified role
func (cs CommandSlice) ByRole(role Role) CommandSlice {
	var filtered CommandSlice
	for _, cmd := range cs {
		if cmd.Role == role {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}
