//go:build linux

package main

import (
	"archive/tar"
	"bufio"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/net/ipv4"
	_ "modernc.org/sqlite"

	"github.com/ulikunitz/xz"
)

type ConfigTOML struct {
	Config struct {
		HttpPort           int    `toml:"httpPort"`
		CacheDir           string `toml:"cacheDir"`
		AutoDeleteCacheDir bool   `toml:"dutoDeleteCacheDir"`
		DaysToDelete       int    `toml:"daysToDelete"`
		DataDir            string `toml:"dataDir"`
	} `toml:"Config"`
}

type IndexTOML struct {
	Name        string    `toml:"name"`
	Version     string    `toml:"version"`
	Author      string    `toml:"author"`
	Description string    `toml:"description"`
	CreatedAt   time.Time `toml:"createdAt"`
}

type CountingReader struct {
	R     io.Reader
	Total int64
}

func (c *CountingReader) Read(p []byte) (int, error) {
	n, err := c.R.Read(p)
	c.Total += int64(n)
	return n, err
}

type Installed struct {
	Realname     string
	Version      string
	Dependencies []string
	Family       string
	Serial       uint
}

type Peer struct {
	IP   net.IP
	Port int
}

type Quer1 struct {
	Realname    string
	Version     string
	Description string
}

type Manifest struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	Author       string   `json:"author"`
	Family       string   `json:"family"`
	Serial       uint     `json:"serial"`
}

var serialPass uint
var cfg ConfigTOML
var PacketsDir string

