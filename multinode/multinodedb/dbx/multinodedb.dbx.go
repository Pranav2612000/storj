//lint:file-ignore * generated file
// AUTOGENERATED BY storj.io/dbx
// DO NOT EDIT.

package dbx

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4/stdlib"

	"storj.io/storj/private/tagsql"
)

// Prevent conditional imports from causing build failures
var _ = strconv.Itoa
var _ = strings.LastIndex
var _ = fmt.Sprint
var _ sync.Mutex

var (
	WrapErr = func(err *Error) error { return err }
	Logger  func(format string, args ...interface{})

	errTooManyRows       = errors.New("too many rows")
	errUnsupportedDriver = errors.New("unsupported driver")
	errEmptyUpdate       = errors.New("empty update")
)

func logError(format string, args ...interface{}) {
	if Logger != nil {
		Logger(format, args...)
	}
}

type ErrorCode int

const (
	ErrorCode_Unknown ErrorCode = iota
	ErrorCode_UnsupportedDriver
	ErrorCode_NoRows
	ErrorCode_TxDone
	ErrorCode_TooManyRows
	ErrorCode_ConstraintViolation
	ErrorCode_EmptyUpdate
)

type Error struct {
	Err         error
	Code        ErrorCode
	Driver      string
	Constraint  string
	QuerySuffix string
}

func (e *Error) Error() string {
	return e.Err.Error()
}

func wrapErr(e *Error) error {
	if WrapErr == nil {
		return e
	}
	return WrapErr(e)
}

func makeErr(err error) error {
	if err == nil {
		return nil
	}
	e := &Error{Err: err}
	switch err {
	case sql.ErrNoRows:
		e.Code = ErrorCode_NoRows
	case sql.ErrTxDone:
		e.Code = ErrorCode_TxDone
	}
	return wrapErr(e)
}

func unsupportedDriver(driver string) error {
	return wrapErr(&Error{
		Err:    errUnsupportedDriver,
		Code:   ErrorCode_UnsupportedDriver,
		Driver: driver,
	})
}

func emptyUpdate() error {
	return wrapErr(&Error{
		Err:  errEmptyUpdate,
		Code: ErrorCode_EmptyUpdate,
	})
}

func tooManyRows(query_suffix string) error {
	return wrapErr(&Error{
		Err:         errTooManyRows,
		Code:        ErrorCode_TooManyRows,
		QuerySuffix: query_suffix,
	})
}

func constraintViolation(err error, constraint string) error {
	return wrapErr(&Error{
		Err:        err,
		Code:       ErrorCode_ConstraintViolation,
		Constraint: constraint,
	})
}

type driver interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (tagsql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

var (
	notAPointer     = errors.New("destination not a pointer")
	lossyConversion = errors.New("lossy conversion")
)

type DB struct {
	tagsql.DB
	dbMethods

	Hooks struct {
		Now func() time.Time
	}
}

func Open(driver, source string) (db *DB, err error) {
	var sql_db *sql.DB
	switch driver {
	case "pgx":
		sql_db, err = openpgx(source)
	default:
		return nil, unsupportedDriver(driver)
	}
	if err != nil {
		return nil, makeErr(err)
	}
	defer func(sql_db *sql.DB) {
		if err != nil {
			sql_db.Close()
		}
	}(sql_db)

	if err := sql_db.Ping(); err != nil {
		return nil, makeErr(err)
	}

	db = &DB{
		DB: tagsql.Wrap(sql_db),
	}
	db.Hooks.Now = time.Now

	switch driver {
	case "pgx":
		db.dbMethods = newpgx(db)
	default:
		return nil, unsupportedDriver(driver)
	}

	return db, nil
}

func (obj *DB) Close() (err error) {
	return obj.makeErr(obj.DB.Close())
}

func (obj *DB) Open(ctx context.Context) (*Tx, error) {
	tx, err := obj.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, obj.makeErr(err)
	}

	return &Tx{
		Tx:        tx,
		txMethods: obj.wrapTx(tx),
	}, nil
}

func (obj *DB) NewRx() *Rx {
	return &Rx{db: obj}
}

func DeleteAll(ctx context.Context, db *DB) (int64, error) {
	tx, err := db.Open(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err == nil {
			err = db.makeErr(tx.Commit())
			return
		}

		if err_rollback := tx.Rollback(); err_rollback != nil {
			logError("delete-all: rollback failed: %v", db.makeErr(err_rollback))
		}
	}()
	return tx.deleteAll(ctx)
}

type Tx struct {
	Tx tagsql.Tx
	txMethods
}

type dialectTx struct {
	tx tagsql.Tx
}

func (tx *dialectTx) Commit() (err error) {
	return makeErr(tx.tx.Commit())
}

func (tx *dialectTx) Rollback() (err error) {
	return makeErr(tx.tx.Rollback())
}

type pgxImpl struct {
	db      *DB
	dialect __sqlbundle_pgx
	driver  driver
}

