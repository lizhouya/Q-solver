package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync"
)

const (
	mobileStateSchemaVersion = 1
	mobileTokenBytes         = 16
	mobileStatePendingFile   = "mobile-state.pending.enc"
)

var errUnknownMobileStateSchema = errors.New("不支持的 Mobile 状态版本")

// MobileRuntimeState 是仅供后端运行时使用的 Mobile 状态。
type MobileRuntimeState struct {
	StableLink bool
	Token      string
	ClearSeq   uint64
	Generation uint64
}

type mobileDiskState struct {
	SchemaVersion int    `json:"schemaVersion"`
	StableLink    bool   `json:"stableLink"`
	Token         string `json:"token"`
	ClearSeq      uint64 `json:"clearSeq"`
	Generation    uint64 `json:"generation"`
}

// MobileStateManager 管理与普通 Config 分离的加密 Mobile 状态。
type MobileStateManager struct {
	mu          sync.Mutex
	appDir      string
	random      io.Reader
	loaded      bool
	current     MobileRuntimeState
	currentSlot int
}

// NewMobileStateManager 创建使用应用配置目录的 Mobile 状态管理器。
func NewMobileStateManager() *MobileStateManager {
	return newMobileStateManager(appConfigDir(), rand.Reader)
}

func newMobileStateManager(appDir string, random io.Reader) *MobileStateManager {
	return &MobileStateManager{
		appDir:      appDir,
		random:      random,
		currentSlot: -1,
	}
}

func mobileStateFileName(slot int) string {
	return fmt.Sprintf("mobile-state.%d.enc", slot)
}

// LoadOrCreate 加载最新有效状态；没有历史状态时创建默认固定链接状态。
func (m *MobileStateManager) LoadOrCreate() (MobileRuntimeState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.loaded {
		return m.current, nil
	}
	if err := os.MkdirAll(m.appDir, 0755); err != nil {
		return MobileRuntimeState{}, errors.New("无法准备 Mobile 状态目录")
	}

	type slotState struct {
		slot  int
		state mobileDiskState
	}
	valid := make([]slotState, 0, 2)
	existing := 0
	for slot := 0; slot < 2; slot++ {
		state, exists, err := m.readSlot(slot)
		if !exists {
			continue
		}
		existing++
		if errors.Is(err, errUnknownMobileStateSchema) {
			return MobileRuntimeState{}, errUnknownMobileStateSchema
		}
		if err == nil {
			valid = append(valid, slotState{slot: slot, state: state})
		}
	}

	if len(valid) == 0 {
		if existing != 0 {
			return MobileRuntimeState{}, errors.New("Mobile 状态无有效副本")
		}
		token, err := m.newToken()
		if err != nil {
			return MobileRuntimeState{}, errors.New("无法生成 Mobile Token")
		}
		initial := MobileRuntimeState{
			StableLink: true,
			Token:      token,
			Generation: 1,
		}
		if err := m.commit(initial, 0); err != nil {
			return MobileRuntimeState{}, err
		}
		return m.current, nil
	}

	selected := valid[0]
	if len(valid) == 2 {
		if valid[0].state.Generation == valid[1].state.Generation {
			return MobileRuntimeState{}, errors.New("Mobile 状态代次冲突")
		}
		if valid[1].state.Generation > valid[0].state.Generation {
			selected = valid[1]
		}
	}

	runtimeState, err := m.runtimeState(selected.state)
	if err != nil {
		return MobileRuntimeState{}, err
	}
	m.current = runtimeState
	m.currentSlot = selected.slot
	m.loaded = true
	return m.current, nil
}

// AdvanceClearSeq 持久化并返回下一个清屏边界。
func (m *MobileStateManager) AdvanceClearSeq() (MobileRuntimeState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureLoaded(); err != nil {
		return MobileRuntimeState{}, err
	}
	candidate, err := m.nextCandidate()
	if err != nil {
		return MobileRuntimeState{}, err
	}
	if err := m.commit(candidate, 1-m.currentSlot); err != nil {
		return MobileRuntimeState{}, err
	}
	return m.current, nil
}

// ResetLink 轮换 Token，并同时推进清屏边界。
func (m *MobileStateManager) ResetLink() (MobileRuntimeState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureLoaded(); err != nil {
		return MobileRuntimeState{}, err
	}
	candidate, err := m.nextCandidate()
	if err != nil {
		return MobileRuntimeState{}, err
	}
	candidate.Token, err = m.newTokenDifferent(m.current.Token)
	if err != nil {
		return MobileRuntimeState{}, err
	}
	if err := m.commit(candidate, 1-m.currentSlot); err != nil {
		return MobileRuntimeState{}, err
	}
	return m.current, nil
}

// SetStableLink 切换固定链接模式；模式改变时轮换 Token 并推进清屏边界。
func (m *MobileStateManager) SetStableLink(enabled bool) (MobileRuntimeState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureLoaded(); err != nil {
		return MobileRuntimeState{}, err
	}
	if m.current.StableLink == enabled {
		return m.current, nil
	}
	candidate, err := m.nextCandidate()
	if err != nil {
		return MobileRuntimeState{}, err
	}
	candidate.StableLink = enabled
	candidate.Token, err = m.newTokenDifferent(m.current.Token)
	if err != nil {
		return MobileRuntimeState{}, err
	}
	if err := m.commit(candidate, 1-m.currentSlot); err != nil {
		return MobileRuntimeState{}, err
	}
	return m.current, nil
}

