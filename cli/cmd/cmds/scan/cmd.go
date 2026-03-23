package scan

type ScanCmd struct {
	All       AllCmd       `cmd:"" help:"Scan for files matching filename and content patterns."`
	Blueprint BlueprintCmd `cmd:"" help:"Scan for projects by their blueprints."`
	Earthfile EarthfileCmd `cmd:"" help:"Scan for projects by their Earthfiles."`
}
