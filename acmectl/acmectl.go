package acmectl

import (
	"fmt"
	"io"
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

type WinEvt struct {
	Offset int
	Del    bool
	Err    error
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

	ctl := &AcmeCtl{}
	ctl.Win = win
	ctl.id = id

	return ctl, nil
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

func (ctl *AcmeCtl) WinEvtChannel() chan WinEvt {
	evtChan := make(chan WinEvt)

	go func() {
		for evt := range ctl.Win.EventChan() {
			switch evt.C2 {
			case 'I':
				err := ctl.Win.Ctl("addr=dot")
				if err != nil {
					evtChan <- WinEvt{-1, false, err}
				}

				runeOffset, _, err := ctl.Win.ReadAddr()
				if err != nil {
					evtChan <- WinEvt{-1, false, err}
				}

				evtChan <- WinEvt{runeOffset, false, nil}

			case 'x', 'X':
				if string(evt.Text) == "Del" {
					ctl.Win.Ctl("delete")
					evtChan <- WinEvt{-1, true, nil}
				}
			}

			ctl.Win.WriteEvent(evt)
		}
	}()

	return evtChan
}

func (ctl AcmeCtl) WindowId() int {
	return ctl.id
}

func (ctl *AcmeCtl) ClearBody() error {
	err := ctl.Win.Addr(",")
	if err != nil {
		return err
	}

	_, err = ctl.Win.Write("data", nil)
	if err != nil {
		return err
	}	

	err = ctl.Win.Ctl("clean")
	if err != nil {
		return err
	}
	
	return nil
}

func (ctl *AcmeCtl) GotoAddr(addr string) error {
	err := ctl.Win.Fprintf("addr", addr)
	if err != nil {
		return err
	}

	err = ctl.Win.Ctl("dot=addr")
	if err != nil {
		return err
	}
	
	err = ctl.Win.Ctl("show")
	if err != nil {
		return err
	}

	return nil
}

func (ctl AcmeCtl) ReadBody() ([]byte, error) {
	rwin, err := acme.Open(ctl.id, nil)

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
