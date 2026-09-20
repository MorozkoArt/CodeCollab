package db

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Connector interface {
	Connect(ctx context.Context, dbConfig DbConfig) (Executor, error)
	Close()
}

type Transactor interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	BeginFunc(ctx context.Context, f func(tx pgx.Tx) error) error
	BeginTxFunc(ctx context.Context, txOptions pgx.TxOptions, f func(tx pgx.Tx) error) error
}

type Executor interface {
	Ping(ctx context.Context) error
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
	Close()
	Transactor
}

type SQL interface {
	Executor
	Scan(
		ctx context.Context,
		executor pgxscan.Querier,
		dest any,
		query string,
		args ...any,
	) error
}

type DBSql struct {
	Executor
}

func NewDBSql(exec Executor) *DBSql {
	return &DBSql{exec}
}

func (d *DBSql) Scan(
	ctx context.Context,
	executor pgxscan.Querier,
	dest any,
	query string,
	args ...any,
) error {
	if executor == nil {
		executor = d
	}

	return pgxscan.Select(ctx, executor, dest, query, args...)
}

type DB interface {
	Connector
	SQL() SQL
	Builder() squirrel.StatementBuilderType
}

type DBClient struct {
	sql     SQL
	builder squirrel.StatementBuilderType
}

func NewDBClient(ctx context.Context, dbConfig DbConfig) (*DBClient, error) {
	client := &DBClient{}

	exec, err := client.Connect(ctx, dbConfig)
	if err != nil {
		return nil, err
	}

	client.sql = NewDBSql(exec)
	client.builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	return client, nil
}

func (d *DBClient) Connect(ctx context.Context, dbConfig DbConfig) (Executor, error) {
	// TODO: сейчас реализованы простые настройки, можно также настроить:
	// pool_max_conns: integer greater than 0
	// pool_min_conns: integer 0 or greater
	// pool_max_conn_lifetime: duration string
	// pool_max_conn_idle_time: duration string
	// pool_health_check_period: duration string
	// pool_max_conn_lifetime_jitter: duration string
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host(),
		dbConfig.Port(),
		dbConfig.User(),
		dbConfig.Pass(),
		dbConfig.Name(),
		dbConfig.SSLMode(),
	)

	conf, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	pol, err := pgxpool.ConnectConfig(ctx, conf)

	return pol, err
}

func (d *DBClient) Close() {
	if d.sql == nil {
		return
	}

	d.sql.Close()
}

func (d *DBClient) SQL() SQL {
	return d.sql
}

func (d *DBClient) Builder() squirrel.StatementBuilderType {
	return d.builder
}
