/*
Sniperkit-Bot
- Status: analyzed
*/

package borges

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/core-retrieval.v0/model"
	"gopkg.in/src-d/go-queue.v1"

	"github.com/sniperkit/snk.fork.borges/storage"
)

const (
	testEndpoint   = "https://some.endpoint.com"
	testJobRetries = 5
)

func TestProducerSuite(t *testing.T) {
	suite.Run(t, new(ProducerSuite))
}

type ProducerSuite struct {
	BaseQueueSuite
	mentionsQueue queue.Queue
}

func (s *ProducerSuite) SetupSuite() {
	s.BaseQueueSuite.SetupSuite()

	assert := require.New(s.T())
	q, err := s.broker.Queue(fmt.Sprintf("mentions_test_%d", time.Now().UnixNano()))
	assert.NoError(err)

	s.mentionsQueue = q
}

func (s *ProducerSuite) newProducer() *Producer {
	storer := storage.FromDatabase(s.DB)
	return NewProducer(NewMentionJobIter(s.mentionsQueue, storer),
		s.queue, queue.PriorityNormal, testJobRetries)
}

func (s *ProducerSuite) newJob() *queue.Job {
	j, err := queue.NewJob()
	s.Assert().NoError(err)

	m := &model.Mention{
		VCS:      model.GIT,
		Provider: "TEST_PROVIDER",
		Endpoint: testEndpoint,
	}
	err = j.Encode(m)
	s.Assert().NoError(err)

	return j
}

func (s *ProducerSuite) TestStartStop() {
	assert := require.New(s.T())
	p := s.newProducer()

	err := s.mentionsQueue.Publish(s.newJob())
	assert.NoError(err)

	go p.Start()

	time.Sleep(time.Millisecond * 100)

	awnd := 1
	iter, err := s.queue.Consume(awnd)
	j, err := iter.Next()
	assert.NoError(err)
	assert.NotNil(j)

	p.Stop()
}

func (s *ProducerSuite) TestStartStop_TwoEqualsJobs() {
	assert := require.New(s.T())
	p := s.newProducer()

	err := s.mentionsQueue.Publish(s.newJob())
	assert.NoError(err)

	err = s.mentionsQueue.Publish(s.newJob())
	assert.NoError(err)

	go p.Start()

	time.Sleep(time.Millisecond * 100)
	awnd := 1
	iter, err := s.queue.Consume(awnd)
	j, err := iter.Next()
	assert.NoError(err)
	assert.NotNil(j)

	var jobOne Job
	assert.NoError(j.Decode(&jobOne))

	iter, err = s.queue.Consume(awnd)
	assert.NoError(err)
	j, err = iter.Next()
	assert.NoError(err)
	assert.NotNil(j)

	var jobTwo Job
	assert.NoError(j.Decode(&jobOne))

	p.Stop()
	assert.Equal(jobOne.RepositoryID, jobTwo.RepositoryID)
}

func (s *ProducerSuite) TestStartStop_ErrorNoNotifier() {
	p := NewProducer(&DummyJobIter{}, s.queue, queue.PriorityNormal, testJobRetries)

	go p.Start()

	time.Sleep(time.Millisecond * 100)
	p.Stop()
}

func (s *ProducerSuite) TestStartStop_noNotifier() {
	assert := require.New(s.T())
	p := s.newProducer()

	go p.Start()

	time.Sleep(time.Millisecond * 100)

	awnd := 1
	iter, err := s.queue.Consume(awnd)
	j, err := iter.Next()
	assert.NoError(err)
	assert.NotNil(j)

	p.Stop()
}

type DummyJobIter struct{}

func (j DummyJobIter) Close() error        { return errors.New("SOME CLOSE ERROR") }
func (j DummyJobIter) Next() (*Job, error) { return &Job{RepositoryID: uuid.Nil}, nil }
