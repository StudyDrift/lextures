package learningevents

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/lrsconfig"
	"github.com/lextures/lextures/server/internal/repos/lrsforwardjobs"
	"github.com/lextures/lextures/server/internal/repos/xapistatements"
	"github.com/lextures/lextures/server/internal/service/caliper"
	"github.com/lextures/lextures/server/internal/service/xapi"
)

// Emitter records xAPI/Caliper learning events when the platform feature is enabled.
type Emitter struct {
	Pool *pgxpool.Pool
	Cfg  config.Config
}

func (e Emitter) enabled() bool {
	return e.Cfg.XAPIEmissionEnabled
}

func (e Emitter) baseIRI() string {
	origin := strings.TrimRight(strings.TrimSpace(e.Cfg.PublicWebOrigin), "/")
	if origin == "" {
		origin = "https://lextures.local"
	}
	return origin
}

// Payload is stored in analytics.xapi_statements.full_json.
type Payload struct {
	XAPI    json.RawMessage `json:"xapi"`
	Caliper json.RawMessage `json:"caliper"`
}

type emitParams struct {
	StatementID   uuid.UUID
	ActorEmail    string
	ActorName     string
	VerbID        string
	ObjectID      string
	ObjectType    string
	ObjectTitle   string
	CourseID      *uuid.UUID
	CourseCode    string
	CaliperType   string
	CaliperAction string
	Score         *float64
	Success       *bool
	OrgID         uuid.UUID
}

func (e Emitter) emit(ctx context.Context, p emitParams) {
	if !e.enabled() || e.Pool == nil {
		return
	}
	if p.StatementID == uuid.Nil {
		p.StatementID = uuid.New()
	}
	now := time.Now().UTC()
	base := e.baseIRI()
	courseIRI := ""
	if p.CourseCode != "" {
		courseIRI = base + "/courses/" + p.CourseCode
	}
	actorIRI := base + "/users/" + p.ActorEmail
	if e.Cfg.LRSAnonymizeActors {
		actorIRI = base + "/users/" + xapi.ActorHash(p.ActorEmail, true)
	}
	stmt := xapi.BuildStatement(xapi.BuildInput{
		StatementID: p.StatementID,
		ActorEmail:  p.ActorEmail,
		ActorName:   p.ActorName,
		Anonymize:   e.Cfg.LRSAnonymizeActors,
		VerbID:      p.VerbID,
		ObjectID:    p.ObjectID,
		ObjectType:  p.ObjectType,
		ObjectTitle: p.ObjectTitle,
		CourseIRI:   courseIRI,
		Score:       p.Score,
		Success:     p.Success,
		Timestamp:   now,
	})
	xRaw, err := xapi.MarshalStatement(stmt)
	if err != nil {
		slog.Warn("learningevents.xapi_marshal", "err", err)
		return
	}
	cal := caliper.BuildEvent(caliper.BuildInput{
		EventID:    uuid.New(),
		EventType:  p.CaliperType,
		Action:     p.CaliperAction,
		ActorIRI:   actorIRI,
		ObjectIRI:  p.ObjectID,
		ObjectName: p.ObjectTitle,
		CourseIRI:  courseIRI,
		Score:      p.Score,
		Timestamp:  now,
	})
	cRaw, err := caliper.MarshalEvent(cal)
	if err != nil {
		slog.Warn("learningevents.caliper_marshal", "err", err)
		return
	}
	full, err := json.Marshal(Payload{XAPI: xRaw, Caliper: cRaw})
	if err != nil {
		return
	}
	var scoreF *float32
	if p.Score != nil {
		v := float32(*p.Score)
		scoreF = &v
	}
	objType := p.ObjectType
	objTitle := p.ObjectTitle
	row := xapistatements.Row{
		StatementID:     p.StatementID,
		ActorHash:       xapi.ActorHash(p.ActorEmail, e.Cfg.LRSAnonymizeActors),
		VerbID:          p.VerbID,
		ObjectID:        p.ObjectID,
		ObjectType:      &objType,
		ObjectTitle:     &objTitle,
		ResultScore:     scoreF,
		ResultSuccess:   p.Success,
		ContextCourseID: p.CourseID,
		StoredAt:        now,
		FullJSON:        full,
	}
	if err := xapistatements.Insert(ctx, e.Pool, row); err != nil {
		slog.Warn("learningevents.store", "err", err, "statement_id", p.StatementID)
		return
	}
	slog.Info("xapi statement stored",
		"statement_id", p.StatementID,
		"verb", p.VerbID,
		"actor_hash", row.ActorHash,
		"object_id", p.ObjectID,
	)
	e.enqueueForward(ctx, p.OrgID, p.StatementID, now)
}

func (e Emitter) enqueueForward(ctx context.Context, orgID, statementID uuid.UUID, storedAt time.Time) {
	if orgID == uuid.Nil || len(e.Cfg.PlatformSecretsKey) != 32 {
		return
	}
	eps, _, err := lrsconfig.ListEnabled(ctx, e.Pool, orgID)
	if err != nil || len(eps) == 0 {
		return
	}
	ids := make([]uuid.UUID, 0, len(eps))
	for _, ep := range eps {
		ids = append(ids, ep.ID)
	}
	if err := lrsforwardjobs.EnqueueForEndpoints(ctx, e.Pool, statementID, storedAt, ids); err != nil {
		slog.Warn("learningevents.lrs_enqueue", "err", err)
	}
}

