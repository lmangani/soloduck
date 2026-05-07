// Soloduck is an interactive DuckDB shell implemented in Solod (So) and linked
// against libduckdb. It loosely follows the DuckDB CLI (see
// https://duckdb.org/docs/lts/clients/cli/overview ).
package main

import (
	"github.com/lmangani/soloduck/duckdb"
	"solod.dev/so/bufio"
	"solod.dev/so/flag"
	"solod.dev/so/io"
	"solod.dev/so/mem"
	"solod.dev/so/os"
	"solod.dev/so/strconv"
	"solod.dev/so/strings"
)

// Upper bound for displayed column width (must stay modest: dash rules are O(width) per segment).
const maxDuckboxColWidth = 256

// dashSegmentMax is the longest ASCII '-' run per cell border segment (width + inner padding).
const dashSegmentMax = maxDuckboxColWidth + 2

func clampDuckboxWidth(w int) int {
	if w < 0 {
		return 0
	}
	if w > maxDuckboxColWidth {
		return maxDuckboxColWidth
	}
	return w
}

// measureCellWidth estimates display width for table layout without trusting len(formatCell(...)):
// Solod/allocator string lengths can be wrong for some cells and previously blew column widths to the cap.
func measureCellWidth(alloc mem.Allocator, res *duckdb.Result, r, c int) int {
	nul, err := res.IsNull(r, c)
	if err != nil {
		return 1
	}
	if nul {
		return len("NULL")
	}
	typ, err := res.ColumnType(c)
	if err != nil {
		s, fr := formatCell(alloc, res, r, c)
		lw := len(s)
		if fr {
			mem.FreeString(alloc, s)
		}
		return clampDuckboxWidth(lw)
	}
	switch typ {
	case duckdb.ColBoolean:
		v, e := res.Bool(r, c)
		if e != nil {
			return 1
		}
		if v {
			return len("true")
		}
		return len("false")
	case duckdb.ColTinyInt, duckdb.ColSmallInt, duckdb.ColInteger, duckdb.ColBigInt,
		duckdb.ColUTinyInt, duckdb.ColUSmallInt, duckdb.ColUInteger, duckdb.ColUBigInt,
		duckdb.ColHugeInt, duckdb.ColUHugeInt, duckdb.ColIntegerLiteral:
		v, e := res.Int64(r, c)
		if e != nil {
			return 1
		}
		var buf [32]byte
		tmp := strconv.FormatInt(buf[:0], v, 10)
		return len(tmp)
	case duckdb.ColFloat, duckdb.ColDouble:
		v, e := res.Float64(r, c)
		if e != nil {
			return 1
		}
		var buf [64]byte
		tmp := strconv.FormatFloat(buf[:0], v, 'g', -1, 64)
		return clampDuckboxWidth(len(tmp))
	case duckdb.ColDecimal, duckdb.ColBigNum:
		s, fr := formatCell(alloc, res, r, c)
		lw := len(s)
		if fr {
			mem.FreeString(alloc, s)
		}
		return clampDuckboxWidth(lw)
	default:
		s, fr := formatCell(alloc, res, r, c)
		lw := len(s)
		if fr {
			mem.FreeString(alloc, s)
		}
		return clampDuckboxWidth(lw)
	}
}

// printASCIIHyphens prints ASCII '-' exactly n times (bounded). Do not use strings.Repeat for
// rules: the Solod/C Repeat implementation can stall or mis-size dashes for multi-column tables.
func printASCIIHyphens(n int) {
	if n <= 0 {
		return
	}
	if n > dashSegmentMax {
		n = dashSegmentMax
	}
	for i := 0; i < n; i++ {
		print("-")
	}
}

var sqlKeywords = []string{
	"ALTER", "AND", "AS", "ASC", "ATTACH", "BETWEEN", "BY", "CASE", "CAST", "COPY",
	"COUNT", "CREATE", "DELETE", "DESC", "DESCRIBE", "DISTINCT", "DROP", "EXPLAIN",
	"FALSE", "FROM", "FULL", "GROUP", "HAVING", "IN", "INNER", "INSERT", "INTO", "JOIN",
	"LEFT", "LIMIT", "NOT", "NULL", "ON", "OR", "ORDER", "OUTER", "PRAGMA", "SELECT",
	"SET", "SHOW", "SUM", "TABLE", "TRUE", "UNION", "UPDATE", "VALUES", "VIEW", "WHERE",
}

