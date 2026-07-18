package service

import "testing"

func TestBuildTrackIdentityNormalizesCaseAndSpaces(t *testing.T) {
	left := buildTrackIdentity(" Song  Name ", " Artist ", " Album ")
	right := buildTrackIdentity("song name", "artist", "album")
	if left != right {
		t.Fatalf("expected normalized identities to match, got %q and %q", left, right)
	}
}

func TestBuildTrackIdentityUsesTrimmedCloudFilename(t *testing.T) {
	cloudIdentity := buildTrackIdentity("Demo Song", "", "")
	publicIdentity := buildTrackIdentity("Demo Song", "", "")
	if cloudIdentity != publicIdentity {
		t.Fatalf("expected cloud and public identities to match, got %q and %q", cloudIdentity, publicIdentity)
	}
}

func TestMergeMetadataPrefersExtractedValues(t *testing.T) {
	base := TrackInfo{Title: "demo-song", Artist: "", Album: "", Duration: 0}
	merged := mergeMetadata(base, extractedTrackMetadata{Title: "Demo Song", Artist: "Tester", Album: "Album A", Duration: 123})
	if merged.Title != "Demo Song" {
		t.Fatalf("expected title to be replaced, got %q", merged.Title)
	}
	if merged.Artist != "Tester" || merged.Album != "Album A" || merged.Duration != 123 {
		t.Fatalf("unexpected merged metadata: %+v", merged)
	}
}