func (m *MobileStateManager) ensureLoaded() error {
	if !m.loaded || m.currentSlot < 0 || m.currentSlot > 1 {
		return errors.New("Mobile 状态尚未加载")
	}
	return nil
}

func (m *MobileStateManager) nextCandidate() (MobileRuntimeState, error) {
	if m.current.ClearSeq == math.MaxUint64 {
		return MobileRuntimeState{}, errors.New("Mobile 清屏序号已达上限")
	}
	if m.current.Generation == math.MaxUint64 {
		return MobileRuntimeState{}, errors.New("Mobile 状态代次已达上限")
	}
	candidate := m.current
	candidate.ClearSeq++
	candidate.Generation++
	return candidate, nil
}

func (m *MobileStateManager) newTokenDifferent(current string) (string, error) {
	for attempt := 0; attempt < 3; attempt++ {
		token, err := m.newToken()
		if err != nil {
			return "", errors.New("无法生成 Mobile Token")
		}
		if token != current {
			return token, nil
		}
	}
	return "", errors.New("无法生成不同的 Mobile Token")
}

func (m *MobileStateManager) newToken() (string, error) {
	if m.random == nil {
		return "", errors.New("随机源不可用")
	}
	data := make([]byte, mobileTokenBytes)
	if _, err := io.ReadFull(m.random, data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func (m *MobileStateManager) runtimeState(state mobileDiskState) (MobileRuntimeState, error) {
	token := state.Token
	if !state.StableLink {
		var err error
		token, err = m.newToken()
		if err != nil {
			return MobileRuntimeState{}, errors.New("无法生成临时 Mobile Token")
		}
	}
	return MobileRuntimeState{
		StableLink: state.StableLink,
		Token:      token,
		ClearSeq:   state.ClearSeq,
		Generation: state.Generation,
	}, nil
}

func (m *MobileStateManager) commit(candidate MobileRuntimeState, slot int) error {
	diskState := mobileDiskState{
		SchemaVersion: mobileStateSchemaVersion,
		StableLink:    candidate.StableLink,
		Token:         candidate.Token,
		ClearSeq:      candidate.ClearSeq,
		Generation:    candidate.Generation,
	}
	if !candidate.StableLink {
		diskState.Token = ""
	}
	if err := validateMobileDiskState(diskState); err != nil {
		return errors.New("拒绝写入无效 Mobile 状态")
	}

	plaintext, err := json.Marshal(diskState)
	if err != nil {
		return errors.New("无法序列化 Mobile 状态")
	}
	ciphertext, err := encrypt(plaintext)
	if err != nil {
		return errors.New("无法加密 Mobile 状态")
	}
	pendingPath := filepath.Join(m.appDir, mobileStatePendingFile)
	_ = os.Remove(pendingPath)
	if err := m.writeAndSync(pendingPath, ciphertext); err != nil {
		return err
	}
	defer os.Remove(pendingPath)

	verified, exists, err := readMobileStateFile(pendingPath)
	if err != nil || !exists || verified != diskState {
		return errors.New("Mobile 状态回读验证失败")
	}

	destination := filepath.Join(m.appDir, mobileStateFileName(slot))
	if err := os.Remove(destination); err != nil && !os.IsNotExist(err) {
		return errors.New("无法发布 Mobile 状态")
	}
	if err := os.Rename(pendingPath, destination); err != nil {
		return errors.New("无法发布 Mobile 状态")
	}

	// Rename 是提交线性化点；成功后不再执行可能让方法返回失败的 I/O。
	m.current = candidate
	m.currentSlot = slot
	m.loaded = true
	return nil
}

func (m *MobileStateManager) writeAndSync(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return errors.New("无法写入 Mobile 状态")
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return errors.New("无法写入 Mobile 状态")
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return errors.New("无法同步 Mobile 状态")
	}
	if err := file.Close(); err != nil {
		return errors.New("无法关闭 Mobile 状态")
	}
	return nil
}

func (m *MobileStateManager) readSlot(slot int) (mobileDiskState, bool, error) {
	return readMobileStateFile(filepath.Join(m.appDir, mobileStateFileName(slot)))
}

func readMobileStateFile(path string) (mobileDiskState, bool, error) {
	ciphertext, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return mobileDiskState{}, false, nil
	}
	if err != nil {
		return mobileDiskState{}, true, errors.New("无法读取 Mobile 状态")
	}
	plaintext, err := decrypt(ciphertext)
	if err != nil {
		return mobileDiskState{}, true, errors.New("无法解密 Mobile 状态")
	}
	var state mobileDiskState
	if err := json.Unmarshal(plaintext, &state); err != nil {
		return mobileDiskState{}, true, errors.New("无法解析 Mobile 状态")
	}
	if state.SchemaVersion != mobileStateSchemaVersion {
		return mobileDiskState{}, true, errUnknownMobileStateSchema
	}
	if err := validateMobileDiskState(state); err != nil {
		return mobileDiskState{}, true, errors.New("Mobile 状态字段无效")
	}
	return state, true, nil
}

func validateMobileDiskState(state mobileDiskState) error {
	if state.SchemaVersion != mobileStateSchemaVersion || state.Generation == 0 {
		return errors.New("invalid metadata")
	}
	if state.StableLink {
		decoded, err := hex.DecodeString(state.Token)
		if err != nil || len(decoded) != mobileTokenBytes {
			return errors.New("invalid stable token")
		}
		return nil
	}
	if state.Token != "" {
		return errors.New("temporary token persisted")
	}
	return nil
}
