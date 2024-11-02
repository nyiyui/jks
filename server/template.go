package server

import (
	"github.com/Masterminds/sprig/v3"
	"github.com/google/safehtml/template"
)

func (s *Server) parseTemplates() error {
	t := template.New("").Funcs(template.FuncMap(sprig.FuncMap()))
	t, err := t.ParseGlob("server/templates/*.html")
	if err != nil {
		return err
	}
	s.tp = t
	return nil
}
