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

	"golang.org/x/net/ipv4"
	_ "modernc.org/sqlite"

	"github.com/ulikunitz/xz"
)

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

func main() {

	uid := os.Getuid()
	if uid != 0 {
		fmt.Println("please, run packet as root")
		return
	}

	if len(os.Args) < 2 {
		fmt.Println("invalid syntax")
		return
	}

	cmd := os.Args[1]

	switch cmd {
	case "install":
		if len(os.Args) < 3 {
			fmt.Println("usage: packets install <name>")
			return
		}

		db, err := sql.Open("sqlite", "/opt/packets/packets/index.db")
		if err != nil {
			log.Fatal(err)
			return
		}
		defer db.Close()

		nameToQuery := os.Args[2]

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
		if len(os.Args) < 3 {
			fmt.Println("usage: packets serve <option>\navaiable options: init, stop")
			return
		}
		switch os.Args[2] {
		case "init":

			var sockets [2]string
			sockets[0] = "/opt/packets/packets/udpsocket"
			sockets[1] = "/opt/packets/packets/httpsocket"

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
			pidfiles[0] = "/opt/packets/packets/http.pid"
			pidfiles[1] = "/opt/packets/packets/udp.pid"

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

	fmt.Println("Reading manifest.json...")

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
	version := manifest.Version
	dependenc := manifest.Dependencies
	family := manifest.Family
	fmt.Printf("Installing %s...\n", name)

	var destDir = fmt.Sprintf("/opt/packets/%s", name)

	f, err := os.Open(packagepath)
	if err != nil {
		return err
	}

	defer f.Close()

	xzr, err := xz.NewReader(f)
	if err != nil {
		return err
	}

	tr := tar.NewReader(xzr)

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
			err = os.Chmod(absPath, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
		default:
			fmt.Printf("ignored type: %c\n", hdr.Typeflag)
		}

		fmt.Printf("\r\033[2KUnpacking: %s ", filepath.Base(rel))

	}

	wManifest, err := os.Open(fmt.Sprintf("%s/manifest.json", destDir))
	if err != nil {
		log.Println(err)
	}
	defer wManifest.Close()

	json_dcoder := json.NewDecoder(wManifest)

	var wManifestOPen Manifest

	if err := json_dcoder.Decode(&wManifestOPen); err != nil {
		log.Println(err)
	}

	wManifestOPen.Serial = serial

	jsonData, err := json.Marshal(wManifestOPen)
	if err != nil {
		log.Println(err)
	}

	wManifest.Write(jsonData)

	script := fmt.Sprintf("%s/postinstall.sh", destDir)

	fmt.Println("\nMaking post install configuration...")
	cmd := exec.Command(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error exec postinstall script %e", err)
	}

	fmt.Printf("Package %s fully installed you maybe run: \"source ~/.bashrc \"\n", name)

	var insert = Installed{
		Realname:     name,
		Version:      version,
		Dependencies: dependenc,
		Family:       family,
		Serial:       serial,
	}

	if err := AddToInstalledDB(insert); err != nil {
		return err
	}
	return nil
}

func GetPackageByMirror(mirror string, realname string) error {

	db, err := sql.Open("sqlite", "/opt/packets/packets/index.db")
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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err := os.Remove(fmt.Sprintf("/var/cache/packets/%s", filename))
		if os.IsNotExist(err) {
			return fmt.Errorf("failed to download, status code not 200OK")
		} else if err != nil {
			return err
		}
		return fmt.Errorf("failed to download, status code not 200OK")
	}

	if err := os.MkdirAll("/var/cache/packets", 0755); err != nil {
		log.Fatal("error creating file for package ", err)
		return err
	}

	out, err := os.Create(fmt.Sprintf("/var/cache/packets/%s", filename))
	if err != nil {
		log.Fatal("error creating package ", err)
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		err := os.Remove(fmt.Sprintf("/var/cache/packets/%s", filename))
		if err != nil {
			return err
		}
		return err
	}

	err = Validate(filename, realname)
	if err != nil {
		return err
	}

	if os.Args[1] == "upgrade" {
		if err := Upgrade(fmt.Sprintf("/var/cache/packets/%s", filename), os.Args[2]); err != nil {
			return err
		}
		return nil
	}
	err = Install(fmt.Sprintf("/var/cache/packets/%s", filename), serial)
	if err != nil {
		return err
	}
	return nil

}
func ResolvDependencies(realname string) {

	db, err := sql.Open("sqlite", "/opt/packets/packets/index.db")
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

	_, err := os.Stat("/opt/packets/packets/index.db")
	if os.IsNotExist(err) {
		fmt.Println("cant find index.db, please use sync first")
	}

	db, err := sql.Open("sqlite", "/opt/packets/packets/index.db")
	if err != nil {
		log.Fatal("cant find index.db, please use sync first")
	}
	defer db.Close()

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
				if err := Upgrade(fmt.Sprintf("/var/cache/packets/%s", filename), os.Args[2]); err != nil {
					log.Fatal(err)
					return
				}
				return
			}
			Install(fmt.Sprintf("/var/cache/packets/%s", filename), serial)
			return

		}
		fmt.Println("Asking in LAN for the package")
		peers := AskLAN(filename)
		answers := len(peers)
		if answers != 0 {
			for _, p := range peers {
				fmt.Printf("Downloading from %s\n", p.IP)
				if err := GetPackageByMirror(fmt.Sprintf("http://%s:%d/%s", p.IP, p.Port, filename), realname); err == nil {
					break
				}
				fmt.Printf("Download failed!\n")
			}
		}
		fmt.Printf("Downloading from %s\n", mirrors)
		if err := GetPackageByMirror(mirrors, realname); err == nil {
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
						if err := Upgrade(fmt.Sprintf("/var/cache/packets/%s", filename), os.Args[2]); err != nil {
							log.Fatal(err)
							return
						}
					}
					if os.Args[1] == "upgrade" {
						if err := Upgrade(fmt.Sprintf("/var/cache/packets/%s", filename), os.Args[2]); err != nil {
							log.Fatal(err)
							return
						}
						break
					}
					Install(fmt.Sprintf("/var/cache/packets/%s", filename), serial)
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
			if err := GetPackageByMirror(v, realname); err == nil {
				break
			}

		}
	}

}

