package gha

type GhaCmd struct {
	Create CreateCmd `cmd:"" help:"Create a new GHA authentication entry."`
}
