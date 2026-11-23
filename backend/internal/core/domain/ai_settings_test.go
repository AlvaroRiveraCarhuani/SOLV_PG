package domain_test

import (
	"testing"

	"solv-backend/internal/core/domain"
)

func TestClassifyAIEngine_CustomProvider_LAN_vs_Cloud(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		baseURL  string
		expected string
	}{
		{
			name:     "ollama_lan always local_gpu",
			provider: domain.AIProviderOllamaLAN,
			model:    "llama3",
			baseURL:  "http://10.0.0.1:11434",
			expected: domain.AIEngineProfileLocalGPU,
		},
		{
			name:     "custom_lan localhost",
			provider: "custom_local_server",
			model:    "qwen2.5",
			baseURL:  "http://localhost:11434/v1",
			expected: domain.AIEngineProfileLocalGPU,
		},
		{
			name:     "custom_lan 127.0.0.1",
			provider: "custom_box",
			model:    "qwen2.5",
			baseURL:  "http://127.0.0.1:8000/v1",
			expected: domain.AIEngineProfileLocalGPU,
		},
		{
			name:     "custom_lan 10.x subnet",
			provider: "custom_facultad",
			model:    "llama-3.3-70b",
			baseURL:  "http://10.20.1.50:11434/v1",
			expected: domain.AIEngineProfileLocalGPU,
		},
		{
			name:     "custom_lan 172.16-31.x subnet",
			provider: "custom_lab",
			model:    "codellama",
			baseURL:  "http://172.20.5.10:8080/v1",
			expected: domain.AIEngineProfileLocalGPU,
		},
		{
			name:     "custom_lan 192.168.x subnet",
			provider: "custom_campus",
			model:    "mistral",
			baseURL:  "http://192.168.1.100:11434/v1",
			expected: domain.AIEngineProfileLocalGPU,
		},
		{
			name:     "custom_lan .local domain",
			provider: "custom_mdns",
			model:    "deepseek-coder",
			baseURL:  "http://gpu-rig.local:11434/v1",
			expected: domain.AIEngineProfileLocalGPU,
		},
		{
			name:     "custom_lan .lan domain",
			provider: "custom_internal_lan",
			model:    "phi-4",
			baseURL:  "http://ai-server.lan:8000/v1",
			expected: domain.AIEngineProfileLocalGPU,
		},
		{
			name:     "custom_lan hostname without dot",
			provider: "custom_box_nodot",
			model:    "llama3.2",
			baseURL:  "http://gpushop:11434/v1",
			expected: domain.AIEngineProfileLocalGPU,
		},
		{
			name:     "custom sin base URL nunca es local_gpu",
			provider: "custom_pendiente",
			model:    "llama-3.3-70b",
			baseURL:  "",
			expected: domain.AIEngineProfileFastCloud,
		},
		{
			name:     "custom sin base URL con modelo de razonamiento no es local_gpu",
			provider: "custom_pendiente",
			model:    "deepseek-r1",
			baseURL:  "",
			expected: domain.AIEngineProfileQueuedOrReasoning,
		},
		{
			name:     "custom cloud public domain HTTPS fast_cloud",
			provider: "custom_inception",
			model:    "mercury-v1",
			baseURL:  "https://api.inceptionlabs.ai/v1",
			expected: domain.AIEngineProfileFastCloud,
		},
		{
			name:     "custom cloud public domain HTTPS mistral",
			provider: "custom_mistral",
			model:    "mistral-large-latest",
			baseURL:  "https://api.mistral.ai/v1",
			expected: domain.AIEngineProfileFastCloud,
		},
		{
			name:     "custom cloud with reasoning model",
			provider: "custom_partner",
			model:    "deepseek-reasoner",
			baseURL:  "https://partner-ai.example.com/v1",
			expected: domain.AIEngineProfileQueuedOrReasoning,
		},
		{
			name:     "custom cloud with free model token",
			provider: "custom_community",
			model:    "meta-llama/llama-3.3-70b:free",
			baseURL:  "https://community-hub.org/api/v1",
			expected: domain.AIEngineProfileQueuedOrReasoning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.ClassifyAIEngine(tt.provider, tt.model, tt.baseURL)
			if got != tt.expected {
				t.Errorf("ClassifyAIEngine(%q, %q, %q) = %q, want %q",
					tt.provider, tt.model, tt.baseURL, got, tt.expected)
			}
		})
	}
}

func TestIsPrivateOrLocalHost(t *testing.T) {
	privateURLs := []string{
		"http://localhost:11434",
		"http://127.0.0.1:8000",
		"http://[::1]:11434",
		"http://10.0.0.1",
		"http://172.16.0.1",
		"http://172.31.255.255",
		"http://192.168.0.1:11434",
		"http://my-gpu.local/v1",
		"http://ollama.lan:8000",
		"http://gpushop:11434/v1",
		"http://0.0.0.0:11434",
		"http://[::]:11434",
		"http://169.254.169.254/latest/meta-data",
		"http://[fe80::1]:11434",
		"http://[fd00::1]:11434",
	}

	for _, u := range privateURLs {
		if !domain.IsPrivateOrLocalHost(u) {
			t.Errorf("IsPrivateOrLocalHost(%q) = false, want true", u)
		}
	}

	publicURLs := []string{
		"https://api.inceptionlabs.ai/v1",
		"https://api.mistral.ai/v1",
		"https://openrouter.ai/api/v1",
		"https://api.openai.com/v1",
		"https://172.32.0.1",
		"https://8.8.8.8",
		"https://api.openai.com/v1",
	}

	for _, u := range publicURLs {
		if domain.IsPrivateOrLocalHost(u) {
			t.Errorf("IsPrivateOrLocalHost(%q) = true, want false", u)
		}
	}
}
