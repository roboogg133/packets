package main

const (
	PacketsVersion = "0.1.0"
	PacketsSerial  = 0
)

const (
	ConfigurationDir       = "/etc/packets"
	InternalDB             = ConfigurationDir + "/internal.db"
	SourceDB               = ConfigurationDir + "/source.db"
	PacketsUsername        = "packets"
	HomeDir                = "/var/lib/packets"
	PackageRootDir         = "/var/lib/packets/packages"
	PackageBuildDepsFS     = "/var/lib/packets/buildPackages"
	LockFileName           = "packet.lock"
	NumberOfTryAttempts    = 4
	UserHomeDirPlaceholder = "{{ USER HOME FOLDER }}"
	UsernamePlaceholder    = "{{ USERNAME }}"
)

const (
	PrefixForLocations = "https://"
	PrefixForPackages  = "pkg"
)
