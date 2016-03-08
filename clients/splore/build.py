#!/usr/bin/python

import os, os.path, subprocess, sys

sys.path.append(os.path.abspath('../../tools'))

import noms.symlink as symlink

def check_node_version():
	version_string = subprocess.check_output(['node', '--version'])
	if (version_string.find("v5")) != 0:
		print version_string + " is the wrong version of node. Must be v5 or greater"
		exit()

def main():
	check_node_version()
	symlink.Force('../../js/.babelrc', os.path.abspath('.babelrc'))
	symlink.Force('../../js/.eslintrc', os.path.abspath('.eslintrc'))
	symlink.Force('../../js/.flowconfig', os.path.abspath('.flowconfig'))

	subprocess.check_call(['npm', 'install'], shell=False)
	subprocess.check_call(['npm', 'run', 'build'], env=os.environ, shell=False)


if __name__ == "__main__":
	main()
