package service

import "testing"

func TestIsFaceIDNoFaceError(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{name: "reported FaceID failure", message: "ComfyUI task failed: InsightFace: No face detected.", want: true},
		{name: "alternate wording", message: "INSIGHTFACE: no face found in image", want: true},
		{name: "unrelated InsightFace error", message: "InsightFace model could not be loaded", want: false},
		{name: "unrelated ComfyUI error", message: "CUDA out of memory", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isFaceIDNoFaceError(test.message); got != test.want {
				t.Fatalf("isFaceIDNoFaceError(%q) = %v, want %v", test.message, got, test.want)
			}
		})
	}
}
