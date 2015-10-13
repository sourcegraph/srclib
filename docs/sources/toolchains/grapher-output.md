page_title: Grapher Output

# Grapher Output

Src will invoke the grapher, providing a JSON representation of a source unit (`*unit.SourceUnit`)
in through stdin.

## Output Schema

The output is a single JSON object with three fields that represent lists of
Definitions, References, and Documentation data respectively. This should be printed to stdout.

[[.code "https://raw.githubusercontent.com/sourcegraph/srclib/bf4ec15991ed05161dad3694f8729d48c5124844/graph/output.pb.go" 15 20]]

### Def Object Structure
[[.code "https://raw.githubusercontent.com/sourcegraph/srclib/bf4ec15991ed05161dad3694f8729d48c5124844/graph/def.pb.go" 73 122]]

### Ref Object Structure
[[.code "https://raw.githubusercontent.com/sourcegraph/srclib/bf4ec15991ed05161dad3694f8729d48c5124844/graph/ref.pb.go" 14 44]]

### Docs Object Structure
[[.code "https://raw.githubusercontent.com/sourcegraph/srclib/bf4ec15991ed05161dad3694f8729d48c5124844/graph/doc.pb.go" 14 30]]

## Example: Grapher output on [jashkenas/underscore](https://github.com/jashkenas/underscore)
```json
{
  "Defs": [
    {
      "Path": "commonjs/test/arrays.js",
      "TreePath": "-commonjs/test/arrays.js",
      "Kind": "module",
      "Exported": true,
      "Data": {
        "Kind": "commonjs-module",
        "Key": {
          "namespace": "commonjs",
          "module": "test/arrays.js",
          "path": ""
        },
        "jsgSymbolData": {
          "nodejs": {
            "moduleExports": true
          }
        },
        "Type": "{}",
        "IsFunc": false
      },
      "Name": "test/arrays",
      "File": "test/arrays.js",
      "DefStart": 0,
      "DefEnd": 0
    },
    ...
  ],
  "Refs" : [
    {
      "DefRepo": "",
      "DefUnitType": "",
      "DefUnit": "",
      "DefPath": "commonjs/underscore.js/-/union",
      "File": "test/arrays.js",
      "Start": 7610,
      "End": 7615
    },
    ...
  ],
  "Docs" : [
    {
      "Path": "commonjs/test/vendor/qunit.js/-/jsDump/parsers/functionArgs",
      "Format": "",
      "Data": "function calls it internally, it's the arguments part of the function"
    },
    ...
  ]
```
