package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/iter8-tools/iter8/base"
	id "github.com/iter8-tools/iter8/driver"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKAssert(t *testing.T) {
	os.Chdir(t.TempDir())
	tests := []cmdTestCase{
		// k launch
		{
			name:   "k launch",
			cmd:    fmt.Sprintf("k launch -c iter8 --chartsParentDir %v --noDownload --set tasks={http} --set http.url=https://httpbin.org/get --set http.duration=2s", base.CompletePath("../", "")),
			golden: base.CompletePath("../testdata", "output/klaunch.txt"),
		},
		// k run
		{
			name: "k run",
			cmd:  "k run -g default --namespace default",
		},
		// k assert
		{
			name:   "k assert",
			cmd:    "k assert -c completed -c nofailure -c slos",
			golden: base.CompletePath("../testdata", "output/kassert.txt"),
		},
	}

	// mock the environment
	base.SetupWithMock(t)
	// fake kube cluster
	*kd = *id.NewFakeKubeDriver(settings)
	byteArray, _ := ioutil.ReadFile(base.CompletePath("../testdata", id.ExperimentPath))
	kd.Clientset.CoreV1().Secrets("default").Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "default",
		},
		StringData: map[string]string{id.ExperimentPath: string(byteArray)},
	}, metav1.CreateOptions{})

	kd.Clientset.BatchV1().Jobs("default").Create(context.TODO(), &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-1-job",
			Namespace: "default",
		},
	}, metav1.CreateOptions{})

	runTestActionCmd(t, tests)
}
