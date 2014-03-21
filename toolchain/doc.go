// Package toolchain provides a generic interface around language-specific tasks
// and tools, such as dependency resolution and graphing.
//
// To create a toolchain and make it accessible to Sourcegraph, implement (at a
// minimum) the Toolchain interface and call the Register function at init time
// from the new toolchain's package.
package toolchain
