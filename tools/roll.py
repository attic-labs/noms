#!/usr/bin/python

# Copyright 2016 Attic Labs, Inc. All rights reserved.
# Licensed under the Apache License, version 2.0:
# http://www.apache.org/licenses/LICENSE-2.0

import argparse, os, os.path, subprocess, sys, shutil, urlparse


def main():
  parser = argparse.ArgumentParser(description='Dependency snapshotter')
  parser.add_argument('url')
  parser.add_argument('--path', help=(
      'path to store the dependency at, defaults to vendor/[url without protocol]'))
  parser.add_argument('--incl', action='append', help=(
      'subdirectories of the dependency to check out, relative to the path. '
      'Defaults to root. Evaluated before --excl.'))
  parser.add_argument('--excl', action='append', help=(
      'subdirectories of the dependency to delete after checking out, relative to the path. '
      'Defaults to nothing. Evaluated after --incl.'))
  parser.add_argument('--version', default='HEAD', help=(
    'version of the dependency to snapshot, defaults to HEAD'))

  args = parser.parse_args()

  url = urlparse.urlparse(args.url)
  if url.scheme == '':
    print 'Invalid url: no scheme'
    sys.exit(1)

  def rel(subdir):
    if subdir is not None and os.path.isabs(subdir):
      print 'subdirectory %s must be a relative path' % subdir
      sys.exit(1)
    return subdir

  incl = None
  if args.incl is not None:
    incl = [rel(i) for i in args.incl]

  excl = None
  if args.excl is not None:
    excl = [rel(e) for e in args.excl]

  if not os.path.isdir('.git'):
    print '%s must be run from the root of a repository' % sys.argv[0]
    sys.exit(1)

  path = url.path
  if path.startswith('/'):
    path = path[1:]
  if path.endswith('.git'):
    path = path[0:len(path) - 4]

  depdir = args.path
  if depdir is None:
    depdir = os.path.join('vendor', url.netloc, path)

  shutil.rmtree(depdir, True)
  parent = os.path.dirname(depdir)
  if not os.path.isdir(parent):
    os.makedirs(parent)
  os.chdir(parent)

  # Kinda sucks to clone entire repo to get a particular version, but:
  # http://stackoverflow.com/questions/3489173/how-to-clone-git-repository-with-specific-revision-changeset
  subprocess.check_call(['git', 'clone', args.url])

  os.chdir(os.path.basename(depdir))
  subprocess.check_call(['git', 'reset', '--hard', args.version])
  head = subprocess.check_output(['git', 'rev-parse', 'HEAD']).strip()

  f = open('.version', 'w')
  f.write('%s\n%s\n' % (args.url, head))
  f.close()

  shutil.rmtree('.git')

  if os.path.isdir('vendor'):
    deps = [dirName for dirName, _, files in os.walk('vendor') if files]
    if deps:
      print '\nWarning!'
      print ' %s contains one or more dependencies which will need to be vendored as well:' % args.url
      print ' -', '\n - '.join(deps)
    shutil.rmtree('vendor')

  if incl is not None:
    inclPaths = []
    inclParentToName = {}
    for dir in incl:
      if not os.path.isdir(dir):
        print 'Warning: --incl directory %s does not exist, skipping.' % dir
      else:
        path = os.path.abspath(dir)
        parent, name = os.path.split(path)
        inclPaths.append(path)
        inclParentToName[parent] = name

    def inclPathStartsWith(dir):
      for p in inclPaths:
        if p.startswith(dir):
          return True
      return False

    for (dirpath, dirnames, _) in os.walk(os.getcwd()):
      if dirpath in inclParentToName:
        inclName = inclParentToName[dirpath]
        # Don't descend into the included subdirectory.
        dirnames.remove(inclName)
      elif not inclPathStartsWith(dirpath):
        # Remove directories that aren't an ancestor of the included.
        print 'rm subdirectory: %s' % dirpath
        shutil.rmtree(dirpath)

  if excl is not None:
    for dir in excl:
      if not os.path.isdir(dir):
        print 'Warning: --excl directory %s does not exist, skipping.' % dir
      else:
        exclPath = os.path.abspath(dir)
        print 'rm subdirectory: %s' % exclPath
        shutil.rmtree(exclPath)


if __name__ == '__main__':
  main()
