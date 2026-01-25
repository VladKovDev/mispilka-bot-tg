package command

// AllCommands contains all available bot commands
var AllCommands = CommandSlice{
	{Name: "start", Description: "Начать работу с ботом", Role: RolePublic},
	{Name: "restart", Description: "Перезапустить бота", Role: RolePublic},
	{Name: "users", Description: "Список пользователей", Role: RoleAdmin},
	{Name: "scenarios", Description: "Show all scenarios", Role: RoleAdmin},
	{Name: "create_scenario", Description: "Create a new scenario", Role: RoleAdmin},
	{Name: "set_default_scenario", Description: "Set default scenario", Role: RoleAdmin},
	{Name: "delete_scenario", Description: "Delete a scenario", Role: RoleAdmin},
	{Name: "demo_scenario", Description: "Demonstrate a scenario", Role: RoleAdmin},
	{Name: "create_broadcast", Description: "Create a broadcast", Role: RoleAdmin},
	{Name: "send_broadcast", Description: "Send a broadcast", Role: RoleAdmin},
}

// GetPublicCommands returns all commands available to public users
func GetPublicCommands() CommandSlice {
	return AllCommands.ByRole(RolePublic)
}

// GetAdminCommands returns all admin-only commands
func GetAdminCommands() CommandSlice {
	return AllCommands.ByRole(RoleAdmin)
}
