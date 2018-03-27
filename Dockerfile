FROM           golang:latest
MAINTAINER     no-reply@noms.io

COPY           . src/github.com/attic-labs/noms

RUN            go install $(go list ./...)
RUN            apt-get update && apt-get install less

VOLUME         /data
EXPOSE         8000

CMD            ["noms", "serve", "/data"]
