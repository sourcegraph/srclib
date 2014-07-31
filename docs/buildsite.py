#!/usr/bin/env python

import jinja2
import os

if __name__ == "__main__":
  # Load templates
  template_env = jinja2.Environment(loader = jinja2.FileSystemLoader(os.path.join(os.path.dirname(__file__), 'theme')))
  index_template = template_env.get_template('home.html')
  community_template = template_env.get_template('community.html')

  with open('site/index.html', 'w') as f:
    f.write(index_template.render(
      page="home"
    ))

  with open('site/community.html', 'w') as f:
    f.write(community_template.render(
      page="community"
    ))