func (obj *pgxImpl) Rebind(s string) string {
	return obj.dialect.Rebind(s)
}

func (obj *pgxImpl) logStmt(stmt string, args ...interface{}) {
	pgxLogStmt(stmt, args...)
}

func (obj *pgxImpl) makeErr(err error) error {
	constraint, ok := obj.isConstraintError(err)
	if ok {
		return constraintViolation(err, constraint)
	}
	return makeErr(err)
}

type pgxDB struct {
	db *DB
	*pgxImpl
}

func newpgx(db *DB) *pgxDB {
	return &pgxDB{
		db: db,
		pgxImpl: &pgxImpl{
			db:     db,
			driver: db.DB,
		},
	}
}

func (obj *pgxDB) Schema() string {
	return `CREATE TABLE members (
	id bytea NOT NULL,
	email text NOT NULL,
	name text NOT NULL,
	password_hash bytea NOT NULL,
	created_at timestamp with time zone NOT NULL,
	PRIMARY KEY ( id )
);
CREATE TABLE nodes (
	id bytea NOT NULL,
	name text NOT NULL,
	public_address text NOT NULL,
	api_secret bytea NOT NULL,
	PRIMARY KEY ( id )
);`
}

func (obj *pgxDB) wrapTx(tx tagsql.Tx) txMethods {
	return &pgxTx{
		dialectTx: dialectTx{tx: tx},
		pgxImpl: &pgxImpl{
			db:     obj.db,
			driver: tx,
		},
	}
}

type pgxTx struct {
	dialectTx
	*pgxImpl
}

func pgxLogStmt(stmt string, args ...interface{}) {
	// TODO: render placeholders
	if Logger != nil {
		out := fmt.Sprintf("stmt: %s\nargs: %v\n", stmt, pretty(args))
		Logger(out)
	}
}

type pretty []interface{}

func (p pretty) Format(f fmt.State, c rune) {
	fmt.Fprint(f, "[")
nextval:
	for i, val := range p {
		if i > 0 {
			fmt.Fprint(f, ", ")
		}
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				fmt.Fprint(f, "NULL")
				continue
			}
			val = rv.Elem().Interface()
		}
		switch v := val.(type) {
		case string:
			fmt.Fprintf(f, "%q", v)
		case time.Time:
			fmt.Fprintf(f, "%s", v.Format(time.RFC3339Nano))
		case []byte:
			for _, b := range v {
				if !unicode.IsPrint(rune(b)) {
					fmt.Fprintf(f, "%#x", v)
					continue nextval
				}
			}
			fmt.Fprintf(f, "%q", v)
		default:
			fmt.Fprintf(f, "%v", v)
		}
	}
	fmt.Fprint(f, "]")
}

type Member struct {
	Id           []byte
	Email        string
	Name         string
	PasswordHash []byte
	CreatedAt    time.Time
}

func (Member) _Table() string { return "members" }

type Member_Update_Fields struct {
	Email        Member_Email_Field
	Name         Member_Name_Field
	PasswordHash Member_PasswordHash_Field
}

type Member_Id_Field struct {
	_set   bool
	_null  bool
	_value []byte
}

func Member_Id(v []byte) Member_Id_Field {
	return Member_Id_Field{_set: true, _value: v}
}

func (f Member_Id_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Member_Id_Field) _Column() string { return "id" }

type Member_Email_Field struct {
	_set   bool
	_null  bool
	_value string
}

func Member_Email(v string) Member_Email_Field {
	return Member_Email_Field{_set: true, _value: v}
}

func (f Member_Email_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Member_Email_Field) _Column() string { return "email" }

type Member_Name_Field struct {
	_set   bool
	_null  bool
	_value string
}

func Member_Name(v string) Member_Name_Field {
	return Member_Name_Field{_set: true, _value: v}
}

func (f Member_Name_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Member_Name_Field) _Column() string { return "name" }

type Member_PasswordHash_Field struct {
	_set   bool
	_null  bool
	_value []byte
}

func Member_PasswordHash(v []byte) Member_PasswordHash_Field {
	return Member_PasswordHash_Field{_set: true, _value: v}
}

func (f Member_PasswordHash_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Member_PasswordHash_Field) _Column() string { return "password_hash" }

type Member_CreatedAt_Field struct {
	_set   bool
	_null  bool
	_value time.Time
}

func Member_CreatedAt(v time.Time) Member_CreatedAt_Field {
	return Member_CreatedAt_Field{_set: true, _value: v}
}

func (f Member_CreatedAt_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Member_CreatedAt_Field) _Column() string { return "created_at" }

type Node struct {
	Id            []byte
	Name          string
	PublicAddress string
	ApiSecret     []byte
}

func (Node) _Table() string { return "nodes" }

type Node_Update_Fields struct {
	Name Node_Name_Field
}

type Node_Id_Field struct {
	_set   bool
	_null  bool
	_value []byte
}

