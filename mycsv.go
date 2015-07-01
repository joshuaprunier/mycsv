package main

import (
	"database/sql"
	"database/sql/driver"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"./csv"
	_ "github.com/go-sql-driver/mysql" // Go MySQL driver
	"golang.org/x/crypto/ssh/terminal"
)

// 25MB
const flushSize = 26214400

// Type definitions
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

func main() {
	start := time.Now()

	// Profiling flags
	var cpuprofile = flag.String("debug_cpu", "", "write cpu profile to file")
	var memprofile = flag.String("debug_mem", "", "write memory profile to this file")

	// Database flags
	dbUser := flag.String("user", "", "Username")
	dbPass := flag.String("pass", "", "Password")
	dbHost := flag.String("host", "", "Database")
	dbPort := flag.String("port", "3306", "Port")
	dbTLS := flag.Bool("tls", false, "TLS")

	csvFile := flag.String("file", "", "CSV filename")
	sqlQuery := flag.String("query", "", "MySQL query")

	// Parse flags
	flag.Parse()

	// Print usage
	if flag.NFlag() == 0 {
		name := os.Args[0]

		fmt.Println()
		fmt.Println("	", name, "USAGE:")
		fmt.Println()
		flag.PrintDefaults()
		fmt.Println()

		os.Exit(0)
	}

	// Check if an output file was supplied otherwise use standard out
	var dest io.Writer
	var err error
	if *csvFile == "" {
		fmt.Println("CSV output will be written to standard out")
		dest = os.Stdout
	} else {
		f, err := os.Open(*csvFile)
		if err == nil {
			fmt.Println(*csvFile, "already exists!")
			fmt.Println("Please remove it or use a different filename")
			os.Exit(1)
		}
		f.Close()
		dest, err = os.Create(*csvFile)
	}

	// If query not provided read from standard in
	var query string
	if *sqlQuery == "" {
		fmt.Println("You must supply a query")
		os.Exit(1)
	} else {
		query = *sqlQuery
	}

	// Make sure the query is a select
	if query[0:6] != "select" {
		fmt.Println("Query must be a select!")
		os.Exit(1)
	}

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
		fmt.Println("You must provide a user name!")
		os.Exit(1)
	}

	// If password is blank prompt user
	if *dbPass == "" {
		fmt.Println("Enter password: ")
		//	if runtime.GOOS == "windows" {
		//	} else {
		//	}
		//	fmt.Println(runtime.GOOS)
		//	fmt.Println(os.Stdin.Fd())
		pwd, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		checkErr(err)
		*dbPass = string(pwd)
	}

	// Populate dbInfo struct with cli flags
	dbi := dbInfo{user: *dbUser, pass: *dbPass, host: *dbHost, port: *dbPort, TLS: *dbTLS}

	// Create a *sql.DB connection to the source database
	db, err := dbi.Connect()
	defer db.Close()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Start reading & writing
	dataChan := make(chan []NullRawBytes)
	quitChan := make(chan bool)
	goChan := make(chan bool)
	go readRows(db, query, dataChan, quitChan, goChan)
	writeCSV(dest, dataChan, goChan)

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

	fmt.Println()
	fmt.Println("Total runtime =", time.Since(start))
}

// Pass the buck error catching
func checkErr(e error) {
	if e != nil {
		log.Panic(e)
	}
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
func readRows(db *sql.DB, query string, dataChan chan []NullRawBytes, quitChan chan bool, goChan chan bool) {
	rows, err := db.Query(query)
	defer rows.Close()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	cols, err := rows.Columns()
	checkErr(err)

	headers := make([]NullRawBytes, len(cols))
	for i, col := range cols {
		headers[i] = NullRawBytes{RawBytes: []byte(col), Valid: true}
	}
	dataChan <- headers
	<-goChan

	// Need to scan into empty interface since we don't know how many columns or their types
	scanVals := make([]interface{}, len(cols))
	vals := make([]NullRawBytes, len(cols))
	cpy := make([]NullRawBytes, len(cols))
	for i := range vals {
		scanVals[i] = &vals[i]
	}

	var rowNum int64
	for rows.Next() {
		err := rows.Scan(scanVals...)
		checkErr(err)

		copy(cpy, vals)
		dataChan <- cpy
		rowNum++

		// Block until writeRows() signals it is safe to proceed
		// This is necessary because sql.RawBytes is a memory pointer and rows.Next() will loop and change the memory address before writeRows can properly process the values
		<-goChan
	}

	err = rows.Err()
	checkErr(err)

	close(dataChan)
	quitChan <- true

	fmt.Println(rowNum, "rows inserted")
}

// writeCSV writes csv output
func writeCSV(w io.Writer, dataChan chan []NullRawBytes, goChan chan bool) {
	out := csv.NewWriter(w)
	out.Quotes = '"'
	out.Escape = '\\'
	out.QuoteEmpty = true

	// Range over row results from readRows()
	var cnt int
	for data := range dataChan {
		a := make([]string, 0)
		for _, row := range data {
			cnt += len(row.RawBytes)
			if row.Valid {
				a = append(a, string(row.RawBytes))
			} else {
				a = append(a, string("\\N"))
			}
		}

		// Write CSV
		err := out.Write(a)
		checkErr(err)

		// Flush CSV writer contents
		if cnt > flushSize {
			out.Flush()
			err = out.Error()
			checkErr(err)
		}

		// Allow read function to unblock and loop over rows
		goChan <- true
	}

	// Flush CSV writer contents
	out.Flush()
	err := out.Error()
	checkErr(err)

}
