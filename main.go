package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rwynn/gtm"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/mgo.v2"
)

var OplogCollection = "oplog.rs"

// Items is a list item
type Item struct {
	Id   string `mgo:"_id"`
	Name string `mgo:"name"`
	List string `mgo:"list"`
}

// Reload Dansguardian configurations
func Reload() {
	err := Write()
	if err != nil {
		log.Error("Failed to write new files; not reloading Dansguardian", "error", err)
		return
	}

	// Tell dansguardian to reload
	cmd := exec.Command("/usr/sbin/dansguardian", "-g")
	err = cmd.Run()
	if err != nil {
		log.Error("Failed to reload dansguardian", "error", err)
	}

	return
}

// Write out the new files
func Write() error {
	var err error
	var result Item

	// Create the temp files
	wl, err := os.Create("/etc/dansguardian/exceptionlist_db")
	if err != nil {
		log.Error("Failed to create temporary whitelist", "error", err)
		return err
	}
	defer wl.Close()

	gl, err := os.Create("/etc/dansguardian/greylist_db")
	if err != nil {
		log.Error("Failed to create temporary greylist", "error", err)
		return err
	}
	defer gl.Close()

	bl, err := os.Create("/etc/dansguardian/blacklist_db")
	if err != nil {
		log.Error("Failed to create temporary blacklist", "error", err)
		return err
	}
	defer bl.Close()

	// Get the list
	session, err := mgo.Dial(os.Getenv("MONGO_URL"))
	if err != nil {
		log.Error("Failed to connect to MongoDB", "error", err)
		return err
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	i := session.DB("dg").C("lists").Find(nil).Iter()
	for i.Next(&result) {
		switch result.List {
		case "whitelist":
			_, err = wl.WriteString(fmt.Sprintf("%s\n", result.Name))
		case "greylist":
			_, err = gl.WriteString(fmt.Sprintf("%s\n", result.Name))
		case "blacklist":
			_, err = bl.WriteString(fmt.Sprintf("%s\n", result.Name))
		}
	}

	return err
}

func main() {
	db, err := mgo.Dial(os.Getenv("MONGO_URL"))
	if err != nil {
		log.Error("Failed to connect to MongoDB", "error", err)
		os.Exit(1)
		return
	}
	defer db.Close()

	ops, errs := gtm.Tail(db, &gtm.Options{
		OpLogCollectionName: &OplogCollection,
		Filter: func(op *gtm.Op) bool {
			return op.Namespace == "dg.lists"
		},
	})

	for {
		select {
		case err = <-errs:
			log.Error("Error from oplog tail", "error", err)
		case <-ops:
			log.Info("Change event")
			Reload()
		}
	}
}
