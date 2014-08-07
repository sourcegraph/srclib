#!/usr/bin/env python

import jinja2
import os
import re
import shlex

import mkdocs.build
from mkdocs.build import build
from mkdocs.config import load_config

# Wrap some functions to allow custom commands in markdown
convert_markdown_original = mkdocs.build.convert_markdown
def convert_markdown_new(source):

  def expand(match):
    args = shlex.split(match.groups()[0])

    # Source code embeds
    if args[0] == ".code":
      lines = open("../" + args[1]).read().splitlines()
      if len(args) == 4:
        lines = lines[int(args[2]) - 1:int(args[3]) + 1]
      return "```go\n" + "\n".join(lines) + "\n```"

    # No matching logic
    else:
      return match.group(0)
  source = re.sub("\[\[(.*)\]\]", expand, source)

  return convert_markdown_original(source)

# Hotpatch in the markdown conversion wrapper
mkdocs.build.convert_markdown = convert_markdown_new




if __name__ == "__main__":
  # Build documentation
  config = load_config(options=None)
  build(config)

  # Load templates
  template_env = jinja2.Environment(loader = jinja2.FileSystemLoader(os.path.join(os.path.dirname(__file__), 'theme')))
  index_template = template_env.get_template('home.html')
  community_template = template_env.get_template('community.html')
  about_template = template_env.get_template('about.html')

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

  # About page
  with open('site/about.html', 'w') as f:
    f.write(about_template.render(
      page="about"
    ))
