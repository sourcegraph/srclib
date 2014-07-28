page_title: Dependency Resolution Output

# Dependency Resolution Output

The output of the dependency resolution tool should be printed to standard output.

## Output Schema
The schema of the dependency resolution output should be an array of `DependencyTarget` objects,
with the structure of each object being as follows.

```go
type DependencyTarget struct {
	SourceUnit	SourceUnitKey
	Version	string
}
```

## Example: rails@rubygem_deps.v0.json
```json
[
	{
		“SourceUnit” : {
			“Repo” : “bitbucket.org/ged/ruby-pg”,
			“Unit” : “pg”,
			“UnitType” : “rubygem”
		},
		“Version” : “\u003e= 0.11.0”
	},
	...
]
```
