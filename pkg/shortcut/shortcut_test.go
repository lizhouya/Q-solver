package shortcut

import "testing"

func TestFunctionKeyComboID(t *testing.T) {
	for _, test := range []struct {
		name string
		want string
	}{
		{name: "F1", want: "112"},
		{name: "F10", want: "121"},
		{name: "F12", want: "123"},
		{name: "F24", want: "135"},
	} {
		got, ok := FunctionKeyComboID(test.name)
		if !ok || got != test.want {
			t.Fatalf("%s: want %s, got %s ok=%v", test.name, test.want, got, ok)
		}
	}
	for _, invalid := range []string{"", "F0", "F25", "Ctrl+F10", "f10", "Ctrl+J"} {
		if got, ok := FunctionKeyComboID(invalid); ok {
			t.Fatalf("%q 不得自动迁移，实际 %q", invalid, got)
		}
	}
}

func TestReadableNameFromComboID(t *testing.T) {
	for _, test := range []struct {
		combo string
		want  string
	}{
		{combo: "121", want: "F10"},
		{combo: "123", want: "F12"},
		{combo: "74+160+162", want: "Ctrl+Shift+J"},
	} {
		got, ok := ReadableNameFromComboID(test.combo)
		if !ok || got != test.want {
			t.Fatalf("%s: want %s, got %s ok=%v", test.combo, test.want, got, ok)
		}
	}
	for _, invalid := range []string{"", "Cmd+J", "121+bad", "121+121"} {
		if got, ok := ReadableNameFromComboID(invalid); ok {
			t.Fatalf("%q 不应解析成功，实际 %q", invalid, got)
		}
	}
}
