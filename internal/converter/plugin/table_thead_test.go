package plugin

import (
	"strings"
	"testing"

	convpkg "github.com/JohannesKaufmann/html-to-markdown/v2/converter"
)

func TestHandleTableWithThead(t *testing.T) {
	plugin := &ConfluencePlugin{}
	conv := convpkg.NewConverter(convpkg.WithPlugins(plugin))
	plugin.Init(conv)

	html := `<table><thead><tr><th>Параметр</th><th>Тип</th><th>Значение</th><th>Обязательность</th></tr></thead><tbody><tr><td>status</td><td>string(10)</td><td>"SUCCESS"</td><td>+</td></tr></tbody></table>`
	got, err := conv.ConvertString(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, header := range []string{"Параметр", "Тип", "Значение", "Обязательность"} {
		if !strings.Contains(got, header) {
			t.Fatalf("expected header %q in %q", header, got)
		}
	}

	headerIdx := strings.Index(got, "Параметр")
	statusIdx := strings.Index(got, "status")
	sepIdx := strings.Index(got, "---|")
	if headerIdx < 0 || statusIdx < 0 || sepIdx < 0 {
		t.Fatalf("expected header row, separator, and data row in %q", got)
	}
	if !(headerIdx < sepIdx && sepIdx < statusIdx) {
		t.Fatalf("expected header, then separator, then data row; got %q", got)
	}
}