func main() {

	out, _ := exec.Command("uname", "-s").Output()
	if uname := strings.TrimSpace(string(out)); uname == "OpenTTY" {
		PacketsDir = "/mnt"
	} else {
		PacketsDir = "/etc/packets"
	}

	_, err := os.Stat(filepath.Join(PacketsDir, "config.toml"))
	if err == os.ErrNotExist {
		fmt.Println("can't find config.toml, generating a blank one")

		cfg.Config.HttpPort = 9123
		cfg.Config.AutoDeleteCacheDir = false
		cfg.Config.CacheDir = "/var/cache/packets"
		cfg.Config.DataDir = "/opt/packets"
		cfg.Config.DaysToDelete = -1

		file, err := os.Create(filepath.Join(PacketsDir, "config.toml"))
		if err != nil {
			log.Fatal(err)
		}

		encoder := toml.NewEncoder(file)

		if err := encoder.Encode(cfg); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Operation Sucess!")

		file.Close()
	}

	_, err = toml.DecodeFile(filepath.Join(PacketsDir, "config.toml"), &cfg)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) < 2 {
		fmt.Println("invalid syntax")
		return
	}

	cmd := os.Args[1]

	switch cmd {
	case "install":
		if os.Getuid() != 0 {
			fmt.Println("please, run as root")
			return
		}

		if len(os.Args) < 3 {
			fmt.Println("usage: packets install <name>")
			return
		}

		db, err := sql.Open("sqlite", filepath.Join(PacketsDir, "index.db"))
		if err != nil {
			log.Fatal(err)
			return
		}
		defer db.Close()

		nameToQuery := os.Args[2]
		var exist bool
		db.QueryRow("SELECT EXISTS(SELECT 1 FROM packages WHERE realname = ?  LIMIT 1)", nameToQuery).Scan(&exist)
		if exist {
			QueryInstall(nameToQuery)
			return
		}

		rows, err := db.Query("SELECT realname, version, description FROM packages WHERE name = ?", nameToQuery)
		if err != nil {
			if strings.Contains(err.Error(), "file is not a database (26)") {
				fmt.Println("index.db corrupted")
				return
			}
			log.Panic(err)
			return
		}

		defer rows.Close()

		var pkgs []Quer1
		for rows.Next() {
			var q Quer1
			if err := rows.Scan(&q.Realname, &q.Version, &q.Description); err != nil {
				log.Fatal(err)
			}
			pkgs = append(pkgs, q)
		}
		switch len(pkgs) {
		case 0:
			fmt.Printf("can't find any results for %s\n", nameToQuery)
			return
		case 1:
			fmt.Printf("Founded 1 package for %s \n", nameToQuery)

			fmt.Printf("Downloading %s \n", pkgs[0].Realname)
			QueryInstall(pkgs[0].Realname)
			return

		default:
			fmt.Printf("Found %d versions of %s\n Select 1\n", len(pkgs), nameToQuery)
			for i, q := range pkgs {
				fmt.Printf("[%d] %s : %s\n %s\n", i, q.Realname, q.Version, q.Description)
			}
			var choice int

			fmt.Fscan(bufio.NewReader(os.Stdin), &choice)
			if choice > len(pkgs) || choice < 0 {
				fmt.Println("invalid option")
				return
			}

			QueryInstall(pkgs[choice].Realname)
			return
		}

	case "serve":
		if os.Getuid() != 0 {
			fmt.Println("please, run as root")
			return
		}

		if len(os.Args) < 3 {
			fmt.Println("usage: packets serve <option>\navaiable options: init, stop")
			return
		}
		switch os.Args[2] {
		case "init":

			var sockets [2]string
			sockets[0] = filepath.Join(PacketsDir, "udpsocket")
			sockets[1] = filepath.Join(PacketsDir, "httpsocket")

			for _, v := range sockets {
				abs, _ := filepath.Abs(v)
				cmd := exec.Command(abs)
				cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
				if err := cmd.Start(); err != nil {
					log.Fatalf("failed to start %s: %v", v, err)
				}

			}
			return

		case "stop":

			var pidfiles [2]string
			pidfiles[0] = filepath.Join(PacketsDir, "http.pid")
			pidfiles[1] = filepath.Join(PacketsDir, "udp.pid")

			for _, v := range pidfiles {
				data, err := os.ReadFile(v)
				if err != nil {
					fmt.Println("cant read PID:", err)
					return
				}
				pid, _ := strconv.Atoi(string(data))
				syscall.Kill(pid, syscall.SIGTERM)
			}
			return
		default:
			return
		}
	case "sync":
		if os.Getuid() != 0 {
			fmt.Println("please, run as root")
			return
		}

		if len(os.Args) < 3 {
			fmt.Println("Starting to sync with https://servidordomal.fun/mirror/index.db")
			if err := Sync("https://servidordomal.fun/mirror/index.db"); err != nil {
				fmt.Println("failed to sync with https://servidordomal.fun/mirror/index.db : ", err)
				return
			}
			fmt.Println("Sucessifully sync!")
			return
		}

		syncurl := os.Args[2]

		fmt.Printf("Starting to sync with %s\n", syncurl)
		if err := Sync(syncurl); err != nil {
			fmt.Printf("failed to sync with %s : %e ", syncurl, err)
			return
		}
		fmt.Println("Sucessifully sync!")
		return

	case "remove":
		if os.Getuid() != 0 {
			fmt.Println("please, run as root")
			return
		}

		if len(os.Args) < 3 {
			fmt.Println("usage: packets remove <package-name>")
			return
		}

		err := Unninstall(os.Args[2])
		if err != nil {
			log.Fatal(err)
			return
		}
		return
	case "list":

		if err := ListPackets(); err != nil {
			return
		}
		return

	case "upgrade":

		if os.Getuid() != 0 {
			fmt.Println("please, run as root")
			return
		}

		if len(os.Args) < 3 {
			fmt.Println("usage: packets upgrade <realname>")
			return
		}

		og_realname := os.Args[2]

		db, err := sql.Open("sqlite", filepath.Join(PacketsDir, "index.db"))
		if err != nil {
			log.Fatal(err)
			return
		}
		defer db.Close()

		idb, err := sql.Open("sqlite", filepath.Join(PacketsDir, "installed.db"))
		if err != nil {
			log.Fatal(err)
			return
		}
		defer idb.Close()

		var family string
		if err := idb.QueryRow("SELECT family FROM packages WHERE realname = ?", og_realname).Scan(&family); err != nil {
			log.Fatal("line 239", err)
			return
		}

		var neo_realname string

		if err := db.QueryRow("SELECT realname FROM packages WHERE family = ? ORDER BY serial DESC LIMIT 1", family).Scan(&neo_realname); err != nil {
			log.Fatal("line 245", err)
			return
		}

		if neo_realname == og_realname {
			fmt.Printf("%s is up to date!\n", og_realname)
			return
		}

		if err := db.QueryRow("SELECT serial FROM packages WHERE family = ? ORDER BY serial DESC LIMIT 1", family).Scan(&serialPass); err != nil {
			log.Fatal("line 255", err)
			return
		}

		fmt.Println("founded upgrade")
		QueryInstall(neo_realname)

	default:
		fmt.Printf(" %s it's not a command\n", cmd)
		return
	}
}

