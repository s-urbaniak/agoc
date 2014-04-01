package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/s-urbaniak/agoc/acmectl"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	swin, err := acmectl.CurrentWindow()
	if err != nil {
		log.Fatal(err)
	}
	defer swin.CloseFiles()

	cwin, err := newAgocWindow()
	if err != nil {
		log.Fatal(err)
	}
	defer cwin.CloseFiles()

	deletes := merge(cwin.DelChan(), swin.DelChan())
	go func() {
		<-deletes
		os.Exit(0)
	}()

	offsets := sevents(swin.OffsetChan())
	debounced := debouncer(offsets, 300*time.Millisecond)
	looper(cwin, swin, debounced)
}

func newAgocWindow() (*acmectl.AcmeCtl, error) {
	cwin, err := acmectl.New()
	if err != nil {
		return nil, err
	}

	pwd, _ := os.Getwd()
	cwin.Name(pwd + "/+agoc")
	cwin.Ctl("clean")
	cwin.Fprintf("tag", "Get ")
	return cwin, nil
}

func sevents(offsetEvents <-chan acmectl.OffsetEvt) chan int {
	offsets := make(chan int)

	go func() {
		for evt := range offsetEvents {
			if evt.Err != nil {
				log.Fatal(evt.Err)
			}

			offsets <- evt.Offset
		}
	}()

	return offsets
}

func looper(cwin *acmectl.AcmeCtl, swin *acmectl.AcmeCtl, offsets chan int) {
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
			cwin.Fprintf("body", "%s\n", err)
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
		}()
	}
}