func Node_Id(v []byte) Node_Id_Field {
	return Node_Id_Field{_set: true, _value: v}
}

func (f Node_Id_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Node_Id_Field) _Column() string { return "id" }

type Node_Name_Field struct {
	_set   bool
	_null  bool
	_value string
}

func Node_Name(v string) Node_Name_Field {
	return Node_Name_Field{_set: true, _value: v}
}

func (f Node_Name_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Node_Name_Field) _Column() string { return "name" }

type Node_PublicAddress_Field struct {
	_set   bool
	_null  bool
	_value string
}

func Node_PublicAddress(v string) Node_PublicAddress_Field {
	return Node_PublicAddress_Field{_set: true, _value: v}
}

func (f Node_PublicAddress_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Node_PublicAddress_Field) _Column() string { return "public_address" }

type Node_ApiSecret_Field struct {
	_set   bool
	_null  bool
	_value []byte
}

func Node_ApiSecret(v []byte) Node_ApiSecret_Field {
	return Node_ApiSecret_Field{_set: true, _value: v}
}

func (f Node_ApiSecret_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Node_ApiSecret_Field) _Column() string { return "api_secret" }

func toUTC(t time.Time) time.Time {
	return t.UTC()
}

func toDate(t time.Time) time.Time {
	// keep up the minute portion so that translations between timezones will
	// continue to reflect properly.
	return t.Truncate(time.Minute)
}

//
// runtime support for building sql statements
//

type __sqlbundle_SQL interface {
	Render() string

	private()
}

type __sqlbundle_Dialect interface {
	Rebind(sql string) string
}

type __sqlbundle_RenderOp int

const (
	__sqlbundle_NoFlatten __sqlbundle_RenderOp = iota
	__sqlbundle_NoTerminate
)

func __sqlbundle_Render(dialect __sqlbundle_Dialect, sql __sqlbundle_SQL, ops ...__sqlbundle_RenderOp) string {
	out := sql.Render()

	flatten := true
	terminate := true
	for _, op := range ops {
		switch op {
		case __sqlbundle_NoFlatten:
			flatten = false
		case __sqlbundle_NoTerminate:
			terminate = false
		}
	}

	if flatten {
		out = __sqlbundle_flattenSQL(out)
	}
	if terminate {
		out += ";"
	}

	return dialect.Rebind(out)
}

func __sqlbundle_flattenSQL(x string) string {
	// trim whitespace from beginning and end
	s, e := 0, len(x)-1
	for s < len(x) && (x[s] == ' ' || x[s] == '\t' || x[s] == '\n') {
		s++
	}
	for s <= e && (x[e] == ' ' || x[e] == '\t' || x[e] == '\n') {
		e--
	}
	if s > e {
		return ""
	}
	x = x[s : e+1]

	// check for whitespace that needs fixing
	wasSpace := false
	for i := 0; i < len(x); i++ {
		r := x[i]
		justSpace := r == ' '
		if (wasSpace && justSpace) || r == '\t' || r == '\n' {
			// whitespace detected, start writing a new string
			var result strings.Builder
			result.Grow(len(x))
			if wasSpace {
				result.WriteString(x[:i-1])
			} else {
				result.WriteString(x[:i])
			}
			for p := i; p < len(x); p++ {
				for p < len(x) && (x[p] == ' ' || x[p] == '\t' || x[p] == '\n') {
					p++
				}
				result.WriteByte(' ')

				start := p
				for p < len(x) && !(x[p] == ' ' || x[p] == '\t' || x[p] == '\n') {
					p++
				}
				result.WriteString(x[start:p])
			}

			return result.String()
		}
		wasSpace = justSpace
	}

	// no problematic whitespace found
	return x
}

// this type is specially named to match up with the name returned by the
// dialect impl in the sql package.
type __sqlbundle_postgres struct{}

func (p __sqlbundle_postgres) Rebind(sql string) string {
	type sqlParseState int
	const (
		sqlParseStart sqlParseState = iota
		sqlParseInStringLiteral
		sqlParseInQuotedIdentifier
		sqlParseInComment
	)

	out := make([]byte, 0, len(sql)+10)

	j := 1
	state := sqlParseStart
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		switch state {
		case sqlParseStart:
			switch ch {
			case '?':
				out = append(out, '$')
				out = append(out, strconv.Itoa(j)...)
				state = sqlParseStart
				j++
				continue
			case '-':
				if i+1 < len(sql) && sql[i+1] == '-' {
					state = sqlParseInComment
				}
			case '"':
				state = sqlParseInQuotedIdentifier
			case '\'':
				state = sqlParseInStringLiteral
			}
		case sqlParseInStringLiteral:
			if ch == '\'' {
				state = sqlParseStart
			}
		case sqlParseInQuotedIdentifier:
			if ch == '"' {
				state = sqlParseStart
			}
		case sqlParseInComment:
			if ch == '\n' {
				state = sqlParseStart
			}
		}
		out = append(out, ch)
	}

	return string(out)
}

