package duckdb

import "solod.dev/so/c"

//so:embed duckdb.h
var duckdb_h string

//so:embed duckdb.c
var duckdb_c string

//so:extern so_duckdb_db
type dbHandle struct{}

//so:extern so_duckdb_stmt
type stmtHandle struct{}

//so:extern so_duckdb_result
type resultHandle struct{}

//so:extern
func so_duckdb_open(path string, out *dbHandle) int {
	_, _ = path, out
	return 0
}

//so:extern
func so_duckdb_open_memory(out *dbHandle) int {
	_ = out
	return 0
}

//so:extern
func so_duckdb_close(db *dbHandle) {
	_ = db
}

//so:extern
func so_duckdb_interrupt(db *dbHandle) {
	_ = db
}

//so:extern
func so_duckdb_library_version() *c.ConstChar {
	return nil
}

//so:extern
func so_duckdb_query(db *dbHandle, query string, out *resultHandle) int {
	_, _, _ = db, query, out
	return 0
}

//so:extern
func so_duckdb_query_void(db *dbHandle, query string) int {
	_, _ = db, query
	return 0
}

//so:extern
func so_duckdb_prepare(db *dbHandle, query string, out *stmtHandle) int {
	_, _, _ = db, query, out
	return 0
}

//so:extern
func so_duckdb_prepare_error(stmt *stmtHandle) *c.ConstChar {
	_ = stmt
	return nil
}

//so:extern
func so_duckdb_stmt_close(stmt *stmtHandle) {
	_ = stmt
}

//so:extern
func so_duckdb_stmt_clear(stmt *stmtHandle) int {
	_ = stmt
	return 0
}

//so:extern
func so_duckdb_bind_null(stmt *stmtHandle, index int) int {
	_, _ = stmt, index
	return 0
}

//so:extern
func so_duckdb_bind_bool(stmt *stmtHandle, index int, value bool) int {
	_, _, _ = stmt, index, value
	return 0
}

//so:extern
func so_duckdb_bind_int64(stmt *stmtHandle, index int, value int64) int {
	_, _, _ = stmt, index, value
	return 0
}

//so:extern
func so_duckdb_bind_double(stmt *stmtHandle, index int, value float64) int {
	_, _, _ = stmt, index, value
	return 0
}

//so:extern
func so_duckdb_bind_varchar(stmt *stmtHandle, index int, value string) int {
	_, _, _ = stmt, index, value
	return 0
}

//so:extern
func so_duckdb_stmt_exec(stmt *stmtHandle, out *resultHandle) int {
	_, _ = stmt, out
	return 0
}

//so:extern
func so_duckdb_result_close(res *resultHandle) {
	_ = res
}

//so:extern
func so_duckdb_result_error(res *resultHandle) *c.ConstChar {
	_ = res
	return nil
}

//so:extern
func so_duckdb_result_error_type(res *resultHandle) int32 {
	_ = res
	return 0
}

//so:extern
func so_duckdb_result_statement_type(res *resultHandle) int32 {
	_ = res
	return 0
}

//so:extern
func so_duckdb_result_row_count(res *resultHandle) int {
	_ = res
	return 0
}

//so:extern
func so_duckdb_result_rows_changed(res *resultHandle) int {
	_ = res
	return 0
}

//so:extern
func so_duckdb_result_column_count(res *resultHandle) int {
	_ = res
	return 0
}

//so:extern
func so_duckdb_result_column_name(res *resultHandle, col int) *c.ConstChar {
	_, _ = res, col
	return nil
}

//so:extern
func so_duckdb_column_type(res *resultHandle, col int) int32 {
	_, _ = res, col
	return 0
}

//so:extern
func so_duckdb_column_data(res *resultHandle, col int) any {
	_, _ = res, col
	return nil
}

//so:extern
func so_duckdb_nullmask_data(res *resultHandle, col int) any {
	_, _ = res, col
	return nil
}

//so:extern
func so_duckdb_column_logical_type(res *resultHandle, col int) any {
	_, _ = res, col
	return nil
}

//so:extern
func so_duckdb_logical_type_destroy(lt any) {
	_ = lt
}

//so:extern
func so_duckdb_value_is_null(res *resultHandle, col int, row int) bool {
	_, _, _ = res, col, row
	return false
}

//so:extern
func so_duckdb_value_bool(res *resultHandle, col int, row int) bool {
	_, _, _ = res, col, row
	return false
}

//so:extern
func so_duckdb_value_int64(res *resultHandle, col int, row int) int64 {
	_, _, _ = res, col, row
	return 0
}

//so:extern
func so_duckdb_value_double(res *resultHandle, col int, row int) float64 {
	_, _, _ = res, col, row
	return 0
}

//so:extern
func so_duckdb_value_string(res *resultHandle, col int, row int) *c.Char {
	_, _, _ = res, col, row
	return nil
}

//so:extern
func so_duckdb_string_free(ptr *c.Char) {
	_ = ptr
}
