#!/usr/bin/env python

# Copyright 2016 Attic Labs, Inc. All rights reserved.
# Licensed under the Apache License, version 2.0:
# http://www.apache.org/licenses/LICENSE-2.0

'''
This script builds the Noms Go binaries for several OS/ARCH combinations and
generates a tar.gz of the binaries for each of those platforms.
'''

import copy
import os
import os.path
import platform
import re
import shutil
import subprocess

# The list of platforms for which we should execute builds on the following packages
PLATFORMS = [
    'darwin-amd64',
    'linux-amd64',
    'linux-arm',
]

# The list of Go packages for which we should build binaries
PACKAGES = [
    './cmd/noms',
    './cmd/noms-ui',
    './samples/go/blob-get',
    './samples/go/counter',
    './samples/go/csv/csv-analyze',
    './samples/go/csv/csv-export',
    './samples/go/csv/csv-import',
    './samples/go/demo-server',
    './samples/go/hr',
    './samples/go/json-import',
    './samples/go/nomdex',
    './samples/go/noms-merge',
    './samples/go/nomdex',
    './samples/go/poke',
    './samples/go/url-fetch',
    './samples/go/xml-import',
]

def call_with_env_and_cwd(cmd, env, cwd):
    """Executes a subprocess, waits for it to finish and asserts the return code is zero"""
    print cmd
    proc = subprocess.Popen(cmd, env=env, cwd=cwd, shell=False)
    proc.wait()
    assert proc.returncode == 0

def get_go_install_path():
    """Returns the full path to the go installation location"""
    go_install_path = '/usr/local/go/bin'
    if platform.system() == "Darwin":
        go_install_path = '/usr/local/bin'
    assert os.path.exists(go_install_path)
    return go_install_path

def get_go_bin_string():
    """Returns the full path to the go binary"""
    go_bin = os.path.join(get_go_install_path(), 'go')
    assert os.path.exists(go_bin)
    return go_bin

def main():
    """Asserts environment variables and file system is appropriate before executing builds"""
    # Workspace is the root of the Jenkins builder, e.g. "/var/lib/jenkins/workspace/builder".
    workspace = os.getenv('WORKSPACE')
    assert workspace

    # Git SHA revision identifier to insert into built binaries
    noms_rev = os.getenv('NOMS_REVISION')
    assert noms_rev

    noms_src_dir = os.path.join(workspace, 'src/github.com/attic-labs/noms')
    assert os.path.exists(noms_src_dir)

    noms_output_dir = os.path.join(noms_src_dir, 'build_output')
    if os.path.exists(noms_output_dir):
        shutil.rmtree(noms_output_dir)
    os.mkdir(noms_output_dir)
    assert os.path.exists(noms_output_dir)

    cmd_path = '%s:%s' % (os.getenv('PATH'), get_go_install_path())

    platform_pattern = re.compile(r'^(?P<os>\w+)-(?P<arch>\w+)$')
    for osarch in PLATFORMS:
        platform_match = platform_pattern.match(osarch)
        assert platform_match

        platform_output_dir = os.path.join(noms_output_dir, osarch)
        os.mkdir(platform_output_dir)
        assert os.path.exists(platform_output_dir)

        env = copy.copy(os.environ)
        env.update({
            'GOOS': platform_match.group('os'),
            'GOARCH': platform_match.group('arch'),
            'GOPATH': workspace,
            'PATH': cmd_path,
        })

        # Using 'go build' instead of 'go install' per the recommendation:
        # http://dave.cheney.net/2015/08/22/cross-compilation-with-go-1-5
        for pkg in PACKAGES:
            # cmd: go build -o outputFile -ldFlags "-X file.Constant=value" package
            cmd = ([get_go_bin_string(), 'build', '-o',
                    os.path.join(platform_output_dir, os.path.basename(pkg)),
                    '-ldflags',
                    '-X github.com/attic-labs/noms/go/constants.NomsGitSHA=' + noms_rev,
                    pkg])
            call_with_env_and_cwd(cmd, env, noms_src_dir)

        platform_targz_file = os.path.join(noms_output_dir, 'noms-%s-%s' % (noms_rev, osarch))
        shutil.make_archive(platform_targz_file, 'gztar', platform_output_dir)
        shutil.rmtree(platform_output_dir)

if __name__ == '__main__':
    main()