func main() {
	var (
		help     bool
		version  bool
		execSQL  string
		csvFlag  bool
		jsonFlag bool
		readonly bool
		batch    bool
	)
	fs := flag.NewFlagSet("soloduck", flag.ExitOnError)
	fs.BoolVar(&help, "help", false, "show help and exit")
	fs.BoolVar(&version, "version", false, "print DuckDB library version and exit")
	fs.StringVar(&execSQL, "c", "", "run SQL then exit (same as official duckdb -c)")
	fs.BoolVar(&csvFlag, "csv", false, "set output mode to csv (non-interactive / initial REPL mode)")
	fs.BoolVar(&jsonFlag, "json", false, "set output mode to json")
	fs.BoolVar(&readonly, "readonly", false, "request read-only open (not fully wired; emits a warning)")
	fs.BoolVar(&batch, "batch", false, "read SQL from stdin with no prompts (pipes and scripts)")
	_ = fs.Parse(os.Args[1:])

	if help {
		printUsage()
		os.Exit(0)
	}
	if version {
		println(duckdb.LibraryVersion())
		os.Exit(0)
	}
	if readonly {
		println("soloduck: warning: -readonly is not enforced yet (opens read-write).")
	}

	dbPath := ":memory:"
	if fs.NArg() >= 1 {
		dbPath = fs.Arg(0)
	}
	if len(execSQL) == 0 && fs.NArg() >= 2 {
		execSQL = fs.Arg(1)
	}

	db, err := duckdb.Open(dbPath)
	if err != nil {
		println("duckdb open failed")
		os.Exit(1)
	}
	defer db.Close()

	initialMode := "duckbox"
	if jsonFlag {
		initialMode = "json"
	} else if csvFlag {
		initialMode = "csv"
	}

	if len(execSQL) > 0 {
		runSQL(db, execSQL, initialMode)
		os.Exit(0)
	}

	if batch {
		if fs.NArg() >= 2 {
			println("soloduck: -batch allows at most one argument (database path)")
			os.Exit(1)
		}
		var alloc mem.Allocator
		script := readStdinAll(alloc)
		processScript(&db, script, &initialMode, alloc)
		mem.FreeString(alloc, script)
		os.Exit(0)
	}

	printBanner(dbPath)
	repl(&db, initialMode)
	os.Exit(0)
}

func printUsage() {
	println("Usage: soloduck [OPTIONS] [FILENAME] [SQL]")
	println("")
	println("Loosely matches the official DuckDB CLI; see:")
	println("  https://duckdb.org/docs/lts/clients/cli/overview")
	println("  https://duckdb.org/docs/lts/clients/cli/arguments")
	println("")
	println("Options:")
	println("  -help          show this help")
	println("  -version       print duckdb_library_version() and exit")
	println("  -c SQL         run SQL and exit (-s in official CLI; same idea)")
	println("  -csv           initial output mode: csv")
	println("  -json          initial output mode: json")
	println("  -batch          run SQL from stdin (no interactive prompt; use with pipes)")
	println("  -readonly      reserved (read-only open not implemented)")
	println("")
	println("With no FILENAME, connects to an in-memory database (:memory:).")
	println("Optional second argument SQL runs once and exits (non-interactive).")
	println("Otherwise starts an interactive SQL shell: type statements ending with `;`,")
	println("see SELECT results as boxed tables (duckbox). Use `.mode ascii` for plain | pipes.")
	println("Dot-commands: .help  .exit  .quit  .open [. --readonly] [PATH]  .read FILE")
	println("              .tables  .schema [TABLE]  .mode MODE  .complete [PREFIX]")
}

func printBanner(dbPath string) {
	println("DuckDB " + duckdb.LibraryVersion() + " (soloduck - Solod CLI demo)")
	println("Interactive SQL: statements end with `;`; SELECT results print as tables (duckbox).")
	println(`Enter ".help" for dot commands and .mode (csv, json, markdown, ascii, ...).`)
	if dbPath == ":memory:" {
		println("Connected to a transient in-memory database.")
		println(`Use ".open FILENAME" to reopen on a persistent database.`)
	} else {
		println("Connected to database at " + dbPath)
	}
}

