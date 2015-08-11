# mycsv
A command line tool for writing CSV files from MySQL queries. MySQL has a "select into outfile" feature to write CSV files but the files are written to the local file system and are owned by the MySQL user. This requires giving users access to the database machine, setting up an NFS mount or creating an automated process to retrieve the CSV files. mycsv allows users or a process to write CSV files directly to a destination.

**Features:**
* Customizable CSV format
* Write initial header line based on columns in query
* Cross platform (linux, darwin & windows)*
* Supports SSL configured MySQL accounts
* Designed for interative and non-interactive use
* Write to a file destination or stdout for redirection or piping
* Stdin redirection to obtain query from pipe or file**


\*64bit binaries provided by build script. Minor changes to the build script can provide 32bit binaries.

**Stdin redirection not supported on windows

Installation
--
```shell
$ go get -d github.com/joshuaprunier/mycsv

$ ./build.sh

Building mycsv version
1.0.30-421b387-20150810.155045


Building Linux
        mycsv - OK

Building Windows
        mycsv.exe - OK

Building Darwin
        mycsv - OK

Done!

$ ls -R bin
bin:
darwin  linux  windows

bin/darwin:
mycsv

bin/linux:
mycsv

bin/windows:
mycsv.exe
```

Usage
--
```shell
mycsv version 1.0.30-421b387-20150810.155045

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
```

Examples
--
##### Basic usage
```shell
mycsv -user=jprunier -pass= -host=db1 -file=my.csv \
-query="select * from test.table1 where filter in ('1', 'test', 'another')"
```
<br>
##### Customize header line column names
```shell
mycsv -user=jprunier -pass= -host=db1 -file=my.csv \
-query="select a as 'column1', b, c as 'column c' from test.table2"
```
<br>
##### UTF8 output and do not write the colum header line
```shell
mycsv -user=jprunier -pass= -host=db1 -file=my.csv -charset=utf8 -header=false\
-query="select * from test.table1 where filter in ('1', 'test', 'another')"
```
<br>
##### Change CSV format - pipe delimited, fields single quoted, lines terminated by carriage return & newline
```shell
mycsv -user=jprunier -pass= -host=db1 -file=my.csv -d="|" -q="'" -t="\r\n"\
-query="select * from test.table1 where filter in ('1', 'test', 'another')"
```
<br>
##### Get query from stdin
```shell
echo "select * from test.table1 where filter in ('1', 'test', 'another')" |\
mycsv -user=jprunier -pass= -host=db1 -file=my.csv
```
or
```shell
echo "select * from test.table1 where filter in ('1', 'test', 'another')" > myquery
mycsv -user=jprunier -pass= -host=db1 -file=my.csv < myquery
```
<br>
##### Pipe stdout to change \N to the word NULL and write to a file
```shell
mycsv -user=jprunier -pass=mypass -host=db1 \
-query="select * from test.table1 where filter in ('1', 'test', 'another')" | sed 's/\\N/NULL/g' > my.csv
```
<br>
##### Show information during execution
```shell
mycsv -user=jprunier -pass=mypass -host=db1 -file=my.csv -query="select * from test.table3 limit 100000" -v
CSV output will be written to my.csv
A '.' will be shown for every 10,000 CSV rows written
..........
100001 rows written
Total runtime = 10.269565988s

```

License
--
[MIT] (LICENSE)

