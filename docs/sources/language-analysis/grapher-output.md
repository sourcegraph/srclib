page_title: Grapher Output

# Grapher Output
The scanner should invoke the grapher with enough information about the source unit. The output of the grapher should be saved in a file with the name {UnitName}@{UnitType}_graph.{SchemaVersion}.json

## Output Schema
The output is a single JSON object with three fields that represent lists of Definitions, References, and Documentation data respectively.

```go
type Graph struct {
	Defs		[]*Def
	Refs		[]*Ref
	Docs		[]*Doc
}

type Def struct {

	// This path should be unique for all symbols in the Source Unit.
	Path		string

	// Similar to the path, but does not have a uniqueness guarantee. However, this should be used to convey some level of structural information. For example, given a method’s tree path, removing the last component should provide the TreePath for the class
	TreePath	string

	Name		string

	Location	Location

	// Whether or not the definition is located in test code
	Test		bool

	// Whether or not the definition is exported through the public api, or only available locally
	Exported	bool

	// A flexible field intended to hold as much language specific data as possible. Enough data should be stored to be able to implement the specifications of the Formatter API
	Data		types.JsonText
}

type Ref struct {
	// The definition that this reference points to
	Target		DefKey
	// Is this reference the original definition or a redefinition
	IsDef		bool

	Location	Location
}

type Doc struct  {
	// A link to the definition that this docstring describes
	Def		DefKey

	// The MIME-type that the documentation is stored in. Valid formats include ‘text/html’, ‘text/plain’, ‘text/x-markdown’, text/x-rst‘
	Format		string

	// The actual documentation text
	Data		string

	// Location where the docstring was extracted from. Leave blank for undefined location
	Location	*Location
}
```

## Example: rails@rubygem_graph.v0.json
```json
{
	“Defs” : [
		{
			“Path” : “RailsGuides/Markdown/Renderer/$methods/convert_notes”,
			“TreePath” : “./RailsGuides/Markdown/Renderer/convert_notes”,

			“Name” : “convert_notes”,
			“Location” : {
				“File” : “guides/rails_guides/markdown/renderer.rb”,
				“StartByte” : 1624,
				“EndByte” : 2558
			},

			“Test” : false,
			“Exported” : true,
			“Data” : {
				"RubyKind": "method",
				"TypeString" : "RailsGuides::Markdown::Renderer#convert_notes",
				"Module": "RailsGuides",
				"RubyPath": "RailsGuides::Markdown::Renderer#convert_notes",
			}
		},
		...
	],
	“Refs” : [
		{
			“Target” : {
				“Path” : “TestController”
			},
			“IsDef” : true,
			“Location” : {
				“File” : “guides/bug_report_templates/action_controller_gem.rb”,
				“StartByte” : 488,
				“EndByte” : 502
			}
		},
		...
	],
	“Docs” : [
		{
			“Def” : {
			},
			“Format” : “text/html”,
			“Data” : “This code is based directly on the Text gem...”,
			“Location” : {
				“File” : "guides/rails_guides/levenshtein.rb",
				“StartByte” : 0,
				“EndByte” : 0
			}
		},
		...
	]
}
```
