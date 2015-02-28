package main

import (
	"socks5/client"
	"flag"
	"log"
	"socks5"
)

func main() {
	var (
		clt *client.Client
		addr string
		srvAddr string
		un string
		pwd string
		err error
	)
	
	socks5.IncreaseRlimit()

	flag.StringVar(&addr, "la", "", "local addrress")
	flag.StringVar(&srvAddr, "sa", "", "server address")
	flag.StringVar(&un, "un", "", "username")
	flag.StringVar(&pwd, "pwd", "", "password")

	flag.Parse()
	
	clt = new(client.Client)
	if err = clt.SetAddr(addr); err != nil {
		log.Fatal(err)
	}

	if err = clt.SetSrvAddr(srvAddr); err != nil {
		log.Fatal(err)
	}

	clt.Un = []byte(un)
	clt.Pwd = []byte(pwd)

	log.Println("Coco is running")

	if err = clt.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
