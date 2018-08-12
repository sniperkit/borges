/*
Sniperkit-Bot
- Status: analyzed
*/

package borges

import (
	"gopkg.in/src-d/core-retrieval.v0/model"
	"gopkg.in/src-d/go-queue.v1"
)

type mentionJobIter struct {
	storer RepositoryStore
	q      queue.Queue
	iter   queue.JobIter
}

// NewMentionJobIter returns a JobIter that returns jobs generated from mentions
// received from a queue (e.g. from rovers).
func NewMentionJobIter(q queue.Queue, storer RepositoryStore) JobIter {
	return &mentionJobIter{
		storer: storer,
		q:      q,
	}
}

func (i *mentionJobIter) Next() (*Job, error) {
	if err := i.initIter(); err != nil {
		return nil, err
	}

	mention, j, err := i.getMention()
	if err != nil {
		if queue.ErrAlreadyClosed.Is(err) {
			if err := i.Close(); err != nil {
				return nil, err
			}
		}
		return nil, err
	}

	ID, err := RepositoryID(getEndpoints(mention.Aliases, mention.Endpoint), mention.IsFork, i.storer)
	if err != nil {
		return nil, err
	}

	bj := &Job{RepositoryID: ID}

	if err := j.Ack(); err != nil {
		return nil, err
	}

	return bj, nil
}

// initIter initialize the iterator if it is not already initialized.
func (i *mentionJobIter) initIter() error {
	if i.iter == nil {
		awnd := 1
		iter, err := i.q.Consume(awnd)
		if err != nil {
			return err
		}

		i.iter = iter
	}

	return nil
}

// getMention obtains the next Job from the queue and decodes the mention on it.
// If success, a Mention model is returned. Also the job itself is returned, to
// be able to send back the ACK.
func (i *mentionJobIter) getMention() (m *model.Mention, j *queue.Job, err error) {
	j, err = i.iter.Next()
	if err != nil {
		return
	}
	err = j.Decode(&m)

	return
}

func getEndpoints(aliases []string, mainURL string) []string {
	if aliases == nil {
		return []string{mainURL}
	}

	// if aliases is not nil it also contains the main URL
	return aliases
}

func (i *mentionJobIter) Close() error {
	if i.iter != nil {
		defer func() { i.iter = nil }()
		if err := i.iter.Close(); err != nil {
			return err
		}
	}

	return nil
}
