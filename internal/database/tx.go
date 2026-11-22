package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// txKey is the context key under which WithinTx stores its transaction.
type txKey struct{}

// Querier is the query surface shared by *sqlx.DB and *sqlx.Tx.
type Querier interface {
	sqlx.ExtContext
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
}

// Conn returns the transaction carried by ctx, or db if ctx carries none.
// Repositories run their queries on it so they join a transaction started by
// a service through Transactor.WithinTx.
func Conn(ctx context.Context, db *sqlx.DB) Querier {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return db
}

// Transactor runs functions in a database transaction. It is safe for
// concurrent use.
type Transactor struct {
	db *sqlx.DB
}

// NewTransactor returns a Transactor that starts transactions on db.
func NewTransactor(db *sqlx.DB) *Transactor {
	return &Transactor{db: db}
}

// WithinTx runs fn in a transaction carried by the context passed to fn, so
// queries must use that context, not ctx. It commits if fn returns nil, and
// otherwise rolls back and returns fn's error unchanged. A call inside another
// WithinTx joins the outer transaction.
func (t *Transactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return fn(ctx)
	}

	tx, err := t.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // A no-op once committed.

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}

	return tx.Commit()
}
