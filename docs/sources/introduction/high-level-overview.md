page_title: High Level Overview

# High Level Overview

## Source Units
The primary element of organization is the source unit. Source units correspond to individual Pip Packages, NPM Packages, RubyGems, or other forms of package distribution. Source units must also be associated with a single VCS repository, to allow the actual extraction of code. Repositories can, however, contain multiple Source Units - for example the Rails repository contains multiple ruby gems.

## Code Graph
For each source unit, a graph is generated from the associated code, consisting of
**definitions** and **references**. A definition is the original point in
code where an object is defined, be it a function, module, or class. References are
anywhere that symbol is then later used in code - references then link back to the
original definition. References can link to definitions in other source units and repositories.

## Toolchain
A primary tool called ‘src’ will serve as a harness for all of the individual language toolchains. It will also serve as the API endpoint, for users to query for data. Each language-specific toolchain is composed of five parts - any language that implements these can be automatically used with srclib.

1. **Scanner** - Traverses the directory tree looking for source units. Invokes the grapher and dependency lister on each unit.
2. **Grapher** - Produces the code graph through static analysis.
3. **Dependency Lister** - Lists all of the dependencies that a source unit uses - for example the raw PipPackage or NPM package names.
4. **Dependency Resolver** - Resolves raw dependency names to VCS repository urls.
5. **Formatter** - Exports functions that the API uses to transform raw language-specific data into a generic, useable structure. This is the only tool that must be written in Go.

This separation allows certain parts to be implemented before others, allowing incremental benefits to accrue before the toolchain is complete. For example, the grapher could be implemented before proper dependency resolution, allowing jump to definition and documentation only within a local codebase. The dependency resolution could be implemented without the grapher, allowing dependency tracking, without code graphing.

The src tool, after invoking the various elements of the toolchain, will extract other information, such as the repository url, the commit id, and use blaming tools to find authorship information for definitions and references.
