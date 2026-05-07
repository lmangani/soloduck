//go:build ignore
#include "duckdb.h"

static int so_duckdb_open_impl(const char* path, so_duckdb_db* out) {
    if (!out) {
        return 1;
    }
    out->db = NULL;
    out->conn = NULL;
    out->open = false;

    if (duckdb_open(path, &out->db) != DuckDBSuccess) {
        return 1;
    }
    if (duckdb_connect(out->db, &out->conn) != DuckDBSuccess) {
        duckdb_close(&out->db);
        return 1;
    }
    out->open = true;
    return 0;
}

int so_duckdb_open(const char* path, so_duckdb_db* out) {
    return so_duckdb_open_impl(path, out);
}

int so_duckdb_open_memory(so_duckdb_db* out) {
    return so_duckdb_open_impl(NULL, out);
}

void so_duckdb_close(so_duckdb_db* db) {
    if (!db || !db->open) {
        return;
    }
    duckdb_disconnect(&db->conn);
    duckdb_close(&db->db);
    db->open = false;
}

void so_duckdb_interrupt(so_duckdb_db* db) {
    if (!db || !db->open) {
        return;
    }
    duckdb_interrupt(db->conn);
}

const char* so_duckdb_library_version(void) {
    return duckdb_library_version();
}

int so_duckdb_query(so_duckdb_db* db, const char* query, so_duckdb_result* out) {
    if (!db || !db->open || !out) {
        return 1;
    }
    duckdb_state st = duckdb_query(db->conn, query, &out->result);
    out->open = true;
    return st == DuckDBSuccess ? 0 : 1;
}

int so_duckdb_query_void(so_duckdb_db* db, const char* query) {
    if (!db || !db->open) {
        return 1;
    }
    return duckdb_query(db->conn, query, NULL) == DuckDBSuccess ? 0 : 1;
}

int so_duckdb_prepare(so_duckdb_db* db, const char* query, so_duckdb_stmt* out) {
    if (!db || !db->open || !out) {
        return 1;
    }
    out->stmt = NULL;
    out->open = false;
    if (duckdb_prepare(db->conn, query, &out->stmt) != DuckDBSuccess) {
        return 1;
    }
    out->open = true;
    return 0;
}

const char* so_duckdb_prepare_error(so_duckdb_stmt* stmt) {
    if (!stmt || !stmt->open) {
        return NULL;
    }
    return duckdb_prepare_error(stmt->stmt);
}

void so_duckdb_stmt_close(so_duckdb_stmt* stmt) {
    if (!stmt || !stmt->open) {
        return;
    }
    duckdb_destroy_prepare(&stmt->stmt);
    stmt->open = false;
}

int so_duckdb_stmt_clear(so_duckdb_stmt* stmt) {
    if (!stmt || !stmt->open) {
        return 1;
    }
    return duckdb_clear_bindings(stmt->stmt) == DuckDBSuccess ? 0 : 1;
}

int so_duckdb_bind_null(so_duckdb_stmt* stmt, int index) {
    if (!stmt || !stmt->open || index <= 0) {
        return 1;
    }
    return duckdb_bind_null(stmt->stmt, (idx_t)index) == DuckDBSuccess ? 0 : 1;
}

int so_duckdb_bind_bool(so_duckdb_stmt* stmt, int index, bool value) {
    if (!stmt || !stmt->open || index <= 0) {
        return 1;
    }
    return duckdb_bind_boolean(stmt->stmt, (idx_t)index, value) == DuckDBSuccess ? 0 : 1;
}

int so_duckdb_bind_int64(so_duckdb_stmt* stmt, int index, int64_t value) {
    if (!stmt || !stmt->open || index <= 0) {
        return 1;
    }
    return duckdb_bind_int64(stmt->stmt, (idx_t)index, value) == DuckDBSuccess ? 0 : 1;
}

int so_duckdb_bind_double(so_duckdb_stmt* stmt, int index, double value) {
    if (!stmt || !stmt->open || index <= 0) {
        return 1;
    }
    return duckdb_bind_double(stmt->stmt, (idx_t)index, value) == DuckDBSuccess ? 0 : 1;
}

