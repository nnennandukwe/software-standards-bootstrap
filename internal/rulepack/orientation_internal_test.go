package rulepack

import (
	"strings"
	"testing"
)

func TestOrientationCollectionLimits(t *testing.T) {
	tests := []struct {
		name string
		set  func(*Orientation, int)
	}{
		{name: "areas", set: func(value *Orientation, count int) { value.Areas = make([]OrientationArea, count) }},
		{name: "prerequisites", set: func(value *Orientation, count int) { value.Prerequisites = make([]OrientationPrerequisite, count) }},
		{name: "documents", set: func(value *Orientation, count int) { value.Documents = make([]OrientationDocument, count) }},
		{name: "related artifacts", set: func(value *Orientation, count int) { value.RelatedArtifactIDs = make([]string, count) }},
		{name: "guidance", set: func(value *Orientation, count int) { value.Guidance = make([]OrientationGuidance, count) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			atLimit := &Orientation{}
			test.set(atLimit, OrientationMaxEntries)
			if diagnostics := validateOrientationCollectionLimits(orientationPath, atLimit); len(diagnostics) != 0 {
				t.Fatalf("at limit: %#v", diagnostics)
			}

			overLimit := &Orientation{}
			test.set(overLimit, OrientationMaxEntries+1)
			diagnostics := validateOrientationCollectionLimits(orientationPath, overLimit)
			if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "maximum is 32") {
				t.Fatalf("over limit: %#v", diagnostics)
			}
		})
	}
}

func TestOrientationTextAndPathLimits(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		max         int
		wantInvalid bool
	}{
		{name: "text at limit", value: strings.Repeat("界", OrientationMaxTextRunes), max: OrientationMaxTextRunes},
		{name: "text over limit", value: strings.Repeat("界", OrientationMaxTextRunes+1), max: OrientationMaxTextRunes, wantInvalid: true},
		{name: "label at limit", value: strings.Repeat("界", OrientationMaxLabelRunes), max: OrientationMaxLabelRunes},
		{name: "label over limit", value: strings.Repeat("界", OrientationMaxLabelRunes+1), max: OrientationMaxLabelRunes, wantInvalid: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := validateOrientationText(orientationPath, "text", test.value, test.max, "text")
			if gotInvalid := len(diagnostics) != 0; gotInvalid != test.wantInvalid {
				t.Fatalf("diagnostics = %#v, want invalid %v", diagnostics, test.wantInvalid)
			}
		})
	}

	atLimit := strings.Repeat("a/", 511) + "aa"
	if len(atLimit) != OrientationMaxPathBytes {
		t.Fatalf("path fixture length = %d", len(atLimit))
	}
	if diagnostics := validateOrientationPath(orientationPath, "path", atLimit); len(diagnostics) != 0 {
		t.Fatalf("path at limit: %#v", diagnostics)
	}
	overLimit := atLimit + "a"
	if diagnostics := validateOrientationPath(orientationPath, "path", overLimit); len(diagnostics) != 1 {
		t.Fatalf("path over limit: %#v", diagnostics)
	}
}

func TestOrientationEvidenceLimits(t *testing.T) {
	for _, test := range []struct {
		name        string
		count       int
		wantInvalid bool
	}{
		{name: "one", count: 1},
		{name: "sixteen", count: OrientationMaxEvidence},
		{name: "zero", count: 0, wantInvalid: true},
		{name: "seventeen", count: OrientationMaxEvidence + 1, wantInvalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := validateOrientationEvidenceLimit(orientationPath, "evidence", test.count)
			if gotInvalid := len(diagnostics) != 0; gotInvalid != test.wantInvalid {
				t.Fatalf("diagnostics = %#v, want invalid %v", diagnostics, test.wantInvalid)
			}
		})
	}
}
