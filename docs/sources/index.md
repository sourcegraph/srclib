# What is Srclib?

The srclib project aims to provide the following functionality in a language agnostic fashion:

1. Jump to definition support, both within local code bases, as well as across online repositories
2. Given a definition, find examples of its use across open source code
3. Search and find documentation for modules and functions
4. Expose an API that makes it easy to query the results of static analysis
5. Be architected in a way that makes it painless to integrate new languages

# Next Steps
### Download an Editor Plugin

If you are interested in using the editor plugins that we have available, check out the
[Editors](installation/editor-plugins.md) page, and download the plugin for your favorite editor.

### Add Support For Your Favorite Editor

If you want to help build/improve editor plugins, or simply hack on srclib, first read
through the docs on the [src](src/overview.md) tool. Then, check out the [API](api/overview.md).

### Help Add More Languages
Finally, if you want to help build out the language analysis infrastructure, make sure you're
familiar with the [src](src/overview.md) executable. Then, you should read closely over our
[Language Analysis](language-analysis/overview.md) section.