// this type is specially named to match up with the name returned by the
// dialect impl in the sql package.
type __sqlbundle_sqlite3 struct{}

func (s __sqlbundle_sqlite3) Rebind(sql string) string {
	return sql
}

// this type is specially named to match up with the name returned by the
// dialect impl in the sql package.
type __sqlbundle_cockroach struct{}

func (p __sqlbundle_cockroach) Rebind(sql string) string {
	type sqlParseState int
	const (
		sqlParseStart sqlParseState = iota
		sqlParseInStringLiteral
		sqlParseInQuotedIdentifier
		sqlParseInComment
	)

	out := make([]byte, 0, len(sql)+10)

	j := 1
	state := sqlParseStart
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		switch state {
		case sqlParseStart:
			switch ch {
			case '?':
				out = append(out, '$')
				out = append(out, strconv.Itoa(j)...)
				state = sqlParseStart
				j++
				continue
			case '-':
				if i+1 < len(sql) && sql[i+1] == '-' {
					state = sqlParseInComment
				}
			case '"':
				state = sqlParseInQuotedIdentifier
			case '\'':
				state = sqlParseInStringLiteral
			}
		case sqlParseInStringLiteral:
			if ch == '\'' {
				state = sqlParseStart
			}
		case sqlParseInQuotedIdentifier:
			if ch == '"' {
				state = sqlParseStart
			}
		case sqlParseInComment:
			if ch == '\n' {
				state = sqlParseStart
			}
		}
		out = append(out, ch)
	}

	return string(out)
}

// this type is specially named to match up with the name returned by the
// dialect impl in the sql package.
type __sqlbundle_pgx struct{}

func (p __sqlbundle_pgx) Rebind(sql string) string {
	type sqlParseState int
	const (
		sqlParseStart sqlParseState = iota
		sqlParseInStringLiteral
		sqlParseInQuotedIdentifier
		sqlParseInComment
	)

	out := make([]byte, 0, len(sql)+10)

	j := 1
	state := sqlParseStart
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		switch state {
		case sqlParseStart:
			switch ch {
			case '?':
				out = append(out, '$')
				out = append(out, strconv.Itoa(j)...)
				state = sqlParseStart
				j++
				continue
			case '-':
				if i+1 < len(sql) && sql[i+1] == '-' {
					state = sqlParseInComment
				}
			case '"':
				state = sqlParseInQuotedIdentifier
			case '\'':
				state = sqlParseInStringLiteral
			}
		case sqlParseInStringLiteral:
			if ch == '\'' {
				state = sqlParseStart
			}
		case sqlParseInQuotedIdentifier:
			if ch == '"' {
				state = sqlParseStart
			}
		case sqlParseInComment:
			if ch == '\n' {
				state = sqlParseStart
			}
		}
		out = append(out, ch)
	}

	return string(out)
}

// this type is specially named to match up with the name returned by the
// dialect impl in the sql package.
type __sqlbundle_pgxcockroach struct{}

func (p __sqlbundle_pgxcockroach) Rebind(sql string) string {
	type sqlParseState int
	const (
		sqlParseStart sqlParseState = iota
		sqlParseInStringLiteral
		sqlParseInQuotedIdentifier
		sqlParseInComment
	)

	out := make([]byte, 0, len(sql)+10)

	j := 1
	state := sqlParseStart
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		switch state {
		case sqlParseStart:
			switch ch {
			case '?':
				out = append(out, '$')
				out = append(out, strconv.Itoa(j)...)
				state = sqlParseStart
				j++
				continue
			case '-':
				if i+1 < len(sql) && sql[i+1] == '-' {
					state = sqlParseInComment
				}
			case '"':
				state = sqlParseInQuotedIdentifier
			case '\'':
				state = sqlParseInStringLiteral
			}
		case sqlParseInStringLiteral:
			if ch == '\'' {
				state = sqlParseStart
			}
		case sqlParseInQuotedIdentifier:
			if ch == '"' {
				state = sqlParseStart
			}
		case sqlParseInComment:
			if ch == '\n' {
				state = sqlParseStart
			}
		}
		out = append(out, ch)
	}

	return string(out)
}

type __sqlbundle_Literal string

func (__sqlbundle_Literal) private() {}

func (l __sqlbundle_Literal) Render() string { return string(l) }

type __sqlbundle_Literals struct {
	Join string
	SQLs []__sqlbundle_SQL
}

func (__sqlbundle_Literals) private() {}

func (l __sqlbundle_Literals) Render() string {
	var out bytes.Buffer

	first := true
	for _, sql := range l.SQLs {
		if sql == nil {
			continue
		}
		if !first {
			out.WriteString(l.Join)
		}
		first = false
		out.WriteString(sql.Render())
	}

	return out.String()
}

