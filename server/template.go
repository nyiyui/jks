package server

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/safehtml"
	"github.com/google/safehtml/template"
	"github.com/google/safehtml/uncheckedconversions"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

//go:embed layouts
var layoutsFS embed.FS

//go:embed templates
var templatesFS embed.FS

type stringConstant string

func (s *Server) renderTemplate(path stringConstant, w http.ResponseWriter, r *http.Request, data map[string]interface{}) {
	t, ok := s.tps[string(path)]
	if !ok {
		panic("template not found")
		return
	}
	if data == nil {
		data = map[string]interface{}{}
	}
	data["login"], _ = r.Context().Value(LoginUserDataKey).(githubUserData)
	err := t.Execute(w, data)
	if err != nil {
		log.Printf("template error: %s", err)
		http.Error(w, "template error", 500)
		return
	}
}

func (s *Server) parseTemplates() error {
	matches, err := fs.Glob(templatesFS, "templates/*.html")
	if err != nil {
		return err
	}
	s.tps = map[string]*template.Template{}
	for _, match := range matches {
		_, basename := filepath.Split(match)
		s.tps[basename], err = s.parseTemplate(basename)
		if err != nil {
			return fmt.Errorf("parse %s: %w", basename, err)
		}
	}
	return nil
}

func (s *Server) parseTemplate(basename string) (*template.Template, error) {
	t := template.New(basename).
		Funcs(template.FuncMap(sprig.FuncMap())).
		Funcs(template.FuncMap{
			"styleTopHeight": func(top, height string) safehtml.Style {
				return safehtml.StyleFromProperties(safehtml.StyleProperties{
					Top:    top,
					Height: height,
				})
			},
			"genRange": func(count uint) []uint {
				items := make([]uint, count)
				for i := uint(0); i < count; i++ {
					items[i] = i
				}
				return items
			},
			"renderMarkdown": func(s string) (safehtml.HTML, error) {
				md := goldmark.New(goldmark.WithExtensions(extension.Linkify))
				var buf bytes.Buffer
				err := md.Convert([]byte(s), &buf)
				if err != nil {
					return safehtml.HTML{}, err
				}
				return uncheckedconversions.HTMLFromStringKnownToSatisfyTypeContract(buf.String()), nil
			},
			"formatDatetimeLocal": func(t time.Time) string {
				return t.Format("2006-01-02T15:04")
			},
			"formatUser": func(t time.Time) string {
				abs := t.Format("2006-01-02 15:04")
				rel := t.Sub(time.Now()).Round(time.Minute)
				rel2 := rel.String()
				return fmt.Sprintf("%s (%s)", abs, rel2[:len(rel2)-2])
			},
			"splitNoteTitle": func(s string) string {
				lines := strings.SplitN(s, "\n", 2)
				if len(lines) == 1 {
					return lines[0]
				}
				return lines[0]
			},
			"splitNoteBody": func(s string) string {
				lines := strings.SplitN(s, "\n", 2)
				if len(lines) == 1 {
					return ""
				}
				return lines[1]
			},
		})
	t, err := t.ParseFS(template.TrustedFSFromEmbed(layoutsFS), "layouts/*.html")
	if err != nil {
		return nil, err
	}
	t, err = t.ParseFS(template.TrustedFSFromEmbed(templatesFS), fmt.Sprintf("templates/%s", basename))
	if err != nil {
		return nil, err
	}
	return t, nil
}
