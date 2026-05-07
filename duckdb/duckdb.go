// Package duckdb provides native DuckDB integration for Solod.
//
// This implementation ships inside [soloduck] so applications can link libduckdb while using
// stock solod.dev from [upstream Solod]—the core repo does not ship this package.
//
// This package is designed around DuckDB's C API with explicit resource
// ownership and predictable lifecycles. Query execution follows duckdb_query
// semantics: after any call that fills a result object, Result.Close must run
// even when the query fails, so errors attached to the result are released correctly.
//
// Follow the DuckDB C overview flow (install libduckdb, use duckdb.h): open a database,
// connect, run queries, disconnect and close. See:
//   - https://duckdb.org/docs/current/clients/c/overview
//   - https://duckdb.org/docs/current/clients/c/connect
//
// Query/result details align with:
// https://duckdb.org/docs/current/clients/c/query
//
// Building translated programs that import this package requires DuckDB headers
// and library to be available to the C compiler and linker (e.g. `-lduckdb`).
//
// [soloduck]: https://github.com/lmangani/soloduck
// [upstream Solod]: https://github.com/solod-dev/solod
package duckdb

import (
	"solod.dev/so/c"
	"solod.dev/so/errors"
	"solod.dev/so/mem"
	"solod.dev/so/strings"
)

var (
	ErrOpen       = errors.New("duckdb: open failed")
	ErrClosed     = errors.New("duckdb: connection is closed")
	ErrQuery      = errors.New("duckdb: query failed")
	ErrPrepare    = errors.New("duckdb: prepare failed")
	ErrBind       = errors.New("duckdb: bind failed")
	ErrExec       = errors.New("duckdb: execute failed")
	ErrNullValue  = errors.New("duckdb: null value")
	ErrInvalidCol = errors.New("duckdb: invalid column index")
	ErrInvalidRow = errors.New("duckdb: invalid row index")
)

// Conn is a connection to a DuckDB database instance.
type Conn struct {
	db     dbHandle
	closed bool
}

// ConnResult is a helper struct for returning
// a Conn and an error from a function.
type ConnResult struct {
	val Conn
	err error
}

// Stmt is a prepared statement bound to a connection.
type Stmt struct {
	stmt   stmtHandle
	closed bool
}

// StmtResult is a helper struct for returning
// a Stmt and an error from a function.
type StmtResult struct {
	val Stmt
	err error
}

// Result owns a DuckDB query result.
type Result struct {
	res    resultHandle
	closed bool
}

// ResultResult is a helper struct for returning
// a Result and an error from a function.
type ResultResult struct {
	val Result
	err error
}

// Rows is a row iterator over a Result.
type Rows struct {
	result *Result
	row    int
}

// LibraryVersion returns duckdb_library_version() (e.g. "v1.5.2").
// The string is owned by DuckDB; do not free it.
func LibraryVersion() string {
	p := so_duckdb_library_version()
	if p == nil {
		return ""
	}
	return c.String(p)
}

// Open creates a new DuckDB database handle and connection from a path string.
// Use a file path for an on-disk database or ":memory:" for an in-memory database string path.
//
// For the same in-memory setup as the C examples using duckdb_open(NULL), use [OpenInMemory].
func Open(path string) (Conn, error) {
	var db dbHandle
	if so_duckdb_open(path, &db) != 0 {
		return Conn{}, ErrOpen
	}
	return Conn{db: db}, nil
}

// OpenInMemory opens an ephemeral database using duckdb_open(NULL) and duckdb_connect,
// matching the Startup & Shutdown C examples.
func OpenInMemory() (Conn, error) {
	var db dbHandle
	if so_duckdb_open_memory(&db) != 0 {
		return Conn{}, ErrOpen
	}
	return Conn{db: db}, nil
}

// Close closes the connection and frees underlying DuckDB resources.
func (c *Conn) Close() error {
	if c.closed {
		return nil
	}
	so_duckdb_close(&c.db)
	c.closed = true
	return nil
}

