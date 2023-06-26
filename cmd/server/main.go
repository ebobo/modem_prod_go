package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ebobo/modem_prod_go/pkg/server"
	sqlitestore "github.com/ebobo/modem_prod_go/pkg/store/sqlite"
	"github.com/ebobo/modem_prod_go/pkg/utility"
	"github.com/jessevdk/go-flags"
)

var opt struct {
	HTTPAddr   string `short:"h" long:"http-addr" default:":9090" description:"http listen address" required:"yes"`
	SqliteFile string `long:"sqlite-file" env:"SQLITE_FILE" default:"modems.db" description:"sqlite file"`
}

func main() {
	_, err := flags.ParseArgs(&opt, os.Args)
	if err != nil {
		log.Fatalf("error parsing flags: %v", err)
	}

	db, created, err := sqlitestore.New(opt.SqliteFile)
	if err != nil {
		log.Fatalf("error connect to sqlite: %v", err)
	}

	//some test data
	if created {
		modems := utility.GenerateFakeModems(20)

		for _, modem := range modems {
			err = db.AddModem(modem)
			if err != nil {
				log.Fatalf("error adding modem to database: %v", err)
			}
		}

	} else {
		log.Println("db already exists")
	}

	server := server.New(server.Config{
		HTTPListenAddr: opt.HTTPAddr,
		DB:             db,
	})

	e := server.Start()
	if e != nil {
		log.Fatalf("error starting server: %v", e)
	}

	// Block forever
	// Capture Ctrl-C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	server.Shutdown()
}
