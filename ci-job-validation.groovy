def cico_retries = 16
def cico_retry_interval = 60
def ci_git_repo = 'https://github.com/ceph/ceph-csi'
def ci_git_branch = 'ci/centos'
def git_repo = 'https://github.com/ceph/ceph-csi'
def ref = 'ci/centos'
def git_since = 'ci/centos'
def base = ''
def doc_change = 0
// private, internal container image repository
def cached_image = 'registry-ceph-csi.apps.ocp.ci.centos.org/ceph-csi'
def use_pulled_image = 'USE_PULLED_IMAGE=yes'

node('cico-workspace') {
	stage('checkout ci repository') {
		if (params.ghprbPullId != null) {
			ref = "pull/${ghprbPullId}/merge"
		}
		checkout([$class: 'GitSCM', branches: [[name: 'FETCH_HEAD']],
			userRemoteConfigs: [[url: "${ci_git_repo}", refspec: "${ref}"]]])
	}

	stage('checkout PR') {
		if (params.ghprbPullId != null) {
			ref = "pull/${ghprbPullId}/merge"
		}
		if (params.ghprbTargetBranch != null) {
			git_since = "${ghprbTargetBranch}"
		}

		sh "git clone --depth=1 --branch='${git_since}' '${git_repo}' ~/build/ceph-csi"
		if (ref != git_since) {
			sh "cd ~/build/ceph-csi && git fetch origin ${ref} && git checkout -b ${ref} FETCH_HEAD"
		}
	}

	stage('check doc-only change') {
		doc_change = sh(
			script: "cd ~/build/ceph-csi && \${OLDPWD}/scripts/skip-doc-change.sh origin/${git_since}",
			returnStatus: true)
	}
	// if doc_change (return value of skip-doc-change.sh is 1, do not run the other stages
	if (doc_change == 1) {
		currentBuild.result = 'SUCCESS'
		return
	}

	stage('reserve bare-metal machine') {
		def firstAttempt = true
		retry(30) {
			if (!firstAttempt) {
				sleep(time: 5, unit: "MINUTES")
			}
			firstAttempt = false
			cico = sh(
				script: "cico node get -f value -c hostname -c comment --release=8 --retry-count=${cico_retries} --retry-interval=${cico_retry_interval}",
				returnStdout: true
			).trim().tokenize(' ')
			env.CICO_NODE = "${cico[0]}.ci.centos.org"
			env.CICO_SSID = "${cico[1]}"
		}
	}

	try {
		stage('prepare bare-metal machine') {
			if (params.ghprbTargetBranch != null) {
				base = "--base=${ghprbTargetBranch}"
			}
			sh 'scp -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no ./prepare.sh root@${CICO_NODE}:'
			sh "ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@${CICO_NODE} ./prepare.sh --workdir=/opt/build/go/src/github.com/ceph/ceph-csi --gitrepo=${ci_git_repo} --ref=${ref} ${base}"
		}

		// - check if the PR modifies the container image files
		// - pull the container image from the repository of no
		//   modifications are detected
		stage('pull container image') {
			def rebuild_container = sh(
				script: "cd ~/build/ceph-csi && \${OLDPWD}/scripts/container-needs-rebuild.sh test origin/${git_since}",
				returnStatus: true)
			if (rebuild_container == 10) {
				// container needs rebuild, don't pull
				use_pulled_image = 'USE_PULLED_IMAGE=no'
				return
			}

			withCredentials([usernamePassword(credentialsId: 'container-registry-auth', usernameVariable: 'CREDS_USER', passwordVariable: 'CREDS_PASSWD')]) {
				sh "ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@${CICO_NODE} 'podman pull --creds=${CREDS_USER}:${CREDS_PASSWD} ${cached_image}:test'"
			}
		}
		stage('test') {
			sh "ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no root@${CICO_NODE} 'cd /opt/build/go/src/github.com/ceph/ceph-csi && make ENV_CSI_IMAGE_NAME=${cached_image} ${use_pulled_image}'"
		}
	}

	finally {
		stage('return bare-metal machine') {
			sh 'cico node done ${CICO_SSID}'
		}
	}
}
