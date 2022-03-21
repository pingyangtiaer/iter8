package cmd

import (
	"fmt"
	"testing"

	"github.com/iter8-tools/iter8/base"
	id "github.com/iter8-tools/iter8/driver"
)

func TestKLaunch(t *testing.T) {
	srv := id.SetupWithRepo(t)

	tests := []cmdTestCase{
		// Launch, base case, values from CLI
		{
			name:   "basic k launch",
			cmd:    fmt.Sprintf("k launch -c load-test-http --repoURL %v --set url=https://httpbin.org/get --set duration=2s", srv.URL()),
			golden: base.CompletePath("../testdata", "output/klaunch.txt"),
		},
	}

	runTestActionCmd(t, tests)

}