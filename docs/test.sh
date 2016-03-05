#!/bin/sh

set -e

# Create site dir if it does not exist
mkdir -p site

# Python server
cd site
python -m SimpleHTTPServer &
cd ..

# Kill python server on exit
trap "exit" INT TERM
trap "kill 0" EXIT

while true; do
  echo "Building site..."
  sassc theme/styles.scss theme/styles.css
  #mkdocs build
  python buildsite.py

  sleep 1
done
