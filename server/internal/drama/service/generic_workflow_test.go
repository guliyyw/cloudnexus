package service

import (
	"strings"
	"testing"

	"github.com/cloudnexus/server/pkg/model"
)

func TestGenericCharacterSemanticLabelUsesAssetDescription(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   string
	}{
		{name: "Captain Nova", prompt: "elderly female spaceship captain", want: "elderly woman"},
		{name: "Milo", prompt: "young boy wearing a red raincoat", want: "boy"},
		{name: "Unit Seven", prompt: "friendly android robot", want: "robot character"},
		{name: "Snow", prompt: "small white dog with a blue collar", want: "dog"},
	}
	for _, test := range tests {
		if got := characterSemanticLabel(test.name, test.prompt); got != test.want {
			t.Fatalf("characterSemanticLabel(%q) = %q, want %q", test.name, got, test.want)
		}
	}
}

func TestGenericProjectVisualStyleIsNotAlwaysPhotorealistic(t *testing.T) {
	project := &model.DramaProject{Description: "A cel-shaded anime adventure"}
	got := projectVisualStyleDirective(project, "two heroes on a bridge")
	if !strings.Contains(got, "anime") || strings.Contains(got, "photorealistic") {
		t.Fatalf("unexpected anime style directive: %q", got)
	}
}

func TestGenericCheckpointSelectionRespectsProjectStyle(t *testing.T) {
	available := []string{
		"RealVisXL_V5.0_fp16.safetensors",
		"sd_xl_base_1.0.safetensors",
	}
	got := selectImageCheckpoint("sd_xl_base_1.0.safetensors", available, "anime series")
	if got != "sd_xl_base_1.0.safetensors" {
		t.Fatalf("anime project selected photorealistic checkpoint: %q", got)
	}
	got = selectImageCheckpoint("sd_xl_base_1.0.safetensors", available, "live action cinematic short")
	if got != "RealVisXL_V5.0_fp16.safetensors" {
		t.Fatalf("live action project did not select photorealistic checkpoint: %q", got)
	}
}

func TestGenericAssetReferencePromptPreservesAnimeStyle(t *testing.T) {
	asset := &model.DramaAsset{
		Type:            "character",
		Name:            "Astra",
		ReferencePrompt: "anime teenage heroine, red jacket, silver hair",
	}
	prompt := buildAssetReferencePrompt(asset, asset.ReferencePrompt, "cel-shaded anime adventure")
	if !strings.Contains(prompt, "anime character design") || strings.Contains(prompt, "photorealistic") {
		t.Fatalf("unexpected anime asset prompt: %q", prompt)
	}
	settings := assetReferenceSettings(ImageGenerationSettings{}, "anime")
	if strings.Contains(settings.NegativePrompt, "anime,") {
		t.Fatalf("anime style was added to the negative prompt: %q", settings.NegativePrompt)
	}
}