type __sqlbundle_Condition struct {
	// set at compile/embed time
	Name  string
	Left  string
	Equal bool
	Right string

	// set at runtime
	Null bool
}

func (*__sqlbundle_Condition) private() {}

func (c *__sqlbundle_Condition) Render() string {
	// TODO(jeff): maybe check if we can use placeholders instead of the
	// literal null: this would make the templates easier.

	switch {
	case c.Equal && c.Null:
		return c.Left + " is null"
	case c.Equal && !c.Null:
		return c.Left + " = " + c.Right
	case !c.Equal && c.Null:
		return c.Left + " is not null"
	case !c.Equal && !c.Null:
		return c.Left + " != " + c.Right
	default:
		panic("unhandled case")
	}
}

type __sqlbundle_Hole struct {
	// set at compiile/embed time
	Name string

	// set at runtime or possibly embed time
	SQL __sqlbundle_SQL
}

func (*__sqlbundle_Hole) private() {}

func (h *__sqlbundle_Hole) Render() string {
	if h.SQL == nil {
		return ""
	}
	return h.SQL.Render()
}

//
// end runtime support for building sql statements
//

func (obj *pgxImpl) Create_Node(ctx context.Context,
	node_id Node_Id_Field,
	node_name Node_Name_Field,
	node_public_address Node_PublicAddress_Field,
	node_api_secret Node_ApiSecret_Field) (
	node *Node, err error) {
	defer mon.Task()(&ctx)(&err)
	__id_val := node_id.value()
	__name_val := node_name.value()
	__public_address_val := node_public_address.value()
	__api_secret_val := node_api_secret.value()

	var __embed_stmt = __sqlbundle_Literal("INSERT INTO nodes ( id, name, public_address, api_secret ) VALUES ( ?, ?, ?, ? ) RETURNING nodes.id, nodes.name, nodes.public_address, nodes.api_secret")

	var __values []interface{}
	__values = append(__values, __id_val, __name_val, __public_address_val, __api_secret_val)

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	node = &Node{}
	err = obj.driver.QueryRowContext(ctx, __stmt, __values...).Scan(&node.Id, &node.Name, &node.PublicAddress, &node.ApiSecret)
	if err != nil {
		return nil, obj.makeErr(err)
	}
	return node, nil

}

func (obj *pgxImpl) Create_Member(ctx context.Context,
	member_id Member_Id_Field,
	member_email Member_Email_Field,
	member_name Member_Name_Field,
	member_password_hash Member_PasswordHash_Field) (
	member *Member, err error) {
	defer mon.Task()(&ctx)(&err)

	__now := obj.db.Hooks.Now().UTC()
	__id_val := member_id.value()
	__email_val := member_email.value()
	__name_val := member_name.value()
	__password_hash_val := member_password_hash.value()
	__created_at_val := __now

	var __embed_stmt = __sqlbundle_Literal("INSERT INTO members ( id, email, name, password_hash, created_at ) VALUES ( ?, ?, ?, ?, ? ) RETURNING members.id, members.email, members.name, members.password_hash, members.created_at")

	var __values []interface{}
	__values = append(__values, __id_val, __email_val, __name_val, __password_hash_val, __created_at_val)

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	member = &Member{}
	err = obj.driver.QueryRowContext(ctx, __stmt, __values...).Scan(&member.Id, &member.Email, &member.Name, &member.PasswordHash, &member.CreatedAt)
	if err != nil {
		return nil, obj.makeErr(err)
	}
	return member, nil

}

func (obj *pgxImpl) Get_Node_By_Id(ctx context.Context,
	node_id Node_Id_Field) (
	node *Node, err error) {
	defer mon.Task()(&ctx)(&err)

	var __embed_stmt = __sqlbundle_Literal("SELECT nodes.id, nodes.name, nodes.public_address, nodes.api_secret FROM nodes WHERE nodes.id = ?")

	var __values []interface{}
	__values = append(__values, node_id.value())

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	node = &Node{}
	err = obj.driver.QueryRowContext(ctx, __stmt, __values...).Scan(&node.Id, &node.Name, &node.PublicAddress, &node.ApiSecret)
	if err != nil {
		return (*Node)(nil), obj.makeErr(err)
	}
	return node, nil

}

func (obj *pgxImpl) All_Node(ctx context.Context) (
	rows []*Node, err error) {
	defer mon.Task()(&ctx)(&err)

	var __embed_stmt = __sqlbundle_Literal("SELECT nodes.id, nodes.name, nodes.public_address, nodes.api_secret FROM nodes")

	var __values []interface{}

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	__rows, err := obj.driver.QueryContext(ctx, __stmt, __values...)
	if err != nil {
		return nil, obj.makeErr(err)
	}
	defer __rows.Close()

	for __rows.Next() {
		node := &Node{}
		err = __rows.Scan(&node.Id, &node.Name, &node.PublicAddress, &node.ApiSecret)
		if err != nil {
			return nil, obj.makeErr(err)
		}
		rows = append(rows, node)
	}
	if err := __rows.Err(); err != nil {
		return nil, obj.makeErr(err)
	}
	return rows, nil

}

