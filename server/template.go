package server

import (
	"github.com/google/safehtml/template"
)

func (s *Server) parseTemplates() error {
	t := template.New("")
	t, err := t.ParseGlob("server/templates/*.html")
	if err != nil {
		return err
	}
	s.tp = t
	return nil
}
