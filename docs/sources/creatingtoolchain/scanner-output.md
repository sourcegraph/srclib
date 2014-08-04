page_title: Scanner Output

# Scanner Output

The scanner should descend through the directory on which it was invoked,
searching for source units. For every source unit, a file should be created in
the `.sourcegraph-data` directory, following the naming convention
`{UnitName}@{UnitType}_info.{SchemaVersion}.json`.

## Output Schema

The data in the output file should consist of a single JSON object that conforms
to the data in this struct.

```go
type SourceUnit struct {
	// The name of the package
	UnitName	string
	// The type of source unit - eg: "PipPackage", "RubyGem", "CommonJSPackage"
	UnitType	string

	// Optional field - used to store language specific information about the source unit
	Data		*types.JsonText

	// List of files in the SourceUnit, relative to directory in which the scanner was invoked. Can be an empty list. Used to derive blame information
	Files		[]*string
}
```

## Example: rails@rubygem_info.v0.json

```json
{
	"UnitName" : "rails",
	"UnitType" : "rubygem",

	"Data" : {
		"Description" : "Ruby on Rails is a full-stack web framework opt...",
		"Homepage": "http://www.rubyonrails.org",
		...
	},
	"Files" : [
		"activerecord/CHANGELOG.md",
		"activerecord/MIT-LICENSE",
		...
	]
}
```
