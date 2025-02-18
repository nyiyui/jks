package server

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/safehtml"
	"github.com/google/safehtml/template"
	"github.com/google/safehtml/uncheckedconversions"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"nyiyui.ca/jks/layout"
	"nyiyui.ca/jks/storage"
	seekbackStorage "nyiyui.ca/seekback-server/storage"
)

var buildInfo debug.BuildInfo
var vcsInfo string

//go:embed layouts
var layoutsFS embed.FS

//go:embed templates
var templatesFS embed.FS

type stringConstant string

func init() {
	buildInfo2, _ := debug.ReadBuildInfo()
	if buildInfo2 != nil {
		buildInfo = *buildInfo2
	}
}

func (s *Server) renderTemplate(path stringConstant, w http.ResponseWriter, r *http.Request, data map[string]interface{}) {
	t, ok := s.tps[string(path)]
	if !ok {
		panic("template not found")
		return
	}
	t, err := t.Clone()
	if err != nil {
		panic("template clone error") // t should not have been executed yet
	}
	if data == nil {
		data = map[string]interface{}{}
	}
	data["login"], _ = r.Context().Value(LoginUserDataKey).(githubUserData)
	data["tzloc"] = getTimeLocation(r)
	t = t.Funcs(template.FuncMap{
		"timezone": func() string {
			loginSession, err := s.store.Get(r, "login")
			if err != nil {
				return ""
			}
			tzName, _ := loginSession.Values["timezone"].(string)
			return tzName
		},
	})
	err = t.Execute(w, data)
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
			"isActivity": func(v layout.Box) bool {
				_, ok := v.(storage.Activity)
				return ok
			},
			"isPlan": func(v layout.Box) bool {
				_, ok := v.(storage.Plan)
				return ok
			},
			"toActivity": func(v layout.Box) storage.Activity {
				return v.(storage.Activity)
			},
			"toPlan": func(v layout.Box) storage.Plan {
				return v.(storage.Plan)
			},
			"styleTopHeight": func(top, height string) safehtml.Style {
				return safehtml.StyleFromProperties(safehtml.StyleProperties{
					Top:    top,
					Height: height,
				})
			},
			"genRange": func(count int) []int {
				items := make([]int, count)
				for i := 0; i < count; i++ {
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
			"formatDayLong": func(loc *time.Location, t time.Time) string {
				return t.In(loc).Format("2006-01-02 Mon")
			},
			"formatDay": func(loc *time.Location, t time.Time) string {
				return t.In(loc).Format("2006-01-02")
			},
			"formatHM": func(loc *time.Location, t time.Time) string {
				return t.In(loc).Format("15:04")
			},
			"formatDatetimeLocalHTML": func(loc *time.Location, t time.Time) string {
				return t.In(loc).Format("2006-01-02T15:04")
			},
			"formatUser": func(loc *time.Location, t time.Time) string {
				t = t.In(loc)
				abs := t.Format("2006-01-02 15:04")
				rel := t.Sub(time.Now()).Round(time.Minute)
				rel2 := rel.String()
				return fmt.Sprintf("%s (%s)", abs, rel2[:len(rel2)-2])
			},
			"formatYearMonth": func(loc *time.Location, t time.Time) string {
				return t.In(loc).Format("2006 Jan")
			},
			"splitNoteTitle": func(s string) string {
				lines := strings.SplitN(s, "\n", 2)
				return strings.TrimSpace(lines[0])
			},
			"splitNoteBody": func(s string) string {
				lines := strings.SplitN(s, "\n", 2)
				if len(lines) == 1 {
					return ""
				}
				return strings.TrimSpace(lines[1])
			},
			"buildInfo": func() debug.BuildInfo {
				return buildInfo
			},
			"vcsInfo": func() string {
				return vcsInfo
			},
			"isSamplePreview": func(v Event) bool {
				_, ok := v.(seekbackStorage.SamplePreview)
				return ok
			},
			"asSamplePreview": func(v Event) seekbackStorage.SamplePreview {
				return v.(seekbackStorage.SamplePreview)
			},
			"timezone": func() string { return "" }, // dummy, replaced with real closure during render
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
