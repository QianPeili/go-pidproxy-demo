package main

import (
	"os"
	"os/signal"
	"syscall"
	"log"
	"strconv"
	"io/ioutil"
)

const (
	EnvProcessState = "APP_CHILD"
)

func writePid(pid int, name string) {
	data := strconv.Itoa(pid)
	ioutil.WriteFile(name, []byte(data), 0644)
}

func main() {
	writePid(os.Getpid(), os.Args[1])
	ch := make(chan os.Signal, 10)
	ppid := os.Getppid()
	if os.Getenv(EnvProcessState) != "" && ppid > 1 {
		if err := syscall.Kill(ppid, syscall.SIGTERM); err != nil {
			log.Printf("app: kill parent process failed: %s", err.Error())
		}
	}
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
	for {
		sig := <-ch
		log.Printf("app: receive signal %d.", sig)
		switch sig {
		case syscall.SIGTERM, syscall.SIGINT:
			log.Printf("app: process %d exit.", os.Getpid())
			os.Exit(0)
		case syscall.SIGUSR2:
			os.Setenv(EnvProcessState, "true")
			process, err := os.StartProcess(os.Args[0], os.Args, &os.ProcAttr{
				Env:   os.Environ(),
				Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			})
			if err != nil {
				log.Printf("app: fork child process failed: %s.\n", err.Error())
			} else {
				log.Printf("app: process restart with pid: %d.\n", process.Pid)
			}
		}
	}
}
