#!/bin/sh
#
# dependencies for the tests are listed in scripts/lint-text.sh
#

# exit on error
set -e

# make sure all updates get installed
sudo yum -y update

# install packages from base CentOS (prevent updates from SCL)
sudo yum -y install \
	make \
	gcc \
	; # empty line for 'git blame'

# make the golang scl available
sudo yum -y install centos-release-scl epel-release

sudo yum -y install \
	golang \
	/usr/bin/shellcheck \
	rh-ruby26 \
	yamllint \
	; # empty line for 'git blame'

scl enable rh-ruby26 'gem install mdl'
curl -L https://git.io/get_helm.sh | bash
go get github.com/securego/gosec/cmd/gosec
go get github.com/golang/dep/cmd/dep
