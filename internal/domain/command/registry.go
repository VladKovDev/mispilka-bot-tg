package command

// AllCommands contains all available bot commands
var AllCommands = CommandSlice{
	{Name: "start", Description: "Начать работу с ботом", Role: RolePublic},
	{Name: "users", Description: "Список пользователей", Role: RoleAdmin},
}

// GetPublicCommands returns all commands available to public users
func GetPublicCommands() CommandSlice {
	return AllCommands.ByRole(RolePublic)
}

// GetAdminCommands returns all admin-only commands
func GetAdminCommands() CommandSlice {
	return AllCommands.ByRole(RoleAdmin)
}
