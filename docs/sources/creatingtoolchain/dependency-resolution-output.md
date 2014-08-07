page_title: Dependency Resolution Output

# Dependency Resolution Output

The output of the dependency resolution tool should be printed to standard
output.

## Output Schema

The schema of the dependency resolution output should be an array of
`Resolution` objects, with the structure of each object being as follows.

[[.code "dep/resolve.go" "Resolution"]]

The `Raw` field is language specific, but the Target field follows the following format.

[[.code "dep/resolve.go" "ResolvedTarget"]]

If an error occurred during resolution, a detailed description should be placed in the `Error` field.

## Example

> Updated Example needed
