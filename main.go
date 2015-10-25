package main

import (
	"os"
	"os/exec"

	"github.com/termie/go-shutil"
	"gopkg.in/fsnotify.v1"
	log "gopkg.in/inconshreveable/log15.v2"
)

// Reload Dansguardian configurations
func Reload(fileName string) {
	var err error

	// Rename each tmp file -- ignore errors
	switch fileName {
	case "/tmp/dg/whitelist":
		err = shutil.CopyFile("/tmp/dg/whitelist", "/etc/dansguardian/exceptionlist_db", false)
	case "/tmp/dg/greylist":
		err = shutil.CopyFile("/tmp/dg/greylist", "/etc/dansguardian/greylist_db", false)
	case "/tmp/dg/blacklist":
		err = shutil.CopyFile("/tmp/dg/blacklist", "/etc/dansguardian/blacklist_db", false)
	default:
		log.Error("Unhandled list", fileName)
		return
	}
	if err != nil {
		log.Error("Failed to move list into place:", "error", err)
		return
	}

	// Tell dansguardian to reload
	cmd := exec.Command("/usr/sbin/dansguardian", "-g")
	err = cmd.Run()
	if err != nil {
		log.Error("Failed to reload dansguardian", "error", err)
	}
}

func main() {
	os.Mkdir("/tmp/dg", 0777)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error("Failed to create filesystem watcher:", "error", err)
		return
	}

	w.Add("/tmp/dg/")

	log.Info("Watching dansguardian configuration tmp files")
	for {
		e := <-w.Events
		log.Debug("Event received", "event", e)
		if e.Op == fsnotify.Write {
			log.Info("Reloading dansguardian...", "list", e.Name)
			Reload(e.Name)
		}
	}
}