// readStdinAll reads all of stdin for -batch (clone so builder buffers can be freed).
func readStdinAll(alloc mem.Allocator) string {
	br := bufio.NewReader(alloc, os.Stdin)
	defer br.Free()
	acc := strings.NewBuilder(alloc)
	for {
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			_, werr := acc.WriteString(line)
			if werr != nil {
				break
			}
		}
		mem.FreeString(alloc, line)
		if err == io.EOF {
			break
		}
		if err != nil {
			println("stdin read failed")
			break
		}
	}
	raw := acc.String()
	out := strings.Clone(alloc, raw)
	acc.Free()
	return out
}

func repl(db *duckdb.Conn, mode string) {
	var alloc mem.Allocator
	br := bufio.NewReader(alloc, os.Stdin)
	defer br.Free()

	acc := strings.NewBuilder(alloc)
	defer acc.Free()

	m := normalizeMode(mode)
	for {
		if acc.Len() == 0 {
			print("D ")
		} else {
			print("  ")
		}

		raw, rerr := br.ReadString('\n')
		if rerr != nil && rerr != io.EOF {
			mem.FreeString(alloc, raw)
			println("stdin read failed")
			os.Exit(1)
		}

		line := strings.TrimSuffix(strings.TrimSuffix(raw, "\n"), "\r")
		mem.FreeString(alloc, raw)

		if rerr == io.EOF && line == "" && acc.Len() == 0 {
			println()
			return
		}

		trim := strings.TrimSpace(line)
		if acc.Len() == 0 && handleDot(db, trim, &m, alloc) {
			continue
		}
		if acc.Len() == 0 && handleMeta(trim) {
			continue
		}

		if !writeLine(&acc, line) {
			os.Exit(1)
		}
		if strings.HasSuffix(trim, ";") {
			sql := strings.TrimSpace(acc.String())
			acc.Reset()
			runSQL(*db, sql, m)
		}
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

func handleDot(db *duckdb.Conn, line string, mode *string, alloc mem.Allocator) bool {
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
		runSQL(*db, "SHOW TABLES;", *mode)
	case ".schema":
		if len(arg) == 0 {
			runSQL(*db, "SHOW TABLES;", *mode)
		} else {
			runSQL(*db, "DESCRIBE "+arg+";", *mode)
		}
	case ".open":
		a := strings.TrimSpace(arg)
		ro := false
		for strings.HasPrefix(a, "--readonly") {
			ro = true
			a = strings.TrimSpace(strings.TrimPrefix(a, "--readonly"))
		}
		if ro {
			println("soloduck: .open --readonly not enforced; opening read-write.")
		}
		a = unquotePath(a)
		reopenDB(db, a)
	case ".read":
		if len(arg) == 0 {
			println("usage: .read FILE")
		} else {
			runReadFile(db, arg, mode, alloc)
		}
	case ".mode":
		if len(arg) == 0 {
			println("usage: .mode csv|json|markdown|duckbox|ascii|column|line|table")
			println("(duckbox = Unicode tables like the DuckDB CLI; ascii = plain | tables)")
			println("(see https://duckdb.org/docs/lts/clients/cli/output_formats )")
		} else {
			nm := normalizeMode(arg)
			if nm == "unknown" {
				println("unknown mode:", arg)
			} else {
				*mode = nm
				cur := *mode
				println("output mode:", cur)
			}
		}
	case ".complete":
		var a mem.Allocator
		completeSQL(a, arg)
	default:
		println("unknown command:", cmd, "(try .help)")
	}
	return true
}

func unquotePath(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

func reopenDB(db *duckdb.Conn, path string) {
	_ = db.Close()
	var n duckdb.Conn
	var err error
	if len(path) == 0 {
		n, err = duckdb.OpenInMemory()
	} else {
		n, err = duckdb.Open(path)
	}
	if err != nil {
		println("duckdb open failed")
		os.Exit(1)
	}
	*db = n
	if len(path) == 0 {
		println("Connected to a transient in-memory database.")
	} else {
		println("Connected to database at " + path)
	}
}

func runReadFile(db *duckdb.Conn, path string, mode *string, alloc mem.Allocator) {
	data, err := os.ReadFile(alloc, path)
	if err != nil {
		println("cannot read file")
		return
	}
	defer mem.FreeSlice(alloc, data)
	processScript(db, string(data), mode, alloc)
}

func processScript(db *duckdb.Conn, content string, mode *string, alloc mem.Allocator) {
	acc := strings.NewBuilder(alloc)
	defer acc.Free()
	start := 0
	for i := 0; i <= len(content); i++ {
		if i == len(content) || content[i] == '\n' {
			raw := content[start:i]
			start = i + 1
			line := strings.TrimSuffix(raw, "\r")
			trim := strings.TrimSpace(line)
			if acc.Len() == 0 && handleDot(db, trim, mode, alloc) {
				continue
			}
			if acc.Len() == 0 && handleMeta(trim) {
				continue
			}
			if len(trim) == 0 && acc.Len() == 0 {
				continue
			}
			if !writeLine(&acc, line) {
				return
			}
			if strings.HasSuffix(strings.TrimSpace(line), ";") {
				sql := strings.TrimSpace(acc.String())
				acc.Reset()
				runSQL(*db, sql, *mode)
			}
		}
	}
}

func normalizeMode(arg string) string {
	switch arg {
	case "csv", "json", "markdown", "duckbox", "column", "ascii":
		return arg
	}
	var alloc mem.Allocator
	lo := strings.ToLower(alloc, strings.TrimSpace(arg))
	defer mem.FreeString(alloc, lo)
	switch lo {
	case "csv":
		return "csv"
	case "json":
		return "json"
	case "markdown":
		return "markdown"
	case "md":
		return "markdown"
	case "duckbox", "box", "table":
		return "duckbox"
	case "ascii", "pipes":
		return "ascii"
	case "column", "line", "list":
		return "column"
	default:
		return "unknown"
	}
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
	println(`Dot commands (subset - see https://duckdb.org/docs/lts/clients/cli/dot_commands ): `)
	println(`  .help | .h`)
	println(`  .exit | .quit | .q`)
	println(`  .open [--readonly] [PATH]   (:memory: or empty PATH -> in-memory)`)
	println(`  .read FILE                   run FILE like interactive input`)
	println(`  .tables                      SHOW TABLES`)
	println(`  .schema [TABLE]              DESCRIBE TABLE (or list tables)`)
	println(`  .mode csv|json|markdown|duckbox|ascii|column|line|table`)
	println(`  .complete [PREFIX]           SQL keyword hints (not full CLI autocomplete)`)
	println("")
	println("Enter SQL ending with `;`. Without `;`, press Enter to continue on the next line.")
	println("Exit: Ctrl-D at an empty prompt, or .exit / .quit")
	println("Also: help | quit (without leading dot)")
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
		outMode := normalizeMode(mode)
		if outMode == "unknown" {
			outMode = "duckbox"
		}
		printResult(a, &res, outMode)
	} else {
		var nbuf [32]byte
		nch := res.RowsChanged()
		ns := strconv.FormatInt(nbuf[:0], int64(nch), 10)
		var a mem.Allocator
		ns2 := strings.Clone(a, ns)
		println("OK", ns2, "rows")
		mem.FreeString(a, ns2)
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
	switch mode {
	case "csv":
		printResultCSV(alloc, res, rows, cols)
	case "json":
		printResultJSON(alloc, res, rows, cols)
	case "markdown":
		printResultTable(alloc, res, rows, cols, true)
	case "column":
		printResultColumn(alloc, res, rows, cols)
	case "ascii":
		printResultTable(alloc, res, rows, cols, false)
	case "duckbox":
		printResultDuckbox(alloc, res, rows, cols)
	default:
		printResultDuckbox(alloc, res, rows, cols)
	}
}

func computeColumnWidths(alloc mem.Allocator, res *duckdb.Result, rows, cols int, widths []int) {
	for c := 0; c < cols; c++ {
		widths[c] = 0
	}
	for c := 0; c < cols; c++ {
		nm, err := res.ColumnName(c)
		if err != nil {
			nm = "?"
		}
		lw := len(previewStr(nm, maxDuckboxColWidth))
		if lw > widths[c] {
			widths[c] = lw
		}
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			lw := measureCellWidth(alloc, res, r, c)
			if lw > widths[c] {
				widths[c] = lw
			}
		}
	}
	for c := 0; c < cols; c++ {
		widths[c] = clampDuckboxWidth(widths[c])
	}
}

// computeDuckboxWidths includes column names, physical type labels, and cells (DuckDB CLI duckbox).
// widths must have length >= cols (caller allocates so backing storage survives for Solod/C stack slices).
func computeDuckboxWidths(alloc mem.Allocator, res *duckdb.Result, rows, cols int, widths []int) {
	for c := 0; c < cols; c++ {
		widths[c] = 0
	}
	for c := 0; c < cols; c++ {
		nm, err := res.ColumnName(c)
		if err != nil {
			nm = "?"
		}
		lw := len(previewStr(nm, maxDuckboxColWidth))
		if lw > widths[c] {
			widths[c] = lw
		}
		typ, err := res.ColumnType(c)
		tlab := "unknown"
		if err == nil {
			tlab = duckdb.PhysicalTypeLabel(typ)
		}
		if len(tlab) > widths[c] {
			widths[c] = len(tlab)
		}
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			lw := measureCellWidth(alloc, res, r, c)
			if lw > widths[c] {
				widths[c] = lw
			}
		}
	}
	for c := 0; c < cols; c++ {
		widths[c] = clampDuckboxWidth(widths[c])
	}
}

