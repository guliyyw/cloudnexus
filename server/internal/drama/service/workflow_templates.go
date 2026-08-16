package service

import (
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

//go:embed workflows/*.json
var dramaWorkflowTemplates embed.FS

var exactWorkflowPlaceholder = regexp.MustCompile(`^\{\{([a-zA-Z0-9_]+)\}\}$`)

func renderDramaWorkflowTemplate(name string, values map[string]interface{}) (map[string]interface{}, error) {
	data, err := dramaWorkflowTemplates.ReadFile("workflows/" + name)
	if err != nil {
		return nil, fmt.Errorf("read workflow template %s: %w", name, err)
	}
	var workflow map[string]interface{}
	if err := json.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("parse workflow template %s: %w", name, err)
	}
	rendered, err := renderWorkflowValue(workflow, values)
	if err != nil {
		return nil, fmt.Errorf("render workflow template %s: %w", name, err)
	}
	result, ok := rendered.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("workflow template %s did not produce an object", name)
	}
	return result, nil
}

func renderWorkflowValue(value interface{}, values map[string]interface{}) (interface{}, error) {
	switch typed := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			rendered, err := renderWorkflowValue(item, values)
			if err != nil {
				return nil, err
			}
			result[key] = rendered
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(typed))
		for index, item := range typed {
			rendered, err := renderWorkflowValue(item, values)
			if err != nil {
				return nil, err
			}
			result[index] = rendered
		}
		return result, nil
	case string:
		match := exactWorkflowPlaceholder.FindStringSubmatch(typed)
		if len(match) == 2 {
			replacement, ok := values[match[1]]
			if !ok {
				return nil, fmt.Errorf("missing placeholder %q", match[1])
			}
			return replacement, nil
		}
		for key, replacement := range values {
			typed = strings.ReplaceAll(typed, "{{"+key+"}}", fmt.Sprint(replacement))
		}
		if strings.Contains(typed, "{{") {
			return nil, fmt.Errorf("unresolved placeholder in %q", typed)
		}
		return typed, nil
	default:
		return value, nil
	}
}

