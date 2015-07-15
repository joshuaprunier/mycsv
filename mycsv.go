package main

import (
	"database/sql"
	"database/sql/driver"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/ssh/terminal"

	_ "github.com/go-sql-driver/mysql"
)

const (
	// Amount of CSV write data to buffer between flushes
	flushSize = 26214400 // 25MB

	// Timeout length where ctrl+c is ignored
	// If a second ctrl+c is sent before the timeout the program exits
	signalTimeout = 3 // Seconds

	// Timeout length to wait for a query string sent via stdin
	stdinTimeout = 10 // Milliseconds
)

type (
	// dbInfo contains information necessary to connect to a database
	dbInfo struct {
		user string
		pass string
		host string
		port string
		TLS  bool
	}

	// NullRawBytes represents sql.RawBytes which may be null
	NullRawBytes struct {
		RawBytes sql.RawBytes
		Valid    bool
	}
)

// ShowUsage prints a help screen
func showUsage() {
	fmt.Println(`
	mycsv usage:
	mycsv DB_COMMANDS [CSV OUTPUT FLAGS] [DEBUG FLAGS] [CSV OUTFILE] query

	EXAMPLES:
	mycsv -user=jprunier -p -file=my.csv -query="select * from jjp.example_table where filter in ('1', 'test', 'another')"
	echo "select * from mysql.plugins" | -user=jprunier -pmypass -hremotedb -tls > my.csv
	mycsv -user=jprunier -p -file=my.csv -d="|" -q="'" < queryfile

	DATABASE FLAGS
	==============
	-user: Username (required)
	-pass: Password (interactive prompt if blank)
	-host: Database Host (localhost assumed if blank)
	-port: Port (3306 default)
	-tls:  Use TLS/SSL for database connection (false default)

	CSV FLAGS
	=========
	-file: CSV output filename (Write to stdout if not supplied)
	-query: MySQL query (required, can be sent via stdin redirection)
	-header: Print initial column name header line (true default)
	-d: CSV field delimiter ("," default)
	-q: CSV quote character ("\"" default)
	-e: CSV escape character ("\\" default)
	-t: CSV line terminator ("\n" default)
	-v: Print more information (false default)

	DEBUG FLAGS
	===========
	-debug_cpu: CPU debugging filename
	-debug_mem: Memory debugging filename

	`)
}

func main() {
	start := time.Now()

	// Profiling flags
	var cpuprofile = flag.String("debug_cpu", "", "CPU debugging filename")
	var memprofile = flag.String("debug_mem", "", "Memory debugging filename")

	// Database flags
	dbUser := flag.String("user", "", "Username (required)")
	dbPass := flag.String("pass", "", "Password (interactive prompt if blank)")
	dbHost := flag.String("host", "", "Database Host (localhost assumed if blank)")
	dbPort := flag.String("port", "3306", "Port")
	dbTLS := flag.Bool("tls", false, "Use TLS/SSL for database connection")

	// CSV format flags
	csvDelimiter := flag.String("d", ",", "CSV field delimiter")
	csvQuote := flag.String("q", "\"", "CSV quote character")
	csvEscape := flag.String("e", "\\", "CSV escape character")
	csvTerminator := flag.String("t", "\n", "CSV line terminator")

	csvHeader := flag.Bool("header", true, "Print initial column name header line")
	csvFile := flag.String("file", "", "CSV output filename")
	sqlQuery := flag.String("query", "", "MySQL query")
	verbose := flag.Bool("v", false, "Print more information")

	// Parse flags
	flag.Parse()

	// Print usage
	if flag.NFlag() == 0 {
		showUsage()

		os.Exit(0)
	}

	// If query not provided read from standard in
	var query string
	queryChan := make(chan string)
	defer close(queryChan)
	if *sqlQuery == "" {
		go func() {
			b, err := ioutil.ReadAll(os.Stdin)
			checkErr(err)

			queryChan <- string(b)
		}()

		select {
		case q := <-queryChan:
			query = q
		case <-time.After(time.Millisecond * stdinTimeout):
			fmt.Fprintln(os.Stderr, "You must supply a query")
			os.Exit(1)
		}
	} else {
		query = *sqlQuery
	}

	// Make sure the query is a select
	if strings.ToLower(query[0:6]) != "select" {
		fmt.Fprintln(os.Stderr, "Query must be a select!")
		os.Exit(1)
	}

	// Check if an output file was supplied otherwise use standard out
	var writeTo string
	var writerDest io.Writer
	var err error
	if *csvFile == "" {
		writeTo = "standard out"
		writerDest = os.Stdout
	} else {
		f, err := os.Open(*csvFile)
		if err == nil {
			fmt.Fprintln(os.Stderr, *csvFile, "already exists!")
			fmt.Fprintln(os.Stderr, "Please remove it or use a different filename")
			os.Exit(1)
		}
		f.Close()
		writerDest, err = os.Create(*csvFile)
		writeTo = *csvFile
	}

	// Create a new CSV writer
	var i int
	CSVWriter := NewWriter(writerDest)
	CSVWriter.Delimiter, i = utf8.DecodeLastRuneInString(*csvDelimiter)
	if i == 0 {
		fmt.Fprintln(os.Stderr, "You must supply a valid delimiter character")
		os.Exit(1)
	}
	CSVWriter.Quote, i = utf8.DecodeLastRuneInString(*csvQuote)
	if i == 0 {
		fmt.Fprintln(os.Stderr, "You must supply a valid quote character")
		os.Exit(1)
	}
	CSVWriter.Escape, i = utf8.DecodeLastRuneInString(*csvEscape)
	if i == 0 {
		fmt.Fprintln(os.Stderr, "You must supply a valid escape character")
		os.Exit(1)
	}
	CSVWriter.Terminator = *csvTerminator

	if *verbose {
		fmt.Println("CSV output will be written to", writeTo)
	}

	// Check if Stdin has been redirected and reset so the user can be prompted for a password
	checkStdin()

	// Listen for SIGINT (ctrl+c)
	catchNotifications()

	// CPU Profiling
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		checkErr(err)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Default to localhost if no host or socket provided
	if *dbHost == "" {
		*dbHost = "127.0.0.1"
	}

	// Need to provide a target
	if *dbUser == "" {
		fmt.Fprintln(os.Stderr, "You must provide a user name!")
		os.Exit(1)
	}

	// If password is blank prompt user
	if *dbPass == "" {
		fmt.Println("Enter password: ")
		pwd, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			if err != io.EOF {
				checkErr(err)
			}
		}
		*dbPass = string(pwd)
	}

	// Populate dbInfo struct with cli flags
	dbi := dbInfo{user: *dbUser, pass: *dbPass, host: *dbHost, port: *dbPort, TLS: *dbTLS}

	// Create a *sql.DB connection to the source database
	db, err := dbi.Connect()
	defer db.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Start reading & writing
	dataChan := make(chan []NullRawBytes)
	quitChan := make(chan bool)
	goChan := make(chan bool)
	go readRows(db, query, dataChan, quitChan, goChan, *csvHeader)
	rowCount := writeCSV(CSVWriter, dataChan, goChan)

	<-quitChan
	close(quitChan)
	close(goChan)

	// Memory Profiling
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		checkErr(err)
		pprof.WriteHeapProfile(f)
		defer f.Close()
	}

	if *verbose {
		fmt.Println()
		fmt.Println(rowCount, "rows written")
		fmt.Println("Total runtime =", time.Since(start))
	}
}

