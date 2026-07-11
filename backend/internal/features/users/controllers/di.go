package users_controllers

import (
	users_services "dockvol-backend/internal/features/users/services"
	cache_utils "dockvol-backend/internal/util/cache"
)

var userController = &UserController{
	users_services.GetUserService(),
	cache_utils.NewRateLimiter(),
}

var settingsController = &SettingsController{
	users_services.GetSettingsService(),
}

var managementController = &ManagementController{
	users_services.GetManagementService(),
}

func GetUserController() *UserController {
	return userController
}

func GetSettingsController() *SettingsController {
	return settingsController
}

func GetManagementController() *ManagementController {
	return managementController
}
