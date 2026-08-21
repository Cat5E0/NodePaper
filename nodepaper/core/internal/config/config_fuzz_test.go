package config

import "testing"

func FuzzParseDoesNotPanic(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("version: 1\nprofile: cumcm\nsource: paper.md\n"),
		[]byte("version: [\n"),
		{},
		[]byte("\x00\xff\xfe"),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(data)
	})
}