// Pass the buck error catching
func checkErr(e error) {
	if e != nil {
		log.Panic(e)
	}
}

func catchNotifications() {
	state, err := terminal.GetState(int(os.Stdin.Fd()))
	checkErr(err)

	// Trap for SIGINT
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	var timer time.Time
	go func() {
		for sig := range sigChan {
			if time.Now().Sub(timer) < time.Second*signalTimeout {
				terminal.Restore(int(os.Stdin.Fd()), state)
				os.Exit(0)
			}

			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, sig, "signal caught!")
			fmt.Fprintf(os.Stderr, "Send signal again within %v seconds to exit\n", signalTimeout)
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "")

			timer = time.Now()
		}
	}()
}

// Create a database connection object
func (dbi *dbInfo) Connect() (*sql.DB, error) {
	var db *sql.DB
	var err error
	if dbi.TLS {
		db, err = sql.Open("mysql", dbi.user+":"+dbi.pass+"@tcp("+dbi.host+":"+dbi.port+")/?allowCleartextPasswords=1&tls=skip-verify")
	} else {
		db, err = sql.Open("mysql", dbi.user+":"+dbi.pass+"@tcp("+dbi.host+":"+dbi.port+")/?allowCleartextPasswords=1")
	}
	checkErr(err)

	// Ping database to verify credentials
	err = db.Ping()

	return db, err
}

// Scan impliments the Scanner interface
func (nb *NullRawBytes) Scan(value interface{}) error {
	if value == nil {
		nb.RawBytes, nb.Valid = []byte(string(0x0)), false
		return nil
	}

	switch v := value.(type) {
	case []byte:
		nb.RawBytes = v
	}
	nb.Valid = true

	return nil
}

// Value impliments the sql.driver Valuer interface
func (nb NullRawBytes) Value() (driver.Value, error) {
	if !nb.Valid {
		return nil, nil
	}
	return nb.RawBytes, nil
}

// readRows executes a query and returns the rows
func readRows(db *sql.DB, query string, dataChan chan []NullRawBytes, quitChan chan bool, goChan chan bool, csvHeader bool) {
	rows, err := db.Query(query)
	defer rows.Close()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	cols, err := rows.Columns()
	checkErr(err)

	if csvHeader {
		headers := make([]NullRawBytes, len(cols))
		for i, col := range cols {
			headers[i] = NullRawBytes{RawBytes: []byte(col), Valid: true}
		}
		dataChan <- headers
		<-goChan
	}

	// Need to scan into empty interface since we don't know how many columns or their types
	scanVals := make([]interface{}, len(cols))
	vals := make([]NullRawBytes, len(cols))
	cpy := make([]NullRawBytes, len(cols))
	for i := range vals {
		scanVals[i] = &vals[i]
	}

	for rows.Next() {
		err := rows.Scan(scanVals...)
		checkErr(err)

		copy(cpy, vals)
		dataChan <- cpy

		// Block until writeRows() signals it is safe to proceed
		// This is necessary because sql.RawBytes is a memory pointer and rows.Next() will loop and change the memory address before writeRows can properly process the values
		<-goChan
	}

	err = rows.Err()
	checkErr(err)

	close(dataChan)
	quitChan <- true
}

// writeCSV writes csv output
func writeCSV(w *Writer, dataChan chan []NullRawBytes, goChan chan bool) uint {
	var cnt uint
	// Range over row results from readRows()
	for data := range dataChan {
		// Write CSV
		size, err := w.Write(data)
		checkErr(err)

		cnt++

		// Flush CSV writer contents
		if size > flushSize {
			w.Flush()
			err = w.Error()
			checkErr(err)
		}

		// Allow read function to unblock and loop over rows
		goChan <- true
	}

	// Flush CSV writer contents
	w.Flush()
	err := w.Error()
	checkErr(err)

	return cnt
}
