############################################################
# Dockerfile to build sandbox for executing 3rd-party code.
############################################################

# Using Docker's official golang base image.
# https://hub.docker.com/_/golang/
FROM golang:1.5

# Per https://github.com/attic-labs/noms/pull/970
ENV GO15VENDOREXPERIMENT 1

# Tweak git config to access github with token
ARG github_token
RUN git config --global url."https://${github_token}:x-oauth-basic@github.com/".insteadOf "https://github.com/"

# Fetch noms
RUN go get -u -t github.com/attic-labs/noms/...

RUN go install ./...
