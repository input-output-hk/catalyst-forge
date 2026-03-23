package github

type GitRepoConfig struct {
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
	Path       string `json:"path,omitempty"`
}

type BrewConfig struct {
	Template     string         `json:"template"`
	Tap          GitRepoConfig  `json:"tap"`
	Templates    *GitRepoConfig `json:"templates,omitempty"`
	Description  string         `json:"description"`
	BinaryName   string         `json:"binary_name"`
	TemplatesUrl string         `json:"templates_url,omitempty"` // Deprecated: use Templates instead
}

type ReleaseConfig struct {
	Prefix string      `json:"prefix"`
	Name   string      `json:"name"`
	Brew   *BrewConfig `json:"brew,omitempty"`
}
