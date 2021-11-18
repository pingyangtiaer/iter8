package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/antonmedv/expr"
	"github.com/iter8-tools/iter8/base"
	"github.com/iter8-tools/iter8/base/log"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run an experiment",
	Long:  `Run an experiment. This command will read the experiment spec from the local file named experiment.yaml, and write the result of the experiment run to the local file named result.yaml.`,
	Example: `
	# download the load-test experiment
	iter8 hub -e load-test
	
	cd load-test

	# run it
	iter8 run
	`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Logger.Trace("build called")
		exp, err := build(false)
		log.Logger.Trace("build finished")
		if err != nil {
			log.Logger.Error("experiment build failed")
			os.Exit(1)
		} else {
			log.Logger.Info("starting experiment run")
			err := exp.run()
			if err != nil {
				log.Logger.Error("experiment failed")
			} else {
				log.Logger.Info("experiment completed successfully")
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(runCmd)
}

// Run an experiment
func (e *experiment) run() error {
	var err error
	if e.Result == nil {
		e.InitResults()
	}
	if e.Result.StartTime == nil {
		err = e.setStartTime()
		if err != nil {
			return err
		}
	}
	for i, t := range e.tasks {
		log.Logger.Info("task " + fmt.Sprintf("%v: %v", i+1, t.GetName()) + " : started")
		shouldRun := true
		// if task has a condition
		if cond := base.GetIf(t); cond != nil {
			// condition evaluates to false ... then shouldRun is false
			program, err := expr.Compile(*cond, expr.Env(e), expr.AsBool())
			if err != nil {
				log.Logger.WithStackTrace(err.Error()).Error("unable to compile if clause")
				return err
			}

			output, err := expr.Run(program, e)
			if err != nil {
				log.Logger.WithStackTrace(err.Error()).Error("unable to run if clause")
				return err
			}

			shouldRun = output.(bool)
		}
		if shouldRun {
			err = t.Run(e.Experiment)
			if err != nil {
				log.Logger.Error("task " + fmt.Sprintf("%v: %v", i+1, t.GetName()) + " : " + "failure")
				e.failExperiment()
				return err
			}
			log.Logger.Info("task " + fmt.Sprintf("%v: %v", i+1, t.GetName()) + " : " + "completed")
		} else {
			log.Logger.WithStackTrace(fmt.Sprint("if clause evaluated to false: ", *base.GetIf(t))).Info("task " + fmt.Sprintf("%v: %v", i+1, t.GetName()) + " : " + "skipped")
		}

		e.incrementNumCompletedTasks()
		err = writeResult(e)
		if err != nil {
			return err
		}
	}
	return nil

}

// write experiment result to file
func writeResult(r *experiment) error {
	rBytes, err := yaml.Marshal(r.Result)
	if err != nil {
		log.Logger.WithStackTrace(err.Error()).Error("unable to marshal experiment result")
		return errors.New("unable to marshal experiment result")
	}
	err = ioutil.WriteFile(experimentResultPath, rBytes, 0664)
	if err != nil {
		log.Logger.WithStackTrace(err.Error()).Error("unable to write experiment result")
		return err
	}
	return err
}

func (e *experiment) setStartTime() error {
	if e.Result == nil {
		log.Logger.Warn("setStartTime called on an experiment object without results")
		e.Experiment.InitResults()
	}
	return nil
}

func (e *experiment) failExperiment() error {
	if e.Result == nil {
		log.Logger.Warn("failExperiment called on an experiment object without results")
		e.Experiment.InitResults()
	}
	e.Result.Failure = true
	return nil
}

func (e *experiment) incrementNumCompletedTasks() error {
	if e.Result == nil {
		log.Logger.Warn("incrementNumCompletedTasks called on an experiment object without results")
		e.Experiment.InitResults()
	}
	e.Result.NumCompletedTasks++
	return nil
}

/*
// Run the given action.
func (a *Action) Run(ctx context.Context) error {
	for i := 0; i < len(*a); i++ {
		log.Info("------ task starting")
		shouldRun := true
		exp, err := GetExperimentFromContext(ctx)
		if err != nil {
			return err
		}
		// if task has a condition
		if cond := (*a)[i].GetIf(); cond != nil {
			// condition evaluates to false ... then shouldRun is false
			program, err := expr.Compile(*cond, expr.Env(exp), expr.AsBool())
			if err != nil {
				return err
			}

			output, err := expr.Run(program, exp)
			if err != nil {
				return err
			}

			shouldRun = output.(bool)
		}
		if shouldRun {
			err := (*a)[i].Run(ctx)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
*/