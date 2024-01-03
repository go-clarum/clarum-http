package itests

import (
	clarumcore "github.com/goclarum/clarum/core"
	clarumhttp "github.com/goclarum/clarum/http"
	"os"
	"testing"
	"time"
)

var testClient = clarumhttp.Http().Client().
	Name("testClient").
	BaseUrl("http://localhost:8083/myApp").
	Timeout(2000 * time.Millisecond).
	Build()

var firstTestServer = clarumhttp.Http().Server().
	Name("firstTestServer").
	Port(8083).
	Build()

var secondTestServer = clarumhttp.Http().Server().
	Name("secondTestServer").
	Port(8084).
	Build()

func TestMain(m *testing.M) {
	clarumcore.Setup()

	result := m.Run()

	clarumcore.Finish()

	os.Exit(result)
}