// Interrupt requests interruption of the current query on this connection (duckdb_interrupt).
func (c *Conn) Interrupt() {
	if c.closed {
		return
	}
	so_duckdb_interrupt(&c.db)
}

// Query executes SQL and returns a materialized result set.
//
// If the returned error is non-nil, the [Result] still wraps the DuckDB result
// object (including error details); the caller must call [Result.Close].
func (c *Conn) Query(query string) (Result, error) {
	if c.closed {
		return Result{}, ErrClosed
	}
	var res resultHandle
	rc := so_duckdb_query(&c.db, query, &res)
	out := Result{res: res}
	if rc != 0 {
		return out, ErrQuery
	}
	return out, nil
}

// Exec executes SQL and returns the number of changed rows.
func (c *Conn) Exec(query string) (int, error) {
	res, err := c.Query(query)
	if err != nil {
		_ = res.Close()
		return 0, err
	}
	n := res.RowsChanged()
	closeErr := res.Close()
	return n, closeErr
}

// ExecSQL runs SQL without retaining a result set (duckdb_query with a NULL result pointer).
// Use this for DDL or statements where row metadata is not needed.
//
// Errors return [ErrQuery] without an error message; use [Conn.Query] when diagnostics are required.
func (c *Conn) ExecSQL(sql string) error {
	if c.closed {
		return ErrClosed
	}
	if so_duckdb_query_void(&c.db, sql) != 0 {
		return ErrQuery
	}
	return nil
}

// Prepare creates a prepared statement.
func (c *Conn) Prepare(query string) (Stmt, error) {
	if c.closed {
		return Stmt{}, ErrClosed
	}
	var stmt stmtHandle
	if so_duckdb_prepare(&c.db, query, &stmt) != 0 {
		return Stmt{}, ErrPrepare
	}
	return Stmt{stmt: stmt}, nil
}

// Close releases prepared statement resources.
func (s *Stmt) Close() error {
	if s.closed {
		return nil
	}
	so_duckdb_stmt_close(&s.stmt)
	s.closed = true
	return nil
}

// ClearBindings clears all existing parameter bindings.
func (s *Stmt) ClearBindings() error {
	if s.closed {
		return ErrClosed
	}
	if so_duckdb_stmt_clear(&s.stmt) != 0 {
		return ErrBind
	}
	return nil
}

// BindNull binds NULL to parameter index (1-based).
func (s *Stmt) BindNull(index int) error {
	if s.closed {
		return ErrClosed
	}
	if so_duckdb_bind_null(&s.stmt, index) != 0 {
		return ErrBind
	}
	return nil
}

// BindBool binds a bool value to parameter index (1-based).
func (s *Stmt) BindBool(index int, value bool) error {
	if s.closed {
		return ErrClosed
	}
	if so_duckdb_bind_bool(&s.stmt, index, value) != 0 {
		return ErrBind
	}
	return nil
}

// BindInt64 binds an int64 value to parameter index (1-based).
func (s *Stmt) BindInt64(index int, value int64) error {
	if s.closed {
		return ErrClosed
	}
	if so_duckdb_bind_int64(&s.stmt, index, value) != 0 {
		return ErrBind
	}
	return nil
}

// BindFloat64 binds a float64 value to parameter index (1-based).
func (s *Stmt) BindFloat64(index int, value float64) error {
	if s.closed {
		return ErrClosed
	}
	if so_duckdb_bind_double(&s.stmt, index, value) != 0 {
		return ErrBind
	}
	return nil
}

// BindString binds a string value to parameter index (1-based).
func (s *Stmt) BindString(index int, value string) error {
	if s.closed {
		return ErrClosed
	}
	if so_duckdb_bind_varchar(&s.stmt, index, value) != 0 {
		return ErrBind
	}
	return nil
}

