package providers

#AWS: {
	// ECR contains the configuration for AWS ECR.
	ecr: #AWSECR

	// Role contains the role to assume.
	role: string

	// Region contains the region to use.
	region: string
}

#AWSECR: {
	// AutoCreate contains whether to automatically create ECR repositories.
	autoCreate?: bool

	// Registry is the ECR registry to login to during CI operations.
	registry?: string
}
