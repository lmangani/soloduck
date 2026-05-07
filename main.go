// Soloduck is an interactive DuckDB shell implemented in Solod (So) and linked
// against libduckdb. It demonstrates solod.dev/so/duckdb in a small CLI.
package main

import (
	"solod.dev/so/bufio"
	"solod.dev/so/duckdb"
	"solod.dev/so/flag"
	"solod.dev/so/io"
	"solod.dev/so/mem"
	"solod.dev/so/os"
	"solod.dev/so/strconv"
	"solod.dev/so/strings"
)

var sqlKeywords = []string{
	"ALTER", "AND", "AS", "ASC", "ATTACH", "BETWEEN", "BY", "CASE", "CAST", "COPY",
	"COUNT", "CREATE", "DELETE", "DESC", "DESCRIBE", "DISTINCT", "DROP", "EXPLAIN",
	"FALSE", "FROM", "FULL", "GROUP", "HAVING", "IN", "INNER", "INSERT", "INTO", "JOIN",
	"LEFT", "LIMIT", "NOT", "NULL", "ON", "OR", "ORDER", "OUTER", "PRAGMA", "SELECT",
	"SET", "SHOW", "SUM", "TABLE", "TRUE", "UNION", "UPDATE", "VALUES", "VIEW", "WHERE",
}

func main() {
	var (
		dbPath  string
		execSQL string
		version bool
	)
	fs := flag.NewFlagSet("soloduck", flag.ExitOnError)
	fs.StringVar(&dbPath, "db", ":memory:", "database path (`:memory:` or file)")
	fs.StringVar(&execSQL, "e", "", "run SQL then exit (non-interactive)")
	fs.BoolVar(&version, "version", false, "print DuckDB library version and exit")
	_ = fs.Parse(os.Args[1:])
	if version {
		println(duckdb.LibraryVersion())
		os.Exit(0)
	}

	db, err := duckdb.Open(dbPath)
	if err != nil {
		println("duckdb open failed")
		os.Exit(1)
	}
	defer db.Close()

	if len(execSQL) > 0 {
		runSQL(db, execSQL, "line")
		os.Exit(0)
	}

	banner()
	repl(db)
	os.Exit(0)
}

func banner() {
	println("soloduck — DuckDB via Solod (libduckdb " + duckdb.LibraryVersion() + ")")
	println("Type SQL ending with `;`, or `help` / `.help`. Ctrl-D to exit.")
}

func repl(db duckdb.Conn) {
	var alloc mem.Allocator
	br := bufio.NewReader(alloc, os.Stdin)
	defer br.Free()

	acc := strings.NewBuilder(alloc)
	defer acc.Free()

	mode := "line"
	for {
		if acc.Len() == 0 {
			print("D ")
		} else {
			print("D ... ")
		}
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if acc.Len() == 0 {
					println()
					return
				}
				break
			}
			println("stdin read failed")
			mem.FreeString(alloc, line)
			os.Exit(1)
		}
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")
		trim := strings.TrimSpace(line)

		if acc.Len() == 0 && handleDot(db, trim, &mode) {
			continue
		}
		if acc.Len() == 0 && handleMeta(trim) {
			continue
		}

		if !writeLine(&acc, line) {
			os.Exit(1)
		}
		if strings.HasSuffix(strings.TrimSpace(line), ";") {
			sql := strings.TrimSpace(acc.String())
			acc.Reset()
			runSQL(db, sql, mode)
		}
		mem.FreeString(alloc, line)
	}
}

func writeLine(b *strings.Builder, line string) bool {
	if _, err := b.WriteString(line); err != nil {
		return false
	}
	_, err := b.WriteString("\n")
	return err == nil
}

func handleMeta(line string) bool {
	switch line {
	case "help", "HELP":
		printHelp()
		return true
	case "quit", "exit", "QUIT", "EXIT":
		os.Exit(0)
		return false
	case "":
		return true
	default:
		return false
	}
}

func handleDot(db duckdb.Conn, line string, mode *string) bool {
	if len(line) == 0 || line[0] != '.' {
		return false
	}
	cmd, arg := split2(line)
	switch cmd {
	case ".help", ".h":
		printHelp()
	case ".quit", ".exit", ".q":
		os.Exit(0)
		return true
	case ".tables":
		runSQL(db, "SHOW TABLES;", *mode)
	case ".schema":
		if len(arg) == 0 {
			runSQL(db, "SHOW TABLES;", *mode)
		} else {
			runSQL(db, "DESCRIBE "+arg+";", *mode)
		}
	case ".mode":
		if arg == "csv" || arg == "line" {
			*mode = arg
			println("output mode:", arg)
		} else {
			println("usage: .mode line|csv")
		}
	case ".complete":
		var a mem.Allocator
		completeSQL(a, arg)
	default:
		println("unknown command:", cmd, "(try .help)")
	}
	return true
}

