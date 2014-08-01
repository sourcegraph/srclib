page_title: Language Analysis Overview

# Language Analysis Overview

> NOTE: Before reading through this section, make sure that you first have a
> proper understanding of the `src` executable. Start with the
> [overview](../src/overview.md).

## Output Location

In the root directory that is being analyzed, there should be a directory
created named ".sourcegraph-data". In this directory, there should be a set of
files generated for each source unit present. The files to generate for each
source unit are listed below.

## Common Data Structures

```go
// All of the information required to uniquely reference a source unit
type SourceUnitKey struct {
	// An id uniquely identifying the repository where this source unit resides
	Repo		string
	// The name of the source unit
	Unit 		string
	// The type of unit, eg: PipPackage, or RubyGem
	UnitType	string
}

// All of the information required to uniquely reference a definition either locally, or across repositories
type DefKey struct {
	// Leave null or undefined to reference the current source unit
	SourceUnit	*SourceUnitKey

	// The path to the definition (unique for all definitions in a unit)
	Path		string
}

// Identify a specific set of characters in a specific file
type Location struct {
	// File path, relative to the directory in which the scanner was invoked.
	File		string

	// Start and end bytes for the identified region. This is measured in bytes, not characters, in order to allow for unicode characters. Set both values to zero to represent an unknown location in a known file
	StartByte	int
	EndByte	int
}
```
