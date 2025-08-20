project: {
	name: "ory-webhook"

	release: {
		docker: {
			on: {
				merge: {}
				tag: {}
			}

			config: {
				tag: _ @forge(name="GIT_HASH_OR_TAG")
			}
		}
	}
}
