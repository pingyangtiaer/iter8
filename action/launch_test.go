package action

import (
	"os"
	"testing"

	"github.com/iter8-tools/iter8/base"
	"github.com/iter8-tools/iter8/driver"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/cli"
)

func TestLocalLaunch(t *testing.T) {
	os.Chdir(t.TempDir())
	base.SetupWithMock(t)

	// fix lOpts
	lOpts := NewLaunchOpts(driver.NewFakeKubeDriver(cli.New()))
	lOpts.RemoteFolderURL = "github.com/sriumcp/iter8.git?ref=ic//charts"
	// // TODO: fix git ref
	// lOpts.RemoteFolderURL = defaultIter8Repo + "?ref=v0.11.0" + "//" + chartsFolderName
	lOpts.ChartName = "iter8"
	lOpts.Values = []string{"tasks={http}", "http.url=https://httpbin.org/get", "http.duration=2s"}

	err := lOpts.LocalRun()
	assert.NoError(t, err)
}

func TestKubeLaunch(t *testing.T) {
	var err error
	os.Chdir(t.TempDir())

	// fix lOpts
	lOpts := NewLaunchOpts(driver.NewFakeKubeDriver(cli.New()))
	lOpts.RemoteFolderURL = "github.com/sriumcp/iter8.git?ref=ic//charts"
	// // TODO: fix git ref
	// lOpts.RemoteFolderURL = defaultIter8Repo + "?ref=v0.11.0" + "//" + chartsFolderName
	lOpts.ChartName = "iter8"
	lOpts.Values = []string{"tasks={http}", "http.url=https://httpbin.org/get", "http.duration=2s", "runner=job"}

	err = lOpts.KubeRun()
	assert.NoError(t, err)

	rel, err := lOpts.Releases.Last(lOpts.Group)
	assert.NotNil(t, rel)
	assert.Equal(t, 1, rel.Version)
	assert.NoError(t, err)
}

func TestLocalLaunchNoDownload(t *testing.T) {
	os.Chdir(t.TempDir())
	base.SetupWithMock(t)

	// fix lOpts
	lOpts := NewLaunchOpts(driver.NewFakeKubeDriver(cli.New()))
	lOpts.ChartsParentDir = base.CompletePath("../", "")
	lOpts.ChartName = "iter8"
	lOpts.NoDownload = true
	lOpts.Values = []string{"tasks={http}", "http.url=https://httpbin.org/get", "http.duration=2s"}

	err := lOpts.LocalRun()
	assert.NoError(t, err)
}

func TestKubeLaunchNoDownload(t *testing.T) {
	var err error
	os.Chdir(t.TempDir())

	// fix lOpts
	lOpts := NewLaunchOpts(driver.NewFakeKubeDriver(cli.New()))
	lOpts.ChartsParentDir = base.CompletePath("../", "")
	lOpts.ChartName = "iter8"
	lOpts.NoDownload = true
	lOpts.Values = []string{"tasks={http}", "http.url=https://httpbin.org/get", "http.duration=2s", "runner=job"}
	// // fixing git ref forever
	// lOpts.RemoteFolderURL = defaultIter8Repo + "?ref=v0.11.0" + "//" + chartsFolderName

	err = lOpts.KubeRun()
	assert.NoError(t, err)

	rel, err := lOpts.Releases.Last(lOpts.Group)
	assert.NotNil(t, rel)
	assert.Equal(t, 1, rel.Version)
	assert.NoError(t, err)
}
