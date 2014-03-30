package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	"code.google.com/p/goplan9/plan9/acme"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	swin, swinid, err := acmeCurrentWin()
	if err != nil {
		log.Fatal(err)
	}
	defer swin.CloseFiles()

	cwin, err := acme.New()
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
	looper(cwin, swinid, debounced)
}

func sevents(win *acme.Win, offsets chan int) {
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

func cevents(win *acme.Win) {
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

func debouncer(inOffset chan int, delay time.Duration) chan int {
	var prevCancel chan bool
	outOffset := make(chan int)

	go func() {
		for curOffset := range inOffset {
			if prevCancel != nil {
				prevCancel <- true
			}

			timeout := make(chan bool, 1)
			cancel := make(chan bool, 1)
			prevCancel = cancel

			go func() {
				time.Sleep(delay)
				timeout <- true
			}()

			go func() {
				select {
				case <-timeout:
					outOffset <- curOffset
				case <-cancel:
				}
			}()
		}
	}()

	return outOffset
}

func looper(cwin *acme.Win, swinid int, offsets chan int) {
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

func acmeCurrentWin() (*acme.Win, int, error) {
	winid := os.Getenv("winid")
	if winid == "" {
		return nil, 0, fmt.Errorf("$winid not set - not running inside acme?")
	}

	id, err := strconv.Atoi(winid)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid $winid %q", winid)
	}

	win, err := acme.Open(id, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot open acme window: %v", err)
	}

	return win, id, nil
}
