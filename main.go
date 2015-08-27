// agoc is a tool for code completion inside the acme editor
package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	acme9 "9fans.net/go/acme"
	"github.com/s-urbaniak/acme"
)

type offset struct {
	id, offset int
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	agoc, err := agoc()
	if err != nil {
		log.Fatal(err)
	}

	looper(agoc)
}

func looper(agoc *acme.Win) {
	logEvt := logEvtChan()
	opened := make(map[int]struct{})
	offsetChan := make(chan offset)

	for {
		select {
		case o := <-offsetChan:
			complete(o, agoc)
		case evt := <-logEvt:
			switch evt.Op {
			case "del":
				delete(opened, evt.ID)
			case "focus":
				if _, ok := opened[evt.ID]; !ok {
					opened[evt.ID] = struct{}{}

					srcChan := src(evt.ID)
					id := evt.ID
					go func() {
						for o := range srcChan {
							offsetChan <- offset{id, o}
						}
					}()
				}
			}
		}
	}
}

func complete(o offset, agoc *acme.Win) {
	srcWin, err := acme.GetWin(o.id)
	if err != nil {
		log.Println(err)
		return
	}

	cmd := exec.Command("gocode", "autocomplete", strconv.Itoa(o.offset))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	go func() {
		defer func() {
			srcWin.CloseFiles()
			stdin.Close()
		}()

		_, err := io.Copy(stdin, srcWin.FileReadWriter("body"))
		if err != nil {
			log.Fatal(err)
		}
	}()

	agoc.ClearBody()

	_, err = io.Copy(agoc.FileReadWriter("body"), stdout)
	if err != nil {
		log.Fatal(err)
	}

	agoc.Fprintf("addr", "#0")
	agoc.Ctl("dot=addr")
	agoc.Ctl("show")
	agoc.Ctl("clean")

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}
}

func logEvtChan() <-chan acme9.LogEvent {
	acmeLog, err := acme9.Log()
	if err != nil {
		log.Fatal(err)
	}

	events := make(chan acme9.LogEvent)

	go func() {
		for {
			evt, err := acmeLog.Read()
			if err != nil {
				log.Fatal(err)
			}

			if strings.HasSuffix(evt.Name, ".go") {
				events <- evt
			}
		}
	}()

	return events
}

func src(id int) <-chan int {
	offset := make(chan int)

	win, err := acme.GetWin(id)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		defer func() {
			win.CloseFiles()
			close(offset)
		}()

		for {
			evt, err := win.ReadEvent()
			if err != nil {
				log.Fatal(err)
			}

			switch evt.C2 {
			case 'I', 'D':
				tagb, err := win.ReadAll("tag")
				if err != nil {
					log.Fatal(err)
				}

				tag := string(tagb)
				if !strings.Contains(tag, "Put") {
					win.Write("tag", []byte(" Put"))
				}

				err = win.Ctl("addr=dot")
				if err != nil {
					log.Fatal(err)
				}

				q0, _, err := win.ReadAddr()
				if err != nil {
					log.Fatal(err)
				}

				offset <- q0
			case 'x', 'X':
				evtText := string(evt.Text)

				if evtText == "Del" || evtText == "Delete" {
					win.Ctl("delete")
					win.WriteEvent(evt)
					return
				}
			}

			win.WriteEvent(evt)
		}
	}()

	return debouncer(offset, 300*time.Millisecond)
}

func agoc() (*acme.Win, error) {
	agoc, err := acme.New()
	if err != nil {
		return nil, err
	}

	agoc.Name("+agoc")
	agoc.Ctl("clean")
	agoc.Fprintf("tag", "Get ")

	go func() {
		for {
			evt, err := agoc.ReadEvent()
			if err != nil {
				log.Fatal(err)
			}

			switch evt.C2 {
			case 'x', 'X':
				evtText := string(evt.Text)

				if evtText == "Del" || evtText == "Delete" {
					agoc.WriteEvent(evt)
					os.Exit(0)
				}
			}

			agoc.WriteEvent(evt)
		}
	}()

	return agoc, nil
}
