#!/usr/bin/env python

import jinja2
import os
import re
import shlex

import mkdocs.build
from mkdocs.build import build
from mkdocs.config import load_config

def line_containing(lines, text):
  for i in range(len(lines)):
    if text.lower() in lines[i].lower():
      return i

# Wrap some functions to allow custom commands in markdown
convert_markdown_original = mkdocs.build.convert_markdown
def convert_markdown_new(source):

  def expand(match):
    args = shlex.split(match.groups()[0])

    # Source code embeds
    if args[0] == ".code":
      lines = open("../" + args[1]).read().splitlines()

      # Short hand for specifying a region
      if len(args) == 3:
        region = args[2]
        args[2] = "START " + region
        args.append("END " + region)

      if len(args) == 4:
        start = 1
        end = len(lines) - 1

        if args[2].isdigit(): start = int(args[2])
        else: start = line_containing(lines, args[2]) + 1

        if args[3].isdigit(): end = int(args[3])
        else: end = line_containing(lines, args[3]) + 1

        #TODO: Also allow regex matching

        lines = lines[start - 1:end]

      # Trim "OMIT" lines
      lines = filter(lambda x: not x.strip().lower().endswith("omit"), lines)

      lines.insert(0, "```go")
      lines.append("```")
      return "\n".join(lines)

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