func split2(line string) (string, string) {
	line = strings.TrimSpace(line)
	i := strings.IndexByte(line, ' ')
	if i < 0 {
		return line, ""
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
}

func printHelp() {
	println("Meta commands:")
	println("  help | .help       this text")
	println("  quit | .quit       exit")
	println("  .tables            SHOW TABLES")
	println("  .schema [table]     DESCRIBE (or show tables)")
	println("  .mode line|csv     result layout")
	println("  .complete [prefix] SQL keyword hints (lightweight autocomplete)")
	println("")
	println("Enter SQL ending with `;`. Use Ctrl-D to exit.")
}

func completeSQL(alloc mem.Allocator, prefix string) {
	prefix = strings.TrimSpace(prefix)
	pu := strings.ToUpper(alloc, prefix)
	defer mem.FreeString(alloc, pu)
	if len(pu) == 0 {
		println("(type a prefix after .complete, e.g. `.complete sel`)")
		return
	}
	var n int
	for _, k := range sqlKeywords {
		if strings.HasPrefix(k, pu) && k != pu {
			println(k)
			n++
		}
	}
	if n == 0 {
		println("(no keywords match)")
	}
}

func runSQL(db duckdb.Conn, sql string, mode string) {
	sql = strings.TrimSpace(sql)
	if len(sql) == 0 {
		return
	}
	res, qerr := db.Query(sql)
	if qerr != nil {
		msg := res.Error()
		if len(msg) > 0 {
			println(msg)
		} else {
			println("query failed")
		}
		_ = res.Close()
		return
	}

	st := res.StatementType()
	if st == duckdb.StatementSelect {
		var a mem.Allocator
		printResult(a, &res, mode)
	} else {
		var nbuf [32]byte
		nch := res.RowsChanged()
		println("OK", strconv.FormatInt(nbuf[:0], int64(nch), 10), "rows")
	}
	_ = res.Close()
}

func printResult(alloc mem.Allocator, res *duckdb.Result, mode string) {
	rows := res.RowCount()
	cols := res.ColumnCount()
	if cols == 0 {
		println("(no result columns)")
		return
	}

	if mode == "csv" {
		var sep string
		for c := 0; c < cols; c++ {
			if c > 0 {
				print(sep)
				sep = ","
			}
			name, err := res.ColumnName(c)
			if err != nil {
				print("?")
				continue
			}
			print(csvEscape(name))
		}
		println()
	} else {
		for c := 0; c < cols; c++ {
			if c > 0 {
				print("\t")
			}
			name, err := res.ColumnName(c)
			if err != nil {
				print("?")
				continue
			}
			print(name)
		}
		println()
		sep := strings.Repeat(alloc, "-", 8*cols)
		println(sep)
		mem.FreeString(alloc, sep)
	}

	for r := 0; r < rows; r++ {
		var sep2 string
		for c := 0; c < cols; c++ {
			if c > 0 {
				if mode == "csv" {
					print(sep2)
					sep2 = ","
				} else {
					print("\t")
				}
			}
			s, free := formatCell(alloc, res, r, c)
			if mode == "csv" {
				print(csvEscape(s))
			} else {
				print(s)
			}
			if free {
				mem.FreeString(alloc, s)
			}
		}
		println()
	}
}

func csvEscape(s string) string {
	if strings.IndexByte(s, ',') < 0 && strings.IndexByte(s, '"') < 0 && strings.IndexByte(s, '\n') < 0 {
		return s
	}
	t := strings.ReplaceAll(nil, s, `"`, `""`)
	return `"` + t + `"`
}

func formatCell(alloc mem.Allocator, res *duckdb.Result, r, c int) (string, bool) {
	n, err := res.IsNull(r, c)
	if err != nil {
		return "?", false
	}
	if n {
		return "NULL", false
	}
	typ, err := res.ColumnType(c)
	if err != nil {
		return "?", false
	}
	switch typ {
	case duckdb.ColBoolean:
		v, e := res.Bool(r, c)
		if e != nil {
			return "?", false
		}
		if v {
			return "true", false
		}
		return "false", false
	case duckdb.ColTinyInt, duckdb.ColSmallInt, duckdb.ColInteger, duckdb.ColBigInt,
		duckdb.ColUTinyInt, duckdb.ColUSmallInt, duckdb.ColUInteger, duckdb.ColUBigInt,
		duckdb.ColHugeInt, duckdb.ColUHugeInt:
		v, e := res.Int64(r, c)
		if e != nil {
			return "?", false
		}
		var buf [32]byte
		return strconv.FormatInt(buf[:0], v, 10), false
	case duckdb.ColFloat, duckdb.ColDouble:
		v, e := res.Float64(r, c)
		if e != nil {
			return "?", false
		}
		var buf [64]byte
		return strconv.FormatFloat(buf[:0], v, 'g', -1, 64), false
	default:
		out, e := res.StringCopy(alloc, r, c)
		if e != nil {
			return "?", false
		}
		return out, true
	}
}
