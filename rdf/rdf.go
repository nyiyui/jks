package rdf

import (
	"fmt"
	"net/url"
	"time"

	"github.com/deiu/rdf2go"
	"nyiyui.ca/jks/storage"
)

func mustJoinPath(base string, elem ...string) string {
	result, err := url.JoinPath(base, elem...)
	if err != nil {
		panic(err)
	}
	return result
}

var rdfType = rdf2go.NewResource("rdf:type")

var xsdDateTime = rdf2go.NewResource("http://www.w3.org/2001/XMLSchema#dateTime")
var xsdBoolean = rdf2go.NewResource("http://www.w3.org/2001/XMLSchema#boolean")

var jksBaseURI = "https://nyiyui.ca/jks/"

func boolToRDF(b bool) rdf2go.Term {
	if b {
		return rdf2go.NewLiteralWithDatatype("true", xsdBoolean)
	}
	return rdf2go.NewLiteralWithDatatype("false", xsdBoolean)
}

type Serializer struct {
	baseURI     string
	taskURI     rdf2go.Term
	activityURI rdf2go.Term
	planURI     rdf2go.Term

	description rdf2go.Term
	quickTitle  rdf2go.Term
	deadline    rdf2go.Term
	due         rdf2go.Term
	forTask     rdf2go.Term
	location    rdf2go.Term
	timeStart   rdf2go.Term
	timeEnd     rdf2go.Term
	done        rdf2go.Term
	durationGe  rdf2go.Term
	durationLt  rdf2go.Term
}

func NewSerializer(baseURI string) *Serializer {
	return &Serializer{
		baseURI:     baseURI,
		taskURI:     rdf2go.NewResource(mustJoinPath(jksBaseURI, "Task")),
		activityURI: rdf2go.NewResource(mustJoinPath(jksBaseURI, "Activity")),
		planURI:     rdf2go.NewResource(mustJoinPath(jksBaseURI, "Plan")),
		description: rdf2go.NewResource(mustJoinPath(jksBaseURI, "description")),
		quickTitle:  rdf2go.NewResource(mustJoinPath(jksBaseURI, "quickTitle")),
		deadline:    rdf2go.NewResource(mustJoinPath(jksBaseURI, "deadline")),
		due:         rdf2go.NewResource(mustJoinPath(jksBaseURI, "due")),
		forTask:     rdf2go.NewResource(mustJoinPath(jksBaseURI, "forTask")),
		location:    rdf2go.NewResource(mustJoinPath(jksBaseURI, "location")),
		timeStart:   rdf2go.NewResource(mustJoinPath(jksBaseURI, "timeStart")),
		timeEnd:     rdf2go.NewResource(mustJoinPath(jksBaseURI, "timeEnd")),
		done:        rdf2go.NewResource(mustJoinPath(jksBaseURI, "done")),
		durationGe:  rdf2go.NewResource(mustJoinPath(jksBaseURI, "durationGe")),
		durationLt:  rdf2go.NewResource(mustJoinPath(jksBaseURI, "durationLt")),
	}
}

func (s *Serializer) GraphURI() string {
	return s.baseURI
}

func (s *Serializer) TaskToRDF(t storage.Task) (*rdf2go.Graph, rdf2go.Term) {
	taskURI := mustJoinPath(s.baseURI, "tasks", fmt.Sprint(t.ID))
	g := rdf2go.NewGraph(taskURI)
	subject := rdf2go.NewResource(taskURI)
	g.AddTriple(subject, rdfType, s.taskURI)
	g.AddTriple(subject, s.description, rdf2go.NewLiteral(t.Description))
	g.AddTriple(subject, s.quickTitle, rdf2go.NewLiteral(t.QuickTitle))
	if t.Deadline != nil {
		g.AddTriple(subject, s.deadline, rdf2go.NewLiteralWithDatatype(t.Deadline.Format(time.RFC3339), xsdDateTime))
	}
	if t.Due != nil {
		g.AddTriple(subject, s.due, rdf2go.NewLiteralWithDatatype(t.Due.Format(time.RFC3339), xsdDateTime))
	}
	return g, subject
}

func (s *Serializer) ActivityToRDF(a storage.Activity) (*rdf2go.Graph, rdf2go.Term) {
	activityURI := mustJoinPath(s.baseURI, "activities", fmt.Sprint(a.ID))
	g := rdf2go.NewGraph(activityURI)
	subject := rdf2go.NewResource(activityURI)
	g.AddTriple(subject, rdfType, s.activityURI)
	g.AddTriple(subject, s.forTask, rdf2go.NewResource(mustJoinPath(s.baseURI, "tasks", fmt.Sprint(a.TaskID))))
	g.AddTriple(subject, s.location, rdf2go.NewLiteral(a.Location))
	g.AddTriple(subject, s.timeStart, rdf2go.NewLiteralWithDatatype(a.TimeStart.Format(time.RFC3339), xsdDateTime))
	g.AddTriple(subject, s.timeEnd, rdf2go.NewLiteralWithDatatype(a.TimeEnd.Format(time.RFC3339), xsdDateTime))
	g.AddTriple(subject, s.done, boolToRDF(a.Done))
	return g, subject
}

func (s *Serializer) PlanToRDF(p storage.Plan) (*rdf2go.Graph, rdf2go.Term) {
	planURI := mustJoinPath(s.baseURI, "plans", fmt.Sprint(p.ID))
	g := rdf2go.NewGraph(planURI)
	subject := rdf2go.NewResource(planURI)
	g.AddTriple(subject, rdfType, s.planURI)
	g.AddTriple(subject, s.forTask, rdf2go.NewResource(mustJoinPath(s.baseURI, "tasks", fmt.Sprint(p.TaskID))))
	g.AddTriple(subject, s.location, rdf2go.NewLiteral(p.Location))
	g.AddTriple(subject, s.timeStart, rdf2go.NewLiteralWithDatatype(p.TimeAtAfter.Format(time.RFC3339), xsdDateTime))
	g.AddTriple(subject, s.timeEnd, rdf2go.NewLiteralWithDatatype(p.TimeBefore.Format(time.RFC3339), xsdDateTime))
	g.AddTriple(subject, s.durationGe, rdf2go.NewLiteralWithDatatype(p.DurationGe.String(), xsdDateTime))
	g.AddTriple(subject, s.durationLt, rdf2go.NewLiteralWithDatatype(p.DurationLt.String(), xsdDateTime))
	return g, subject
}
