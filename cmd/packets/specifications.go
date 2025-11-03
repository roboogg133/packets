package main

const (
	ConfigurationDir       = "/etc/packets"
	InternalDB             = ConfigurationDir + "/internal.db"
	HomeDir                = "/var/lib/packets"
	PackageRootDir         = "/var/lib/packets/packages"
	NumberOfTryAttempts    = 4
	UserHomeDirPlaceholder = "{{ USER HOME FOLDER }}"
	UsernamePlaceholder    = "{{ USERNAME }}"
)
