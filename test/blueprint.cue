version: "1.0"
project: {
	name: "timoni-test"
	release: {
		timoni: {
			on: always: {}
			config: {
				container: "timoni-test"
				tag:       "1.0.0"
			}
		}
	}
}
