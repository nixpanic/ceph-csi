#!/bin/sh
#
# dependencies for the tests are listed in scripts/lint-text.sh
#

# make the golang scl available
sudo yum -y install centos-release-scl epel-release

sudo yum -y install \
	make \
	go-toolset-7-golang \
	/usr/bin/shellcheck \
	rh-ruby26 \
	yamllint \
	; # empty line for 'git blame'

scl enable rh-ruby26 'gem install mdl'
curl -L https://git.io/get_helm.sh | bash
scl enable go-toolset-7 'go get github.com/securego/gosec/cmd/gosec'
scl enable go-toolset-7 'go get github.com/golang/dep/cmd/dep'
