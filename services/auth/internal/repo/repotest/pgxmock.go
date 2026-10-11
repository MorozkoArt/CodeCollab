package repotest

import (
	"context"
	"testing"

	pkgdb "github.com/MorozkoArt/CodeCollab/pkg/db"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/pashagolub/pgxmock"
	"github.com/stretchr/testify/require"
)

var _ pkgdb.SQL = Adapter{}

type Adapter struct {
	pgxmock.PgxConnIface
}

func New(t *testing.T) (pkgdb.SQL, pgxmock.PgxConnIface) {
	t.Helper()

	mock, err := pgxmock.NewConn()
	require.NoError(t, err)

	t.Cleanup(func() { _ = mock.Close(context.Background()) })

	return Adapter{PgxConnIface: mock}, mock
}

func (Adapter) Begin(context.Context) (pgx.Tx, error) { panic("unimplemented") }

func (Adapter) BeginFunc(context.Context, func(tx pgx.Tx) error) error {
	panic("unimplemented")
}

func (Adapter) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	panic("unimplemented")
}

func (Adapter) BeginTxFunc(context.Context, pgx.TxOptions, func(tx pgx.Tx) error) error {
	panic("unimplemented")
}

func (a Adapter) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return a.PgxConnIface.Exec(ctx, query, args...)
}

func (Adapter) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unimplemented")
}

func (a Adapter) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return a.PgxConnIface.QueryRow(ctx, query, args...)
}

func (Adapter) Scan(context.Context, pgxscan.Querier, any, string, ...any) error {
	panic("unimplemented")
}

func (a Adapter) Close() { _ = a.PgxConnIface.Close(context.Background()) }
