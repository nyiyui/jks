package linkdata

import (
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func TestNewLinkDataFromMarkdown(t *testing.T) {
	source := []byte(`take a look at [the example site](https://example.com)`)
	r := text.NewReader(source)
	node := goldmark.New().Parser().Parse(r)
	ld := NewLinkDataFromMarkdown(source, node)
	if len(ld.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(ld.Links))
	}
	if ld.Links[0].Label != "the example site" {
		t.Fatalf("expected label to be 'the example site', got %q", ld.Links[0].Label)
	}
	if ld.Links[0].Destination.String() != "https://example.com" {
		t.Fatalf("expected destination to be 'https://example.com', got %q", ld.Links[0].Destination.String())
	}
}
