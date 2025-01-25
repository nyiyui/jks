package server

import (
	"context"
	"net/url"
	"time"

	"nyiyui.ca/jks/linkdata"
	"nyiyui.ca/jks/storage"
)

func (s *Server) getLinks(link *url.URL, ctx context.Context) (linkdata.LinkData, error) {
	var err error
	var ld linkdata.LinkData
	for _, provider := range s.linkProviders {
		data, err2 := provider.GetLinks(link, ctx)
		if err2 != nil {
			if err == nil {
				err = err2
			}
		} else {
			ld.Links = append(ld.Links, data.Links...)
		}
	}
	return ld, err
}

func (s *Server) syncDatabase(ctx context.Context) error {
	tasksWindow, err := s.st.TaskSearch("", time.Time{}, ctx)
	if err != nil {
		return err
	}
	tasks := make([]storage.Task, 0)
	for offset := 0; true; offset += 1000 {
		tasks2, err := tasksWindow.Get(1000, offset)
		if err != nil {
			return err
		}
		tasks = append(tasks, tasks2...)
	}
	panic("not implemented")
	for _, task := range tasks {
		ld := linkdata.NewLinkDataFromMarkdownSource([]byte(task.Description))
		_ = ld
	}
	panic("not implemented")
}
