package plugin

import (
	"strings"
	"testing"

	convpkg "github.com/JohannesKaufmann/html-to-markdown/v2/converter"
)

func TestHandleTableSimpleCellMultipleSpans(t *testing.T) {
	plugin := &ConfluencePlugin{}
	conv := convpkg.NewConverter(convpkg.WithPlugins(plugin))
	plugin.Init(conv)

	html := `<table><tbody><tr><td><span style="color:var(--ds-text,#000000);">ответ.subjects</span>.<span style="color:var(--ds-text,#000000);">risks.riskCodeName</span></td></tr></tbody></table>`
	got, err := conv.ConvertString(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "ответ.subjects.risks.riskCodeName"
	if !strings.Contains(got, want) {
		t.Fatalf("expected full dotted path %q in %q", want, got)
	}
}
