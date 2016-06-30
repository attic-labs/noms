package integrationtest

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/testify/assert"
	"github.com/attic-labs/testify/suite"
)

type IntegrationSuiteInterface interface {
	suite.TestingSuite
	// DatabaseSpecString returns the spec for the test database.
	DatabaseSpecString() string
	// ValueSpecString returns the spec for the value in the test database.
	ValueSpecString(value string) string

	setPort(port int)
	npmInstall()
}

// SetupDataSuite is the interface to implement if you need to initialize some data in the ChunkStore.
type SetupDataSuite interface {
	SetupData(cs chunks.ChunkStore)
}

// CheckDataSuite is the interface to implement if you want to check the data in the store after the node command has finished.
type CheckDataSuite interface {
	CheckData(cs chunks.ChunkStore)
}

// NodeArgsSuite is the interface to implement if you want to provide extra arguments to node. If this is not implemented we call `node .`
type NodeArgsSuite interface {
	NodeArgs() []string
}

// CheckNodeSuite is the interface to implement if you want to validate the output of the node command.
type CheckNodeSuite interface {
	CheckNode(out string)
}

// IntegrationSuite is used to create a single node js integration test.
type IntegrationSuite struct {
	suite.Suite
	port int
}

// RunIntegrationSuite runs a single integration test.
func RunIntegrationSuite(t *testing.T, s IntegrationSuiteInterface) {
	s.SetT(t)
	s.npmInstall()
	cs := chunks.NewMemoryStore()
	if s, ok := s.(SetupDataSuite); ok {
		s.SetupData(cs)
	}

	runServer(cs, s)

	if s, ok := s.(CheckDataSuite); ok {
		s.CheckData(cs)
	}
}

func runServer(cs chunks.ChunkStore, s IntegrationSuiteInterface) {
	server := datas.NewRemoteDatabaseServer(cs, 0)
	server.Ready = func() {
		s.setPort(server.Port())
		runNode(s)
		server.Stop()
	}
	server.Run()
}

func runNode(s IntegrationSuiteInterface) {
	args := []string{"."}
	if nas, ok := s.(NodeArgsSuite); ok {
		args = append(args, nas.NodeArgs()...)
	}
	out, err := exec.Command("node", args...).Output()
	assert.NoError(s.T(), err)
	if cns, ok := s.(CheckNodeSuite); ok {
		cns.CheckNode(string(out))
	}
}

func (s *IntegrationSuite) setPort(port int) {
	s.port = port
}

func (s *IntegrationSuite) npmInstall() {
	err := exec.Command("npm", "install").Run()
	s.NoError(err)
}

// DatabaseSpecString returns the spec for the database to test against.
func (s *IntegrationSuite) DatabaseSpecString() string {
	return spec.CreateDatabaseSpecString("http", fmt.Sprintf("//localhost:%d", s.port))
}

// ValueSpecString returns the value spec for the value to test against.
func (s *IntegrationSuite) ValueSpecString(value string) string {
	return spec.CreateValueSpecString("http", fmt.Sprintf("//localhost:%d", s.port), value)
}
