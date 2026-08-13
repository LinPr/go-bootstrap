package version

var (
	// Version is the current release version of the go-bootstrap/otel library.
	// Set via ldflags: go build -ldflags "-X github.com/LinPr/go-bootstrap/otel/version.Version=$(git describe --tags --always --dirty)"
	Version = "dev"
)
