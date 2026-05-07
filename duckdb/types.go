package duckdb

// ColType is the physical SQL column type (duckdb_type / DUCKDB_TYPE_*).
// Named ColType (not Type) so translated C does not emit duckdb_Type, which clashes with DuckDB's duckdb_type.
//
// See https://duckdb.org/docs/current/clients/c/types
type ColType int32

// ColTypeResult is a helper struct for returning a ColType and an error from a function.
type ColTypeResult struct {
	val ColType
	err error
}

const (
	ColInvalid        ColType = 0
	ColBoolean        ColType = 1
	ColTinyInt        ColType = 2
	ColSmallInt       ColType = 3
	ColInteger        ColType = 4
	ColBigInt         ColType = 5
	ColUTinyInt       ColType = 6
	ColUSmallInt      ColType = 7
	ColUInteger       ColType = 8
	ColUBigInt        ColType = 9
	ColFloat          ColType = 10
	ColDouble         ColType = 11
	ColTimestamp      ColType = 12
	ColDate           ColType = 13
	ColTime           ColType = 14
	ColInterval       ColType = 15
	ColHugeInt        ColType = 16
	ColUHugeInt       ColType = 32
	ColVarchar        ColType = 17
	ColBlob           ColType = 18
	ColDecimal        ColType = 19
	ColTimestampS     ColType = 20
	ColTimestampMS    ColType = 21
	ColTimestampNS    ColType = 22
	ColEnum           ColType = 23
	ColList           ColType = 24
	ColStruct         ColType = 25
	ColMap            ColType = 26
	ColUUID           ColType = 27
	ColUnion          ColType = 28
	ColBit            ColType = 29
	ColTimeTZ         ColType = 30
	ColTimestampTZ    ColType = 31
	ColArray          ColType = 33
	ColAny            ColType = 34
	ColBigNum         ColType = 35
	ColSQLNull        ColType = 36
	ColStringLiteral  ColType = 37
	ColIntegerLiteral ColType = 38
	ColTimeNS         ColType = 39
	ColGeometry       ColType = 40
)

// StatementType identifies the kind of SQL statement that produced a result.
type StatementType int32

const (
	StatementInvalid     StatementType = 0
	StatementSelect      StatementType = 1
	StatementInsert      StatementType = 2
	StatementUpdate      StatementType = 3
	StatementExplain     StatementType = 4
	StatementDelete      StatementType = 5
	StatementPrepare     StatementType = 6
	StatementCreate      StatementType = 7
	StatementExecute     StatementType = 8
	StatementAlter       StatementType = 9
	StatementTransaction StatementType = 10
	StatementCopy        StatementType = 11
	StatementAnalyze     StatementType = 12
	StatementVariableSet StatementType = 13
	StatementCreateFunc  StatementType = 14
	StatementDrop        StatementType = 15
	StatementExport      StatementType = 16
	StatementPragma      StatementType = 17
	StatementVacuum      StatementType = 18
	StatementCall        StatementType = 19
	StatementSet         StatementType = 20
	StatementLoad        StatementType = 21
	StatementRelation    StatementType = 22
	StatementExtension   StatementType = 23
	StatementLogicalPlan StatementType = 24
	StatementAttach      StatementType = 25
	StatementDetach      StatementType = 26
	StatementMulti       StatementType = 27
)

// ErrorType classifies errors attached to a failed duckdb_query result.
type ErrorType int32

const (
	ErrorInvalid              ErrorType = 0
	ErrorOutOfRange           ErrorType = 1
	ErrorConversion           ErrorType = 2
	ErrorUnknownType          ErrorType = 3
	ErrorDecimal              ErrorType = 4
	ErrorMismatchType         ErrorType = 5
	ErrorDivideByZero         ErrorType = 6
	ErrorObjectSize           ErrorType = 7
	ErrorInvalidType          ErrorType = 8
	ErrorSerialization        ErrorType = 9
	ErrorTransaction          ErrorType = 10
	ErrorNotImplemented       ErrorType = 11
	ErrorExpression           ErrorType = 12
	ErrorCatalog              ErrorType = 13
	ErrorParser               ErrorType = 14
	ErrorPlanner              ErrorType = 15
	ErrorScheduler            ErrorType = 16
	ErrorExecutor             ErrorType = 17
	ErrorConstraint           ErrorType = 18
	ErrorIndex                ErrorType = 19
	ErrorStat                 ErrorType = 20
	ErrorConnection           ErrorType = 21
	ErrorSyntax               ErrorType = 22
	ErrorSettings             ErrorType = 23
	ErrorBinder               ErrorType = 24
	ErrorNetwork              ErrorType = 25
	ErrorOptimizer            ErrorType = 26
	ErrorNullPointer          ErrorType = 27
	ErrorIO                   ErrorType = 28
	ErrorInterrupt            ErrorType = 29
	ErrorFatal                ErrorType = 30
	ErrorInternal             ErrorType = 31
	ErrorInvalidInput         ErrorType = 32
	ErrorOutOfMemory          ErrorType = 33
	ErrorPermission           ErrorType = 34
	ErrorParameterNotResolved ErrorType = 35
	ErrorParameterNotAllowed  ErrorType = 36
	ErrorDependency           ErrorType = 37
	ErrorHTTP                 ErrorType = 38
	ErrorMissingExtension     ErrorType = 39
	ErrorAutoload             ErrorType = 40
	ErrorSequence             ErrorType = 41
	ErrorInvalidConfiguration ErrorType = 42
)
