package linkdata

import (
	"net/url"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type LinkData struct {
	Links []Link
}

type Link struct {
	Label       string
	Destination *url.URL
}

type Backlink struct {
	Source *url.URL
	Label  string
}

func NewLinkDataFromMarkdownSource(source []byte) LinkData {
	r := text.NewReader(source)
	node := goldmark.New().Parser().Parse(r)
	return NewLinkDataFromMarkdown(source, node)
}

func NewLinkDataFromMarkdown(source []byte, node ast.Node) LinkData {
	ld := LinkData{}
	ld.walkGoldmarkNode(source, node)
	return ld
}

func (ld *LinkData) walkGoldmarkNode(source []byte, node ast.Node) {
	switch node.Kind() {
	case ast.KindLink:
		link := node.(*ast.Link)
		destination, err := url.Parse(string(link.Destination))
		if err != nil {
			// ignore invalid links
			return
		}
		l := Link{Destination: destination}
		if link.HasChildren() {
			label := ""
			for child := link.FirstChild(); child != nil; child = child.NextSibling() {
				label += getPlainText(source, child)
			}
			l.Label = label
		}
		ld.Links = append(ld.Links, l)
	case ast.KindAutoLink:
		destination, err := url.Parse(string(node.Text(source)))
		if err != nil {
			// ignore invalid links
			return
		}
		ld.Links = append(ld.Links, Link{Destination: destination})
	default:
		if node.HasChildren() {
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				ld.walkGoldmarkNode(source, child)
			}
		}
	}
}

func getPlainText(source []byte, node ast.Node) string {
	switch node.Kind() {
	case ast.KindText:
		return string(node.(*ast.Text).Value(source))
	case ast.KindCodeSpan:
		return string(node.(*ast.CodeSpan).Text(source))
	default:
		if node.HasChildren() {
			text := ""
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				text += getPlainText(source, child)
			}
			return text
		} else {
			return ""
		}
	}
}
