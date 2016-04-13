#!/usr/bin/python

import os, subprocess, sys

SRC = 'src/main.js'
OUT = 'out.js'

def main():
    env = os.environ
    env['NODE_ENV'] = sys.argv[1]
    env['BABEL_ENV'] = sys.argv[1]
    subprocess.check_call(
            ['node_modules/.bin/webpack',
                '--config', 'node_modules/@attic/webpack-config/index.js', SRC, OUT],
            env=env, shell=False)


if __name__ == "__main__":
    main()