// LoggedIn emits SessionEvent.LoggedIn / xAPI launched.
func (e Emitter) LoggedIn(ctx context.Context, orgID uuid.UUID, email, displayName string) {
	base := e.baseIRI()
	e.emit(ctx, emitParams{
		ActorEmail:    email,
		ActorName:     displayName,
		VerbID:        xapi.VerbLaunched,
		ObjectID:      base + "/platform",
		ObjectType:    "Activity",
		ObjectTitle:   "Lextures",
		CaliperType:   "SessionEvent",
		CaliperAction: caliper.ActionLoggedIn,
		OrgID:         orgID,
	})
}

// CourseEnrollment emits CourseEnrollmentEvent.
func (e Emitter) CourseEnrollment(ctx context.Context, orgID uuid.UUID, courseID uuid.UUID, courseCode, email, displayName string) {
	base := e.baseIRI()
	e.emit(ctx, emitParams{
		ActorEmail:    email,
		ActorName:     displayName,
		VerbID:        xapi.VerbCompleted,
		ObjectID:      base + "/courses/" + courseCode + "/enrollment",
		ObjectType:    "Activity",
		ObjectTitle:   "Course enrollment",
		CourseID:      &courseID,
		CourseCode:    courseCode,
		CaliperType:   "CourseEnrollmentEvent",
		CaliperAction: caliper.ActionEnrolled,
		OrgID:         orgID,
	})
}

// CourseVisited emits NavigationEvent for course_visit audit events.
func (e Emitter) CourseVisited(ctx context.Context, orgID uuid.UUID, courseID uuid.UUID, courseCode, email, displayName string) {
	base := e.baseIRI()
	e.emit(ctx, emitParams{
		ActorEmail:    email,
		ActorName:     displayName,
		VerbID:        xapi.VerbExperienced,
		ObjectID:      base + "/courses/" + courseCode,
		ObjectType:    "Activity",
		ObjectTitle:   "Course",
		CourseID:      &courseID,
		CourseCode:    courseCode,
		CaliperType:   "NavigationEvent",
		CaliperAction: caliper.ActionNavigatedTo,
		OrgID:         orgID,
	})
}

// ContentViewed emits NavigationEvent for content_open.
func (e Emitter) ContentViewed(ctx context.Context, orgID uuid.UUID, courseID uuid.UUID, courseCode, email, displayName, itemID, itemTitle string) {
	base := e.baseIRI()
	e.emit(ctx, emitParams{
		ActorEmail:    email,
		ActorName:     displayName,
		VerbID:        xapi.VerbExperienced,
		ObjectID:      base + "/courses/" + courseCode + "/items/" + itemID,
		ObjectType:    "Activity",
		ObjectTitle:   itemTitle,
		CourseID:      &courseID,
		CourseCode:    courseCode,
		CaliperType:   "NavigationEvent",
		CaliperAction: caliper.ActionNavigatedTo,
		OrgID:         orgID,
	})
}

// QuizAttemptGraded emits passed/failed xAPI verbs for a submitted quiz.
func (e Emitter) QuizAttemptGraded(ctx context.Context, orgID uuid.UUID, courseID uuid.UUID, courseCode, email, displayName, quizItemID, quizTitle string, scorePercent float64, passed bool) {
	base := e.baseIRI()
	verb := xapi.VerbFailed
	if passed {
		verb = xapi.VerbPassed
	}
	scaled := scorePercent / 100.0
	if scaled > 1 {
		scaled = 1
	}
	ok := passed
	e.emit(ctx, emitParams{
		ActorEmail:    email,
		ActorName:     displayName,
		VerbID:        verb,
		ObjectID:      base + "/courses/" + courseCode + "/quizzes/" + quizItemID,
		ObjectType:    "Activity",
		ObjectTitle:   quizTitle,
		CourseID:      &courseID,
		CourseCode:    courseCode,
		CaliperType:   "AssessmentItemEvent",
		CaliperAction: caliper.ActionCompleted,
		Score:         &scaled,
		Success:       &ok,
		OrgID:         orgID,
	})
}

// StoreExternalStatement stores a received xAPI statement (e.g. H5P) in the internal LRS.
func (e Emitter) StoreExternalStatement(ctx context.Context, orgID uuid.UUID, courseID *uuid.UUID, actorEmail, actorName string, verbID, objectID, objectTitle string, rawStatement json.RawMessage) error {
	if !e.enabled() || e.Pool == nil {
		return nil
	}
	stmtID := uuid.New()
	now := time.Now().UTC()
	cal := caliper.BuildEvent(caliper.BuildInput{
		EventID:    uuid.New(),
		EventType:  "Event",
		Action:     caliper.ActionCompleted,
		ActorIRI:   e.baseIRI() + "/users/" + xapi.ActorHash(actorEmail, e.Cfg.LRSAnonymizeActors),
		ObjectIRI:  objectID,
		ObjectName: objectTitle,
		Timestamp:  now,
	})
	cRaw, _ := caliper.MarshalEvent(cal)
	full, err := json.Marshal(Payload{XAPI: rawStatement, Caliper: cRaw})
	if err != nil {
		return err
	}
	title := objectTitle
	row := xapistatements.Row{
		StatementID:     stmtID,
		ActorHash:       xapi.ActorHash(actorEmail, e.Cfg.LRSAnonymizeActors),
		VerbID:          verbID,
		ObjectID:        objectID,
		ObjectTitle:     &title,
		ContextCourseID: courseID,
		StoredAt:        now,
		FullJSON:        full,
	}
	if err := xapistatements.Insert(ctx, e.Pool, row); err != nil {
		return err
	}
	e.enqueueForward(ctx, orgID, stmtID, now)
	return nil
}
