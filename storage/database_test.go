/*
Sniperkit-Bot
- Status: analyzed
*/

package storage

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/core-retrieval.v0/model"
	"gopkg.in/src-d/core-retrieval.v0/test"
	"gopkg.in/src-d/go-kallax.v1"
)

type DatabaseSuite struct {
	test.Suite
	store    *DatabaseStore
	rawStore *model.RepositoryStore
}

func (s *DatabaseSuite) SetupTest() {
	s.Setup()
	s.rawStore = model.NewRepositoryStore(s.DB)
	s.store = FromDatabase(s.DB)
}

func (s *DatabaseSuite) TearDownTest() {
	s.TearDown()
}

func (s *DatabaseSuite) TestGet() {
	require := s.Require()

	expected := s.createRepo(model.Pending, "foo")
	repo, err := s.store.Get(expected.ID)
	require.NoError(err)
	require.Equal(expected.ID, repo.ID)
	require.Equal(expected.Endpoints, repo.Endpoints)
	require.Equal(expected.Status, repo.Status)

	repo, err = s.store.Get(kallax.NewULID())
	require.Equal(kallax.ErrNotFound, err)
}

func (s *DatabaseSuite) TestGetByEndpoints() {
	require := s.Require()

	repos := []*model.Repository{
		s.createRepo(model.Pending, "foo"),
		s.createRepo(model.Pending, "bar"),
		s.createRepo(model.Pending, "baz", "bar"),
		s.createRepo(model.Pending, "baz"),
	}

	result, err := s.store.GetByEndpoints("bar", "baz")
	require.Len(result, 3)
	require.NoError(err)
	require.Equal(repos[1].ID, result[0].ID)
	require.Equal(repos[2].ID, result[1].ID)
	require.Equal(repos[3].ID, result[2].ID)

	result, err = s.store.GetByEndpoints("notfound")
	require.Len(result, 0)
	require.NoError(err)
}

func (s *DatabaseSuite) TestSetStatus() {
	require := s.Require()
	repo := s.createRepo(model.Pending, "foo")

	err := s.store.SetStatus(repo, model.Fetching)
	require.NoError(err)
	require.Equal(model.Fetching, repo.Status)

	repo, err = s.store.Get(repo.ID)
	require.NoError(err)
	require.Equal(model.Fetching, repo.Status)
}

func (s *DatabaseSuite) TestSetEndpoints() {
	require := s.Require()
	repo := s.createRepo(model.Pending, "foo")

	endpoints := []string{"bar", "baz"}
	err := s.store.SetEndpoints(repo, endpoints...)
	require.NoError(err)
	require.Len(repo.Endpoints, 2)
	require.Equal(endpoints, repo.Endpoints)

	repo, err = s.store.Get(repo.ID)
	require.NoError(err)
	require.Equal(endpoints, repo.Endpoints)
}

func (s *DatabaseSuite) TestUpdateFailed() {
	require := s.Require()
	repo := s.createRepo(model.Fetching, "foo")

	err := s.store.UpdateFailed(repo, model.Pending)
	require.NoError(err)
	require.Equal(model.Pending, repo.Status)

	repo, err = s.store.Get(repo.ID)
	require.NoError(err)
	require.Equal(model.Pending, repo.Status)
}

func (s *DatabaseSuite) TestUpdateFetched() {
	require := s.Require()
	repo := s.createRepo(model.Fetching, "foo")
	t := withoutNs(time.Now())

	err := s.store.UpdateFetched(repo, t)
	require.NoError(err)
	require.Len(repo.Endpoints, 1)
	require.Equal(&t, repo.FetchedAt)
	require.Equal(model.Fetched, repo.Status)
	require.NotNil(repo.LastCommitAt)
	require.NotEqual(new(time.Time), repo.LastCommitAt)

	repo, err = s.store.Get(repo.ID)
	require.NoError(err)
	require.Equal(model.Fetched, repo.Status)
	require.NotNil(repo.LastCommitAt)
	require.NotEqual(new(time.Time), repo.LastCommitAt)
}

func (s *DatabaseSuite) TestUpdateWithRefsChanged() {
	require := s.Require()

	repo := s.createRepo(model.Fetching, "foo")
	refs := []*model.Reference{
		model.NewReference(),
		model.NewReference(),
	}

	refs[0].Name = "foo"
	refs[0].Repository = repo
	refs[1].Name = "bar"
	refs[1].Repository = repo

	repo.References = refs

	_, err := s.store.Save(repo)
	require.NoError(err)

	newRef := model.NewReference()
	newRef.Repository = repo
	newRef.Name = "baz"
	repo.References = append(repo.References[1:], newRef)

	repo.Status = model.Fetched
	err = s.store.updateWithRefsChanged(repo, model.Schema.Repository.Status)
	require.NoError(err)

	var refStore model.ReferenceStore
	kallax.StoreFrom(&refStore, s.store.RepositoryStore)
	dbRefs, err := refStore.FindAll(model.NewReferenceQuery().FindByRepository(repo.ID))
	require.NoError(err)

	require.Len(dbRefs, 2)
	refNames := []string{dbRefs[0].Name, dbRefs[1].Name}
	sort.Strings(refNames)
	require.Equal([]string{"bar", "baz"}, refNames)

	r, err := s.store.FindOne(model.NewRepositoryQuery().FindByID(repo.ID))
	require.NoError(err)
	require.Equal(model.Fetched, r.Status)
}

func (s *DatabaseSuite) createRepo(status model.FetchStatus, remotes ...string) *model.Repository {
	repo := model.NewRepository()
	repo.Status = status
	repo.Endpoints = remotes
	s.Require().NoError(s.rawStore.Insert(repo))

	repo.CreatedAt = withoutNs(repo.CreatedAt)
	repo.UpdatedAt = withoutNs(repo.UpdatedAt)

	repo.References = []*model.Reference{
		makeRef(model.NewSHA1("1"), time.Now()),
		makeRef(model.NewSHA1("2"), time.Now().Add(-5*time.Hour)),
	}

	_, err := s.rawStore.Update(repo)
	s.Require().NoError(err)
	return repo
}

func makeRef(hash model.SHA1, time time.Time) *model.Reference {
	ref := model.NewReference()
	ref.Hash = hash
	ref.Time = time
	return ref
}

func withoutNs(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, t.Location())
}

func TestDatabase(t *testing.T) {
	suite.Run(t, new(DatabaseSuite))
}