// Query executes the prepared statement and returns a result set.
//
// If the returned error is non-nil, the [Result] still must be closed.
func (s *Stmt) Query() (Result, error) {
	if s.closed {
		return Result{}, ErrClosed
	}
	var res resultHandle
	rc := so_duckdb_stmt_exec(&s.stmt, &res)
	out := Result{res: res}
	if rc != 0 {
		return out, ErrExec
	}
	return out, nil
}

// Exec executes the prepared statement and returns changed row count.
func (s *Stmt) Exec() (int, error) {
	res, err := s.Query()
	if err != nil {
		_ = res.Close()
		return 0, err
	}
	n := res.RowsChanged()
	closeErr := res.Close()
	return n, closeErr
}

// PrepareError returns the latest prepare error for this statement.
func (s *Stmt) PrepareError() string {
	msg := so_duckdb_prepare_error(&s.stmt)
	if msg == nil {
		return ""
	}
	return c.String(msg)
}

// Close releases the underlying result data.
func (r *Result) Close() error {
	if r.closed {
		return nil
	}
	so_duckdb_result_close(&r.res)
	r.closed = true
	return nil
}

// Error returns the result-level error message, if any.
//
// The pointer is owned by DuckDB; do not free it. It is valid until [Result.Close].
func (r *Result) Error() string {
	msg := so_duckdb_result_error(&r.res)
	if msg == nil {
		return ""
	}
	return c.String(msg)
}

// ErrorType returns the error classification when [Conn.Query] failed.
func (r *Result) ErrorType() ErrorType {
	if r.closed {
		return ErrorInvalid
	}
	return ErrorType(so_duckdb_result_error_type(&r.res))
}

// StatementType returns the statement type that produced this result.
func (r *Result) StatementType() StatementType {
	if r.closed {
		return StatementInvalid
	}
	return StatementType(so_duckdb_result_statement_type(&r.res))
}

// RowCount returns the number of rows in this result.
func (r *Result) RowCount() int {
	return so_duckdb_result_row_count(&r.res)
}

// RowsChanged returns number of rows changed by the statement.
func (r *Result) RowsChanged() int {
	return so_duckdb_result_rows_changed(&r.res)
}

// ColumnCount returns number of columns in this result.
func (r *Result) ColumnCount() int {
	return so_duckdb_result_column_count(&r.res)
}

// ColumnName returns a column name by index.
func (r *Result) ColumnName(col int) (string, error) {
	if col < 0 || col >= r.ColumnCount() {
		return "", ErrInvalidCol
	}
	name := so_duckdb_result_column_name(&r.res, col)
	if name == nil {
		return "", ErrInvalidCol
	}
	return c.String(name), nil
}

// ColumnType returns the physical SQL type of column col (see [ColType] constants).
func (r *Result) ColumnType(col int) (ColType, error) {
	if r.closed {
		return ColInvalid, ErrQuery
	}
	if col < 0 || col >= r.ColumnCount() {
		return ColInvalid, ErrInvalidCol
	}
	return ColType(so_duckdb_column_type(&r.res, col)), nil
}

// ColumnData returns duckdb_column_data: a pointer to columnar data for col.
// Layout depends on [Result.ColumnType] / [ColType]; see DuckDB C documentation.
func (r *Result) ColumnData(col int) any {
	if r.closed || col < 0 || col >= r.ColumnCount() {
		return nil
	}
	return so_duckdb_column_data(&r.res, col)
}

// NullmaskData returns duckdb_nullmask_data for col, or nil if unavailable.
func (r *Result) NullmaskData(col int) any {
	if r.closed || col < 0 || col >= r.ColumnCount() {
		return nil
	}
	return so_duckdb_nullmask_data(&r.res, col)
}

// ColumnLogicalType returns an opaque duckdb_logical_type handle for col (see DuckDB C docs).
// Release it with [DestroyLogicalType].
func (r *Result) ColumnLogicalType(col int) (any, error) {
	if r.closed {
		return nil, ErrQuery
	}
	if col < 0 || col >= r.ColumnCount() {
		return nil, ErrInvalidCol
	}
	h := so_duckdb_column_logical_type(&r.res, col)
	if h == nil {
		return nil, ErrInvalidCol
	}
	return h, nil
}

