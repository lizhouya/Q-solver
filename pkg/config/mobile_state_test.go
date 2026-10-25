package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("test random source failure")
}

func readMobileDiskState(t *testing.T, dir string, slot int) mobileDiskState {
	t.Helper()
	ciphertext, err := os.ReadFile(filepath.Join(dir, mobileStateFileName(slot)))
	if err != nil {
		t.Fatal("failed to read mobile state fixture")
	}
	plaintext, err := decrypt(ciphertext)
	if err != nil {
		t.Fatal("failed to decrypt mobile state fixture")
	}
	var state mobileDiskState
	if err := json.Unmarshal(plaintext, &state); err != nil {
		t.Fatal("failed to decode mobile state fixture")
	}
	return state
}

func writeMobileDiskState(t *testing.T, dir string, slot int, state mobileDiskState) {
	t.Helper()
	plaintext, err := json.Marshal(state)
	if err != nil {
		t.Fatal("failed to encode mobile state fixture")
	}
	ciphertext, err := encrypt(plaintext)
	if err != nil {
		t.Fatal("failed to encrypt mobile state fixture")
	}
	if err := os.WriteFile(filepath.Join(dir, mobileStateFileName(slot)), ciphertext, 0600); err != nil {
		t.Fatal("failed to write mobile state fixture")
	}
}

func TestMobileLoadOrCreateDefaultsToStableAndPersistsEncrypted(t *testing.T) {
	dir := t.TempDir()
	manager := newMobileStateManager(dir, bytes.NewReader(bytes.Repeat([]byte{0x11}, mobileTokenBytes)))

	state, err := manager.LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if !state.StableLink || state.Token == "" || state.ClearSeq != 0 || state.Generation != 1 {
		t.Fatal("unexpected default mobile state")
	}

	ciphertext, err := os.ReadFile(filepath.Join(dir, mobileStateFileName(0)))
	if err != nil {
		t.Fatal("default state file was not created")
	}
	if bytes.Contains(ciphertext, []byte(state.Token)) {
		t.Fatal("mobile token must not be stored as plaintext")
	}

	disk := readMobileDiskState(t, dir, 0)
	if disk.SchemaVersion != mobileStateSchemaVersion || !disk.StableLink || disk.Token == "" || disk.Generation != 1 {
		t.Fatal("unexpected persisted default mobile state")
	}

	reloaded, err := newMobileStateManager(dir, failingReader{}).LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded != state {
		t.Fatal("stable mobile state was not reused")
	}
}

func TestMobileMutationsAlternateSlotsAndSelectNewestGeneration(t *testing.T) {
	dir := t.TempDir()
	manager := newMobileStateManager(dir, bytes.NewReader(bytes.Repeat([]byte{0x22}, mobileTokenBytes*2)))
	initial, err := manager.LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	advanced, err := manager.AdvanceClearSeq()
	if err != nil {
		t.Fatal(err)
	}
	if advanced.ClearSeq != initial.ClearSeq+1 || advanced.Generation != initial.Generation+1 || advanced.Token != initial.Token {
		t.Fatal("clear sequence advance changed unexpected fields")
	}
	if _, err := os.Stat(filepath.Join(dir, mobileStateFileName(0))); err != nil {
		t.Fatal("first mobile state slot missing")
	}
	if _, err := os.Stat(filepath.Join(dir, mobileStateFileName(1))); err != nil {
		t.Fatal("second mobile state slot missing")
	}

	reloaded, err := newMobileStateManager(dir, failingReader{}).LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded != advanced {
		t.Fatal("newest mobile state generation was not selected")
	}
}

func TestMobileTemporaryModeDoesNotPersistTokenAndRotatesOnRestart(t *testing.T) {
	dir := t.TempDir()
	random := append(bytes.Repeat([]byte{0x31}, mobileTokenBytes), bytes.Repeat([]byte{0x32}, mobileTokenBytes)...)
	manager := newMobileStateManager(dir, bytes.NewReader(random))
	initial, err := manager.LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	temporary, err := manager.SetStableLink(false)
	if err != nil {
		t.Fatal(err)
	}
	if temporary.StableLink || temporary.Token == "" || temporary.Token == initial.Token {
		t.Fatal("disabling stable link did not rotate to a temporary token")
	}
	if temporary.ClearSeq != initial.ClearSeq+1 || temporary.Generation != initial.Generation+1 {
		t.Fatal("disabling stable link did not advance the display boundary")
	}
	if disk := readMobileDiskState(t, dir, 1); disk.Token != "" || disk.StableLink {
		t.Fatal("temporary token must not be persisted")
	}

	reloaded, err := newMobileStateManager(dir, bytes.NewReader(bytes.Repeat([]byte{0x33}, mobileTokenBytes))).LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.StableLink || reloaded.Token == "" || reloaded.Token == temporary.Token {
		t.Fatal("temporary token must rotate on process restart")
	}
	if reloaded.ClearSeq != temporary.ClearSeq || reloaded.Generation != temporary.Generation {
		t.Fatal("temporary restart changed persisted counters")
	}
}

