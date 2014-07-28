# Srclib Documentation Site

The srclib documentation files are written in Markdown and converted to a
browsable HTML static site, hosted at [srclib.org](http://srclib.org/).
**[MkDocs](http://www.mkdocs.org/)**, installable through pip, is used to generate the
HTML.

Run `pip install mkdocs` command to install MkDocs.

When testing documentation, you can type `mkdocs serve` to start the server, or `mkdocs build` to generate
the site into a directory labelled `site`.

## Editing
Markdown files are located in the `sources` directory, with subdirectories intended to correspond
with the drop-down menus on the site. The actual categories, however, are specified through configuration
in `mkdocs.yml`.