func systemRegionalStoryboardWorkflow(prompt string, settings ImageGenerationSettings, uploaded []string, references []ComfyReferenceImage) (map[string]interface{}, error) {
	if len(uploaded) != len(references) {
		return nil, fmt.Errorf("uploaded reference count does not match reference metadata")
	}
	characterCount := 0
	sceneCount := 0
	for _, reference := range references {
		switch reference.Kind {
		case "character":
			characterCount++
		case "scene":
			sceneCount++
		}
	}
	if characterCount == 0 && sceneCount == 0 {
		return nil, fmt.Errorf("regional workflow requires character or scene assets")
	}

	workflow, err := renderDramaWorkflowTemplate("storyboard_regional_sdxl_v1.json", map[string]interface{}{
		"checkpoint":      settings.Checkpoint,
		"positive_prompt": prompt,
		"negative_prompt": settings.NegativePrompt,
		"width":           settings.Width,
		"height":          settings.Height,
		"seed":            imageGenerationSeed(settings),
		"steps":           settings.Steps,
		"cfg":             settings.CFG,
		"sampler":         settings.Sampler,
		"scheduler":       settings.Scheduler,
	})
	if err != nil {
		return nil, err
	}

	nextID := 20
	modelRef := []interface{}{"1", 0}
	positiveRef := []interface{}{"2", 0}
	for index, reference := range references {
		if reference.Kind != "scene" {
			continue
		}
		loadID := strconv.Itoa(nextID)
		nextID++
		prepID := strconv.Itoa(nextID)
		nextID++
		loaderID := strconv.Itoa(nextID)
		nextID++
		applyID := strconv.Itoa(nextID)
		nextID++
		workflow[loadID] = workflowNode("LoadImage", map[string]interface{}{"image": uploaded[index]}, "Scene reference: "+reference.Name)
		workflow[prepID] = workflowNode("PrepImageForClipVision", map[string]interface{}{
			"image": []interface{}{loadID, 0}, "interpolation": "LANCZOS", "crop_position": "center", "sharpening": 0.05,
		}, "Prepare scene reference")
		workflow[loaderID] = workflowNode("IPAdapterUnifiedLoader", map[string]interface{}{
			"model": modelRef, "preset": "PLUS (high strength)",
		}, "Load scene IPAdapter")
		workflow[applyID] = workflowNode("IPAdapterAdvanced", map[string]interface{}{
			"model": modelRef, "ipadapter": []interface{}{loaderID, 1}, "image": []interface{}{prepID, 0},
			"weight": 0.3, "weight_type": "style transfer precise", "combine_embeds": "average",
			"start_at": 0.0, "end_at": 0.55, "embeds_scaling": "V only", "clip_vision": []interface{}{"8", 0},
		}, "Apply scene globally")
		modelRef = []interface{}{applyID, 0}
	}

	if characterCount > 0 {
		characterLoaderID := strconv.Itoa(nextID)
		nextID++
		characterLoaderType := "IPAdapterUnifiedLoader"
		characterLoaderInputs := map[string]interface{}{
			"model": modelRef, "preset": "PLUS FACE (portraits)",
		}
		if settings.UseFaceID {
			characterLoaderType = "IPAdapterUnifiedLoaderFaceID"
			characterLoaderInputs = map[string]interface{}{
				"model": modelRef, "preset": "FACEID PLUS V2", "lora_strength": 0.65, "provider": "CPU",
			}
		}
		workflow[characterLoaderID] = workflowNode(characterLoaderType, characterLoaderInputs, "Load character identity IPAdapter")
		modelRef = []interface{}{characterLoaderID, 0}
		characterIndex := 0
		for index, reference := range references {
			if reference.Kind != "character" {
				continue
			}
			loadID := strconv.Itoa(nextID)
			nextID++
			prepID := strconv.Itoa(nextID)
			nextID++
			zeroMaskID := strconv.Itoa(nextID)
			nextID++
			regionMaskID := strconv.Itoa(nextID)
			nextID++
			compositeMaskID := strconv.Itoa(nextID)
			nextID++
			featherMaskID := strconv.Itoa(nextID)
			nextID++
			characterPromptID := strconv.Itoa(nextID)
			nextID++
			characterConditionID := strconv.Itoa(nextID)
			nextID++
			combinedConditionID := strconv.Itoa(nextID)
			nextID++
			applyID := strconv.Itoa(nextID)
			nextID++
			x, regionWidth := characterRegionBounds(characterIndex, characterCount, settings.Width)
			workflow[loadID] = workflowNode("LoadImage", map[string]interface{}{"image": uploaded[index]}, "Character reference: "+reference.Name)
			workflow[prepID] = workflowNode("PrepImageForClipVision", map[string]interface{}{
				"image": []interface{}{loadID, 0}, "interpolation": "LANCZOS", "crop_position": "center", "sharpening": 0.1,
			}, "Prepare character reference")
			workflow[zeroMaskID] = workflowNode("SolidMask", map[string]interface{}{
				"value": 0.0, "width": settings.Width, "height": settings.Height,
			}, "Empty character mask")
			workflow[regionMaskID] = workflowNode("SolidMask", map[string]interface{}{
				"value": 1.0, "width": regionWidth, "height": settings.Height,
			}, "Character region")
			workflow[compositeMaskID] = workflowNode("MaskComposite", map[string]interface{}{
				"destination": []interface{}{zeroMaskID, 0}, "source": []interface{}{regionMaskID, 0},
				"x": x, "y": 0, "operation": "add",
			}, "Place character region")
			workflow[featherMaskID] = workflowNode("FeatherMask", map[string]interface{}{
				"mask": []interface{}{compositeMaskID, 0}, "left": 64, "top": 16, "right": 64, "bottom": 16,
			}, "Feather character region")
			regionLabel := characterRegionLabel(characterIndex, characterCount)
			characterName := strings.TrimSuffix(reference.Name, filepath.Ext(reference.Name))
			characterPrompt := compactRegionalPrompt(reference.Prompt, 260)
			if characterPrompt == "" {
				characterPrompt = characterName
			}
			workflow[characterPromptID] = workflowNode("CLIPTextEncode", map[string]interface{}{
				"text": fmt.Sprintf("one visible %s, %s, located on the %s, medium shot, complete upper body, clear face", characterSemanticLabel(characterName, reference.Prompt), characterPrompt, regionLabel),
				"clip": []interface{}{"1", 1},
			}, "Character regional prompt")
			workflow[characterConditionID] = workflowNode("ConditioningSetMask", map[string]interface{}{
				"conditioning": []interface{}{characterPromptID, 0}, "mask": []interface{}{featherMaskID, 0},
				"strength": 0.72, "set_cond_area": "mask bounds",
			}, "Bind character prompt to region")
			workflow[combinedConditionID] = workflowNode("ConditioningCombine", map[string]interface{}{
				"conditioning_1": positiveRef, "conditioning_2": []interface{}{characterConditionID, 0},
			}, "Combine regional character prompt")
			positiveRef = []interface{}{combinedConditionID, 0}
			applyType := "IPAdapterAdvanced"
			applyInputs := map[string]interface{}{
				"model": modelRef, "ipadapter": []interface{}{characterLoaderID, 1}, "image": []interface{}{prepID, 0},
				"weight": 0.72, "weight_type": "linear", "combine_embeds": "concat",
				"start_at": 0.0, "end_at": 0.82, "embeds_scaling": "K+V w/ C penalty",
				"attn_mask": []interface{}{featherMaskID, 0}, "clip_vision": []interface{}{"8", 0},
			}
			if settings.UseFaceID {
				applyType = "IPAdapterFaceID"
				applyInputs["weight"] = 0.78
				applyInputs["weight_faceidv2"] = 0.85
			}
			workflow[applyID] = workflowNode(applyType, applyInputs, fmt.Sprintf("Bind character %d identity to region", characterIndex+1))
			modelRef = []interface{}{applyID, 0}
			characterIndex++
		}
	}

	sampler := workflow["5"].(map[string]interface{})
	samplerInputs := sampler["inputs"].(map[string]interface{})
	samplerInputs["model"] = modelRef
	samplerInputs["positive"] = positiveRef
	return workflow, nil
}

