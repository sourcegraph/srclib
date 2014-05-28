// Package unit provides a source unit abstraction over distribution packages in
// various languages.
//
// To define a new source unit, call Register with a name and a type. The type
// must implement SourceUnit. To display additional information about the source
// unit, implement Info as well.
package unit
