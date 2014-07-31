# Srclib Documentation Site

The srclib documentation files are written in Markdown and converted to a
browsable HTML static site, hosted at [srclib.org](http://srclib.org/).
**[MkDocs](http://www.mkdocs.org/)**, installable through pip, is used to generate the
HTML.

Run `pip install mkdocs` to install MkDocs.

You will also need SASS, in order to compile the style sheets. This can be obtained with `gem install sass`.
You can then run `sass theme/styles.scss:theme/styles.css` to quickly build the css.

When testing documentation, you can type `mkdocs serve` to start the server,
or `mkdocs build` to build the HTML into the `site` directory.

## Editing
Markdown files are located in the `sources` directory, with subdirectories intended to correspond
with the drop-down menus on the site. The actual categories, however, are specified through configuration
in `mkdocs.yml`.

## Deploying
The site can be deployed to srclib.org simply py running `./deploy.sh`. This builds the SASS stylesheed,
builds the docs, and builds the other static pages of the site. It then uses aws s3 sync to store these in the correct bucket.