func CheckDownloaded(filename string) bool {

	_, err := os.Stat(fmt.Sprintf("/var/cache/packets/%s", filename))
	if os.IsNotExist(err) {
		return false
	} else {
		return true
	}

}

func Validate(filename string, realname string) error {

	db, err := sql.Open("sqlite", "/opt/packets/packets/index.db")
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	downloaded, err := os.Open(fmt.Sprintf("/var/cache/packets/%s", filename))
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
		fmt.Println("fatal, index.db or package is corrupted")

		err := os.Remove(fmt.Sprintf("/var/cache/packets/%s", filename))
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
	_, err = os.Stat("/opt/packets/packets/index.db")

	if os.IsNotExist(err) {
		os.MkdirAll("/opt/packets/packets", 0755)
	}
	f, err := os.Create("/opt/packets/packets/index.db")
	if err != nil {
		return err
	}

	if _, err = io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func AddToInstalledDB(insert Installed) error {
	db, err := sql.Open("sqlite", "/opt/packets/packets/installed.db")
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS packages (realname TEXT NOT NULL UNIQUE PRIMARY KEY, version TEXT NOT NULL, dependencies TEXT, name TEXT)")
	if err != nil {
		return err
	}

	if len(insert.Dependencies) == 0 {
		_, err = db.Exec("INSERT INTO packages (realname, version) VALUES (?, ?)", insert.Realname, insert.Version)
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

	_, err = db.Exec("INSERT INTO packages (realname, version, dependencies) VALUES (?, ?, ?)", insert.Realname, insert.Version, query)
	if err != nil {
		return err
	}

	return nil
}

func Unninstall(realname string) error {
	db, err := sql.Open("sqlite", "/opt/packets/packets/installed.db")
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

	cmd := exec.Command(fmt.Sprintf("/opt/packets/%s/remove.sh", realname))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	if err := os.RemoveAll(fmt.Sprintf("/opt/packets/%s", realname)); err != nil {
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

	db, err := sql.Open("sqlite", "/opt/packets/packets/installed.db")
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

	db, err := sql.Open("sqlite", "/opt/packets/packets/installed.db")
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	rows, err := db.Query("SELECT realname FROM packages")
	if err != nil {
		return err
	}

	var realname string
	defer rows.Close()

	fmt.Println("Installed packages:")
	for rows.Next() {
		rows.Scan(&realname)
		fmt.Println(realname)
	}
	return nil
}

func Upgrade(packagepath string, og_realname string) error {

	db, err := sql.Open("sqlite", "/opt/packets/packets/installed.db")
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
	version := manifest.Version
	dependenc := manifest.Dependencies

	fmt.Printf("Upgrading %s to %s...\n", og_realname, name)

	var destDir = fmt.Sprintf("/opt/packets/%s", og_realname)

	f, err := os.Open(packagepath)
	if err != nil {
		return err
	}

	defer f.Close()

	xzr, err := xz.NewReader(f)
	if err != nil {
		return err
	}

	tr := tar.NewReader(xzr)

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
			err = os.Chmod(absPath, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
		default:
			fmt.Printf("\r\033[2Kignored type: %c", hdr.Typeflag)
		}

		fmt.Printf("\r\033[2KUnpacking: %s ", filepath.Base(rel))

	}

	os.Rename(destDir, fmt.Sprintf("/opt/packets/%s", name))
	destDir = fmt.Sprintf("/opt/packets/%s", name)

	script := fmt.Sprintf("%s/postinstall.sh", destDir)

	fmt.Println("\nMaking post install configuration...")
	cmd := exec.Command(script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error exec postinstall script %e", err)
	}

	fmt.Printf("Package %s fully installed you maybe run: \"source ~/.bashrc \"\n", name)

	var insert Installed

	insert.Realname = name
	insert.Version = version
	insert.Dependencies = dependenc

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

	db, err := sql.Open("sqlite", "/opt/packets/packets/index.db")
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	return nil
}
