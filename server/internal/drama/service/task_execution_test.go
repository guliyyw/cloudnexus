package service

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/cloudnexus/server/pkg/model"
)

func TestVideoDimensionsFromImagePreservesLandscapeAspect(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 1344, 768))
	source.Set(0, 0, color.White)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}

	width, height := videoDimensionsFromImage(encoded.Bytes())
	if width != 832 || height != 480 {
		t.Fatalf("video dimensions = %dx%d, want 832x480", width, height)
	}
}

func TestBuildSegmentImageNegativePromptHonorsSinglePersonCloseUp(t *testing.T) {
	segment := &model.DramaStoryboardSegment{
		Characters:     "妻子",
		Shot:           "face close-up, 人物特写",
		NegativePrompt: "single person, solo portrait, close-up, extreme close-up, missing husband, armor",
	}

	prompt := buildSegmentImageNegativePrompt(segment)
	for _, forbidden := range []string{"single person", "solo portrait", "close-up", "extreme close-up", "missing husband"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("negative prompt contains conflicting term %q: %s", forbidden, prompt)
		}
	}
	if !strings.Contains(prompt, "armor") {
		t.Fatalf("negative prompt lost a valid custom term: %s", prompt)
	}
}

func TestBuildSegmentVideoNegativePromptProtectsGroupShot(t *testing.T) {
	segment := &model.DramaStoryboardSegment{
		Characters: "丈夫, 妻子",
		Shot:       "medium two-shot",
	}

	prompt := buildSegmentVideoNegativePrompt(segment)
	for _, required := range []string{"solo portrait", "cropped second person", "missing character"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("negative prompt missing %q: %s", required, prompt)
		}
	}
}

func TestWanWorkflowUsesRequestedDimensionsAndDuration(t *testing.T) {
	workflow := systemWan22LocalImageToVideoWorkflow("frame.png", "subtle motion", "", 3, 832, 480)
	wanNode := workflow["10"].(map[string]interface{})
	wanInputs := wanNode["inputs"].(map[string]interface{})
	if wanInputs["width"] != 832 || wanInputs["height"] != 480 {
		t.Fatalf("Wan dimensions = %vx%v, want 832x480", wanInputs["width"], wanInputs["height"])
	}
	if wanInputs["length"] != 49 {
		t.Fatalf("Wan frame length = %v, want 49", wanInputs["length"])
	}
	videoNode := workflow["14"].(map[string]interface{})
	videoInputs := videoNode["inputs"].(map[string]interface{})
	if videoInputs["fps"] != 16.0 {
		t.Fatalf("video fps = %v, want 16", videoInputs["fps"])
	}
}