func (obj *pgxImpl) Get_Member_By_Email(ctx context.Context,
	member_email Member_Email_Field) (
	member *Member, err error) {
	defer mon.Task()(&ctx)(&err)

	var __embed_stmt = __sqlbundle_Literal("SELECT members.id, members.email, members.name, members.password_hash, members.created_at FROM members WHERE members.email = ? LIMIT 2")

	var __values []interface{}
	__values = append(__values, member_email.value())

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	__rows, err := obj.driver.QueryContext(ctx, __stmt, __values...)
	if err != nil {
		return nil, obj.makeErr(err)
	}
	defer __rows.Close()

	if !__rows.Next() {
		if err := __rows.Err(); err != nil {
			return nil, obj.makeErr(err)
		}
		return nil, makeErr(sql.ErrNoRows)
	}

	member = &Member{}
	err = __rows.Scan(&member.Id, &member.Email, &member.Name, &member.PasswordHash, &member.CreatedAt)
	if err != nil {
		return nil, obj.makeErr(err)
	}

	if __rows.Next() {
		return nil, tooManyRows("Member_By_Email")
	}

	if err := __rows.Err(); err != nil {
		return nil, obj.makeErr(err)
	}

	return member, nil

}

func (obj *pgxImpl) Get_Member_By_Id(ctx context.Context,
	member_id Member_Id_Field) (
	member *Member, err error) {
	defer mon.Task()(&ctx)(&err)

	var __embed_stmt = __sqlbundle_Literal("SELECT members.id, members.email, members.name, members.password_hash, members.created_at FROM members WHERE members.id = ?")

	var __values []interface{}
	__values = append(__values, member_id.value())

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	member = &Member{}
	err = obj.driver.QueryRowContext(ctx, __stmt, __values...).Scan(&member.Id, &member.Email, &member.Name, &member.PasswordHash, &member.CreatedAt)
	if err != nil {
		return (*Member)(nil), obj.makeErr(err)
	}
	return member, nil

}

func (obj *pgxImpl) Update_Node_By_Id(ctx context.Context,
	node_id Node_Id_Field,
	update Node_Update_Fields) (
	node *Node, err error) {
	defer mon.Task()(&ctx)(&err)
	var __sets = &__sqlbundle_Hole{}

	var __embed_stmt = __sqlbundle_Literals{Join: "", SQLs: []__sqlbundle_SQL{__sqlbundle_Literal("UPDATE nodes SET "), __sets, __sqlbundle_Literal(" WHERE nodes.id = ? RETURNING nodes.id, nodes.name, nodes.public_address, nodes.api_secret")}}

	__sets_sql := __sqlbundle_Literals{Join: ", "}
	var __values []interface{}
	var __args []interface{}

	if update.Name._set {
		__values = append(__values, update.Name.value())
		__sets_sql.SQLs = append(__sets_sql.SQLs, __sqlbundle_Literal("name = ?"))
	}

	if len(__sets_sql.SQLs) == 0 {
		return nil, emptyUpdate()
	}

	__args = append(__args, node_id.value())

	__values = append(__values, __args...)
	__sets.SQL = __sets_sql

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	node = &Node{}
	err = obj.driver.QueryRowContext(ctx, __stmt, __values...).Scan(&node.Id, &node.Name, &node.PublicAddress, &node.ApiSecret)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, obj.makeErr(err)
	}
	return node, nil
}

func (obj *pgxImpl) UpdateNoReturn_Node_By_Id(ctx context.Context,
	node_id Node_Id_Field,
	update Node_Update_Fields) (
	err error) {
	defer mon.Task()(&ctx)(&err)
	var __sets = &__sqlbundle_Hole{}

	var __embed_stmt = __sqlbundle_Literals{Join: "", SQLs: []__sqlbundle_SQL{__sqlbundle_Literal("UPDATE nodes SET "), __sets, __sqlbundle_Literal(" WHERE nodes.id = ?")}}

	__sets_sql := __sqlbundle_Literals{Join: ", "}
	var __values []interface{}
	var __args []interface{}

	if update.Name._set {
		__values = append(__values, update.Name.value())
		__sets_sql.SQLs = append(__sets_sql.SQLs, __sqlbundle_Literal("name = ?"))
	}

	if len(__sets_sql.SQLs) == 0 {
		return emptyUpdate()
	}

	__args = append(__args, node_id.value())

	__values = append(__values, __args...)
	__sets.SQL = __sets_sql

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	_, err = obj.driver.ExecContext(ctx, __stmt, __values...)
	if err != nil {
		return obj.makeErr(err)
	}
	return nil
}

