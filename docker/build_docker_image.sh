#!/bin/bash

VM_NAME="golang_docker_vm"

function no_docker_error {
  echo "Please install docker: www.docker.com"
  exit 1
}

function create_docker_image {
  vm_name=$1
  github_token=$2

  echo "Creating docker image \"${vm_name}\"..."
  docker build --force-rm=true --rm=true --build-arg github_token=${github_token} -t "${vm_name}" - < Dockerfile
}

which docker > /dev/null || no_docker_error

github_token=$1
create_docker_image ${VM_NAME} ${github_token}

echo "Retrieving installed docker images..."
docker images
