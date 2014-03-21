package task2

import (
	"fmt"
	"io"
	"log"
	"log/syslog"
	"net/url"
	"os"
	"path/filepath"

	"sourcegraph.com/sourcegraph/util"
	"sync"
)

var LogDir = filepath.Join(os.TempDir(), "sg-log")
var TagLength = 3

func init() {
	err := os.MkdirAll(LogDir, 0700)
	if err != nil {
		log.Fatal(err)
	}
}

type Context struct {
	Log            *log.Logger
	Stderr, Stdout io.Writer

	Destination string

	tag string

	c []io.Closer
}

func (x *Context) Close() error {
	if len(x.c) == 0 {
		return nil
	}
	for _, c := range x.c {
		err := c.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (x *Context) Child() *Context {
	if x.tag == "" {
		return x
	}

	return newRecordedContext(x.tag + "-" + newRandomTag())
}

var DefaultContext *Context = &Context{
	Log:         log.New(os.Stderr, "", log.Ltime|log.Lmicroseconds),
	Stderr:      os.Stderr,
	Stdout:      os.Stdout,
	Destination: "console",
}

func newRandomTag() string { return util.RandomPrintable(TagLength) }

func NewRecordedContext() *Context {
	return newRecordedContext(newRandomTag())
}

func newRecordedContext(tag string) *Context {
	logURL := fmt.Sprintf("https://papertrailapp.com/groups/591393/events?q=program:%s", url.QueryEscape(tag))
	pw, err := syslog.Dial("udp", "logs.papertrailapp.com:50140", syslog.LOG_INFO, tag)
	if err != nil {
		log.Fatal(err)
	}

	logFile := filepath.Join(LogDir, tag+".log")
	fw, err := os.Create(logFile)
	if err != nil {
		log.Fatal(err)
	}

	mw := io.MultiWriter(fw, pw)

	closersMu.Lock()
	defer closersMu.Unlock()
	closers = append(closers, pw, fw)

	return &Context{
		Log:         log.New(mw, "", 0),
		Stderr:      mw,
		Stdout:      mw,
		Destination: fmt.Sprintf("%s and %s", logURL, logFile),
		tag:         tag,
		c:           []io.Closer{pw, fw},
	}
}

var (
	closers   []io.Closer
	closersMu sync.Mutex
)

func FlushAll() {
	closersMu.Lock()
	defer closersMu.Unlock()
	var w sync.WaitGroup
	for _, c := range closers {
		w.Add(1)
		go func() {
			defer w.Done()
			c.Close()
		}()
	}
	w.Wait()
}
