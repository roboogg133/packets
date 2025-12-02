package main

const (
	ConfigurationDir       = "/etc/packets"
	InternalDB             = ConfigurationDir + "/internal.db"
	SourceDB               = ConfigurationDir + "/source.db"
	PacketsUsername        = "packets"
	HomeDir                = "/var/lib/packets"
	PackageRootDir         = "/var/lib/packets/packages"
	PackageBuildDepsFS     = "/var/lib/packets/buildPackages"
	LockFileName           = "Packet.lock"
	NumberOfTryAttempts    = 4
	UserHomeDirPlaceholder = "{{ USER HOME FOLDER }}"
	UsernamePlaceholder    = "{{ USERNAME }}"
)
