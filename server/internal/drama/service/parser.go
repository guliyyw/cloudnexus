package service

import (
	"regexp"
	"sort"
	"strings"
)

type ParsedStoryboard struct {
	Seq         int    `json:"seq"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Prompt      string `json:"prompt"`
	Dialogue    string `json:"dialogue"`
	SceneAnchor string `json:"scene_anchor"`
	Plot        string `json:"plot"`
}

type ParsedScript struct {
	Preface     string             `json:"preface"`
	Storyboards []ParsedStoryboard `json:"storyboards"`
}

var storyboardMarker = regexp.MustCompile(`(?m)(?:【\s*分镜序号\s*】|\[\s*分镜序号\s*\]|分镜序号)\s*[:：]\s*(\d+)(?:\s*/\s*\d+)?`)

func ParseScript(script string) ParsedScript {
	matches := storyboardMarker.FindAllStringSubmatchIndex(script, -1)
	if len(matches) == 0 {
		trimmed := strings.TrimSpace(script)
		return ParsedScript{Storyboards: []ParsedStoryboard{{Seq: 1, Title: "分镜 1", Content: trimmed, Plot: extractSection(trimmed, "整镜剧情"), Dialogue: extractDialogue(trimmed)}}}
	}

	result := ParsedScript{
		Preface:     strings.TrimSpace(script[:matches[0][0]]),
		Storyboards: make([]ParsedStoryboard, 0, len(matches)),
	}
	for i, m := range matches {
		start := m[0]
		end := len(script)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		block := strings.TrimSpace(script[start:end])
		seqText := script[m[2]:m[3]]
		seq := atoi(seqText)
		if seq == 0 {
			seq = i + 1
		}
		result.Storyboards = append(result.Storyboards, ParsedStoryboard{
			Seq:         seq,
			Title:       "分镜 " + seqText,
			Content:     block,
			Prompt:      synthesizePrompt(block),
			Dialogue:    extractDialogue(block),
			SceneAnchor: extractSection(block, "场景锚定"),
			Plot:        extractSection(block, "整镜剧情"),
		})
	}
	sort.Slice(result.Storyboards, func(i, j int) bool {
		return result.Storyboards[i].Seq < result.Storyboards[j].Seq
	})
	return result
}

func ExtractAssets(preface string) (characters []ParsedAsset, scenes []ParsedAsset) {
	characters = extractAssetBlock(preface, []string{"角色", "人物"})
	scenes = extractAssetBlock(preface, []string{"场景", "地点"})
	return characters, scenes
}

type ParsedAsset struct {
	Name        string
	Description string
}

func extractAssetBlock(text string, keywords []string) []ParsedAsset {
	lines := strings.Split(text, "\n")
	assets := make([]ParsedAsset, 0)
	seen := map[string]bool{}
	for _, raw := range lines {
		line := strings.TrimSpace(strings.Trim(raw, "-*• \t"))
		if line == "" {
			continue
		}
		if !containsAny(line, keywords) && !strings.Contains(line, "：") && !strings.Contains(line, ":") {
			continue
		}
		name, desc := splitNameDescription(line)
		if name == "" || seen[name] {
			continue
		}
		if len([]rune(name)) > 30 {
			continue
		}
		seen[name] = true
		assets = append(assets, ParsedAsset{Name: name, Description: desc})
	}
	return assets
}

func splitNameDescription(line string) (string, string) {
	line = strings.TrimPrefix(line, "【角色】")
	line = strings.TrimPrefix(line, "【人物】")
	line = strings.TrimPrefix(line, "【场景】")
	line = strings.TrimPrefix(line, "【地点】")
	parts := strings.SplitN(line, "：", 2)
	if len(parts) < 2 {
		parts = strings.SplitN(line, ":", 2)
	}
	if len(parts) < 2 {
		return strings.TrimSpace(line), ""
	}
	name := strings.TrimSpace(parts[0])
	desc := strings.TrimSpace(parts[1])
	if name == "角色" || name == "人物" || name == "场景" || name == "地点" {
		nested := strings.SplitN(desc, "：", 2)
		if len(nested) < 2 {
			nested = strings.SplitN(desc, ":", 2)
		}
		if len(nested) == 2 {
			return strings.TrimSpace(nested[0]), strings.TrimSpace(nested[1])
		}
	}
	name = strings.TrimPrefix(name, "角色")
	name = strings.TrimPrefix(name, "人物")
	name = strings.TrimPrefix(name, "场景")
	name = strings.TrimPrefix(name, "地点")
	name = strings.TrimSpace(name)
	return name, desc
}

func extractDialogue(block string) string {
	return extractSection(block, "人物对白")
}

func extractSection(block, name string) string {
	pattern := regexp.MustCompile(`(?s)(?:【\s*` + regexp.QuoteMeta(name) + `\s*】|\[\s*` + regexp.QuoteMeta(name) + `\s*\])\s*[:：]?\s*(.*?)(?:\n\s*(?:【[^】]+】|\[[^\]]+\])|$)`)
	match := pattern.FindStringSubmatch(block)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func synthesizePrompt(block string) string {
	parts := []string{
		extractSection(block, "场景锚定"),
		extractSection(block, "整镜剧情"),
		extractSection(block, "画面提示词"),
		extractSection(block, "提示词"),
	}
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			kept = append(kept, strings.TrimSpace(part))
		}
	}
	return strings.Join(kept, "，")
}

func containsAny(s string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
	}
	return n
}
