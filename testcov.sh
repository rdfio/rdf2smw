#!/bin/bash
set -ex
for pkg in "github.com/rdfio/rdf2smw" "github.com/rdfio/rdf2smw/components"; do
    touch profile_tmp.cov
    go test -v -covermode=count -coverprofile=profile_tmp.cov $pkg || ERROR="Error testing $pkg"
    tail -n +2 profile_tmp.cov >> cover.out || exit "Unable to append coverage for $pkg"
done
