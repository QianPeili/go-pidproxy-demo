package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var PidFile string
var ExecFile string

func main() {
	PidFile = os.Args[1]
	filePath, err := exec.LookPath(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	ExecFile = filePath
	wg := sync.WaitGroup{}
	wg.Add(1)
	go signalHandle(&wg)
	if !isRunning() {
		go start()
	}
	wg.Wait()

}

func signalHandle(wg *sync.WaitGroup) {
	defer wg.Done()

	ch := make(chan os.Signal, 10)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
	for {
		sig := <-ch
		pid := readPid()
		s := sig.(syscall.Signal)
		if err := syscall.Kill(pid, s); err != nil {
			log.Printf("proxy: send signal failed, signal: %s, err: %s.", sig.String(), err.Error())
			return
		}

		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			os.Remove(PidFile)
			return
		case syscall.SIGUSR2:
			state := make(chan int)
			go checkRestartState(state, pid)
			select {
			case <-state:
			case <-time.After(time.Second * 3):
				log.Printf("proxy: app restart failed.")
			}
		}
	}
}

func checkRestartState(state chan int, oldPid int) {
	for {
		pid := readPid()
		if pid > 0 && pid != oldPid {
			state <- pid
		} else {
			time.Sleep(time.Microsecond * 100)
		}
	}
}

func readPid() int {
	data, err := ioutil.ReadFile(PidFile)
	if err != nil && os.IsNotExist(err) {
		return -1
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return -1
	}
	return pid
}

func isRunning() bool {
	pid := readPid()
	if pid <= 0 {
		return false
	}

	return syscall.Kill(pid, 0) == nil
}

func start() {
	process, err := os.StartProcess(ExecFile, []string{ExecFile, PidFile}, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("proxy: app starts in pid: %d\n", process.Pid)
}
