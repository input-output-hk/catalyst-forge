version: "1.0"
project: {
	name: "forge"
	ci: targets: {
		publish: platforms: [
			"linux/amd64",
			"linux/arm64",
		]
		release: platforms: [
			"linux/amd64",
			"linux/arm64",
			"darwin/amd64",
			"darwin/arm64",
		]
	}
}
