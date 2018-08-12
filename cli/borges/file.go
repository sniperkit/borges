/*
Sniperkit-Bot
- Status: analyzed
*/

package main

import (
	"os"

	"gopkg.in/src-d/core-retrieval.v0"

	"github.com/sniperkit/snk.fork.borges"
	"github.com/sniperkit/snk.fork.borges/storage"
)

const (
	fileCmdName      = "file"
	fileCmdShortDesc = "produce jobs from file"
	fileCmdLongDesc  = ""
)

// fileCommand is a producer subcommand.
var fileCommand = &fileCmd{producerSubcmd: newProducerSubcmd(
	fileCmdName,
	fileCmdShortDesc,
	fileCmdLongDesc,
)}

type fileCmd struct {
	producerSubcmd

	filePositionalArgs `positional-args:"true" required:"1"`
}

type filePositionalArgs struct {
	File string `positional-arg-name:"path"`
}

func (c *fileCmd) Execute(args []string) error {
	if err := c.producerSubcmd.init(); err != nil {
		return err
	}
	defer c.broker.Close()

	return c.generateJobs(c.jobIter)
}

func (c *fileCmd) jobIter() (borges.JobIter, error) {
	storer := storage.FromDatabase(core.Database())
	f, err := os.Open(c.File)
	if err != nil {
		return nil, err
	}
	return borges.NewLineJobIter(f, storer), nil
}
