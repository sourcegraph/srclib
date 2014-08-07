page_title: Grapher Output

# Grapher Output

Src will invoke the grapher, providing a JSON representation of a source unit (`*unit.SourceUnit`)
in through stdin.

## Output Schema

The output is a single JSON object with three fields that represent lists of
Definitions, References, and Documentation data respectively. This should be printed to stdout.

[[.code "grapher/grapher.go" "Output"]]

### Def Object Structure
[[.code "graph/def.go" "Def"]]

### Ref Object Structure
[[.code "graph/ref.go" "Ref"]]

### Docs Object Structure
[[.code "graph/doc.go" "Doc"]]

## Example
> Updated Example needed
