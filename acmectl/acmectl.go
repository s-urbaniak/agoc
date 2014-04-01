package acmectl

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"code.google.com/p/goplan9/plan9/acme"
)

var ctlre = regexp.MustCompile(`(.{11} )(.{11} )(.{11} )(.{11} )(.{11} )(.{11} )(.*?) (.*)`)

type AcmeCtl struct {
	*acme.Win
	id int
}

type OffsetEvt struct {
	offset int
	err    error
}

func CurrentWindow() (*AcmeCtl, error) {
	winid := os.Getenv("winid")
	if winid == "" {
		return nil, fmt.Errorf("$winid not set - not running inside acme?")
	}

	id, err := strconv.Atoi(winid)
	if err != nil {
		return nil, fmt.Errorf("invalid $winid %q", winid)
	}

	win, err := acme.Open(id, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot open acme window: %v", err)
	}

	return &AcmeCtl{win, id}, nil
}

func New() (*AcmeCtl, error) {
	ctl := &AcmeCtl{}
	win, err := acme.New()
	if err != nil {
		return nil, err
	}

	ctl.Win = win

	buf, err := win.ReadAll("ctl")
	if err != nil {
		return nil, err
	}

	fields := ctlre.FindAllStringSubmatch(string(buf), -1)
	ctl.id, err = strconv.Atoi(strings.TrimSpace(fields[0][1]))
	if err != nil {
		return nil, err
	}

	return ctl, err
}

func (ctl AcmeCtl) WindowId() int {
	return ctl.id
}

func (ctl AcmeCtl) OffsetChan() <-chan OffsetEvt {
	offsets := make(chan OffsetEvt)

	go func() {
		for evt := range ctl.Win.EventChan() {
			switch evt.C2 {
			case 'I':
				err := ctl.Win.Ctl("addr=dot")
				if err != nil {
					offsets <- OffsetEvt{-1, err}
				}

				runeOffset, _, err := ctl.Win.ReadAddr()
				if err != nil {
					offsets <- OffsetEvt{-1, err}
				}

				offsets <- OffsetEvt{runeOffset, nil}
			}
			ctl.Win.WriteEvent(evt)
		}
	}()

	return offsets
}

func (ctl AcmeCtl) DelChan() <-chan bool {
	del := make(chan bool)

	go func() {
		for evt := range ctl.Win.EventChan() {
			switch evt.C2 {
			case 'x', 'X':
				if string(evt.Text) == "Del" {
					ctl.Win.Ctl("delete")
					del <- true
				}
			}
			ctl.Win.WriteEvent(evt)
		}
	}()

	return del
}

func (ctl *AcmeCtl) ClearBody() {
	ctl.Win.Addr(",")
	ctl.Win.Write("data", nil)
	ctl.Win.Ctl("clean")
}

func (ctl *AcmeCtl) GotoAddr(addr string) {
	ctl.Win.Fprintf("addr", addr)
	ctl.Win.Ctl("dot=addr")
	ctl.Win.Ctl("show")
}
