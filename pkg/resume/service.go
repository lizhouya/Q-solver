package resume

import (
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/prompts"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Service struct {
	mu           sync.RWMutex
	config       config.Config
	resumeBase64 string
}

func NewService(cfg config.Config, cm *config.ConfigManager) *Service {
	s := &Service{
		config: cfg,
	}

	cm.Subscribe(func(newConfig config.Config, oldConfig config.Config) {
		s.mu.Lock()
		s.config = newConfig
		if newConfig.ResumePath != oldConfig.ResumePath {
			s.resumeBase64 = ""
		}
		s.mu.Unlock()
	})

	return s
}

func (s *Service) SelectResume(ctx context.Context) string {
	selection, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title: "选择简历 (PDF)",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "PDF Files",
				Pattern:     "*.pdf",
			},
		},
	})

	if err != nil {
		logger.Printf("选择文件失败: %v\n", err)
		return ""
	}

	if selection == "" {
		return ""
	}

	return selection
}

func (s *Service) ClearResume() {
	s.mu.Lock()
	s.resumeBase64 = ""
	s.mu.Unlock()
	logger.Println("简历缓存已清除")
}

func (s *Service) GetResumeBase64() (string, error) {
	s.mu.RLock()
	cached := s.resumeBase64
	resumePath := s.config.ResumePath
	s.mu.RUnlock()

	if len(cached) > 0 {
		logger.Println("使用缓存的简历 Base64")
		return cached, nil
	}
	if resumePath == "" {
		return "", nil
	}

	fileInfo, err := os.Stat(resumePath)
	if err != nil {
		return "", err
	}

	const maxResumeSize = 5 * 1024 * 1024
	if fileInfo.Size() > maxResumeSize {
		return "", fmt.Errorf("简历文件大小超过 5MB 限制")
	}

	content, err := os.ReadFile(resumePath)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(content)
	s.mu.Lock()
	s.resumeBase64 = encoded
	s.mu.Unlock()
	return encoded, nil
}

func (s *Service) ParseResume(ctx context.Context) (string, error) {
	resumeBase64, err := s.GetResumeBase64()
	if err != nil {
		return "", fmt.Errorf("读取简历失败: %v", err)
	}
	if resumeBase64 == "" {
		return "", fmt.Errorf("请先选择简历文件")
	}

	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	if strings.TrimSpace(cfg.APIKey) == "" {
		return "", fmt.Errorf("请先配置 API Key")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return "", fmt.Errorf("请先选择模型")
	}

	logger.Println("开始通过当前模型解析简历")

	provider, err := llm.NewProvider(cfg)
	if err != nil {
		return "", err
	}
	messages := []llm.Message{
		llm.NewSystemMessage(prompts.ResumeParsePrompt),
		llm.NewMultiPartMessage(llm.RoleUser, []llm.ContentPart{
			llm.TextPart("请将这份简历解析并整理为结构清晰的 Markdown。"),
			llm.PDFPart(resumeBase64),
		}),
	}

	result, err := provider.GenerateContent(ctx, cfg.Model, messages)
	if err != nil {
		logger.Printf("简历解析失败: %v", err)
		return "", err
	}

	content := strings.TrimSpace(result.Content)
	if content == "" {
		return "", fmt.Errorf("模型没有返回简历解析结果")
	}

	return content, nil
}
