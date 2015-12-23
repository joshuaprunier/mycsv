package main

import (
	"database/sql"
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

	"golang.org/x/crypto/ssh/terminal"

	_ "github.com/go-sql-driver/mysql"
)

const (
	// Amount of CSV write data to buffer between flushes.
	flushBufferSize = 26214400 // 25MB

	// Timeout length where ctrl+c is ignored.
	signalTimeout = 3 // Seconds

	// Timeout length to wait for a query string sent via stdin.
	stdinTimeout = 10 // Milliseconds
)

type (
	// dbInfo contains information necessary to connect to a database
	dbInfo struct {
		user    string
		pass    string
		host    string
		port    string
		charset string
		tls     bool
	}
)

// Version information supplied by build script
var versionInformation string

// ShowUsage prints a help screen
func showUsage() {
	fmt.Printf("\tmycsv version %s\n", versionInformation)
	fmt.Println(`
	USAGE:
	mycsv DB_COMMANDS [CSV OUTPUT FLAGS] [DEBUG FLAGS] [CSV OUTFILE] query

	EXAMPLES:
	mycsv -user=jprunier -pass= -file=my.csv -charset=utf8 -query="select * from jjp.example_table where filter in ('1', 'test', 'another')"
	echo "select * from mysql.plugin" | mycsv -user=jprunier -pass=mypass -host=remotedb > my.csv
	mycsv -user=jprunier -pass= -file=my.csv -d="|" -q="'" < queryfile

	DATABASE FLAGS
	==============
	-user: Database Username (required)
	-pass: Database Password (interactive prompt if blank)
	-host: Database Host (localhost assumed if blank)
	-port: Database Port (3306 default)
	-charset: Database character set (binary default)
	-tls: Use TLS, also enables cleartext passwords (default false)


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
	-version: Version information

	`)
}