func ManifestReadXZ(path string) (*Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	xzr, err := xz.NewReader(f)
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(xzr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if strings.HasSuffix(hdr.Name, "/manifest.json") || hdr.Name == "manifest.json" {

			var manifest Manifest
			decoder := json.NewDecoder(tr)
			if err := decoder.Decode(&manifest); err != nil {
				return nil, err
			}
			return &manifest, nil
		}
	}
	return nil, fmt.Errorf("can't find manifest.json")
}

func Install(packagepath string, serial uint) error {

	manifest, err := ManifestReadXZ(packagepath)
	if err != nil {
		log.Panic(err)
	}

	name := manifest.Name

	var destDir = filepath.Join(cfg.Config.DataDir, name)

	f, err := os.Open(packagepath)
	if err != nil {
		return err
	}

	stat, _ := f.Stat()

	totalsize := stat.Size()

	defer f.Close()

	counter := &CountingReader{R: f}

	xzr, err := xz.NewReader(counter)
	if err != nil {
		return err
	}

	tr := tar.NewReader(xzr)

	bar := progressbar.NewOptions64(
		totalsize,
		progressbar.OptionSetDescription("[2/2] Installing ..."),
		progressbar.OptionSetWriter(os.Stdout),
	)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		rel := filepath.Clean(hdr.Name)

		if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			fmt.Println("Ignored :", rel)
			continue
		}

		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		absPath := filepath.Join(destDir, rel)

		switch hdr.Typeflag {

		case tar.TypeDir:
			err = os.MkdirAll(absPath, os.FileMode(hdr.Mode))

			if err != nil {
				return err
			}

		case tar.TypeReg:
			err = os.MkdirAll(filepath.Dir(absPath), 0755)
			if err != nil {
				return err
			}

			out, err := os.Create(absPath)
			if err != nil {
				return err
			}

			_, err = io.Copy(out, tr)
			out.Close()
			if err != nil {
				return err
			}

			bar.Set(int(counter.Total))

			err = os.Chmod(absPath, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
		default:

		}
	}

	bar.Finish()

	manifest.Serial = serial

	jsonData, err := json.Marshal(manifest)
	if err != nil {
		log.Println(err)
	}

	os.WriteFile(fmt.Sprintf("%s/manifest.json", destDir), jsonData, 0777)

	script := fmt.Sprintf("%s/postinstall.sh", destDir)

	os.Chmod(script, 0777)
	os.Chmod(fmt.Sprintf("%s/remove.sh", destDir), 0777)

	fmt.Println("\nMaking post install configuration...")
	cmd := exec.Command(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()

	if err != nil {
		log.Println(err)
		return fmt.Errorf("error exec postinstall script %e", err)
	}

	fmt.Printf("Package %s fully installed\n", name)

	var insert = Installed{
		Realname:     manifest.Name,
		Version:      manifest.Version,
		Dependencies: manifest.Dependencies,
		Family:       manifest.Family,
		Serial:       manifest.Serial,
	}

	if err := AddToInstalledDB(insert); err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func GetPackageByMirror(mirror string, realname string) error {

	db, err := sql.Open("sqlite", filepath.Join(PacketsDir, "index.db"))
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	var serial uint

	if err := db.QueryRow("SELECT serial FROM packages WHERE realname = ?", realname).Scan(&serial); err != nil {
		log.Fatal(err)
	}

	u, _ := url.Parse(mirror)
	filename := path.Base(u.Path)

	resp, err := http.Get(mirror)
	if err != nil {
		log.Panic("error doing get request, do you really have an internet connection?")
		return err
	}

	var domain = mirror
	var link bool

	if cont := strings.Contains(mirror, "https"); cont {
		link = true
		domain = strings.Replace(mirror, "https", "", 1)
	} else {
		link = true
		domain = strings.Replace(mirror, "http", "", 1)
	}
	if link {

		domain = strings.Replace(domain, "://", "", 1)
		slice := strings.SplitN(domain, "/", 2)

		domain = slice[0]
	}

	bar := progressbar.NewOptions64(resp.ContentLength,
		progressbar.OptionSetDescription(fmt.Sprintf("[1/2] Downloading from %s ...", domain)),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionShowCount(),
		progressbar.OptionClearOnFinish(),
	)

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err := os.Remove(filepath.Join(cfg.Config.CacheDir, filename))
		if os.IsNotExist(err) {
			return fmt.Errorf("failed to download, status code not 200OK")
		} else if err != nil {
			return err
		}
		return fmt.Errorf("failed to download, status code not 200OK")
	}

	if err := os.MkdirAll(cfg.Config.CacheDir, 0755); err != nil {
		log.Fatal("error creating file for package ", err)
		return err
	}

	out, err := os.Create(filepath.Join(cfg.Config.CacheDir, filename))
	if err != nil {
		log.Fatal("error creating package ", err)
		return err
	}
	defer out.Close()

	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	if err != nil {
		err := os.Remove(filepath.Join(cfg.Config.CacheDir, filename))
		if err != nil {
			return err
		}
		return err
	}
	bar.Finish()

	err = Validate(filename, realname)
	if err != nil {
		return err
	}

	if os.Args[1] == "upgrade" {
		if err := Upgrade(filepath.Join(cfg.Config.CacheDir, filename), os.Args[2], serialPass); err != nil {
			return err
		}
		return nil
	}
	err = Install(filepath.Join(cfg.Config.CacheDir, filename), serial)
	if err != nil {
		return err
	}
	return nil

}
func ResolvDependencies(realname string) {

	db, err := sql.Open("sqlite", filepath.Join(PacketsDir, "index.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var dependencies *string

	err = db.QueryRow("SELECT dependencies FROM packages WHERE realname = ?", realname).Scan(&dependencies)
	if err != nil {
		log.Panic(err)
		return
	}

	if dependencies == nil {
		return
	}

	dependencie := strings.Fields(*dependencies)

	for _, v := range dependencie {
		err := AlredySatisfied(v)
		if err != nil {
			fmt.Printf("error installing %v : %s", v, err.Error())
			continue
		}
		QueryInstall(v)
	}
}

func QueryInstall(realname string) {

	_, err := os.Stat(filepath.Join(PacketsDir, "index.db"))
	if os.IsNotExist(err) {
		fmt.Println("cant find index.db, please use sync first")
	}

	db, err := sql.Open("sqlite", filepath.Join(PacketsDir, "index.db"))
	if err != nil {
		log.Fatal("cant find index.db, please use sync first")
	}
	defer db.Close()

	simplecheck, err := sql.Open("sqlite", filepath.Join(PacketsDir, "installed.db"))
	if err == nil {

		var exist bool
		simplecheck.QueryRow("SELECT EXISTS(SELECT 1 FROM packages WHERE realname = ?  LIMIT 1)", realname).Scan(&exist)

		if exist {
			fmt.Println("Alredy installed!")
			simplecheck.Close()
			return
		}

		simplecheck.Close()
	}

	var mirrors string

	ResolvDependencies(realname)

	err = db.QueryRow("SELECT mirrors FROM packages WHERE realname = ?", realname).Scan(&mirrors)
	if err != nil {
		log.Panic(err)
		return
	}
	var serial uint
	err = db.QueryRow("SELECT serial FROM packages WHERE realname = ?", realname).Scan(&serial)
	if err != nil {
		log.Panic(err)
		return
	}

	if !strings.Contains(mirrors, " ") {
		u, _ := url.Parse(mirrors)
		filename := path.Base(u.Path)

		fmt.Println("Checking if the package exists")
		if CheckDownloaded(filename) {
			err := Validate(filename, realname)
			if err != nil {
				return
			}
			if os.Args[1] == "upgrade" {
				if err := Upgrade(filepath.Join(cfg.Config.CacheDir, filename), os.Args[2], serialPass); err != nil {
					log.Fatal(err)
					return
				}
				return
			}
			Install(filepath.Join(cfg.Config.CacheDir, filename), serial)
			return

		}
		fmt.Println("Asking in LAN for the package")
		peers := AskLAN(filename)
		answers := len(peers)
		if answers != 0 {
			for _, p := range peers {
				fmt.Printf("Downloading from %s\n", p.IP)
				if err := GetPackageByMirror(fmt.Sprintf("http://%s:%d/%s", p.IP, p.Port, filename), realname); err != nil {
					log.Println(err)
					break
				}
				fmt.Printf("Download failed!\n")
			}
		}
		if err := GetPackageByMirror(mirrors, realname); err != nil {
			log.Println(err)
			return
		}

	} else {

		fmt.Println("A mirror list was found")
		mirrorlist := strings.Fields(mirrors)

		for _, v := range mirrorlist {
			u, _ := url.Parse(v)
			filename := path.Base(u.Path)

			fmt.Printf("Checking for %s", filename)
			if CheckDownloaded(filename) {
				err := Validate(filename, realname)
				if err != nil {
					continue
				} else {
					if os.Args[1] == "upgrade" {
						if err := Upgrade(filepath.Join(cfg.Config.CacheDir, filename), os.Args[2], serialPass); err != nil {
							log.Fatal(err)
							return
						}
						break
					}
					Install(filepath.Join(cfg.Config.CacheDir, filename), serial)
					break
				}
			}
			fmt.Println("Checking for package in LAN")
			peers := AskLAN(filename)
			answers := len(peers)
			if answers != 0 {
				for _, p := range peers {
					fmt.Printf("Downloading from %s\n", v)
					if err := GetPackageByMirror(fmt.Sprintf("http://%s:%d/%s", p.IP, p.Port, filename), realname); err == nil {
						break
					}
					fmt.Printf("Failed!\n")
				}
			}
			fmt.Printf("Downloading from %s\n", v)
			if err := GetPackageByMirror(v, realname); err != nil {
				log.Println(err)
				break
			}

		}
	}

}

func CheckDownloaded(filename string) bool {

	_, err := os.Stat(filepath.Join(cfg.Config.CacheDir, filename))
	if os.IsNotExist(err) {
		return false
	} else {
		return true
	}

}

func Validate(filename string, realname string) error {

	db, err := sql.Open("sqlite", filepath.Join(PacketsDir, "index.db"))
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	downloaded, err := os.Open(filepath.Join(cfg.Config.CacheDir, filename))
	if err != nil {
		log.Fatal("error reading new file")
		return err
	}
	defer downloaded.Close()

	h := sha256.New()
	if _, err := io.Copy(h, downloaded); err != nil {
		fmt.Println("error doing sha256sum")
		return err
	}

	sum := h.Sum(nil)
	hashString := hex.EncodeToString(sum)

	var hashStringDB string

	err = db.QueryRow("SELECT hash FROM packages WHERE realname = ?", realname).Scan(&hashStringDB)
	if err != nil {
		log.Panic(err)
		return err
	}

	if hashString != hashStringDB {
		fmt.Println("tampered package, removing it...\nplease run the command again")

		err := os.Remove(filepath.Join(cfg.Config.CacheDir, filename))
		if err != nil {
			return err
		}
		return fmt.Errorf("the package isn't safe, alredy removed")
	}
	return nil
}

func AskLAN(filename string) []Peer {
	var peers []Peer
	query := []byte("Q:" + filename)

	pc, err := net.ListenPacket("udp", ":0")
	if err != nil {
		log.Panic("error starting udp socket:", err)
	}
	defer pc.Close()

	if pconn := ipv4.NewPacketConn(pc); pconn != nil {
		_ = pconn.SetTTL(1)
	}

	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil {
				continue
			}

			bcast := broadcastAddr(ipnet.IP.To4(), ipnet.Mask)
			dst := &net.UDPAddr{IP: bcast, Port: 1333}

			_, err = pc.WriteTo(query, dst)
			if err != nil {
				log.Printf("[%s] can't send to  %s: %v", ifc.Name, bcast, err)
			}
		}
	}
	_ = pc.SetDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1500)

	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			break
		}
		msg := string(buf[:n])

		if strings.HasPrefix(msg, "H:"+filename) {
			parts := strings.Split(msg, ":")
			port, _ := strconv.Atoi(parts[2])
			fmt.Printf("%s have the package\n", addr)
			peers = append(peers, Peer{IP: addr.(*net.UDPAddr).IP, Port: port})
		}
	}
	return peers
}

