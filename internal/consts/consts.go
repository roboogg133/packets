package consts

import "time"

const (
	DefaultLinux_d  = "/etc/packets"
	DefaultCache_d  = "/var/cache/packets"
	DefaultHttpPort = 9123
	DefaultData_d   = "/opt/packets"
	LANDeadline     = 2 * time.Second
	IndexDB         = "/etc/packets/index.db"
	InstalledDB     = "/etc/packets/installed.db"
	DefaultSyncUrl  = "https://servidordomal.fun/index.db"
)
