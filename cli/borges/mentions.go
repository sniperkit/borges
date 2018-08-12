/*
Sniperkit-Bot
- Status: analyzed
*/

package main

import (
	"gopkg.in/src-d/core-retrieval.v0"

	"github.com/sniperkit/snk.fork.borges"
	"github.com/sniperkit/snk.fork.borges/storage"
)

const (
	mentionsCmdName      = "mentions"
	mentionsCmdShortDesc = "produce jobs from mentions"
	mentionsCmdLongDesc  = ""
)

var mentionsCommand = &mentionsCmd{producerSubcmd: newProducerSubcmd(
	mentionsCmdName,
	mentionsCmdShortDesc,
	mentionsCmdLongDesc,
)}

// mentionsCommand is a producer subcommand.
type mentionsCmd struct {
	producerSubcmd

	MentionsQueue   string `long:"mentions-queue" default:"rovers" description:"queue name used to obtain mentions if the source type is 'mentions'"`
	RepublishBuried bool   `long:"republish-buried" description:"republishes again all buried jobs before starting to listen for mentions, used with --source=mentions"`
}

func (c *mentionsCmd) Execute(args []string) error {
	if err := c.producerSubcmd.init(); err != nil {
		return err
	}
	defer c.broker.Close()

	return c.generateJobs(c.jobIter)
}

func (c *mentionsCmd) jobIter() (borges.JobIter, error) {
	storer := storage.FromDatabase(core.Database())
	q, err := c.broker.Queue(c.MentionsQueue)
	if err != nil {
		return nil, err
	}

	if c.RepublishBuried {
		if err := q.RepublishBuried(); err != nil {
			return nil, err
		}
	}
	return borges.NewMentionJobIter(q, storer), nil
}
