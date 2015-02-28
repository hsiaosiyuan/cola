package socks5

import (
	"syscall"
	"log"
)


func IncreaseRlimit(){
	var (
		err error
		lim *syscall.Rlimit
	)
	
	// details: http://linux.die.net/man/2/setrlimit
	lim = &syscall.Rlimit{
		65535,
		65535,
	}

	// details: http://stackoverflow.com/questions/17817204/how-to-set-ulimit-n-from-a-golang-program
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE,lim)
	if err !=nil{
		log.Println("Error occrred when increasing rlimit: " + err.Error())
		log.Fatal("You may need to run this soft as root.")
	}
}
