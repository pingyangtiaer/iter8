package cmd

import (
	"context"
	"io/ioutil"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	id "github.com/iter8-tools/iter8/driver"

	"github.com/iter8-tools/iter8/base"
)

func TestKReport(t *testing.T) {

	// fake kube cluster
	*kd = *id.NewFakeKubeDriver(settings)
	kd.Revision = 1
	byteArray, _ := ioutil.ReadFile(base.CompletePath("../testdata/assertinputs", "experiment.yaml"))
	kd.Clientset.CoreV1().Secrets("default").Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-1-spec",
			Namespace: "default",
		},
		StringData: map[string]string{"experiment.yaml": string(byteArray)},
	}, metav1.CreateOptions{})

	byteArray, _ = ioutil.ReadFile(base.CompletePath("../testdata/assertinputs", "result.yaml"))
	kd.Clientset.CoreV1().Secrets("default").Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-1-result",
			Namespace: "default",
		},
		StringData: map[string]string{"result.yaml": string(byteArray)},
	}, metav1.CreateOptions{})

	tests := []cmdTestCase{
		// k report
		{
			name:   "k report",
			cmd:    "k report",
			golden: base.CompletePath("../testdata", "output/kreport.txt"),
		},
	}

	runTestActionCmd(t, tests)
}