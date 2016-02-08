# Docker for running importers

# Prerequisites

### Get github auth token
Use shared deployment token if there is one; or go to https://github.com/settings/tokens and genereate a new access token with your account.

### Install docker
follow instructions at https://www.docker.com.

### (non-linux host) Open a docker terminal
This has changed rapidly in recent releases of docker, and is covered as the last step of docker installation above.

With Docker 1.10.0 on Mac, run the "Docker Quickstart Terminal" app, which will setup the VM engine and drop you into a prompt. All following commands must be run inside of a docker terminal.

# Build docker image
The following script will build a docker image named "golang_docker_vm" with noms binaries built from head.

Go path is set to /go
```
sh build_docker.sh ${github_token_from_previous_step}
```

# Run noms binaries
You can now run go binaries in the container:
```
docker run --rm -t golang_docker_vm /go/bin/importer --h=http://host:port  --ds=dataset_name <PATH>
```
