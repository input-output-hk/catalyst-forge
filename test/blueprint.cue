version: "1.0"
project: {
	name: "timoni-test"
	release: {
		timoni: {
			on: always: {}
			config: {
				container: "timoni-test"
				tag:       "v1.0.0"
			}
		}
	}
}
