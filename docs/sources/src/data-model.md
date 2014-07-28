# Sourcegraph data model

### Source unit

A source unit is code that Sourcegraph processes as one logical unit. It is
conceptually similar to a compilation unit.

A source unit consists of a type, a set of files and directories, along with
metadata specific to the type of source unit. Typically a source unit is the
level of code structure that creates a package or binary product that other
projects can depend on.

Examples include:

* A Go package
* A Node.js package
* A Ruby gem
* A Python pip package

A file or directory may appear in more than one source unit. For example, a
frontend JavaScript library and a node.js package might include the same
JavaScript file (to make it usable in the browser and in node.js, respectively).
Or 2 Ruby gems in the same repository might refer to the same `.rb` file.

Examples of things that would NOT be good source units and the reasons why:

* Things that you can import, include, or require in source code are not
  necessarily good source units. For example, a Python package can be imported
  in source code (`import mypkg`), but the information necessary to build a
  Python package is not contained within the package itself (it's usually in
  setup.py). And when an external project wants to depend on a Python package,
  they first must specify the dependency to the Python pip package containing
  that Python package. So, the pip package is the right level to use as the
  source unit.


NOTE: We haven't yet found a succinct way to describe a source unit. Modify this
document as we come up with clearer explanations.