// Geospatial is a simple benchmarking utility to benchmark spatial geometry
// in MySQL.
package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/alecthomas/kingpin.v2"
)

// GeoCommand stores all values provided on the command command line. These
// values are passed to the command functions listed below.
type GeoCommand struct {
	User      string   // name of the MySQL user
	Password  string   // MySQL user's password
	Host      string   // MySQL host IP/DNS name, or localhost
	Port      int      // MySQL port
	Schema    string   // MySQL database/schema to use
	Table     string   // the MySQL table name for addresses
	Quiet     bool     // whether to show non-error output
	Lon       string   // Longitude to use for Select
	Lat       string   // Latitude to use for Select
	InFile    *os.File // File name to use for Load
	Postal    string   // Postal code to prepend to the file's codes for Load
	QueryType string   // Type of query to use for Select.  Can be inline, stored, or spatial
}

// Distance calculates the distance between two points
func (gc *GeoCommand) Distance(context *kingpin.ParseContext) error {
	dlon := 11.8686483
	dlat := 47.2261598
	dtlon, _ := strconv.ParseFloat(gc.Lon, 64)
	dtlat, _ := strconv.ParseFloat(gc.Lat, 64)

	fmt.Printf("Method 1: %f\nMethod 2: %f\n", Distance(dlon, dlat, dtlon, dtlat), Distance2(dlon, dlat, dtlon, dtlat))
	return nil
}

// Load a file into the MySQL database
func (gc *GeoCommand) Load(ctx *kingpin.ParseContext) error {

	db := gc.Connect()

	// Open a buffered CSV reader
	reader := csv.NewReader(bufio.NewReader(gc.InFile))
	// Read off the header row
	_, err := reader.Read()
	if err != nil {
		log.Fatal(err.Error())
		return err
	}

	for {
		row, err := reader.Read()

		if err == io.EOF {
			// Done
			break
		}

		// IF a postal code was provided, it should be prepended to the
		// postal code in the row.
		p := ""
		if gc.Postal != "" {
			p = fmt.Sprintf("%s-", gc.Postal)
		} else {
			p = row[8]
		}

		// Set the table selection
		query := fmt.Sprintf(InsertQuery, gc.Table,
			row[0],
			row[1],
			row[2],
			row[3],
			row[4],
			row[5],
			row[6],
			row[7],
			p)
		_, err = db.Exec(query)

		if err != nil {
			fmt.Println(row)
			fmt.Println(query)
			return err
		}
	}

	return nil
}

// Benchmark runs the benchmark tests
func (gc *GeoCommand) Benchmark(context *kingpin.ParseContext) error {
	db := gc.Connect()
	defer db.Close()
	return nil
}

// Select uses the query statement above to fetch rows by the lon/lat provided
// on the command line.
func (gc *GeoCommand) Select(context *kingpin.ParseContext) error {
	db := gc.Connect()
	defer db.Close()

	var query string
	if gc.QueryType == "inline" {
		query = InlineQuery
	} else if gc.QueryType == "stored" {
		query = StoredFuncQuery
	} else if gc.QueryType == "spatial" {
		query = SpatialFuncQuery
	} else {
		panic(fmt.Sprintf("GeoCommand.Select - unknown query type: %s\n", gc.QueryType))
	}

	// Search the database for the username provided
	// If it exists grab the password for validation
	q := fmt.Sprintf(query, gc.Table, gc.Lon, gc.Lat)
	fmt.Printf("%s\n", q)
	start := time.Now()
	rows, err := db.Query(q)
	queryTime := time.Since(start)

	if err != nil {
		log.Fatal(err.Error())
		return err
	}

	counter := 0
	start = time.Now()
	for rows.Next() {
		counter++
		if !gc.Quiet {
			var (
				rID      string
				distance float64
			)
			if err := rows.Scan(&rID, &distance); err != nil {
				log.Fatal(err)
			}

			fmt.Fprintf(os.Stdout, "%s - %f\n", rID, distance)
		}
	}
	fetchTime := time.Since(start)
	fmt.Printf("Query time: %s  Fetch time: %s\n", queryTime, fetchTime)
	return nil
}

// Seed generates feeds with comments for a random number of days
func (gc *GeoCommand) Seed(context *kingpin.ParseContext) error {
	db := gc.Connect()
	defer db.Close()
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := MaxDays; i > 0; i-- {
		// N days ago
		day := DaysAgo(i)
		CreateFeeds(i, day, rnd, db)
	}
	return nil
}

func configureApp(app *kingpin.Application) {
	gc := &GeoCommand{}

	// General flags
	app.Flag("quiet", "Run in quiet mode.  Errors will still be shown.").
		Short('q').
		BoolVar(&gc.Quiet)
	app.Flag("user", "MSQL user").
		Default("geo_user").
		StringVar(&gc.User)
	app.Flag("password", "MSQL password").
		Default("geo_password").
		StringVar(&gc.Password)
	app.Flag("host", "MSQL host").
		Default("127.0.0.1").
		StringVar(&gc.Host)
	app.Flag("port", "MSQL port").
		Default("3306").
		IntVar(&gc.Port)
	app.Flag("schema", "MSQL schema/database").
		Default("geo_data").
		StringVar(&gc.Schema)
	app.Flag("table", "MSQL table").
		Default("addr_inno").
		StringVar(&gc.Table)

	// Select command and args
	selectCmd := app.Command("select", "Search for a location by lon lat.").Action(gc.Select)
	selectCmd.Arg("lon", "Longitude in the form degrees.minutes [DD.MMMMMMMM]").
		Required().
		StringVar(&gc.Lon)
	selectCmd.Arg("lat", "Latitude in the form degrees.minutes [DD.MMMMMMMM]").
		Required().
		StringVar(&gc.Lat)
	selectCmd.Flag("query", "Type of query to use [stored, inline, spatial]").
		Required().
		EnumVar(&gc.QueryType, "inline", "stored", "spatial")

	// Load command and args
	loadCmd := app.Command("load", "Load data from a CSV file.").Action(gc.Load)
	loadCmd.Flag("postal", "PostalCode to prepend to file postal codes").
		StringVar(&gc.Postal)
	loadCmd.Arg("file", "File to load").
		Required().
		OpenFileVar(&gc.InFile, os.O_RDONLY, 0666)

	// Distance command
	distCmd := app.Command("distance", "Calc distance between two points.").Action(gc.Distance)
	distCmd.Arg("lon", "Longitude in the form degrees.minutes [DD.MMMMMMMM]").
		Required().
		StringVar(&gc.Lon)
	distCmd.Arg("lat", "Latitude in the form degrees.minutes [DD.MMMMMMMM]").
		Required().
		StringVar(&gc.Lat)

	app.Command("seed", "Seed the database with feeds and comments").Action(gc.Seed)
}

func main() {
	kingpin.CommandLine.Help = "Time tests for MySQL Geospatial data."
	kingpin.CommandLine.HelpFlag.Short('h')
	app := kingpin.New("modular", "My modular application.")
	//app.UsageTemplate(kingpin.CompactUsageTemplate)
	app.Version("0.0.1")
	app.Author("David Skyberg")
	configureApp(app)
	kingpin.MustParse(app.Parse(os.Args[1:]))
}