func TestMobileResetAndStableModeEnableRotateLinkAndClear(t *testing.T) {
	dir := t.TempDir()
	random := make([]byte, mobileTokenBytes*4)
	for i := range random {
		random[i] = byte(i + 1)
	}
	manager := newMobileStateManager(dir, bytes.NewReader(random))
	initial, err := manager.LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	reset, err := manager.ResetLink()
	if err != nil {
		t.Fatal(err)
	}
	if reset.Token == initial.Token || reset.ClearSeq != initial.ClearSeq+1 || reset.Generation != initial.Generation+1 {
		t.Fatal("reset did not rotate the link and display boundary")
	}
	temporary, err := manager.SetStableLink(false)
	if err != nil {
		t.Fatal(err)
	}
	stable, err := manager.SetStableLink(true)
	if err != nil {
		t.Fatal(err)
	}
	if !stable.StableLink || stable.Token == temporary.Token || stable.ClearSeq != temporary.ClearSeq+1 || stable.Generation != temporary.Generation+1 {
		t.Fatal("enabling stable link did not create a fresh persistent link")
	}

	same, err := manager.SetStableLink(true)
	if err != nil {
		t.Fatal(err)
	}
	if same != stable {
		t.Fatal("setting the current stable mode should be idempotent")
	}
}

func TestMobileSingleCorruptSlotFallsBackToValidSlot(t *testing.T) {
	dir := t.TempDir()
	manager := newMobileStateManager(dir, bytes.NewReader(bytes.Repeat([]byte{0x41}, mobileTokenBytes)))
	initial, err := manager.LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	advanced, err := manager.AdvanceClearSeq()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, mobileStateFileName(1)), []byte("truncated"), 0600); err != nil {
		t.Fatal("failed to corrupt test slot")
	}

	reloaded, err := newMobileStateManager(dir, failingReader{}).LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded != initial || reloaded == advanced {
		t.Fatal("manager did not fall back to the remaining valid slot")
	}
}

func TestMobileExistingInvalidSlotsFailClosed(t *testing.T) {
	for _, test := range []struct {
		name  string
		slots []int
	}{
		{name: "single invalid slot", slots: []int{0}},
		{name: "both invalid slots", slots: []int{0, 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, slot := range test.slots {
				if err := os.WriteFile(filepath.Join(dir, mobileStateFileName(slot)), []byte("invalid"), 0600); err != nil {
					t.Fatal("failed to create invalid test slot")
				}
			}
			if _, err := newMobileStateManager(dir, bytes.NewReader(bytes.Repeat([]byte{0x51}, mobileTokenBytes))).LoadOrCreate(); err == nil {
				t.Fatal("existing invalid mobile state must fail closed")
			}
		})
	}
}

func TestMobileUnknownSchemaFailsClosedEvenWithAnotherValidSlot(t *testing.T) {
	dir := t.TempDir()
	writeMobileDiskState(t, dir, 0, mobileDiskState{
		SchemaVersion: mobileStateSchemaVersion,
		StableLink:    true,
		Token:         "0123456789abcdef0123456789abcdef",
		Generation:    1,
	})
	writeMobileDiskState(t, dir, 1, mobileDiskState{
		SchemaVersion: mobileStateSchemaVersion + 1,
		StableLink:    true,
		Token:         "fedcba9876543210fedcba9876543210",
		Generation:    2,
	})

	if _, err := newMobileStateManager(dir, failingReader{}).LoadOrCreate(); err == nil {
		t.Fatal("unknown decrypted schema must fail closed")
	}
}

func TestMobileOverflowDoesNotChangeCurrent(t *testing.T) {
	for _, test := range []struct {
		name     string
		clearSeq uint64
		gen      uint64
	}{
		{name: "clear sequence", clearSeq: math.MaxUint64, gen: 1},
		{name: "generation", clearSeq: 1, gen: math.MaxUint64},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			writeMobileDiskState(t, dir, 0, mobileDiskState{
				SchemaVersion: mobileStateSchemaVersion,
				StableLink:    true,
				Token:         "0123456789abcdef0123456789abcdef",
				ClearSeq:      test.clearSeq,
				Generation:    test.gen,
			})
			manager := newMobileStateManager(dir, bytes.NewReader(bytes.Repeat([]byte{0x61}, mobileTokenBytes)))
			before, err := manager.LoadOrCreate()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := manager.AdvanceClearSeq(); err == nil {
				t.Fatal("counter overflow must fail")
			}
			if manager.current != before {
				t.Fatal("counter overflow changed current mobile state")
			}
		})
	}
}

