package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunRun(t *testing.T) {
	// valid run task... should succeed
	rt := &runTask{
		TaskMeta: TaskMeta{
			Run: StringPointer("echo hello"),
		},
	}

	exp := &Experiment{
		Tasks:  []Task{rt},
		Result: &ExperimentResult{},
	}
	exp.initResults()
	err := rt.run(exp)
	assert.NoError(t, err)
}