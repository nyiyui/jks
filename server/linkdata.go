package server

import (
	"context"
	"net/url"
	"strconv"
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
	taskURL, err := url.Parse("/task/")
	if err != nil {
		panic(err)
	}
	activityURL, err := url.Parse("/activity/")
	if err != nil {
		panic(err)
	}

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
	for _, task := range tasks {
		ld := linkdata.NewLinkDataFromMarkdownSource([]byte(task.Description))
		rowURl := taskURL.JoinPath(strconv.FormatInt(task.ID, 10))
		if err != nil {
			panic(err)
		}
		err = s.st.ReplaceLinks(rowURl, ld.Links, ctx)
		if err != nil {
			return err
		}
	}

	activitiesWindow, err := s.st.ActivityRange(time.Time{}, time.Now(), ctx)
	if err != nil {
		return err
	}
	activities := make([]storage.Activity, 0)
	for offset := 0; true; offset += 1000 {
		activities2, err := activitiesWindow.Get(1000, offset)
		if err != nil {
			return err
		}
		activities = append(activities, activities2...)
	}
	for _, activity := range activities {
		ld := linkdata.NewLinkDataFromMarkdownSource([]byte(activity.Note))
		rowURL := activityURL.JoinPath(strconv.FormatInt(activity.ID, 10))
		if err != nil {
			panic(err)
		}
		err = s.st.ReplaceLinks(rowURL, ld.Links, ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
