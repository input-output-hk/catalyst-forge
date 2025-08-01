package scan

type ScanCmd struct {
	Blueprint BlueprintCmd `cmd:"" help:"Scan for projects by their blueprints."`
	Earthfile EarthfileCmd `cmd:"" help:"Scan for projects by their Earthfiles."`
}
