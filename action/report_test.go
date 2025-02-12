package action

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/iter8-tools/iter8/base"
	"github.com/iter8-tools/iter8/driver"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/cli"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLocalReportText(t *testing.T) {
	os.Chdir(t.TempDir())
	// fix rOpts
	rOpts := NewReportOpts(driver.NewFakeKubeDriver(cli.New()))
	rOpts.RunDir = base.CompletePath("../", "testdata/assertinputs")

	err := rOpts.LocalRun(os.Stdout)
	assert.NoError(t, err)
}

func TestLocalReportTextNoInsights(t *testing.T) {
	os.Chdir(t.TempDir())
	// fix rOpts
	rOpts := NewReportOpts(driver.NewFakeKubeDriver(cli.New()))
	rOpts.RunDir = base.CompletePath("../", "testdata/assertinputs/noinsights")

	err := rOpts.LocalRun(os.Stdout)
	assert.NoError(t, err)
}

func TestLocalReportHTML(t *testing.T) {
	os.Chdir(t.TempDir())
	// fix rOpts
	rOpts := NewReportOpts(driver.NewFakeKubeDriver(cli.New()))
	rOpts.RunDir = base.CompletePath("../", "testdata/assertinputs")
	rOpts.OutputFormat = HTMLOutputFormatKey

	err := rOpts.LocalRun(os.Stdout)
	assert.NoError(t, err)
}

func TestKubeReportText(t *testing.T) {
	os.Chdir(t.TempDir())
	base.SetupWithMock(t)
	// fix rOpts
	rOpts := NewReportOpts(driver.NewFakeKubeDriver(cli.New()))

	byteArray, _ := ioutil.ReadFile(base.CompletePath("../testdata/assertinputs", driver.ExperimentPath))
	rOpts.Clientset.CoreV1().Secrets("default").Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "default",
		},
		StringData: map[string]string{driver.ExperimentPath: string(byteArray)},
	}, metav1.CreateOptions{})

	err := rOpts.KubeRun(os.Stdout)
	assert.NoError(t, err)
}

func TestLocalReportHTMLNoInsights(t *testing.T) {
	os.Chdir(t.TempDir())
	// fix rOpts
	rOpts := NewReportOpts(driver.NewFakeKubeDriver(cli.New()))
	rOpts.RunDir = base.CompletePath("../", "testdata/assertinputs/noinsights")
	rOpts.OutputFormat = HTMLOutputFormatKey
	err := rOpts.LocalRun(os.Stdout)
	assert.NoError(t, err)
}
