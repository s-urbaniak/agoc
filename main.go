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

	cwin, err := acmectl.New()
	if err != nil {
		log.Fatal(err)
	}
	pwd, _ := os.Getwd()
	cwin.Name(pwd + "/+agoc")
	cwin.Ctl("clean")
	cwin.Fprintf("tag", "Get ")

	offsets := swin.OffsetChan()

	go sevents(swin, offsets)
	go cevents(cwin)
	debounced := debouncer(offsets, 300*time.Millisecond)
	looper(cwin, swin, debounced)
}

func sevents(win *acmectl.AcmeCtl, offsets chan int) {
	for evt := range win.EventChan() {
		switch evt.C2 {
		case 'x', 'X':
			if string(evt.Text) == "Del" {
				win.Ctl("delete")
				os.Exit(0)
			}
		}
		win.WriteEvent(evt)
	}
}

func cevents(win *acmectl.AcmeCtl) {
	for evt := range win.EventChan() {
		switch evt.C2 {
		case 'x', 'X':
			if string(evt.Text) == "Del" {
				win.Ctl("delete")
				os.Exit(0)
			}
		}
		win.WriteEvent(evt)
	}
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

		cwin.ClearBody()

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
			_, err := cwin.Write("body", buf.Bytes())
			if err != nil {
				log.Fatal(err)
			}

			cwin.GotoAddr("#0")
			cwin.Ctl("clean")
		}()
	}
}
