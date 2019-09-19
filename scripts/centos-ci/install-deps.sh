#!/bin/sh
#
# dependencies for the tests are listed in scripts/lint-text.sh
#

# make the golang scl available
sudo yum -y install centos-release-scl epel-release

sudo yum -y install \
	make \
	/usr/bin/go \
	/usr/bin/shellcheck \
	rh-ruby26 \
	yamllint \
	; # empty line for 'git blame'

scl enable rh-ruby26 'gem install mdl'
curl -L https://git.io/get_helm.sh | bash
go get github.com/securego/gosec/cmd/gosec
go get github.com/golang/dep/cmd/dep
