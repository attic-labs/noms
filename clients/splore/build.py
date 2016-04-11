#!/usr/bin/python

import os, subprocess

def main():
	subprocess.check_call(['npm', 'install'], shell=False)
	subprocess.check_call(['npm', 'run', 'build'], env=os.environ, shell=False)


if __name__ == "__main__":
	main()
