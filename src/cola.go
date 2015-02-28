package main

import (
	"flag"
	"log"
	"socks5/server"
	"time"
	"socks5"
)

func main() {
	var (
		cfgFile string
		cfg     *server.Config
		err     error
		srv     *server.Server
	)
	
	socks5.IncreaseRlimit()

	flag.StringVar(&cfgFile, "c", "", "conf file")
	flag.Parse()

	if cfg, err = server.NewConfig(cfgFile); err != nil {
		log.Fatal(err)
	}

	srv = &server.Server{
		cfg,
		time.Now(),
	}

	log.Println("Cola is running")

	if err = srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
