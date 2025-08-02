# 📦 Packets – Custom Package Manager for Linux

> A fast and minimal package manager written in Go with Lua hooks, local network discovery, and SQLite-based indexing.

---

## 📘 Overview

**Packets** is a lightweight package manager for Linux, written in Go. It supports:

- Installation and removal of packages
- Dependency resolution and upgrading
- `.tar.zst` compressed packages with `manifest.toml` metadata
- Lua-based install/remove hooks
- Local cache with SHA-256 validation
- Peer-to-peer discovery over LAN
- Remote package syncing via HTTP
- SQLite-based local database

---

## 📁 Directory Structure

| Path                  | Description                      |
|-----------------------|----------------------------------|
| `/etc/packets/`       | Configuration files              |
| `/opt/packets/`       | Installed package data           |
| `/var/cache/packets/` | Cached `.tar.zst` package files  |

(This can be changed in `/etc/packets/config.toml`)

---

# Available Commands

| Command                   | Description                                                                |
|---------------------------|----------------------------------------------------------------------------|
|`packets install <name>`	|    Install a package (resolves dependencies, executes Lua install hook)    |
|`packets remove <name>`	|    Remove a package (executes Lua remove hook)                             |
|`packets upgrade <name>`	|    Upgrade a package by checking family and serial in the manifest         |
|`packets sync [url]`	    |    Synchronize index.db from remote HTTP source                            |
|`packets serve init/stop`  |    Starts and stop the LAN service daemon                                  |
|`packets list`	            |    List all installed packages                                             |

# 📦 Package Format

Packages must be compressed as .tar.zst and include:


- ├── manifest.toml       # Package metadata
- ├── data/               # Files to install
- ├── install.lua         # Lua install hook
- └── remove.lua          # Lua remove hook


## Example manifest.toml
``[Info]
name = "packets"
version = "1.0.0"
description = "offline and online packetmanager"
dependencies = []
author = "robo"
family = "1f84ca15-5077-4f1d-a370-0ec860766eb2"
serial = 0

[Hooks]
install = "install.lua"
remove = "remove.lua"``

--
# 🔄 Installation Process

    Check if package is already cached and validated via SHA-256.

    If not, search the package:

        Via LAN: Sends UDP broadcast (Q:filename) to peers.

        Via HTTP: Downloads from configured mirrors.

    Decompress .tar.zst, install files.

    Execute Lua install hook.

# 🧩 Core Features
✅ Dependency Resolution

Installs required dependencies listed in the manifest.
## 🌐 LAN Discovery

Broadcasts package request to devices in the same network via UDP.
## 📡 Remote Download

Downloads package via HTTP if not found on LAN.
## 🔒 Security

    SHA-256 checksum validation

    Path validation to avoid exploits (..)

    Safe, sandboxed Lua runtime with limited API

## 🛠️ Allowed Lua API (install/remove hooks)

To ensure security, only a limited set of safe functions are exposed in Lua hooks:

os.remove(path)
os.rename(old, new)
os.copy(source, target)
os.symlink(source, target)
io.open(path, mode)
path_join(...)  -- Safely join path segments

### Note: Dangerous functions like os.execute, os.getenv, etc. are removed.
## 🗃️ Databases

    index.db: Available packages (after sync)

    installed.db: Packages currently installed

#⚠️ Restrictions & Notes

    Linux only (//go:build linux)

    Root permissions required for most commands

    Changing dataDir triggers prompt to migrate installed packages

    Binaries in binDir are not automatically moved if path changes

    Do not manually edit lastDataDir

