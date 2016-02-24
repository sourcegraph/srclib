#!/usr/bin/env python

import jinja2
import os

import mkdocs.build
from mkdocs.build import build
from mkdocs.config import load_config

if __name__ == "__main__":
  # Build documentation
  config = load_config(options=None)
  build(config)

  # Load templates
  template_env = jinja2.Environment(loader = jinja2.FileSystemLoader(os.path.join(os.path.dirname(__file__), 'theme')))
  index_template = template_env.get_template('home.html')
  community_template = template_env.get_template('community.html')

  # Home page
  with open('site/index.html', 'w') as f:
    f.write(index_template.render(
      page="home"
    ))

  # Community page
  with open('site/community.html', 'w') as f:
    f.write(community_template.render(
      page="community"
    ))
