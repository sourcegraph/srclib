# Srcfile

TODO(sqs): this is totally old and inaccurate

A Srcfile configures how srclib handles a source tree. The file consists of
instructions, one per line. Order does not matter. Lines starting with `#` are
comments.

Example:

```
# Skip all handling of Ruby gem source units
SKIP ALL rubygem:*

# Skip graphing for the Python pip package named "foo"
SKIP graph PipPackage:foo

# Override to use a custom Python grapher for the Python pip package named "foo"
TOOL depresolve PipPackage:foo github.com/alice/srclib-python-custom`

# Set custom config visible to all tools building Go packages
CONFIG GoPackage:* BaseImportPath "github.com/foo/bar"`

# Required instruction until this file format becomes more stable,
# to set the right expectations
I-UNDERSTAND-THIS-FORMAT-IS-PRERELEASE-AND-SUBJECT-TO-CHANGE
```

## Instructions

### SKIP <tool|ALL> unit

### TOOL <tool|ALL> unit preferred-tool

### CONFIG unit key json-value

### I-UNDERSTAND-THIS-FORMAT-IS-PRERELEASE-AND-SUBJECT-TO-CHANGE

This format is prerelease and **will** change. If you add a Srcfile to your
project, you should be prepared to update it frequently whenever the format
changes. This instruction will be removed when the format becomes more stable.
