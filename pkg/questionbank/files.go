package questionbank

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"Q-Solver/pkg/llm"
)

func newID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
func validID(id string) bool {
	b, err := hex.DecodeString(id)
	return err == nil && len(b) == 12 && strings.ToLower(id) == id
}
func safeName(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, value)
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > 40 {
		value = string(runes[:40])
	}
	value = strings.TrimRight(value, ". ")
	if value == "" {
		return "未命名"
	}
	base := strings.ToUpper(strings.SplitN(value, ".", 2)[0])
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || base == "CLOCK$" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9') {
		value = "_" + value
	}
	return value
}
func decodeImage(dataURL string) ([]byte, string, error) {
	if len(dataURL) > 28*1024*1024 {
		return nil, "", fmt.Errorf("单张截图超过 20 MiB")
	}
	mime, data := llm.ParseBase64DataURL(dataURL)
	if (mime != "image/png" && mime != "image/jpeg") || !strings.HasPrefix(dataURL, "data:"+mime+";base64,") {
		return nil, "", fmt.Errorf("仅支持 PNG/JPEG base64 截图")
	}
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, "", fmt.Errorf("截图编码无效")
	}
	if len(b) > 20*1024*1024 {
		return nil, "", fmt.Errorf("单张截图超过 20 MiB")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 80000000 {
		return nil, "", fmt.Errorf("截图格式或尺寸无效")
	}
	if mime != "image/"+format {
		return nil, "", fmt.Errorf("截图实际格式与声明不一致")
	}
	ext := ".png"
	if format == "jpeg" {
		ext = ".jpg"
	}
	return b, ext, nil
}

// atomicJSON never truncates the previous version if writing the replacement fails.
func atomicJSON(path string, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, append(data, '\n'))
}
func atomicWrite(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".writing-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// safeChild rejects traversal and symlinks before reading user-editable manifests.
func safeChild(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || strings.Contains(relative, `\`) {
		return "", fmt.Errorf("无效归档相对路径")
	}
	parts := strings.Split(relative, "/")
	path := root
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.Contains(part, ":") {
			return "", fmt.Errorf("无效归档相对路径")
		}
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("归档路径不能使用符号链接")
		}
	}
	return path, nil
}
func readLimited(path string, max int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > max {
		return nil, fmt.Errorf("归档文件过大或不是普通文件")
	}
	return os.ReadFile(path)
}

// mkdirWithin requires the chosen root to still exist and rejects redirected children.
func mkdirWithin(root, relative string) error {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("输出根目录不存在或已变更，请恢复目录后重试保存")
	}
	path := root
	for _, part := range strings.Split(filepath.ToSlash(relative), "/") {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `\:`) {
			return fmt.Errorf("无效归档目录")
		}
		path = filepath.Join(path, part)
		if err = os.Mkdir(path, 0700); err != nil && !os.IsExist(err) {
			return err
		}
		info, err = os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("归档目录被文件或链接占用")
		}
	}
	return nil
}
