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

# install Go from https://go-repo.io/ as golang-1.13 does not work with the e2e tests
sudo rpm --import https://mirror.go-repo.io/centos/RPM-GPG-KEY-GO-REPO
curl -s https://mirror.go-repo.io/centos/go-repo.repo | sudo tee /etc/yum.repos.d/go-repo.repo
sudo yum -y install golang-1.12 || sudo yum -y downgrade golang-1.12

sudo yum -y install \
	/usr/bin/shellcheck \
	rh-ruby26 \
	yamllint \
	; # empty line for 'git blame'

scl enable rh-ruby26 'gem install mdl'
curl -L https://git.io/get_helm.sh | bash
go get github.com/securego/gosec/cmd/gosec
go get github.com/golang/dep/cmd/dep