func duckboxPrintRuleTop(widths []int, cols int) {
	print("\xe2\x94\x8c")
	for c := 0; c < cols; c++ {
		if c > 0 {
			print("\xe2\x94\xac")
		}
		w := clampDuckboxWidth(widths[c])
		printASCIIHyphens(w + 2)
	}
	println("\xe2\x94\x90")
}

func duckboxPrintRuleSep(widths []int, cols int) {
	print("\xe2\x94\x9c")
	for c := 0; c < cols; c++ {
		if c > 0 {
			print("\xe2\x94\xbc")
		}
		w := clampDuckboxWidth(widths[c])
		printASCIIHyphens(w + 2)
	}
	println("\xe2\x94\xa4")
}

func duckboxPrintRuleBottom(widths []int, cols int) {
	print("\xe2\x94\x94")
	for c := 0; c < cols; c++ {
		if c > 0 {
			print("\xe2\x94\xb4")
		}
		w := clampDuckboxWidth(widths[c])
		printASCIIHyphens(w + 2)
	}
	println("\xe2\x94\x98")
}

func duckboxPadPrint(s string, w int, align byte) {
	w = clampDuckboxWidth(w)
	if len(s) > w {
		s = previewStr(s, w)
	}
	pad := w - len(s)
	if pad < 0 {
		pad = 0
	}
	if pad > maxDuckboxColWidth {
		pad = maxDuckboxColWidth
	}
	switch align {
	case 'r':
		for i := 0; i < pad; i++ {
			print(" ")
		}
		print(s)
	case 'c':
		left := pad / 2
		right := pad - left
		for i := 0; i < left; i++ {
			print(" ")
		}
		print(s)
		for i := 0; i < right; i++ {
			print(" ")
		}
	default:
		print(s)
		for i := 0; i < pad; i++ {
			print(" ")
		}
	}
}