// DestroyLogicalType destroys a handle returned by [Result.ColumnLogicalType].
func DestroyLogicalType(lt any) {
	if lt == nil {
		return
	}
	so_duckdb_logical_type_destroy(lt)
}

// IsNull reports whether a value at (row, col) is null.
func (r *Result) IsNull(row int, col int) (bool, error) {
	if row < 0 || row >= r.RowCount() {
		return false, ErrInvalidRow
	}
	if col < 0 || col >= r.ColumnCount() {
		return false, ErrInvalidCol
	}
	return so_duckdb_value_is_null(&r.res, col, row), nil
}

// Bool returns a bool value from (row, col).
func (r *Result) Bool(row int, col int) (bool, error) {
	isNull, err := r.IsNull(row, col)
	if err != nil {
		return false, err
	}
	if isNull {
		return false, ErrNullValue
	}
	return so_duckdb_value_bool(&r.res, col, row), nil
}

// Int64 returns an int64 value from (row, col).
func (r *Result) Int64(row int, col int) (int64, error) {
	isNull, err := r.IsNull(row, col)
	if err != nil {
		return 0, err
	}
	if isNull {
		return 0, ErrNullValue
	}
	return so_duckdb_value_int64(&r.res, col, row), nil
}

// Float64 returns a float64 value from (row, col).
func (r *Result) Float64(row int, col int) (float64, error) {
	isNull, err := r.IsNull(row, col)
	if err != nil {
		return 0, err
	}
	if isNull {
		return 0, ErrNullValue
	}
	return so_duckdb_value_double(&r.res, col, row), nil
}

// StringCopy returns a heap-allocated copy of a string value from (row, col).
// The caller owns the returned string and must free it using mem.FreeString.
func (r *Result) StringCopy(a mem.Allocator, row int, col int) (string, error) {
	isNull, err := r.IsNull(row, col)
	if err != nil {
		return "", err
	}
	if isNull {
		return "", ErrNullValue
	}
	ptr := so_duckdb_value_string(&r.res, col, row)
	if ptr == nil {
		return "", ErrQuery
	}
	tmp := c.String(ptr)
	out := strings.Clone(a, tmp)
	so_duckdb_string_free(ptr)
	return out, nil
}

// Rows creates a row iterator over this result.
func (r *Result) Rows() Rows {
	return Rows{
		result: r,
		row:    -1,
	}
}

// Next advances to the next row and reports whether one exists.
func (rs *Rows) Next() bool {
	if rs.result == nil {
		return false
	}
	rs.row++
	return rs.row < rs.result.RowCount()
}

// Row returns the current row index.
func (rs *Rows) Row() int {
	return rs.row
}

// IsNull reports whether current row value at col is null.
func (rs *Rows) IsNull(col int) (bool, error) {
	if rs.result == nil {
		return false, ErrQuery
	}
	return rs.result.IsNull(rs.row, col)
}

// Bool reads a bool from current row at col.
func (rs *Rows) Bool(col int) (bool, error) {
	if rs.result == nil {
		return false, ErrQuery
	}
	return rs.result.Bool(rs.row, col)
}

// Int64 reads an int64 from current row at col.
func (rs *Rows) Int64(col int) (int64, error) {
	if rs.result == nil {
		return 0, ErrQuery
	}
	return rs.result.Int64(rs.row, col)
}

// Float64 reads a float64 from current row at col.
func (rs *Rows) Float64(col int) (float64, error) {
	if rs.result == nil {
		return 0, ErrQuery
	}
	return rs.result.Float64(rs.row, col)
}

// StringCopy reads a string from current row at col and clones it.
// The caller owns the returned string and must free it using mem.FreeString.
func (rs *Rows) StringCopy(a mem.Allocator, col int) (string, error) {
	if rs.result == nil {
		return "", ErrQuery
	}
	return rs.result.StringCopy(a, rs.row, col)
}
