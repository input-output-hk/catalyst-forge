version: "1.0"
project: {
	name: "foundry-operator"
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
