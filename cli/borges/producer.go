/*
Sniperkit-Bot
- Status: analyzed
*/

package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jessevdk/go-flags"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
	"gopkg.in/src-d/go-queue.v1"
	_ "gopkg.in/src-d/go-queue.v1/amqp"

	"github.com/sniperkit/snk.fork.borges"
)

const (
	producerCmdName      = "producer"
	producerCmdShortDesc = "create new jobs and put them into the queue"
	producerCmdLongDesc  = ""
)

var producerCommand = &producerCmd{simpleCommand: newSimpleCommand(
	producerCmdName,
	producerCmdShortDesc,
	producerCmdLongDesc,
)}

type producerCmd struct {
	simpleCommand
}

type producerSubcmd struct {
	command
	broker queue.Broker
	queue  queue.Queue

	Priority    uint8 `long:"priority" default:"4" description:"priority used to enqueue jobs, goes from 0 (lowest) to :MAX: (highest)"`
	JobsRetries int   `long:"job-retries" default:"5" description:"number of times a falied job should be processed again before reject it"`
}

func newProducerSubcmd(name, short, long string) producerSubcmd {
	return producerSubcmd{command: newCommand(
		name,
		short,
		long,
	)}
}

func (c *producerSubcmd) init() error {
	c.command.init()

	err := checkPriority(c.Priority)
	if err != nil {
		return err
	}

	c.broker, err = queue.NewBroker(c.Broker)
	if err != nil {
		return err
	}

	c.queue, err = c.broker.Queue(c.Queue)
	if err != nil {
		return err
	}

	return nil
}

type getIterFunc func() (borges.JobIter, error)

func (c *producerSubcmd) generateJobs(getIter getIterFunc) error {
	ji, err := getIter()
	if err != nil {
		return err
	}
	defer ioutil.CheckClose(ji, &err)

	p := borges.NewProducer(ji, c.queue, queue.Priority(c.Priority), c.JobsRetries)

	p.Start()

	return err
}

// Changes the priority description and default on runtime as it is not
// possible to create a dynamic tag
func setPrioritySettings(c *flags.Command) {
	options := c.Options()

	for _, o := range options {
		if o.LongName == "priority" {
			o.Default[0] = strconv.Itoa((int(queue.PriorityNormal)))
			o.Description = strings.Replace(
				o.Description, ":MAX:", strconv.Itoa(int(queue.PriorityUrgent)), 1)
		}
	}
}

func checkPriority(prio uint8) error {
	if prio > uint8(queue.PriorityUrgent) {
		return fmt.Errorf("Priority must be between 0 and %d", queue.PriorityUrgent)
	}

	return nil
}

var producerSubcommands = []ExecutableCommand{
	mentionsCommand,
	fileCommand,
	republishCommand,
}

func init() {
	c, err := parser.AddCommand(
		producerCommand.name,
		producerCommand.shortDescription,
		producerCommand.longDescription,
		producerCommand)

	if err != nil {
		panic(err)
	}

	for _, subcommand := range producerSubcommands {
		s, err := c.AddCommand(
			subcommand.Name(),
			subcommand.ShortDescription(),
			subcommand.LongDescription(),
			subcommand,
		)

		if err != nil {
			panic(err)
		}

		setPrioritySettings(s)
	}
}
