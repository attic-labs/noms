Contributing to Noms
====================

## License

Noms is open source software, licensed under the [Apache License, Version 2.0](LICENSE).

## Contributing code

Due to legal reasons, all contributors must sign a contributor license agreement, either for an [individual](http://noms.io/ca_individual.html) or [corporation](http://noms.io/ca_corporation.html), before a pull request can be accepted.

## Languages

* Use Python, JS, or Go.
* Shell script is not allowed.

## Coding style

* JS follows the [Airbnb Style Guide](https://github.com/airbnb/javascript)
* Go uses gofmt, advisable to hook into your editor
* Tag PRs with either "toward: #<bug>" or "fixes: #<bug>" to help establish context for why a change is happening

## Running the tests

You can use `go test` command, e.g:

* `go test $(go list ./... | grep -v /vendor/)` should run every tests except from vendor packages.

For JS code, We have a python script to run all js tests.

* `python tools/run-all-js-tests.py`

## Recommended Chrome extensions
* Hides generated (Go) files from GitHub pull requests: [GitHub PR gen hider](https://chrome.google.com/webstore/detail/mhemmopgidccpkibohejfhlbkggdcmhf).
