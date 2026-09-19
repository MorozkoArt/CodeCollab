package repo_test

import (
	"context"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/pashagolub/pgxmock"
)

type adapter struct {
	pgxmock.PgxConnIface
}

func (adapter) Begin(ctx context.Context) (pgx.Tx, error) {
	panic("unimplemented")
}

func (adapter) BeginFunc(ctx context.Context, f func(tx pgx.Tx) error) error {
	panic("unimplemented")
}

func (adapter) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	panic("unimplemented")
}

func (adapter) BeginTxFunc(ctx context.Context, txOptions pgx.TxOptions, f func(tx pgx.Tx) error) error {
	panic("unimplemented")
}

func (a adapter) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return a.PgxConnIface.Exec(ctx, query, args...)
}

func (adapter) Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error) {
	panic("unimplemented")
}

func (a adapter) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return a.PgxConnIface.QueryRow(ctx, query, args...)
}

func (adapter) Scan(ctx context.Context, executor pgxscan.Querier, dest any, query string, args ...any) error {
	panic("unimplemented")
}

func (a adapter) Close() {
	a.PgxConnIface.Close(context.Background())
}
