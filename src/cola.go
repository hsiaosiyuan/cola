package main

import (
	"flag"
	"log"
	"socks5"
	"time"
)

func main() {
	var (
		cfgFile string
		cfg     *socks5.Config
		err     error
		server  *socks5.Server
	)

	flag.StringVar(&cfgFile, "c", "", "conf file")
	flag.Parse()

	if cfg, err = socks5.NewConfig(cfgFile); err != nil {
		log.Fatal(err)
	}

	server = &socks5.Server{
		cfg,
		time.Now(),
	}

	log.Println("Server is running")

	if err = server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