func broadcastAddr(ip net.IP, mask net.IPMask) net.IP {
	b := make(net.IP, len(ip))
	for i := range ip {
		b[i] = ip[i] | ^mask[i]
	}
	return b
}

func Sync(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = os.Stat(filepath.Join(PacketsDir, "index.db"))

	if os.IsNotExist(err) {
		os.MkdirAll("/etc/packets", 0755)
	}
	f, err := os.Create(filepath.Join(PacketsDir, "index.db"))
	if err != nil {
		return err
	}

	if _, err = io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func AddToInstalledDB(insert Installed) error {
	db, err := sql.Open("sqlite", filepath.Join(PacketsDir, "installed.db"))
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS packages (realname TEXT NOT NULL UNIQUE PRIMARY KEY, version TEXT NOT NULL, dependencies TEXT, name TEXT, family TEXT NOT NULL, serial INTEGER)")
	if err != nil {
		return err
	}

	if len(insert.Dependencies) == 0 {
		_, err = db.Exec("INSERT INTO packages (realname, version, family, serial) VALUES (?, ?, ?, ?)", insert.Realname, insert.Version, insert.Family, insert.Serial)
		if err != nil {
			return err
		}
		return nil
	}
	var query string
	for _, v := range insert.Dependencies {

		query = query + v + " "
	}

	query = query[:len(query)-1]

	_, err = db.Exec("INSERT INTO packages (realname, version, dependencies, family, serial) VALUES (?, ?, ?, ?, ?)", insert.Realname, insert.Version, query, insert.Family, insert.Serial)
	if err != nil {
		return err
	}

	return nil
}

func Unninstall(realname string) error {
	db, err := sql.Open("sqlite", filepath.Join(PacketsDir, "installed.db"))
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	var exist bool

	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM packages WHERE realname = ?  LIMIT 1)", realname).Scan(&exist)
	if err != nil {
		return err
	}

	if !exist {
		return fmt.Errorf("this package isn't installed")
	}
	fmt.Printf("Sure you will remove %s ? y/n ", realname)
	var answer string
	fmt.Scanf("%s", &answer)

	if answer != "y" && answer != "Y" {
		return fmt.Errorf("operation cancelled")
	}

	cmd := exec.Command(filepath.Join(cfg.Config.DataDir, realname, "remove.sh"))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	if err := os.RemoveAll(filepath.Join(cfg.Config.DataDir, realname)); err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM packages WHERE realname =  ?", realname)
	if err != nil {
		return err
	}

	fmt.Println("Sucessifully removed")
	return nil
}

