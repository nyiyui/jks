package server

import (
	"fmt"
	"path/filepath"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/safehtml/template"
)

func (s *Server) parseTemplates() error {
	matches, err := filepath.Glob("server/templates/*.html")
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
		Funcs(template.FuncMap(sprig.FuncMap()))
	t, err := t.ParseGlob("server/layouts/*.html")
	if err != nil {
		return nil, err
	}
	ts, err := template.TrustedSourceFromConstantDir("server/templates", template.TrustedSource{}, basename)
	if err != nil {
		return nil, err
	}
	t, err = t.ParseFilesFromTrustedSources(ts)
	if err != nil {
		return nil, err
	}
	return t, nil
}