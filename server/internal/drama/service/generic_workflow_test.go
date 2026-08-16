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

func TestCharacterViewPromptsGenerateOnePersonPerView(t *testing.T) {
	asset := &model.DramaAsset{
		Type:            "character",
		Name:            "Astra",
		Description:     "anime teenage heroine, red jacket, silver hair",
		ReferencePrompt: "teenage heroine with silver hair and a red jacket",
	}
	prompts := buildCharacterViewPrompts(asset, asset.ReferencePrompt, "cel-shaded anime adventure")
	if len(prompts) != 3 {
		t.Fatalf("character view prompt count = %d, want 3", len(prompts))
	}
	orientations := []string{"STRICT ORIENTATION: FRONT VIEW ONLY", "STRICT ORIENTATION: LEFT PROFILE ONLY", "STRICT ORIENTATION: BACK VIEW ONLY"}
	for index, prompt := range prompts {
		if !strings.Contains(prompt, "anime full-body studio character reference") || strings.Contains(prompt, "photorealistic") {
			t.Fatalf("unexpected anime view prompt %d: %q", index, prompt)
		}
		if !strings.Contains(prompt, "exactly one full-body depiction") || !strings.Contains(prompt, orientations[index]) {
			t.Fatalf("view prompt %d is not isolated to one orientation: %q", index, prompt)
		}
		for _, forbidden := range []string{"triptych", "contact sheet", "three full-body views"} {
			if strings.Contains(prompt, forbidden) {
				t.Fatalf("view prompt %d leaked multi-view instruction %q: %q", index, forbidden, prompt)
			}
		}
	}
	if !strings.Contains(characterViewOrientationNegative(1), "front-facing") || !strings.Contains(characterViewOrientationNegative(2), "visible face") {
		t.Fatal("profile and rear view negatives must explicitly reject front-facing output")
	}
	settings := characterViewReferenceSettings(ImageGenerationSettings{}, "anime")
	if strings.Contains(settings.NegativePrompt, "anime,") {
		t.Fatalf("anime style was added to the negative prompt: %q", settings.NegativePrompt)
	}
	for _, required := range []string{"multiple people", "duplicate person", "triptych", "contact sheet"} {
		if !strings.Contains(settings.NegativePrompt, required) {
			t.Fatalf("turnaround negative prompt is missing %q: %q", required, settings.NegativePrompt)
		}
	}
}

func TestCharacterReferencePromptExcludesNonVisualDescription(t *testing.T) {
	asset := &model.DramaAsset{
		Type: "character",
		Name: "Mother",
		Description: strings.Join([]string{
			"年龄：58岁",
			"外貌：圆脸，头发盘起，微胖丰满",
			"服装：红色棉质家居服，碎花围裙，棉拖鞋",
			"性格：热情开朗，眼神充满慈爱",
			"音色建议：慈祥老年女声，温和响亮",
		}, "\n"),
	}
	prompts := buildCharacterViewPrompts(asset, "", "realistic family drama")
	prompt := strings.Join(prompts, "\n")
	for _, forbidden := range []string{"性格", "热情开朗", "音色", "老年女声", "cinematic reference photo"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("character prompt leaked non-visual or conflicting text %q: %q", forbidden, prompt)
		}
	}
	for _, required := range []string{"58岁", "圆脸", "红色棉质家居服", "photorealistic full-body studio character reference"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("character prompt lost visual detail %q: %q", required, prompt)
		}
	}
}

func TestCharacterReferencePromptPrefersCuratedAppearance(t *testing.T) {
	asset := &model.DramaAsset{
		Type:            "character",
		Description:     "性格：热情开朗\n音色建议：慈祥女声",
		ReferencePrompt: "58-year-old Chinese woman, round face, dark hair in a low bun, red cotton lounge set, floral apron, beige slippers, neutral expression, closed mouth",
	}
	prompt := strings.Join(buildCharacterViewPrompts(asset, asset.ReferencePrompt, "realistic"), "\n")
	if !strings.Contains(prompt, "58-year-old Chinese woman") {
		t.Fatalf("curated visual prompt was not used: %q", prompt)
	}
	if strings.Contains(prompt, "热情开朗") || strings.Contains(prompt, "慈祥女声") {
		t.Fatalf("description polluted curated visual prompt: %q", prompt)
	}
}

func TestStoryboardAssetPromptDoesNotLeakCharacterSheetLayout(t *testing.T) {
	asset := model.DramaAsset{
		Type:            "character",
		Description:     "Chinese woman, short black hair, cream sweater",
		ReferencePrompt: "professional character turnaround model sheet, exactly three full-body views",
	}
	prompt := storyboardAssetPrompt(asset)
	if strings.Contains(strings.ToLower(prompt), "turnaround") || strings.Contains(strings.ToLower(prompt), "three full-body views") {
		t.Fatalf("storyboard character prompt leaked asset-sheet layout: %q", prompt)
	}
	if !strings.Contains(prompt, "cream sweater") {
		t.Fatalf("storyboard character prompt lost stable appearance: %q", prompt)
	}
}
