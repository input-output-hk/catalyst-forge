version: "1.0"
project: {
	name: "foundry-api"
	ci: targets: {
		publish: {
			args: {
				version: string | *"dev" @env(name="GIT_TAG",type="string")
			}
		}
		release: {
			args: {
				version: string | *"dev" @env(name="GIT_TAG",type="string")
			}
		}
	}
}