func (obj *pgxImpl) Update_Member_By_Id(ctx context.Context,
	member_id Member_Id_Field,
	update Member_Update_Fields) (
	member *Member, err error) {
	defer mon.Task()(&ctx)(&err)
	var __sets = &__sqlbundle_Hole{}

	var __embed_stmt = __sqlbundle_Literals{Join: "", SQLs: []__sqlbundle_SQL{__sqlbundle_Literal("UPDATE members SET "), __sets, __sqlbundle_Literal(" WHERE members.id = ? RETURNING members.id, members.email, members.name, members.password_hash, members.created_at")}}

	__sets_sql := __sqlbundle_Literals{Join: ", "}
	var __values []interface{}
	var __args []interface{}

	if update.Email._set {
		__values = append(__values, update.Email.value())
		__sets_sql.SQLs = append(__sets_sql.SQLs, __sqlbundle_Literal("email = ?"))
	}

	if update.Name._set {
		__values = append(__values, update.Name.value())
		__sets_sql.SQLs = append(__sets_sql.SQLs, __sqlbundle_Literal("name = ?"))
	}

	if update.PasswordHash._set {
		__values = append(__values, update.PasswordHash.value())
		__sets_sql.SQLs = append(__sets_sql.SQLs, __sqlbundle_Literal("password_hash = ?"))
	}

	if len(__sets_sql.SQLs) == 0 {
		return nil, emptyUpdate()
	}

	__args = append(__args, member_id.value())

	__values = append(__values, __args...)
	__sets.SQL = __sets_sql

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	member = &Member{}
	err = obj.driver.QueryRowContext(ctx, __stmt, __values...).Scan(&member.Id, &member.Email, &member.Name, &member.PasswordHash, &member.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, obj.makeErr(err)
	}
	return member, nil
}

func (obj *pgxImpl) Delete_Node_By_Id(ctx context.Context,
	node_id Node_Id_Field) (
	deleted bool, err error) {
	defer mon.Task()(&ctx)(&err)

	var __embed_stmt = __sqlbundle_Literal("DELETE FROM nodes WHERE nodes.id = ?")

	var __values []interface{}
	__values = append(__values, node_id.value())

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	__res, err := obj.driver.ExecContext(ctx, __stmt, __values...)
	if err != nil {
		return false, obj.makeErr(err)
	}

	__count, err := __res.RowsAffected()
	if err != nil {
		return false, obj.makeErr(err)
	}

	return __count > 0, nil

}

func (obj *pgxImpl) Delete_Member_By_Id(ctx context.Context,
	member_id Member_Id_Field) (
	deleted bool, err error) {
	defer mon.Task()(&ctx)(&err)

	var __embed_stmt = __sqlbundle_Literal("DELETE FROM members WHERE members.id = ?")

	var __values []interface{}
	__values = append(__values, member_id.value())

	var __stmt = __sqlbundle_Render(obj.dialect, __embed_stmt)
	obj.logStmt(__stmt, __values...)

	__res, err := obj.driver.ExecContext(ctx, __stmt, __values...)
	if err != nil {
		return false, obj.makeErr(err)
	}

	__count, err := __res.RowsAffected()
	if err != nil {
		return false, obj.makeErr(err)
	}

	return __count > 0, nil

}

func (impl pgxImpl) isConstraintError(err error) (
	constraint string, ok bool) {
	if e, ok := err.(*pgconn.PgError); ok {
		if e.Code[:2] == "23" {
			return e.ConstraintName, true
		}
	}
	return "", false
}

func (obj *pgxImpl) deleteAll(ctx context.Context) (count int64, err error) {
	defer mon.Task()(&ctx)(&err)
	var __res sql.Result
	var __count int64
	__res, err = obj.driver.ExecContext(ctx, "DELETE FROM nodes;")
	if err != nil {
		return 0, obj.makeErr(err)
	}

	__count, err = __res.RowsAffected()
	if err != nil {
		return 0, obj.makeErr(err)
	}
	count += __count
	__res, err = obj.driver.ExecContext(ctx, "DELETE FROM members;")
	if err != nil {
		return 0, obj.makeErr(err)
	}

	__count, err = __res.RowsAffected()
	if err != nil {
		return 0, obj.makeErr(err)
	}
	count += __count

	return count, nil

}

type Rx struct {
	db *DB
	tx *Tx
}

func (rx *Rx) UnsafeTx(ctx context.Context) (unsafe_tx tagsql.Tx, err error) {
	tx, err := rx.getTx(ctx)
	if err != nil {
		return nil, err
	}
	return tx.Tx, nil
}

func (rx *Rx) getTx(ctx context.Context) (tx *Tx, err error) {
	if rx.tx == nil {
		if rx.tx, err = rx.db.Open(ctx); err != nil {
			return nil, err
		}
	}
	return rx.tx, nil
}

func (rx *Rx) Rebind(s string) string {
	return rx.db.Rebind(s)
}

func (rx *Rx) Commit() (err error) {
	if rx.tx != nil {
		err = rx.tx.Commit()
		rx.tx = nil
	}
	return err
}