int so_duckdb_bind_varchar(so_duckdb_stmt* stmt, int index, const char* value) {
    if (!stmt || !stmt->open || index <= 0) {
        return 1;
    }
    return duckdb_bind_varchar(stmt->stmt, (idx_t)index, value) == DuckDBSuccess ? 0 : 1;
}

int so_duckdb_stmt_exec(so_duckdb_stmt* stmt, so_duckdb_result* out) {
    if (!stmt || !stmt->open || !out) {
        return 1;
    }
    duckdb_state st = duckdb_execute_prepared(stmt->stmt, &out->result);
    out->open = true;
    return st == DuckDBSuccess ? 0 : 1;
}

void so_duckdb_result_close(so_duckdb_result* res) {
    if (!res || !res->open) {
        return;
    }
    duckdb_destroy_result(&res->result);
    res->open = false;
}

const char* so_duckdb_result_error(so_duckdb_result* res) {
    if (!res || !res->open) {
        return NULL;
    }
    return duckdb_result_error(&res->result);
}

int32_t so_duckdb_result_error_type(so_duckdb_result* res) {
    if (!res || !res->open) {
        return 0;
    }
    return (int32_t)duckdb_result_error_type(&res->result);
}

int32_t so_duckdb_result_statement_type(so_duckdb_result* res) {
    if (!res || !res->open) {
        return 0;
    }
    return (int32_t)duckdb_result_statement_type(res->result);
}

int so_duckdb_result_row_count(so_duckdb_result* res) {
    if (!res || !res->open) {
        return 0;
    }
    return (int)duckdb_row_count(&res->result);
}

int so_duckdb_result_rows_changed(so_duckdb_result* res) {
    if (!res || !res->open) {
        return 0;
    }
    return (int)duckdb_rows_changed(&res->result);
}

int so_duckdb_result_column_count(so_duckdb_result* res) {
    if (!res || !res->open) {
        return 0;
    }
    return (int)duckdb_column_count(&res->result);
}

const char* so_duckdb_result_column_name(so_duckdb_result* res, int col) {
    if (!res || !res->open || col < 0) {
        return NULL;
    }
    return duckdb_column_name(&res->result, (idx_t)col);
}

int32_t so_duckdb_column_type(so_duckdb_result* res, int col) {
    if (!res || !res->open || col < 0) {
        return (int32_t)DUCKDB_TYPE_INVALID;
    }
    return (int32_t)duckdb_column_type(&res->result, (idx_t)col);
}

void* so_duckdb_column_data(so_duckdb_result* res, int col) {
    if (!res || !res->open || col < 0) {
        return NULL;
    }
    return duckdb_column_data(&res->result, (idx_t)col);
}

bool* so_duckdb_nullmask_data(so_duckdb_result* res, int col) {
    if (!res || !res->open || col < 0) {
        return NULL;
    }
    return duckdb_nullmask_data(&res->result, (idx_t)col);
}

duckdb_logical_type so_duckdb_column_logical_type(so_duckdb_result* res, int col) {
    if (!res || !res->open || col < 0) {
        return NULL;
    }
    return duckdb_column_logical_type(&res->result, (idx_t)col);
}

void so_duckdb_logical_type_destroy(duckdb_logical_type lt) {
    duckdb_logical_type local = lt;
    duckdb_destroy_logical_type(&local);
}

bool so_duckdb_value_is_null(so_duckdb_result* res, int col, int row) {
    if (!res || !res->open || col < 0 || row < 0) {
        return true;
    }
    return duckdb_value_is_null(&res->result, (idx_t)col, (idx_t)row);
}

bool so_duckdb_value_bool(so_duckdb_result* res, int col, int row) {
    return duckdb_value_boolean(&res->result, (idx_t)col, (idx_t)row);
}

int64_t so_duckdb_value_int64(so_duckdb_result* res, int col, int row) {
    return duckdb_value_int64(&res->result, (idx_t)col, (idx_t)row);
}

double so_duckdb_value_double(so_duckdb_result* res, int col, int row) {
    return duckdb_value_double(&res->result, (idx_t)col, (idx_t)row);
}

char* so_duckdb_value_string(so_duckdb_result* res, int col, int row) {
    return duckdb_value_varchar(&res->result, (idx_t)col, (idx_t)row);
}

void so_duckdb_string_free(char* ptr) {
    if (ptr) {
        duckdb_free(ptr);
    }
}
