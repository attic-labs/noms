GOCMD=go
GOTEST=$(GOCMD) test

# the standard vet set is atomic,bool,buildtags,nilfunc,printf. printf causes a lot of spurious failures, so leave that
# out
test:
	$(GOTEST) -vet=atomic,bool,buildtags,nilfunc ./...