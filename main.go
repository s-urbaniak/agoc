package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/s-urbaniak/apl/acme"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	id, err := acme.GetWinid()
	if err != nil {
		log.Fatal(err)
	}

	swin, err := acme.GetWin(id)
	if err != nil {
		log.Fatal(err)
	}
	defer swin.CloseFiles()

	cwin, err := newAgocWindow()
	if err != nil {
		log.Fatal(err)
	}
	defer cwin.CloseFiles()

	sDelChan, sOffsetChan := make(chan bool), make(chan int)

	go func() {
		for evt := range swin.EvtChannel(acme.Soffset | acme.Sdel) {
			if evt.Err != nil {
				log.Fatal(evt.Err)
			} else if evt.Del {
				sDelChan <- true
			} else {
				sOffsetChan <- evt.Offset
			}
		}
	}()

	cDelChan := make(chan bool)
	go func() {
		for _ = range cwin.EvtChannel(acme.Sdel) {
			cDelChan <- true
		}
	}()

	deletes := merge(cDelChan, sDelChan)
	go func() {
		<-deletes
		os.Exit(0)
	}()

	debounced := debouncer(sOffsetChan, 300*time.Millisecond)
	looper(cwin, swin, debounced)
}

func newAgocWindow() (*acme.Win, error) {
	cwin, err := acme.New()
	if err != nil {
		return nil, err
	}

	pwd, _ := os.Getwd()
	cwin.Name(pwd + "/+agoc")
	cwin.Ctl("clean")
	cwin.Fprintf("tag", "Get ")
	return cwin, nil
}

func looper(cwin *acme.Win, swin *acme.Win, offsets <-chan int) {
	for o := range offsets {
		cmd := exec.Command("gocode", "autocomplete", strconv.Itoa(o))
		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Fatal(err)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}

		if err := cmd.Start(); err != nil {
			cwin.Fprintf("body", "Error: %s\n", err)
		}

		go func() {
			defer stdin.Close()

			body, err := swin.ReadBody()
			if err != nil {
				log.Fatal(err)
			}

			_, err = io.WriteString(stdin, string(body))
			if err != nil {
				log.Fatal(err)
			}
		}()

		go func() {
			defer stdout.Close()

			buf := new(bytes.Buffer)
			buf.ReadFrom(stdout)

			cwin.ClearBody()

			_, err := cwin.Write("body", buf.Bytes())
			if err != nil {
				log.Fatal(err)
			}

			cwin.GotoAddr("#0")
			cwin.Ctl("clean")

			err = cmd.Wait()
			if err != nil {
				log.Fatal(err)
			}
		}()
	}
}