func AlredySatisfied(realname string) error {

	db, err := sql.Open("sqlite", filepath.Join(PacketsDir, "installed.db"))
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	var exist bool

	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM packages WHERE realname = ?  LIMIT 1)", realname).Scan(&exist)
	if err != nil {
		return err
	}

	if !exist {
		return nil
	}
	return fmt.Errorf("conflict")
}

func ListPackets() error {

	db, err := sql.Open("sqlite", filepath.Join(PacketsDir, "installed.db"))
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	rows, err := db.Query("SELECT realname, version FROM packages")
	if err != nil {
		return err
	}

	var realname string
	var version string
	defer rows.Close()

	fmt.Println("Installed packages:")
	for rows.Next() {
		rows.Scan(&realname, &version)
		fmt.Printf("%s	%s\n", realname, version)
	}
	return nil
}

func Upgrade(packagepath string, og_realname string, serial uint) error {

	db, err := sql.Open("sqlite", filepath.Join(PacketsDir, "installed.db"))
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	var exist bool

	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM packages WHERE realname = ?  LIMIT 1)", og_realname).Scan(&exist)
	if err != nil {
		return err
	}

	if !exist {
		return fmt.Errorf("this package isn't installed")
	}

	manifest, err := ManifestReadXZ(packagepath)
	if err != nil {
		log.Panic(err)
	}

	name := manifest.Name

	fmt.Printf("Unpacking (%s) above (%s)\n", og_realname, name)

	var destDir = filepath.Join(cfg.Config.DataDir, og_realname)

	f, err := os.Open(packagepath)
	if err != nil {
		return err
	}
	stats, _ := f.Stat()
	totalSize := stats.Size()
	defer f.Close()

	counter := &CountingReader{R: f}

	xzr, err := xz.NewReader(counter)
	if err != nil {
		return err
	}

	tr := tar.NewReader(xzr)

	bar := progressbar.NewOptions64(
		totalSize,
		progressbar.OptionSetDescription("[2/2] Upgrading ..."),
		progressbar.OptionSetWriter(os.Stdout),
	)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		rel := filepath.Clean(hdr.Name)

		if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			fmt.Println("Ignored :", rel)
			continue
		}

		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		absPath := filepath.Join(destDir, rel)

		switch hdr.Typeflag {

		case tar.TypeDir:
			err = os.MkdirAll(absPath, os.FileMode(hdr.Mode))

			if err != nil {
				return err
			}

		case tar.TypeReg:
			err = os.MkdirAll(filepath.Dir(absPath), 0755)
			if err != nil {
				return err
			}

			out, err := os.Create(absPath)
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			out.Close()
			if err != nil {
				return err
			}

			bar.Set(int(counter.Total))

			err = os.Chmod(absPath, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
		default:

		}

	}
	bar.Finish()

	os.Rename(destDir, filepath.Join(cfg.Config.DataDir, name))
	destDir = filepath.Join(cfg.Config.DataDir, name)

	manifest.Serial = serial

	jsonData, err := json.Marshal(manifest)
	if err != nil {
		log.Println(err)
	}

	os.WriteFile(filepath.Join(destDir, "manifest.json"), jsonData, 0777)

	script := fmt.Sprintf(filepath.Join(destDir, "postinstall.sh"), destDir)

	os.Chmod(script, 0777)
	os.Chmod(fmt.Sprintf(filepath.Join(destDir, "remove.sh"), destDir), 0777)

	fmt.Println("Making post install configuration...")
	cmd := exec.Command(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error exec postinstall script %e", err)
	}

	fmt.Printf("Package %s fully installed you maybe run: \"source ~/.bashrc \"\n", name)

	var insert = Installed{
		Realname:     manifest.Name,
		Version:      manifest.Version,
		Dependencies: manifest.Dependencies,
		Family:       manifest.Family,
		Serial:       manifest.Serial,
	}

	_, err = db.Exec("DELETE FROM packages WHERE realname =  ?", og_realname)
	if err != nil {
		return err
	}

	if err := AddToInstalledDB(insert); err != nil {
		return err
	}
	return nil
}

func SearchUpgrades(name string) error {

	db, err := sql.Open("sqlite", filepath.Join(PacketsDir, "index.db"))
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	return nil
}
