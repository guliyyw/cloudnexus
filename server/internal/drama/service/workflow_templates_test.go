package service

import (
	"reflect"
	"testing"

	"github.com/cloudnexus/server/pkg/model"
)

func TestRenderDramaWorkflowTemplatePreservesTypedValues(t *testing.T) {
	workflow, err := renderDramaWorkflowTemplate("storyboard_regional_sdxl_v1.json", map[string]interface{}{
		"checkpoint":      "sdxl.safetensors",
		"positive_prompt": "two people at a table",
		"negative_prompt": "text",
		"width":           1344,
		"height":          768,
		"seed":            int64(42),
		"steps":           28,
		"cfg":             6.5,
		"sampler":         "euler",
		"scheduler":       "normal",
	})
	if err != nil {
		t.Fatal(err)
	}
	latentInputs := workflow["4"].(map[string]interface{})["inputs"].(map[string]interface{})
	if latentInputs["width"] != 1344 || latentInputs["height"] != 768 {
		t.Fatalf("canvas = %vx%v, want 1344x768", latentInputs["width"], latentInputs["height"])
	}
	samplerInputs := workflow["5"].(map[string]interface{})["inputs"].(map[string]interface{})
	if samplerInputs["seed"] != int64(42) || samplerInputs["cfg"] != 6.5 {
		t.Fatalf("typed sampler values were not preserved: %#v", samplerInputs)
	}
}

func TestRegionalWorkflowSeparatesCharactersAndScene(t *testing.T) {
	settings := ImageGenerationSettings{
		Checkpoint: "sd_xl_base_1.0.safetensors",
		Width:      768, Height: 432, Steps: 24, CFG: 7,
		Sampler: "euler", Scheduler: "normal", NegativePrompt: "text",
	}
	references := []ComfyReferenceImage{
		{Name: "husband.png", Kind: "character"},
		{Name: "wife.png", Kind: "character"},
		{Name: "dining-room.png", Kind: "scene"},
	}
	workflow, err := systemRegionalStoryboardWorkflow(
		"husband on the left, wife on the right",
		settings,
		[]string{"husband.png", "wife.png", "dining-room.png"},
		references,
	)
	if err != nil {
		t.Fatal(err)
	}

	classCounts := make(map[string]int)
	loaderPresets := make(map[string]bool)
	maskX := make(map[int]bool)
	for _, rawNode := range workflow {
		node := rawNode.(map[string]interface{})
		classType := node["class_type"].(string)
		classCounts[classType]++
		inputs := node["inputs"].(map[string]interface{})
		if classType == "IPAdapterUnifiedLoader" {
			loaderPresets[inputs["preset"].(string)] = true
		}
		if classType == "MaskComposite" {
			maskX[inputs["x"].(int)] = true
		}
	}
	if classCounts["IPAdapterAdvanced"] != 3 {
		t.Fatalf("IPAdapterAdvanced count = %d, want 3", classCounts["IPAdapterAdvanced"])
	}
	if classCounts["SolidMask"] != 4 || classCounts["FeatherMask"] != 2 {
		t.Fatalf("unexpected regional mask nodes: %#v", classCounts)
	}
	if classCounts["ConditioningSetMask"] != 2 || classCounts["ConditioningCombine"] != 2 {
		t.Fatalf("regional character prompts were not applied: %#v", classCounts)
	}
	if !loaderPresets["PLUS (high strength)"] || !loaderPresets["PLUS FACE (portraits)"] {
		t.Fatalf("scene and face loaders were not separated: %#v", loaderPresets)
	}
	if !maskX[0] || !maskX[256] {
		t.Fatalf("two-character masks are not left/right bound: %#v", maskX)
	}
	samplerInputs := workflow["5"].(map[string]interface{})["inputs"].(map[string]interface{})
	if reflect.DeepEqual(samplerInputs["model"], []interface{}{"1", 0}) {
		t.Fatal("sampler still uses the unconditioned checkpoint model")
	}
	if reflect.DeepEqual(samplerInputs["positive"], []interface{}{"2", 0}) {
		t.Fatal("sampler does not use regional character conditioning")
	}
}

func TestRegionalWorkflowPrefersFaceIDWhenAvailable(t *testing.T) {
	settings := ImageGenerationSettings{
		Checkpoint: "sd_xl_base_1.0.safetensors",
		Width:      768, Height: 432, Steps: 24, CFG: 7,
		Sampler: "euler", Scheduler: "normal", UseFaceID: true,
	}
	references := []ComfyReferenceImage{
		{Name: "husband.png", Kind: "character"},
		{Name: "wife.png", Kind: "character"},
		{Name: "room.png", Kind: "scene"},
	}
	workflow, err := systemRegionalStoryboardWorkflow(
		"husband on the left, wife on the right",
		settings,
		[]string{"husband.png", "wife.png", "room.png"},
		references,
	)
	if err != nil {
		t.Fatal(err)
	}
	classCounts := make(map[string]int)
	for _, rawNode := range workflow {
		node := rawNode.(map[string]interface{})
		classCounts[node["class_type"].(string)]++
	}
	if classCounts["IPAdapterUnifiedLoaderFaceID"] != 1 || classCounts["IPAdapterFaceID"] != 2 {
		t.Fatalf("FaceID nodes were not selected: %#v", classCounts)
	}
	if classCounts["IPAdapterAdvanced"] != 1 {
		t.Fatalf("scene should remain on standard IPAdapter: %#v", classCounts)
	}
}

func TestFilterAssetsForSegmentUsesCharacterOrder(t *testing.T) {
	assets := []model.DramaAsset{
		{Type: "character", Name: "妻子"},
		{Type: "character", Name: "丈夫"},
		{Type: "scene", Name: "家中餐厅"},
	}
	segment := &model.DramaStoryboardSegment{
		Characters: "丈夫, 妻子",
		Scene:      "家中餐厅",
	}
	filtered := filterAssetsForSegment(assets, segment)
	got := []string{filtered[0].Name, filtered[1].Name, filtered[2].Name}
	want := []string{"丈夫", "妻子", "家中餐厅"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("asset order = %#v, want %#v", got, want)
	}
}

func TestFilterAssetsForSegmentPrefersExplicitSpatialOrder(t *testing.T) {
	assets := []model.DramaAsset{
		{Type: "character", Name: "husband"},
		{Type: "character", Name: "wife"},
		{Type: "scene", Name: "living room"},
	}
	segment := &model.DramaStoryboardSegment{
		Characters:      "husband, wife",
		Scene:           "living room",
		ReferencePrompt: "wife standing on the left side, husband standing on the right side",
	}
	filtered := filterAssetsForSegment(assets, segment)
	got := []string{filtered[0].Name, filtered[1].Name, filtered[2].Name}
	want := []string{"wife", "husband", "living room"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("spatial asset order = %#v, want %#v", got, want)
	}
}

func TestSelectImageCheckpointUpgradesBaseModel(t *testing.T) {
	available := []string{
		"sd_xl_base_1.0.safetensors",
		"RealVisXL_V5.0_fp16.safetensors",
	}
	got := selectImageCheckpoint("sd_xl_base_1.0.safetensors", available)
	if got != "RealVisXL_V5.0_fp16.safetensors" {
		t.Fatalf("checkpoint = %q, want RealVisXL", got)
	}
	if got := selectImageCheckpoint("custom.safetensors", available); got != "custom.safetensors" {
		t.Fatalf("explicit custom checkpoint was replaced: %q", got)
	}
}