func main() {
	start := time.Now()

	// Database flags
	dbUser := flag.String("user", "", "Database Username (required)")
	dbPass := flag.String("pass", "", "Database Password (interactive prompt if blank)")
	dbHost := flag.String("host", "", "Database Host (localhost assumed if blank)")
	dbPort := flag.String("port", "3306", "Database Port")
	dbCharset := flag.String("charset", "binary", "Database character set")
	dbTLS := flag.Bool("tls", false, "Enable TLS & cleartext passwords")

	// CSV formatting flags
	csvFile := flag.String("file", "", "CSV output filename")
	csvQuery := flag.String("query", "", "MySQL query")
	csvHeader := flag.Bool("header", true, "Print initial column name header line")
	csvDelimiter := flag.String("d", `,`, "CSV field delimiter")
	csvQuote := flag.String("q", `"`, "CSV quote character")
	csvEscape := flag.String("e", `\`, "CSV escape character")
	csvTerminator := flag.String("t", "\n", "CSV line terminator")
	verbose := flag.Bool("v", false, "Print more information")

	// Debug flags
	cpuprofile := flag.String("debug_cpu", "", "CPU debugging filename")
	memprofile := flag.String("debug_mem", "", "Memory debugging filename")
	version := flag.Bool("version", false, "Version information")

	// Override default help
	help := flag.Bool("help", false, "Show usage")
	h := flag.Bool("h", false, "Show usage")

	// Parse flags
	flag.Parse()

	// Print usage
	if flag.NFlag() == 0 || *help == true || *h == true {
		showUsage()

		os.Exit(0)
	}

	if *version {
		fmt.Printf("mycsv version %s\n", versionInformation)
		os.Exit(0)
	}

	// If query not provided read from standard in
	var query string
	queryChan := make(chan string)
	defer close(queryChan)
	if *csvQuery == "" {
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
		query = *csvQuery
	}

	// Make sure the query is a select
	if strings.ToLower(query[0:6]) != "select" {
		fmt.Fprintln(os.Stderr, "Query must be a select!")
		os.Exit(1)
	}

	// Create CSV output file if supplied, otherwise use standard out
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
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		writeTo = *csvFile
	}

	// Create a new CSV writer
	CSVWriter := NewWriter(writerDest)
	if *csvDelimiter == `\t` {
		CSVWriter.Delimiter = "\t"
	} else {
		CSVWriter.Delimiter = *csvDelimiter
	}
	CSVWriter.Quote = *csvQuote
	CSVWriter.Escape = *csvEscape

	// Need literal string check here to see all 4 bytes instead of 2 (ascii 13 & 10)
	// Newline is default but check here in case it is manually passed in
	if *csvTerminator == `\r\n` {
		CSVWriter.Terminator = "\r\n"
	} else if *csvTerminator == `\n` {
		CSVWriter.Terminator = "\n"
	} else {
		CSVWriter.Terminator = *csvTerminator
	}

	if *verbose {
		fmt.Println("CSV output will be written to", writeTo)
	}

	// Check if Stdin has been redirected and reset so the user can be prompted for a password
	checkStdin()

	// Catch signals
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

	// Populate dbInfo struct with flag values
	dbi := dbInfo{user: *dbUser, pass: *dbPass, host: *dbHost, port: *dbPort, charset: *dbCharset, tls: *dbTLS}

	// Create a *sql.DB connection to the source database
	db, err := dbi.connect()
	defer db.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Create channels
	dataChan := make(chan []sql.RawBytes)
	quitChan := make(chan bool)
	goChan := make(chan bool)

	// Start reading & writing
	go readRows(db, query, dataChan, quitChan, goChan, *csvHeader)
	rowCount := writeCSV(CSVWriter, dataChan, goChan, *verbose)

	// Block on quitChan until readRows() completes
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

// Catch signals
func catchNotifications() {
	state, err := terminal.GetState(int(os.Stdin.Fd()))
	checkErr(err)

	// Deal with SIGINT
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	var timer time.Time
	go func() {
		for sig := range sigChan {
			// Prevent exiting on accidental signal send
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

// Create and return a database handle
func (dbi *dbInfo) connect() (*sql.DB, error) {
	// Set MySQL driver parameters
	dbParameters := "charset=" + dbi.charset

	// Append cleartext and tls parameters if TLS is specified
	if dbi.tls == true {
		dbParameters = dbParameters + "&allowCleartextPasswords=1&tls=skip-verify"
	}

	db, err := sql.Open("mysql", dbi.user+":"+dbi.pass+"@tcp("+dbi.host+":"+dbi.port+")/?"+dbParameters)
	checkErr(err)

	// Ping database to verify credentials
	err = db.Ping()

	return db, err
}

// readRows executes a query and sends each row over a channel to be consumed
func readRows(db *sql.DB, query string, dataChan chan []sql.RawBytes, quitChan chan bool, goChan chan bool, csvHeader bool) {
	rows, err := db.Query(query)
	defer rows.Close()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	cols, err := rows.Columns()
	checkErr(err)

	// Write columns as a header line
	if csvHeader {
		headers := make([]sql.RawBytes, len(cols))
		for i, col := range cols {
			headers[i] = []byte(col)
		}
		dataChan <- headers
		<-goChan
	}

	// Need to scan into empty interface since we don't know how many columns a query might return
	scanVals := make([]interface{}, len(cols))
	vals := make([]sql.RawBytes, len(cols))
	for i := range vals {
		scanVals[i] = &vals[i]
	}

	for rows.Next() {
		err := rows.Scan(scanVals...)
		checkErr(err)

		dataChan <- vals

		// Block and wait for writeRows() to signal back it has consumed the data
		// This is necessary because sql.RawBytes is a memory pointer and when rows.Next()
		// loops and change the memory address before writeRows can properly process the values
		<-goChan
	}

	err = rows.Err()
	checkErr(err)

	close(dataChan)
	quitChan <- true
}

// writeCSV reads from a channel and writes CSV output
func writeCSV(w *Writer, dataChan chan []sql.RawBytes, goChan chan bool, verbose bool) uint {
	var rowsWritten uint
	var verboseCount uint

	if verbose {
		fmt.Println("A '.' will be shown for every 10,000 CSV rows written")
	}

	// Range over row results from readRows()
	for data := range dataChan {
		// Format the data to CSV and write
		size, err := w.Write(data)
		checkErr(err)

		// Visual write indicator when verbose is enabled
		rowsWritten++
		if verbose {
			verboseCount++
			if verboseCount == 10000 {
				fmt.Printf(".")
				verboseCount = 0
			}
		}

		// Flush CSV writer contents once it exceeds flushBufferSize
		if size > flushBufferSize {
			w.Flush()
			err = w.Error()
			checkErr(err)
		}

		// Signal back to readRows() it can loop and scan the next row
		goChan <- true
	}

	// Flush remaining CSV writer contents
	w.Flush()
	err := w.Error()
	checkErr(err)

	return rowsWritten
}
