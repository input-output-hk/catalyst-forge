package global

// Release contains the configuration for the release of a project.
#Release: {
	// Docs is the configuration for the docs release type.
	docs?: #DocsRelease
}

// DocsRelease contains the configuration for the docs release type.
#DocsRelease: {
	// Bucket is the name of the S3 bucket to upload the docs to.
	bucket: string

	// Path is the subpath within the bucket to upload the docs to.
	path?: string | *""
}