func duckboxIsNumericType(t duckdb.ColType) bool {
	switch t {
	case duckdb.ColTinyInt, duckdb.ColSmallInt, duckdb.ColInteger, duckdb.ColBigInt,
		duckdb.ColUTinyInt, duckdb.ColUSmallInt, duckdb.ColUInteger, duckdb.ColUBigInt,
		duckdb.ColHugeInt, duckdb.ColUHugeInt, duckdb.ColIntegerLiteral,
		duckdb.ColFloat, duckdb.ColDouble, duckdb.ColDecimal, duckdb.ColBigNum:
		return true
	default:
		return false
	}
}

// printResultDuckbox matches DuckDB CLI duckbox: rule, names, centered types, sep, rows (nums right-aligned), bottom.
func printResultDuckbox(alloc mem.Allocator, res *duckdb.Result, rows, cols int) {
	widths := make([]int, cols)
	computeDuckboxWidths(alloc, res, rows, cols, widths)

	duckboxPrintRuleTop(widths, cols)

	print("\xe2\x94\x82")
	for c := 0; c < cols; c++ {
		print(" ")
		nm, err := res.ColumnName(c)
		if err != nil {
			nm = "?"
		}
		label := previewStr(nm, maxDuckboxColWidth)
		duckboxPadPrint(label, widths[c], 'l')
		print(" ")
		print("\xe2\x94\x82")
	}
	println()

	print("\xe2\x94\x82")
	for c := 0; c < cols; c++ {
		print(" ")
		typ, err := res.ColumnType(c)
		tlab := "unknown"
		if err == nil {
			tlab = duckdb.PhysicalTypeLabel(typ)
		}
		duckboxPadPrint(tlab, widths[c], 'c')
		print(" ")
		print("\xe2\x94\x82")
	}
	println()

	duckboxPrintRuleSep(widths, cols)

	for r := 0; r < rows; r++ {
		print("\xe2\x94\x82")
		for c := 0; c < cols; c++ {
			print(" ")
			s, fr := formatCell(alloc, res, r, c)
			typ, _ := res.ColumnType(c)
			al := byte('l')
			if duckboxIsNumericType(typ) {
				al = 'r'
			}
			cell := previewStr(s, widths[c])
			duckboxPadPrint(cell, widths[c], al)
			print(" ")
			print("\xe2\x94\x82")
			if fr {
				mem.FreeString(alloc, s)
			}
		}
		println()
	}

	duckboxPrintRuleBottom(widths, cols)
}

