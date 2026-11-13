package cli

import (
	"fmt"
	"io"

	"github.com/nnennandukwe/software-standards-bootstrap/internal/buildinfo"
)

func runVersion(args []string, stdout, stderr io.Writer) int {
	if args[0] == "version" && len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
		fmt.Fprint(stdout, versionHelp)
		return 0
	}
	if len(args) != 1 {
		fmt.Fprintln(stderr, "error: version doesn't accept arguments")
		fmt.Fprint(stderr, versionHelp)
		return 2
	}
	fmt.Fprintf(
		stdout,
		"ssb %s (commit %s)\n",
		buildinfo.Version,
		buildinfo.Commit,
	)
	return 0
}
