package rulepack

import (
	"bytes"
	"testing"
)

func FuzzSplitFrontmatter(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("---\nid: example\n---\n# Body\n"),
		[]byte("---\r\nid: example\r\n---\r\n# Body\r\n"),
		[]byte("---\nid: example\n---\n---\nsecond marker\n"),
		[]byte("---\n---\n"), []byte(""), []byte("prefix\n---\nid: x\n---\nbody"),
		[]byte("---\nid: x\n---"), []byte("---\nid: x\n--- extra\nbody"),
		[]byte("---\r\nid: x\n---\nbody"), []byte("---\nid: [malformed YAML\n---\nbody\x00"),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		before := bytes.Clone(data)
		header, body, err := splitFrontmatter(data)
		againHeader, againBody, againErr := splitFrontmatter(data)
		if !bytes.Equal(data, before) {
			t.Fatal("split changed its input")
		}
		if (err == nil) != (againErr == nil) || !bytes.Equal(header, againHeader) || !bytes.Equal(body, againBody) {
			t.Fatal("split isn't deterministic")
		}
		if err != nil {
			if header != nil || body != nil {
				t.Fatal("failed split exposed partial content")
			}
			return
		}
		opening, closing := []byte("---\n"), []byte("\n---\n")
		if bytes.HasPrefix(data, []byte("---\r\n")) {
			opening, closing = []byte("---\r\n"), []byte("\r\n---\r\n")
		}
		if !bytes.HasPrefix(data, opening) || bytes.Contains(header, closing) {
			t.Fatal("split skipped the first exact boundary")
		}

		for _, malformed := range [][]byte{append([]byte("prefix\n"), data...), append(bytes.Clone(opening), header...)} {
			if _, _, err := splitFrontmatter(malformed); err == nil {
				t.Fatal("accepted malformed frontmatter boundary")
			}
		}
		joined := append(bytes.Clone(opening), header...)
		joined = append(joined, closing...)
		joined = append(joined, body...)
		if !bytes.Equal(joined, data) {
			t.Fatal("frontmatter/body boundaries changed bytes")
		}
		if len(data) >= len(opening)+len(header)+len(closing) && !bytes.Equal(body, data[len(opening)+len(header)+len(closing):]) {
			t.Fatal("body isn't the exact suffix")
		}
	})
}
