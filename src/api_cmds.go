package src

import (
	"fmt"
	"log"
)

func init() {
	c, err := CLI.AddCommand("api",
		"API",
		"",
		&apiCmd,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.AddCommand("describe",
		"display documentation for the def under the cursor",
		"Returns information about the definition referred to by the cursor's current position in a file.",
		&apiDescribeCmd,
	)
	if err != nil {
		log.Fatal(err)
	}
}

type APICmd struct{}

var apiCmd APICmd

func (c *APICmd) Execute(args []string) error { return nil }

type APIDescribeCmd struct {
	File      string `long:"file" required:"yes" value-name:"FILE"`
	StartByte int    `long:"start-byte" required:"yes" value-name:"BYTE"`
}

var apiDescribeCmd APIDescribeCmd

func (c *APIDescribeCmd) Execute(args []string) error {
	// TODO(sqs): implement these for real
	fmt.Println(`{"Name":"myname","Doc":"mydoc"}\n`)
	return nil
}