func printResultCSV(alloc mem.Allocator, res *duckdb.Result, rows, cols int) {
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
	for r := 0; r < rows; r++ {
		sep = ""
		for c := 0; c < cols; c++ {
			if c > 0 {
				print(sep)
				sep = ","
			}
			s, free := formatCell(alloc, res, r, c)
			print(csvEscape(s))
			if free {
				mem.FreeString(alloc, s)
			}
		}
		println()
	}
}

func printResultJSON(alloc mem.Allocator, res *duckdb.Result, rows, cols int) {
	println("[")
	for r := 0; r < rows; r++ {
		if r > 0 {
			println(",")
		}
		print("  {")
		for c := 0; c < cols; c++ {
			if c > 0 {
				print(", ")
			}
			nm, err := res.ColumnName(c)
			if err != nil {
				nm = "?"
			}
			print(`"`)
			print(jsonEscapeStr(nm))
			print(`":`)
			nul, _ := res.IsNull(r, c)
			if nul {
				print("null")
				continue
			}
			printJSONValue(alloc, res, r, c)
		}
		print("}")
	}
	println()
	println("]")
}

func printJSONValue(alloc mem.Allocator, res *duckdb.Result, r, c int) {
	typ, err := res.ColumnType(c)
	if err != nil {
		print(`""`)
		return
	}
	switch typ {
	case duckdb.ColBoolean:
		v, e := res.Bool(r, c)
		if e != nil {
			print("null")
			return
		}
		if v {
			print("true")
		} else {
			print("false")
		}
	case duckdb.ColTinyInt, duckdb.ColSmallInt, duckdb.ColInteger, duckdb.ColBigInt,
		duckdb.ColUTinyInt, duckdb.ColUSmallInt, duckdb.ColUInteger, duckdb.ColUBigInt,
		duckdb.ColHugeInt, duckdb.ColUHugeInt, duckdb.ColIntegerLiteral:
		v, e := res.Int64(r, c)
		if e != nil {
			print("null")
			return
		}
		var ibuf [32]byte
		is := strconv.FormatInt(ibuf[:0], v, 10)
		cl := strings.Clone(alloc, is)
		print(cl)
		mem.FreeString(alloc, cl)
	case duckdb.ColFloat, duckdb.ColDouble:
		v, e := res.Float64(r, c)
		if e != nil {
			print("null")
			return
		}
		var fbuf [64]byte
		fs := strconv.FormatFloat(fbuf[:0], v, 'g', -1, 64)
		cl := strings.Clone(alloc, fs)
		print(cl)
		mem.FreeString(alloc, cl)
	default:
		s, fr := formatCell(alloc, res, r, c)
		print(`"`)
		print(jsonEscapeStr(s))
		print(`"`)
		if fr {
			mem.FreeString(alloc, s)
		}
	}
}

