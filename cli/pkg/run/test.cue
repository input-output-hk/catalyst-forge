{
    project: ci: targets {
        target: {
            args: {
                foo: "bar"
            }
            platforms: ["linux/amd64"]
            privileged: true
            retries: 3
        }
    }
}