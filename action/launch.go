package action

import (
	"path"

	"github.com/iter8-tools/iter8/base/log"
	"github.com/iter8-tools/iter8/driver"
	"helm.sh/helm/v3/pkg/cli/values"
)

// LaunchOpts are the options used for launching experiments
type LaunchOpts struct {
	// DryRun enables simulating a launch
	DryRun bool
	// RemoteFolderURL is the URL of the remote Iter8 experiment charts folder
	// Remote URLs can be any go-getter URLs like GitHub or GitLab URLs
	// https://github.com/hashicorp/go-getter
	RemoteFolderURL string
	// ChartsParentDir is the directory where `charts` is to be downloaded or is located
	ChartsParentDir string
	// NoDownload disables charts download.
	// With this option turned on, `charts` that are already present locally are reused
	NoDownload bool
	// ChartName is the name of the chart
	ChartName string
	// Options provides the values to be combined with the experiment chart
	values.Options
	// Rundir is the directory where experiment.yaml file is located
	RunDir string
	// KubeDriver enables Kubernetes experiment run
	*driver.KubeDriver
}

// NewHubOpts initializes and returns launch opts
func NewLaunchOpts(kd *driver.KubeDriver) *LaunchOpts {
	return &LaunchOpts{
		DryRun:          false,
		RemoteFolderURL: DefaultRemoteFolderURL(),
		ChartsParentDir: ".",
		NoDownload:      false,
		ChartName:       "",
		Options:         values.Options{},
		RunDir:          ".",
		KubeDriver:      kd,
	}
}

// LocalRun launches a local experiment
func (lOpts *LaunchOpts) LocalRun() error {
	log.Logger.Debug("launch local run started...")
	if !lOpts.NoDownload {
		// download chart from Iter8 hub
		hOpts := &HubOpts{
			RemoteFolderURL: lOpts.RemoteFolderURL,
			ChartsDir:       path.Join(lOpts.ChartsParentDir, chartsFolderName),
		}
		if err := hOpts.LocalRun(); err != nil {
			return err
		}
		log.Logger.Debug("hub complete")
	} else {
		log.Logger.Debug("using `charts` under ", lOpts.ChartsParentDir)
	}

	// gen experiment spec
	gOpts := GenOpts{
		Options:         lOpts.Options,
		ChartsParentDir: lOpts.ChartsParentDir,
		GenDir:          lOpts.RunDir,
		ChartName:       lOpts.ChartName,
	}
	if err := gOpts.LocalRun(); err != nil {
		return err
	}
	log.Logger.Debug("gen complete")

	// all done if this is a dry run
	if lOpts.DryRun {
		log.Logger.Info("dry run complete")
		return nil
	}

	// run experiment locally
	log.Logger.Info("starting local experiment")
	rOpts := &RunOpts{
		RunDir:     lOpts.RunDir,
		KubeDriver: lOpts.KubeDriver,
	}
	return rOpts.LocalRun()
}

// KubeRun launches a Kubernetes experiment
func (lOpts *LaunchOpts) KubeRun() error {
	// initialize kube driver
	if err := lOpts.KubeDriver.Init(); err != nil {
		return err
	}

	if !lOpts.NoDownload {
		// download chart from Iter8 hub
		hOpts := &HubOpts{
			RemoteFolderURL: lOpts.RemoteFolderURL,
			ChartsDir:       path.Join(lOpts.ChartsParentDir, chartsFolderName),
		}
		if err := hOpts.LocalRun(); err != nil {
			return err
		}
		log.Logger.Debug("hub complete")
	} else {
		log.Logger.Debug("using `charts` under ", lOpts.ChartsParentDir)
	}

	// update dependencies
	gOpts := GenOpts{
		Options:         lOpts.Options,
		ChartsParentDir: lOpts.ChartsParentDir,
		GenDir:          lOpts.RunDir,
		ChartName:       lOpts.ChartName,
	}
	driver.UpdateChartDependencies(gOpts.chartDir(), lOpts.EnvSettings)

	return lOpts.KubeDriver.Launch(gOpts.chartDir(), lOpts.Options, lOpts.Group, lOpts.DryRun)
}
