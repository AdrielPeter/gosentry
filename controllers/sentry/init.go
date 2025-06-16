package sentry

import (
	"fmt"

	"github.com/getsentry/sentry-go"
	"gorm.io/gorm"
)

type sentryGormPlugin struct{}

func (p *sentryGormPlugin) Name() string { return "sentry" }

func (p *sentryGormPlugin) Initialize(db *gorm.DB) error {
	cb := func(db *gorm.DB) {
		ctx := db.Statement.Context
		span := sentry.StartSpan(ctx,
			"db.sql.query",
			sentry.WithDescription(db.Dialector.Explain(db.Statement.SQL.String(), db.Statement.Vars...)),
		)
		db.InstanceSet("sentry:span", span) // guarda pra fechar depois
	}

	after := func(db *gorm.DB) {
		if v, ok := db.InstanceGet("sentry:span"); ok {
			v.(*sentry.Span).Finish()
		}
	}

	// aplica nas operações que quiser
	for _, op := range []string{"create", "query", "update", "delete"} {
		db.Callback().Query().Before(fmt.Sprintf("gorm:%s", op)).Register("sentry:before", cb)
		db.Callback().Query().After(fmt.Sprintf("gorm:%s", op)).Register("sentry:after", after)
	}
	return nil
}
