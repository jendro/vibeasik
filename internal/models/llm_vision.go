package models

import (
	"encoding/json"
	"sort"
	"strings"
)

// Dukungan gambar (vision) per model LLM. Nilai bawaan ditebak dari nama model; pengguna bisa menimpanya di
// Pengaturan LLM (disimpan di LLMConfig.VisionJSON). Dipakai fitur yang mengirim screenshot, mis. Galeri Desain.

var visionHints = []string{
	"gpt-4o", "gpt-4.1", "gpt-4-turbo", "gpt-4-vision", "gpt-5", "chatgpt-4o", "o3", "o4",
	"claude", "gemini", "gemma-3", "llava", "bakllava", "pixtral", "mistral-small-3", "mistral-medium",
	"vision", "-vl", "vl-", "qwen-vl", "qvq", "glm-4v", "glm-4.5v", "glm-4.6v", "minicpm-v", "moondream",
	"llama-4", "llama-3.2-11b", "llama-3.2-90b", "grok-4", "grok-2-vision", "kimi-vl", "kimi-k2.5", "internvl", "phi-4-multimodal",
}

var noVisionHints = []string{
	"embed", "tts", "whisper", "audio", "transcribe", "rerank", "moderation", "o3-mini", "o1-mini",
	"gpt-3.5", "deepseek-r1", "deepseek-chat", "deepseek-coder", "codestral", "-coder", "image-generation", "imagen",
}

// GuessModelVision menebak dari nama apakah model menerima input gambar.
func GuessModelVision(model string) bool {
	m := strings.ToLower(model)
	for _, h := range noVisionHints {
		if strings.Contains(m, h) {
			return false
		}
	}
	for _, h := range visionHints {
		if strings.Contains(m, h) {
			return true
		}
	}
	return false
}

func (c *LLMConfig) visionOverrides() map[string]bool {
	out := map[string]bool{}
	if c.VisionJSON != "" {
		_ = json.Unmarshal([]byte(c.VisionJSON), &out)
	}
	return out
}

// ModelSupportsVision: override pengguna bila ada, selain itu tebakan dari nama model.
func (c *LLMConfig) ModelSupportsVision(model string) bool {
	if v, ok := c.visionOverrides()[model]; ok {
		return v
	}
	return GuessModelVision(model)
}

// SetModelVision menyimpan override dukungan gambar untuk satu model.
func (c *LLMConfig) SetModelVision(model string, vision bool) {
	o := c.visionOverrides()
	o[model] = vision
	b, _ := json.Marshal(o)
	c.VisionJSON = string(b)
}

// ModelList mengembalikan daftar model konfigurasi (fallback ke model terpilih).
func (c *LLMConfig) ModelList() []string {
	var list []string
	if c.Models != "" {
		_ = json.Unmarshal([]byte(c.Models), &list)
	}
	if len(list) == 0 && c.SelectedModel != "" {
		list = []string{c.SelectedModel}
	}
	return list
}

func (c *LLMConfig) visionMap(models []string) map[string]bool {
	out := map[string]bool{}
	for _, m := range models {
		out[m] = c.ModelSupportsVision(m)
	}
	return out
}

func (c *LLMConfig) visionManual() []string {
	out := []string{}
	for m := range c.visionOverrides() {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}
