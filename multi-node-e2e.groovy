def cico_retries = 16
def cico_retry_interval = 60
def ci_git_repo = 'https://github.com/nixpanic/ceph-csi'
def ci_git_branch = 'ci/multi-node-e2e'

node('cico-workspace') {
	// environment (not?) set by Jenkins, or not?
	environment {
		GIT_REPO = 'https://github.com/ceph/ceph-csi'
		GIT_BRANCH = 'master'
	}

	stage('checkout ci repository') {
		git url: "${ci_git_repo}",
			branch: "${ci_git_branch}",
			changelog: false
	}

	stage('reserve bare-metal machine') {
		cico = sh(
			script: "cico node get -f value -c hostname -c comment --retry-count ${cico_retries} --retry-interval ${cico_retry_interval}",
			returnStdout: true
		).trim().tokenize(' ')
		env.CICO_NODE = "${cico[0]}.ci.centos.org"
		env.CICO_SSID = "${cico[1]}"
	}

	try {
		stage('prepare bare-metal machine') {
			sh 'scp -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ./prepare.sh ./multi-node-k8s.sh root@${CICO_NODE}:'
			sh 'ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@${CICO_NODE} ./prepare.sh --workdir=/opt/build --gitrepo=${GIT_REPO} --branch=${GIT_BRANCH}'
			sh 'ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@${CICO_NODE} ./multi-node-k8s.sh'
		}

		stage('build') {
			sh 'ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@${CICO_NODE} "export GOPATH=/opt/build/go ; go test github.com/ceph/ceph-csi/e2e --deploy-timeout=10 -timeout=30m -v"'
		}
	}

	finally {
		stage('return bare-metal machine') {
			sh 'cico node done ${CICO_SSID}'
		}
	}
}
