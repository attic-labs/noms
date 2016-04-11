#!/usr/bin/python

import copy, os, os.path, subprocess, sys

sys.path.append(os.path.abspath('../../tools'))
import noms.symlink as symlink

SRC = ['babel-regenerator-runtime', 'src/main.js']
OUT = 'out.js'

def main():
	env = copy.copy(os.environ)
	env['NODE_ENV'] = sys.argv[1]
	env['BABEL_ENV'] = sys.argv[1]

	symlink.Force('../../js/.babelrc', os.path.abspath('.babelrc'))
	symlink.Force('../../js/.eslintrc', os.path.abspath('.eslintrc'))
	symlink.Force('../../js/.flowconfig', os.path.abspath('.flowconfig'))

	subprocess.check_call(['node_modules/.bin/webpack'] + SRC + [OUT], env=env, shell=False)


if __name__ == "__main__":
	main()