func (rx *Rx) Rollback() (err error) {
	if rx.tx != nil {
		err = rx.tx.Rollback()
		rx.tx = nil
	}
	return err
}

func (rx *Rx) All_Node(ctx context.Context) (
	rows []*Node, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.All_Node(ctx)
}

func (rx *Rx) Create_Member(ctx context.Context,
	member_id Member_Id_Field,
	member_email Member_Email_Field,
	member_name Member_Name_Field,
	member_password_hash Member_PasswordHash_Field) (
	member *Member, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.Create_Member(ctx, member_id, member_email, member_name, member_password_hash)

}

func (rx *Rx) Create_Node(ctx context.Context,
	node_id Node_Id_Field,
	node_name Node_Name_Field,
	node_public_address Node_PublicAddress_Field,
	node_api_secret Node_ApiSecret_Field) (
	node *Node, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.Create_Node(ctx, node_id, node_name, node_public_address, node_api_secret)

}

func (rx *Rx) Delete_Member_By_Id(ctx context.Context,
	member_id Member_Id_Field) (
	deleted bool, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.Delete_Member_By_Id(ctx, member_id)
}

func (rx *Rx) Delete_Node_By_Id(ctx context.Context,
	node_id Node_Id_Field) (
	deleted bool, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.Delete_Node_By_Id(ctx, node_id)
}

func (rx *Rx) Get_Member_By_Email(ctx context.Context,
	member_email Member_Email_Field) (
	member *Member, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.Get_Member_By_Email(ctx, member_email)
}

func (rx *Rx) Get_Member_By_Id(ctx context.Context,
	member_id Member_Id_Field) (
	member *Member, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.Get_Member_By_Id(ctx, member_id)
}

func (rx *Rx) Get_Node_By_Id(ctx context.Context,
	node_id Node_Id_Field) (
	node *Node, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.Get_Node_By_Id(ctx, node_id)
}

func (rx *Rx) UpdateNoReturn_Node_By_Id(ctx context.Context,
	node_id Node_Id_Field,
	update Node_Update_Fields) (
	err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.UpdateNoReturn_Node_By_Id(ctx, node_id, update)
}

func (rx *Rx) Update_Member_By_Id(ctx context.Context,
	member_id Member_Id_Field,
	update Member_Update_Fields) (
	member *Member, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.Update_Member_By_Id(ctx, member_id, update)
}

func (rx *Rx) Update_Node_By_Id(ctx context.Context,
	node_id Node_Id_Field,
	update Node_Update_Fields) (
	node *Node, err error) {
	var tx *Tx
	if tx, err = rx.getTx(ctx); err != nil {
		return
	}
	return tx.Update_Node_By_Id(ctx, node_id, update)
}

type Methods interface {
	All_Node(ctx context.Context) (
		rows []*Node, err error)

	Create_Member(ctx context.Context,
		member_id Member_Id_Field,
		member_email Member_Email_Field,
		member_name Member_Name_Field,
		member_password_hash Member_PasswordHash_Field) (
		member *Member, err error)

	Create_Node(ctx context.Context,
		node_id Node_Id_Field,
		node_name Node_Name_Field,
		node_public_address Node_PublicAddress_Field,
		node_api_secret Node_ApiSecret_Field) (
		node *Node, err error)

	Delete_Member_By_Id(ctx context.Context,
		member_id Member_Id_Field) (
		deleted bool, err error)

	Delete_Node_By_Id(ctx context.Context,
		node_id Node_Id_Field) (
		deleted bool, err error)

	Get_Member_By_Email(ctx context.Context,
		member_email Member_Email_Field) (
		member *Member, err error)

	Get_Member_By_Id(ctx context.Context,
		member_id Member_Id_Field) (
		member *Member, err error)

	Get_Node_By_Id(ctx context.Context,
		node_id Node_Id_Field) (
		node *Node, err error)

	UpdateNoReturn_Node_By_Id(ctx context.Context,
		node_id Node_Id_Field,
		update Node_Update_Fields) (
		err error)

	Update_Member_By_Id(ctx context.Context,
		member_id Member_Id_Field,
		update Member_Update_Fields) (
		member *Member, err error)

	Update_Node_By_Id(ctx context.Context,
		node_id Node_Id_Field,
		update Node_Update_Fields) (
		node *Node, err error)
}

type TxMethods interface {
	Methods

	Rebind(s string) string
	Commit() error
	Rollback() error
}

type txMethods interface {
	TxMethods

	deleteAll(ctx context.Context) (int64, error)
	makeErr(err error) error
}

type DBMethods interface {
	Methods

	Schema() string
	Rebind(sql string) string
}

type dbMethods interface {
	DBMethods

	wrapTx(tx tagsql.Tx) txMethods
	makeErr(err error) error
}

func openpgx(source string) (*sql.DB, error) {
	return sql.Open("pgx", source)
}
