page_title: Formatter

# Formatter API

> Note: The requirements for the formatter are still in flux for this beta

The srclib tool will expose a structured API, thus allowing easy access to the
data without manual parsing of the file. In order to derive some generalized
information from language specific data, a formatter must be developed for each
language. This formatter should implement the following transformations, that
convert the raw language data to useful information.

## Definition

The following formatter functions should take in a Def object, and use the
information provided, as well as the language specific Data field, to generate a
language agnostic result.

### GetType(Def)

The purpose of this function is to provide some general understanding of the
"type" of a function. This function can return one of the following.

1. A DefKey object, directly linking an object to its type's definition in code
1. A string containing the name of the type of the definition, for instances
   where the type cannot be expressed by a single DefKey (eg: Array<int>,
   (String, Int))
1. An array, containing a mix of 1 and 2 - this allows for the presence of union
   types in type inference

#### Union Types

Union types can occur for dynamic languages such as JavaScript and Python, where
type inference may be able to narrow down the type of a variable to two or three
possibilities. This should be specified through the array format, where each
type is an element in an array.

### GetRelationships(Def)

Various types of relationships between definitions, such as classical and
prototypical inheritance, can be represented as a mapping from a string to a
list of definitions. This function should export those, using the specific
vocabulary of the language itself.

```json
{
  "Extends" : [
  	DefKey
  ],
  "Implements" : [
  	DefKey,
  	DefKey
  ],
  "Prototype" : [
  	DefKey
  ]
}
```

### GetSignature(Def)

This returns the signature of a callable, including input and return values for
any function, constructor, or method. This is a generic format that provides
support for multiple or single return values, positional arguments, and named
arguments. Any of the three fields can be omitted.

```json
{
	"Returns" : [
		DefKey or String
	],
	"PositionalArguments" : [
		DefKey or String,
		DefKey or String
	]
	"NamedArguments" : {
		"namedarg" : DefKey or String
	}
}
```
