package errors

import (
	"errors"
	clarumcore "github.com/go-clarum/clarum-core"
	clarumhttp "github.com/go-clarum/clarum-http"

	"os"
	"strings"
	"testing"
	"time"
)

var errorsClient = clarumhttp.Http().Client().
	Name("errorsClient").
	Timeout(2000 * time.Millisecond).
	Build()

var errorsServer = clarumhttp.Http().Server().
	Name("errorsServer").
	Port(8083).
	Build()

func TestMain(m *testing.M) {
	clarumcore.Setup()

	result := m.Run()

	clarumcore.Finish()

	os.Exit(result)
}

func checkErrors(t *testing.T, expectedErrors []string, actionErrors ...error) {
	allErrors := errors.Join(actionErrors...)

	if allErrors == nil {
		t.Error("One error expected, but there was none.")
	} else {
		for _, value := range expectedErrors {
			if !strings.Contains(allErrors.Error(), value) {
				t.Errorf("Unexpected errors: %s", allErrors)
			}
		}
	}
}
