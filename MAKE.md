# `src make`

The `src make` command is the top-level task runner for all operations. It
should be run from the top-level source directory of a project. When run, it
determines the tasks that need to be run (based on manual configuration and
automatic detection and scanning) and then runs them.

## Phases and steps

When you run `src make` in a source directory, it executes tasks in 2 phases:
configuration and execution.

### Phase 1. Configuration

First, `src make` determines *what to do* using a combination of the Srcfile
manual configuration file (if present) and the scanners available in the
SRCLIBPATH.

1. Read the manual configuration in Srcfile, if present.
1. Determine which scanners to run, based on the list of available scanners and
   the Srcfile.
1. Run each scanner to produce lists of discovered source units and operations
   to run on them.
1. Merge the manually specified source units and operations with the output from
   the scanners. (Manually specified source units take precedence.)
1. Eliminate source units and operations that are SKIPped in the Srcfile.
1. Cache the final configuration (source units, operations, and global config).
1. Generate a Makefile that, when run, will execute the necessary operations and
   save the results to disk.

The final products of the configuration phase are a cached final configuration
file and a Makefile used in the execution phase.

### Phase 2. Execution

Next, `src make` executes the Makefile produced in the prior configuration
phase.

In the Makefile, each target corresponds to an operation (either on a single
source unit or on the whole project). The targets are JSON files in the
`.srclib-data/COMMIT-ID` directory with names that encode the operation and
source unit (if any):
`${SOURCE_UNIT_NAME}:${SOURCE_UNIT_TYPE}_${OPERATION}.json`.

The final products of the execution phase are the target JSON files containing
the results of executing the operations.