func characterRegionBounds(index, count, width int) (int, int) {
	if count <= 1 {
		return 0, width
	}
	cellWidth := width / count
	overlap := cellWidth / 3
	start := index*cellWidth - overlap
	if start < 0 {
		start = 0
	}
	end := (index+1)*cellWidth + overlap
	if end > width {
		end = width
	}
	return start, end - start
}

func characterSemanticLabel(name, prompt string) string {
	lower := " " + strings.ToLower(name+" "+prompt) + " "
	if containsAny(lower, []string{"dog", "puppy", "犬", "狗"}) {
		return "dog"
	}
	if containsAny(lower, []string{"cat", "kitten", "猫"}) {
		return "cat"
	}
	if containsAny(lower, []string{"robot", "android", "机器人", "机械人"}) {
		return "robot character"
	}
	isChild := containsAny(lower, []string{"child", "kid", "young boy", "young girl", "儿童", "小孩", "男孩", "女孩", "少年", "少女"})
	isOlder := containsAny(lower, []string{"elderly", "older man", "older woman", "senior", "老人", "老年", "爷爷", "奶奶"})
	isFemale := containsAny(lower, []string{
		" woman", " female", " girl", " she ", " her ", "wife", "mother", "girlfriend",
		"女性", "女人", "女孩", "妻子", "母亲", "妈妈", "奶奶", "姐姐", "妹妹", "女友",
	})
	isMale := containsAny(lower, []string{
		" man", " male", " boy", " he ", " his ", "husband", "father", "boyfriend",
		"男性", "男人", "男孩", "丈夫", "父亲", "爸爸", "爷爷", "哥哥", "弟弟", "男友",
	})
	switch {
	case isChild && isFemale:
		return "girl"
	case isChild && isMale:
		return "boy"
	case isOlder && isFemale:
		return "elderly woman"
	case isOlder && isMale:
		return "elderly man"
	case isFemale:
		return "adult woman"
	case isMale:
		return "adult man"
	default:
		return "character"
	}
}

func compactRegionalPrompt(prompt string, limit int) string {
	prompt = strings.Join(strings.Fields(prompt), " ")
	runes := []rune(prompt)
	if len(runes) <= limit {
		return prompt
	}
	return strings.TrimSpace(string(runes[:limit]))
}

func workflowNode(classType string, inputs map[string]interface{}, title string) map[string]interface{} {
	node := map[string]interface{}{"class_type": classType, "inputs": inputs}
	if strings.TrimSpace(title) != "" {
		node["_meta"] = map[string]interface{}{"title": title}
	}
	return node
}
