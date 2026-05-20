package middleware

import "testing"

func TestConvertVolcesRequestToUnified(t *testing.T) {
	raw := map[string]any{
		"model": "doubao-seedance-2-0-260128",
		"content": []any{
			map[string]any{
				"type": "text",
				"text": "first prompt",
			},
			map[string]any{
				"type": "image_url",
				"image_url": map[string]any{
					"url": "https://example.com/ref.png",
				},
				"role": "reference_image",
			},
			map[string]any{
				"type": "text",
				"text": "second prompt",
			},
		},
		"duration":       float64(11),
		"generate_audio": true,
		"ratio":          "16:9",
	}

	unified, err := convertVolcesRequestToUnified(raw)
	if err != nil {
		t.Fatalf("convertVolcesRequestToUnified returned error: %v", err)
	}

	if got := unified["model"]; got != "doubao-seedance-2-0-260128" {
		t.Fatalf("unexpected model: %#v", got)
	}
	if got := unified["prompt"]; got != "first prompt\nsecond prompt" {
		t.Fatalf("unexpected prompt: %#v", got)
	}
	if got := unified["seconds"]; got != "11" {
		t.Fatalf("unexpected seconds: %#v", got)
	}

	metadata, ok := unified["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata missing or invalid: %#v", unified["metadata"])
	}
	if _, exists := metadata["duration"]; exists {
		t.Fatalf("duration should not remain in metadata: %#v", metadata["duration"])
	}
	content, ok := metadata["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("unexpected metadata.content: %#v", metadata["content"])
	}
	if got := metadata["generate_audio"]; got != true {
		t.Fatalf("unexpected generate_audio: %#v", got)
	}
	if got := metadata["ratio"]; got != "16:9" {
		t.Fatalf("unexpected ratio: %#v", got)
	}
}