func jsonEscapeStr(s string) string {
	if strings.IndexByte(s, '"') < 0 && strings.IndexByte(s, '\\') < 0 && strings.IndexByte(s, '\n') < 0 {
		return s
	}
	t := strings.ReplaceAll(nil, s, "\\", "\\\\")
	t = strings.ReplaceAll(nil, t, "\"", "\\\"")
	t = strings.ReplaceAll(nil, t, "\n", "\\n")
	return t
}

func printResultColumn(alloc mem.Allocator, res *duckdb.Result, rows, cols int) {
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
	sepLen := 8 * cols
	if sepLen > dashSegmentMax {
		sepLen = dashSegmentMax
	}
	for i := 0; i < sepLen; i++ {
		print("-")
	}
	println()

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if c > 0 {
				print("\t")
			}
			s, free := formatCell(alloc, res, r, c)
			print(s)
			if free {
				mem.FreeString(alloc, s)
			}
		}
		println()
	}
}

func printResultTable(alloc mem.Allocator, res *duckdb.Result, rows, cols int, markdown bool) {
	widths := make([]int, cols)
	computeColumnWidths(alloc, res, rows, cols, widths)

	for c := 0; c < cols; c++ {
		print("|")
		print(" ")
		nm, err := res.ColumnName(c)
		if err != nil {
			nm = "?"
		}
		label := previewStr(nm, maxDuckboxColWidth)
		print(label)
		for p := len(label); p < widths[c]; p++ {
			print(" ")
		}
		print(" ")
	}
	println("|")

	for c := 0; c < cols; c++ {
		print("|")
		nd := clampDuckboxWidth(widths[c]) + 2
		if nd > dashSegmentMax {
			nd = dashSegmentMax
		}
		if markdown {
			print("-")
			printASCIIHyphens(nd)
			print("-")
		} else {
			printASCIIHyphens(nd)
		}
	}
	println("|")

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			print("|")
			print(" ")
			s, fr := formatCell(alloc, res, r, c)
			disp := previewStr(s, widths[c])
			print(disp)
			for p := len(disp); p < widths[c]; p++ {
				print(" ")
			}
			print(" ")
			if fr {
				mem.FreeString(alloc, s)
			}
		}
		println("|")
	}
}

func previewStr(s string, max int) string {
	if max <= 0 {
		max = 1
	}
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
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
		duckdb.ColHugeInt, duckdb.ColUHugeInt, duckdb.ColIntegerLiteral:
		v, e := res.Int64(r, c)
		if e != nil {
			return "?", false
		}
		var buf [32]byte
		tmp := strconv.FormatInt(buf[:0], v, 10)
		out := strings.Clone(alloc, tmp)
		return out, true
	case duckdb.ColFloat, duckdb.ColDouble:
		v, e := res.Float64(r, c)
		if e != nil {
			return "?", false
		}
		var buf [64]byte
		tmp := strconv.FormatFloat(buf[:0], v, 'g', -1, 64)
		out := strings.Clone(alloc, tmp)
		return out, true
	case duckdb.ColVarchar, duckdb.ColBlob, duckdb.ColStringLiteral:
		out, e := res.StringCopy(alloc, r, c)
		if e != nil {
			return "?", false
		}
		const maxCell = 4096
		if len(out) > maxCell {
			mem.FreeString(alloc, out)
			out = strings.Clone(alloc, "<value too long>")
			return out, true
		}
		return out, true
	default:
		// DuckDB sometimes reports literal / numeric columns with types we do not map;
		// prefer typed accessors before varchar/string copy (avoids bogus huge strings).
		if v, e := res.Int64(r, c); e == nil {
			var buf [32]byte
			tmp := strconv.FormatInt(buf[:0], v, 10)
			out := strings.Clone(alloc, tmp)
			return out, true
		}
		if v, e := res.Float64(r, c); e == nil {
			var buf [64]byte
			tmp := strconv.FormatFloat(buf[:0], v, 'g', -1, 64)
			out := strings.Clone(alloc, tmp)
			return out, true
		}
		out, e := res.StringCopy(alloc, r, c)
		if e != nil {
			return "?", false
		}
		const maxCell = 4096
		if len(out) > maxCell {
			mem.FreeString(alloc, out)
			out = strings.Clone(alloc, "<value too long>")
			return out, true
		}
		return out, true
	}
}
