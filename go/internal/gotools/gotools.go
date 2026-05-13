package gotools

type Tool struct {
	Binary  string
	Package string
}

func (tool Tool) PackageVersion(version string) string {
	return tool.Package + "@" + version
}

func (tool Tool) GoRunArgs(version string, args ...string) []string {
	runArgs := []string{"run", tool.PackageVersion(version)}
	return append(runArgs, args...)
}

const Latest = "latest"

var (
	Dry4Go    = Tool{Binary: "dry4go", Package: "github.com/unclebob/dry4go/cmd/dry4go"}
	CRAP4Go   = Tool{Binary: "crap4go", Package: "github.com/unclebob/crap4go/cmd/crap4go"}
	Mutate4Go = Tool{Binary: "mutate4go", Package: "github.com/unclebob/mutate4go/cmd/mutate4go"}
)
