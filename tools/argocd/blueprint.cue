version: "1.0"
project: {
	name: "forge-argocd"
	release: {
		docker: {
			on: {
				//merge: {}
				//tag: {}
				always: {}
			}
			config: {
				tag: _ @forge(name="GIT_COMMIT_HASH")
			}
		}
	}
}
