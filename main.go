// agoc is a tool for code completion inside the acme editor
package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/s-urbaniak/acme"
)

type swinHandler struct {
	offset chan int
	del    chan struct{}
}

var _ acme.EvtHandler = (*swinHandler)(nil)

func (s swinHandler) BodyInsert(offset int) {
	s.offset <- offset
}

func (s swinHandler) Del() {
	s.del <- struct{}{}
}

func (s swinHandler) Err(err error) {
	log.Fatal(err)
}

type cwinHandler struct {
	win *acme.Win
	del chan struct{}
}

var _ acme.EvtHandler = (*cwinHandler)(nil)

func (c cwinHandler) Del() {
	c.del <- struct{}{}
}

func (c cwinHandler) BodyInsert(offset int) {
	c.win.Fprintf("addr", "#0")
	c.win.Ctl("dot=addr")
	c.win.Ctl("show")
	c.win.Ctl("clean")
}

func (c cwinHandler) Err(err error) {
	log.Fatal(err)
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	id, err := acme.GetWinID()
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

	sDelChan, sOffsetChan := make(chan struct{}), make(chan int)
	swin.HandleEvt(swinHandler{sOffsetChan, sDelChan})

	cDelChan := make(chan struct{})
	cwin.HandleEvt(cwinHandler{cwin, cDelChan})

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
		defer stdout.Close()

		if err != nil {
			log.Fatal(err)
		}

		if err := cmd.Start(); err != nil {
			cwin.Fprintf("body", "Error: %s\n", err)
		}

		go func() {
			defer stdin.Close()

			srcBody, err := swin.ReadBody()
			if err != nil {
				log.Fatal(err)
			}

			_, err = io.WriteString(stdin, string(srcBody))
			if err != nil {
				log.Fatal(err)
			}
		}()

		cwin.ClearBody()
		_, err = io.Copy(cwin.FileWriter("body"), stdout)
		if err != nil {
			log.Fatal(err)
		}

		err = cmd.Wait()
		if err != nil {
			log.Fatal(err)
		}
	}
}
