#!/bin/bash
for testdir in 'js2' 'clients/splore'; do
  pushd $testdir
  npm install
  ./link.sh
  npm build
  npm test
  if [ $? -eq 0 ]
  then
    echo "js tests pass"
  else
    exit 1
  fi
  popd
done
