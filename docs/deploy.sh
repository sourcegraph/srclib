#!/usr/bin/env bash

# Compile the css file
sass theme/styles.scss:theme/styles.css

# Build the doc pages
mkdocs build

# Build the other parts of the site
python buildsite.py

# Sync site with S3 bucket
aws s3 sync site/ s3://srclib.org/
