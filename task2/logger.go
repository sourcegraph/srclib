package task2

import (
	"fmt"
	"io"
	"log"
	"log/syslog"
	"net/url"
	"os"
	"path/filepath"

	"sync"
)

var LogDir = filepath.Join(os.TempDir(), "sg-log")

func init() {
	err := os.MkdirAll(LogDir, 0700)
	if err != nil {
		log.Fatal(err)
	}
}

type Logger struct {
	*log.Logger
	io.Writer
	Destination string
	c           []io.Closer
}

func (x *Logger) Close() error {
	if len(x.c) == 0 {
		return nil
	}
	for _, c := range x.c {
		err := c.Close()
		if err != nil {
			return err
		}
	}
	x.c = nil
	return nil
}

func LogURLForTag(tag string) string {
	return fmt.Sprintf("https://papertrailapp.com/groups/591393/events?q=program:%s", url.QueryEscape(tag))
}

func NewPapertrailLogger(tag string) *syslog.Writer {
	pw, err := syslog.Dial("udp", "logs.papertrailapp.com:50140", syslog.LOG_INFO, tag)
	if err != nil {
		log.Fatal(err)
	}
	return pw
}

func NewLogger(tag string) *Logger {
	logURL := LogURLForTag(tag)
	pw := NewPapertrailLogger(tag)

	logFile := filepath.Join(LogDir, tag+".log")
	fw, err := os.Create(logFile)
	if err != nil {
		log.Fatal(err)
	}

	mw := io.MultiWriter(fw, pw)

	closersMu.Lock()
	defer closersMu.Unlock()
	closers = append(closers, pw, fw)

	return &Logger{
		Logger:      log.New(mw, "", 0),
		Writer:      mw,
		Destination: fmt.Sprintf("%s and %s", logURL, logFile),
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