func TestMobileResetRejectsRepeatedCurrentToken(t *testing.T) {
	dir := t.TempDir()
	same := bytes.Repeat([]byte{0x91}, mobileTokenBytes)
	random := append([]byte{}, same...)
	random = append(random, same...)
	random = append(random, same...)
	random = append(random, same...)
	manager := newMobileStateManager(dir, bytes.NewReader(random))
	before, err := manager.LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ResetLink(); err == nil {
		t.Fatal("连续生成当前 Token 时 Reset 必须失败")
	}
	if manager.current != before {
		t.Fatal("无法生成不同 Token 时不得改变 current")
	}
}

func TestMobileResetRetriesWhenRandomTokenMatchesCurrent(t *testing.T) {
	dir := t.TempDir()
	current := bytes.Repeat([]byte{0x92}, mobileTokenBytes)
	different := bytes.Repeat([]byte{0x93}, mobileTokenBytes)
	random := append([]byte{}, current...)
	random = append(random, current...)
	random = append(random, different...)
	manager := newMobileStateManager(dir, bytes.NewReader(random))
	before, err := manager.LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	after, err := manager.ResetLink()
	if err != nil {
		t.Fatal(err)
	}
	if after.Token == before.Token || after.ClearSeq != before.ClearSeq+1 {
		t.Fatal("Reset 应跳过重复随机值并提交不同 Token")
	}
}

func TestMobileFailuresDoNotChangeCurrent(t *testing.T) {
	t.Run("random source", func(t *testing.T) {
		dir := t.TempDir()
		manager := newMobileStateManager(dir, bytes.NewReader(bytes.Repeat([]byte{0x71}, mobileTokenBytes)))
		before, err := manager.LoadOrCreate()
		if err != nil {
			t.Fatal(err)
		}
		manager.random = failingReader{}
		if _, err := manager.ResetLink(); err == nil {
			t.Fatal("random source failure must be returned")
		}
		if manager.current != before {
			t.Fatal("random source failure changed current mobile state")
		}
	})

	t.Run("candidate write", func(t *testing.T) {
		dir := t.TempDir()
		manager := newMobileStateManager(dir, bytes.NewReader(bytes.Repeat([]byte{0x72}, mobileTokenBytes)))
		before, err := manager.LoadOrCreate()
		if err != nil {
			t.Fatal(err)
		}
		blockedSlot := filepath.Join(dir, mobileStateFileName(1))
		if err := os.Mkdir(blockedSlot, 0700); err != nil {
			t.Fatal("failed to block candidate slot")
		}
		if err := os.WriteFile(filepath.Join(blockedSlot, "keep"), []byte("block removal"), 0600); err != nil {
			t.Fatal("failed to make blocked slot non-empty")
		}
		if _, err := manager.AdvanceClearSeq(); err == nil {
			t.Fatal("candidate publish failure must be returned")
		}
		if manager.current != before || manager.currentSlot != 0 {
			t.Fatal("candidate publish failure changed current mobile state")
		}
		reloaded, err := newMobileStateManager(dir, failingReader{}).LoadOrCreate()
		if err != nil {
			t.Fatal(err)
		}
		if reloaded != before {
			t.Fatal("failed candidate publish became active after restart")
		}
	})
}

func TestMobilePendingWriteFailureLeavesOfficialSlotsUnchanged(t *testing.T) {
	dir := t.TempDir()
	manager := newMobileStateManager(dir, bytes.NewReader(bytes.Repeat([]byte{0x82}, mobileTokenBytes)))
	before, err := manager.LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	pendingPath := filepath.Join(dir, mobileStatePendingFile)
	if err := os.Mkdir(pendingPath, 0700); err != nil {
		t.Fatal("failed to block pending file")
	}
	if err := os.WriteFile(filepath.Join(pendingPath, "keep"), []byte("block removal"), 0600); err != nil {
		t.Fatal("failed to make pending path non-empty")
	}
	if _, err := manager.AdvanceClearSeq(); err == nil {
		t.Fatal("pending write failure must be returned")
	}
	if manager.current != before {
		t.Fatal("pending write failure changed current state")
	}
	reloaded, err := newMobileStateManager(dir, failingReader{}).LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded != before {
		t.Fatal("pending write failure became active after restart")
	}
}

func TestMobileStateIsIndependentFromOrdinaryConfigSave(t *testing.T) {
	dir := t.TempDir()
	manager := newMobileStateManager(dir, bytes.NewReader(bytes.Repeat([]byte{0x81}, mobileTokenBytes)))
	if _, err := manager.LoadOrCreate(); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(dir, mobileStateFileName(0))
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal("failed to read mobile state fixture")
	}

	configManager := &ConfigManager{
		loaded:      true,
		config:      NewDefaultConfig(),
		oldConfig:   NewDefaultConfig(),
		configPath:  filepath.Join(dir, "config"),
		subscribers: []func(NewConfig Config, oldConfig Config){},
	}
	if err := configManager.Save(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal("failed to read mobile state after ordinary config save")
	}
	if !bytes.Equal(before, after) {
		t.Fatal("ordinary config save must not rewrite Mobile state")
	}
}
