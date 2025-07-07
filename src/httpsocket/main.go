package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	pid := os.Getpid()
	if err := os.WriteFile("/opt/packets/packets/http.pid", []byte(fmt.Sprint(pid)), 0644); err != nil {
		fmt.Println("error saving subprocess pid", err)
	}

	fs := http.FileServer(http.Dir("/var/cache/packets"))
	http.Handle("/", fs)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 9123), nil))
}
