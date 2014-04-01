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

	"code.google.com/p/goplan9/plan9/acme"
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

	var offsets = make(chan int, 1)
	go sevents(swin, offsets)
	go cevents(cwin)
	debounced := debouncer(offsets, 300*time.Millisecond)
	looper(cwin, swin.WindowId(), debounced)
}

func sevents(win *acmectl.AcmeCtl, offsets chan int) {
	for evt := range win.EventChan() {
		switch evt.C2 {
		case 'x', 'X':
			if string(evt.Text) == "Del" {
				win.Ctl("delete")
				os.Exit(0)
			}
		case 'I':
			err := win.Ctl("addr=dot")
			if err != nil {
				log.Fatal(err)
			}

			runeOffset, _, err := win.ReadAddr()
			if err != nil {
				log.Fatal(err)
			}

			offsets <- runeOffset
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

func looper(cwin *acmectl.AcmeCtl, swinid int, offsets chan int) {
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

		cwin.Addr(",")
		cwin.Write("data", nil)
		cwin.Ctl("clean")

		if err := cmd.Start(); err != nil {
			cwin.Fprintf("body", "%s\n", err)
		}

		go func() {
			defer stdin.Close()

			body, err := readBody(swinid)
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
			cwin.Fprintf("addr", "#0")
			cwin.Ctl("dot=addr")
			cwin.Ctl("show")
			cwin.Ctl("clean")
		}()
	}
}

func readBody(id int) ([]byte, error) {
	rwin, err := acme.Open(id, nil)

	if err != nil {
		return nil, err
	}

	defer rwin.CloseFiles()

	var body []byte
	buf := make([]byte, 8000)
	for {
		n, err := rwin.Read("body", buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		body = append(body, buf[0:n]...)
	}

	if err != nil {
		return nil, err
	}

	return body, nil
}
