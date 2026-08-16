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

func TestStitchCharacterViewsCreatesExactlyThreePanels(t *testing.T) {
	colors := []color.RGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}}
	encoded := make([][]byte, 0, 3)
	for _, fill := range colors {
		panel := image.NewRGBA(image.Rect(0, 0, 4, 6))
		for y := 0; y < 6; y++ {
			for x := 0; x < 4; x++ {
				panel.Set(x, y, fill)
			}
		}
		var buffer bytes.Buffer
		if err := png.Encode(&buffer, panel); err != nil {
			t.Fatal(err)
		}
		encoded = append(encoded, buffer.Bytes())
	}

	combined, err := stitchCharacterViews(encoded)
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(combined))
	if err != nil {
		t.Fatal(err)
	}
	if config.Width != 12 || config.Height != 6 {
		t.Fatalf("combined turnaround size = %dx%d, want 12x6", config.Width, config.Height)
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

func TestStoryboardImagePromptUsesStaticReferencePromptOnly(t *testing.T) {
	project := &model.DramaProject{Description: "realistic family drama"}
	storyboard := &model.DramaStoryboard{Title: "Entrance confrontation", Prompt: "STORYBOARD_FALLBACK"}
	segment := &model.DramaStoryboardSegment{
		Title:             "Confrontation established",
		Scene:             "apartment entrance",
		Action:            "ACTION_TIMELINE_MARKER: first walks in, then turns around",
		CompositionPrompt: "two-shot, wife left, husband right",
		ReferencePrompt:   "STATIC_IMAGE_MARKER: wife and husband face each other in the entrance",
		VideoPrompt:       "VIDEO_MOTION_MARKER: camera pushes in while both characters move",
	}

	prompt := (&DramaService{}).buildStoryboardImagePrompt(project, storyboard, nil, segment)
	if !strings.Contains(prompt, "STATIC_IMAGE_MARKER") {
		t.Fatalf("image prompt did not use segment reference_prompt: %q", prompt)
	}
	for _, forbidden := range []string{"VIDEO_MOTION_MARKER", "ACTION_TIMELINE_MARKER", "STORYBOARD_FALLBACK"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("image prompt leaked %q: %q", forbidden, prompt)
		}
	}
}

func TestSegmentVideoPromptUsesVideoPrompt(t *testing.T) {
	storyboard := &model.DramaStoryboard{Title: "Entrance confrontation"}
	segment := &model.DramaStoryboardSegment{
		ReferencePrompt: "STATIC_IMAGE_MARKER",
		VideoPrompt:     "VIDEO_MOTION_MARKER: slow push-in",
	}
	prompt := buildSegmentVideoPrompt(storyboard, segment, nil)
	if !strings.Contains(prompt, "VIDEO_MOTION_MARKER") || strings.Contains(prompt, "STATIC_IMAGE_MARKER") {
		t.Fatalf("video prompt fields are not isolated: %q", prompt)
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

func TestVideoDimensionsForPreset(t *testing.T) {
	width, height := videoDimensionsForSettings(nil, VideoGenerationSettings{Model: "minimax_h3", QualityPreset: "fast", SizePreset: "portrait"})
	if width != 512 || height != 896 {
		t.Fatalf("H3 fast portrait dimensions = %dx%d, want 512x896", width, height)
	}
	width, height = videoDimensionsForSettings(nil, VideoGenerationSettings{Model: "minimax_h3", QualityPreset: "standard", SizePreset: "portrait"})
	if width != 576 || height != 1024 {
		t.Fatalf("H3 standard portrait dimensions = %dx%d, want 576x1024", width, height)
	}
	width, height = videoDimensionsForSettings(nil, VideoGenerationSettings{Model: "minimax_h3", QualityPreset: "final", SizePreset: "portrait"})
	if width != 768 || height != 1344 {
		t.Fatalf("H3 final portrait dimensions = %dx%d, want 768x1344", width, height)
	}
	width, height = videoDimensionsForSettings(nil, VideoGenerationSettings{Model: "wan22", SizePreset: "landscape"})
	if width != 832 || height != 480 {
		t.Fatalf("Wan landscape dimensions = %dx%d, want 832x480", width, height)
	}
}

func TestH3QualityPresetControlsStepsAndFastAudio(t *testing.T) {
	for quality, wantSteps := range map[string]int{"fast": 8, "standard": 12, "final": 20} {
		workflow := systemMiniMaxH3ImageToVideoWorkflow(
			"first.png", "", "subtle motion", "", 3, 768, 1344,
			VideoGenerationSettings{Model: "minimax_h3", H3Mode: "i2v", AudioMode: "native", QualityPreset: quality}, nil,
		)
		inputs := workflow["10"].(map[string]interface{})["inputs"].(map[string]interface{})
		if inputs["steps"] != wantSteps {
			t.Fatalf("H3 %s steps = %v, want %d", quality, inputs["steps"], wantSteps)
		}
		_, hasNativeAudio := workflow["16"]
		if quality == "fast" && hasNativeAudio {
			t.Fatal("H3 fast workflow must not decode native audio")
		}
		if quality != "fast" && !hasNativeAudio {
			t.Fatalf("H3 %s workflow should keep selected native audio", quality)
		}
	}
}

func TestH3FL2VAWorkflowUsesFirstAndLastFrames(t *testing.T) {
	workflow := systemMiniMaxH3ImageToVideoWorkflow(
		"first.png", "last.png", "smooth transition", "", 5, 1344, 768,
		VideoGenerationSettings{Model: "minimax_h3", H3Mode: "fl2va", AudioMode: "external"}, nil,
	)
	inputs := workflow["7"].(map[string]interface{})["inputs"].(map[string]interface{})
	if _, ok := inputs["first_frame"]; !ok {
		t.Fatal("H3 FL2VA workflow is missing first_frame")
	}
	if _, ok := inputs["last_frame"]; !ok {
		t.Fatal("H3 FL2VA workflow is missing last_frame")
	}
	loader := workflow["18"].(map[string]interface{})["inputs"].(map[string]interface{})
	if loader["image"] != "last.png" {
		t.Fatalf("last frame loader image = %v, want last.png", loader["image"])
	}
}

func TestSelectSegmentTailFrameOnlyUsesExtractedTail(t *testing.T) {
	media := []model.DramaStoryboardMedia{
		{SegmentID: 10, Kind: "image", Source: "generated", FileID: 100},
		{SegmentID: 10, Kind: "image", Source: "video_tail", FileID: 101},
	}
	frame := selectSegmentTailFrame(10, media)
	if frame.FileID != 101 {
		t.Fatalf("tail frame file = %d, want 101", frame.FileID)
	}
}
