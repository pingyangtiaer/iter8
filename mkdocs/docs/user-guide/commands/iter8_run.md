---
template: main.html
title: "Iter8 Run"
hide:
- toc
---

## iter8 run

run an experiment

### Synopsis

Run an experiment. This command will read the experiment spec from the local file named experiment.yaml, and write the result of the experiment run to the local file named result.yaml.

```
iter8 run [flags]
```

### Examples

```

	# download the load-test experiment
	iter8 hub -e load-test
	
	cd load-test

	# run it
	iter8 run
	
```

### Options

```
  -h, --help   help for run
```

### SEE ALSO

* [iter8](iter8.md)	 - metrics driven experiments

###### Auto generated by spf13/cobra on 17-Nov-2021