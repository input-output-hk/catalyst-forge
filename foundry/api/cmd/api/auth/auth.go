package auth

// AuthCmd represents the auth subcommand category
type AuthCmd struct {
	Generate GenerateCmd `kong:"cmd,help='Generate authentication tokens'"`
	Init     InitCmd     `kong:"cmd,help='Initialize authentication configuration'"`
	Validate ValidateCmd `kong:"cmd,help='Validate authentication tokens'"`
}
