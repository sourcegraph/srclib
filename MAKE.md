# `src make`

The `src make` command is the top-level task runner for the entire analysis
process. It should be run from the top-level source directory of a project. When
run, it determines the tasks that need to be run (based on manual configuration
and automatic detection and scanning) and then runs them.

When you run `src make` in a source directory, it executes tasks in 3 phases:

1. Configure
1. Plan
1. Execute

## Phase 1. Configure

First, `src config` determines *what to do* using a combination of the Srcfile
manual configuration file (if present) and the scanners available in the
SRCLIBPATH.

1. Read the manual configuration in Srcfile, if present.
1. Determine which scanners to run, based on the list of default scanners and
   the Srcfile.
1. Run each scanner to produce a list of discovered source units.
1. Merge the manually specified source units with the output from the scanners.
   (Manually specified source units take precedence.)
1. Eliminate source units that are skipped in the Srcfile.

The final product of the configuration phase is the final configuration file.

## Phase 2. Plan

Next, `src plan` generates a Makefile that, when run, will run the necessary
tasks and save each task's output to a file.

In the Makefile, each target corresponds to a task. The targets are JSON files
in the `.srclib-cache` directory with names that encode the tool
operation and source unit (if any):
`${SOURCE_UNIT_NAME}:${SOURCE_UNIT_TYPE}_${OPERATION}.json`.

The final product of the planning phase is this Makefile.

## Phase 3. Execute

Finally, `src make` executes the Makefile produced in the prior planning
phase.

The final products of the execution phase are the target JSON files containing
the results of executing the tools as specified in the Makefile.
