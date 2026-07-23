package workspace

import "testing"

func TestSupportedGitVersion(t *testing.T) {
	tests := []struct {
		output string
		want   bool
	}{
		{"git version 2.39.0", true},
		{"git version 2.39.5 (Apple Git-154)", true},
		{"git version 2.50.1.windows.1", true},
		{"git version 2.38.9", false},
		{"unexpected", false},
	}
	for _, test := range tests {
		if got := supportedGitVersion(test.output); got != test.want {
			t.Errorf("supportedGitVersion(%q) = %v, want %v", test.output, got, test.want)
		}
	}
}
