package inventory

import (
	"bytes"
	"fmt"
	"path"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func FuzzParseGitTree(f *testing.F) {
	oid := strings.Repeat("a", 40)
	for _, seed := range []string{
		"100644 blob " + oid + " 12\tREADME.md\x00",
		"100755 blob " + strings.Repeat("b", 64) + " 0\tsrc/tool.sh\x00",
		"160000 commit " + oid + " -\tmodule\x00",
		"120000 blob " + oid + " 3\tlink\x00",
		"", "missing tab\x00", "100644 blob\tshort\x00",
		"100644 blob " + oid + " overflow\tfile\x00",
		"100644 blob " + oid + " 1\t../escape\x00",
		"100644 blob " + oid + " 1\t/absolute\x00",
		"100644 blob " + oid + " 1\ta//b\x00",
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		original := bytes.Clone(raw)
		entries, err := parseTree(raw)
		again, againErr := parseTree(raw)
		if !bytes.Equal(raw, original) || (err == nil) != (againErr == nil) || !reflect.DeepEqual(entries, again) {
			t.Fatal("tree parsing changed input or isn't deterministic")
		}
		if err != nil {
			return
		}
		for _, entry := range entries {
			for _, malformed := range []string{
				"100644 blob " + strings.Repeat("a", 40) + " 1 missing-tab\x00",
				"100644 blob\t" + entry.path + "\x00",
				"100644 blob " + entry.oid + " 999999999999999999999999999999\t" + entry.path + "\x00",
			} {
				if _, err := parseTree([]byte(malformed)); err == nil {
					t.Fatalf("accepted malformed tree record %q", malformed)
				}
			}

			if entry.path == "" || entry.path == "." || entry.path == ".." || path.IsAbs(entry.path) || path.Clean(entry.path) != entry.path || strings.HasPrefix(entry.path, "../") {
				t.Fatalf("accepted unsafe tree path %q", entry.path)
			}
			size := strconv.FormatInt(entry.size, 10)
			if entry.size == -1 {
				size = "-"
			}
			record := fmt.Sprintf("%s %s %s %s\t%s\x00", entry.mode, entry.kind, entry.oid, size, entry.path)
			roundTrip, roundTripErr := parseTree([]byte(record))
			if roundTripErr != nil || len(roundTrip) != 1 || roundTrip[0] != entry {
				t.Fatalf("tree record didn't round trip: %v", roundTripErr)
			}
			for _, unsafePath := range []string{"../" + entry.path, "/" + entry.path, entry.path + "/../escape"} {
				malformed := fmt.Sprintf("%s %s %s %s\t%s\x00", entry.mode, entry.kind, entry.oid, size, unsafePath)
				if _, err := parseTree([]byte(malformed)); err == nil {
					t.Fatalf("accepted unsafe tree record %q", malformed)
				}
			}
		}
	})
}
