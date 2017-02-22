package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/alecthomas/kingpin.v2"
)

/**
Spherical "great circle" distance calculations are based on variations
of the Haversine function. Geometry data is typically represented in
degrees (latitude and longitude).  But great circle distance calculaions
are calculated in radians, using the following conversion:
	Radians = Degrees * (pi / 180)

*/

// Mean radius of Earth in kilometers
var Rk float64 = 6371.0

// Mean radius of Earth in miles
var Rm float64 = 3959.0

var updateGeomQuery = `
UPDATE address SET geom = GeomFromText('POINT(11.40045540 47.23693990)') WHERE id='48e42824-f529-11e6-b9aa-6c4008befb36';
`

var createTableQuery = `
GRANT ALL ON *.* TO 'root'@'%' IDENTIFIED BY 'password';

CREATE SCHEMA geo_data;
USE geo_data;
CREATE TABLE addr_isam (
  id INT NOT NULL AUTO_INCREMENT,
  lon decimal(10,7) NOT NULL,
  lat decimal(10,7) NOT NULL,
  geom point NOT NULL,
  rlon_d decimal(10, 7) NOT NULL COMMENT 'lon as decimal radians',
  rlat_d decimal(10, 7) NOT NULL COMMENT 'lat as decimal radians',
  rlon_dd double(10, 7) NOT NULL COMMENT 'lon as double radians',
  rlat_dd double(10, 7) NOT NULL COMMENT 'lat as double radians',
  number varchar(32) DEFAULT NULL,
  street varchar(64) DEFAULT NULL,
  unit varchar(8) DEFAULT NULL,
  city varchar(64) DEFAULT NULL,
  district varchar(64) DEFAULT NULL,
  region varchar(64) DEFAULT NULL,
  postcode varchar(16) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY lon (lon),
  KEY lat (lat),
  SPATIAL KEY geom (geom)
) ENGINE=MyISAM DEFAULT CHARSET='utf8mb4';

CREATE TABLE addr_inno (
  id INT NOT NULL AUTO_INCREMENT,
  lon decimal(10,7) NOT NULL,
  lat decimal(10,7) NOT NULL,
  geom point NOT NULL,
  rlon_d decimal(10, 7) NOT NULL COMMENT 'lon as decimal radians',
  rlat_d decimal(10, 7) NOT NULL COMMENT 'lat as decimal radians',
  rlon_dd double(10, 7) NOT NULL COMMENT 'lon as double radians',
  rlat_dd double(10, 7) NOT NULL COMMENT 'lat as double radians',
  number varchar(32) DEFAULT NULL,
  street varchar(64) DEFAULT NULL,
  unit varchar(8) DEFAULT NULL,
  city varchar(64) DEFAULT NULL,
  district varchar(64) DEFAULT NULL,
  region varchar(64) DEFAULT NULL,
  postcode varchar(16) DEFAULT NULL,
  PRIMARY KEY (id),
  KEY lon (lon),
  KEY lat (lat),
  SPATIAL KEY geom (geom)
) ENGINE=InnoDB DEFAULT CHARSET='utf8mb4';

GRANT ALL ON geo_data.* TO 'geo_user'@'%' IDENTIFIED BY 'geo_password';

CREATE DEFINER = 'geo_user'@'%' FUNCTION distance_loc (lon DECIMAL(12,8), lat DECIMAL(12,8), tlon DECIMAL(12,8), tlat DECIMAL(12,8)) 
	RETURNS DECIMAL(12,8)
	RETURN  3959 * ACOS( SIN( lat ) * SIN( tlat) + COS( tlat ) * COS( lat ) * COS( lon - tlon ) ) ;
`

// Global sql.DB to access the database by all handlers
var rawQuery = `
SELECT id, (3956 * ACOS(COS(RADIANS(%[3]s)) 
			* COS(RADIANS(lat)) 
			* COS(RADIANS(lon) - RADIANS(%[2]s)) 
			+ SIN(RADIANS(%[3]s)) * SIN(RADIANS(lat)) ) ) 
			AS distance FROM %[1]s HAVING distance < $[4]s;
`

// Uses the stored function
var storedFuncQuery = "SELECT id, distance_loc(lon, lat, %[2]s, %[3]s) AS distance FROM %[1]s;"

// Use MySQL spatial function
var spatialFuncQuery = "SELECT id, (st_distance_sphere(geom, POINT(%[2]s, %[3]s))/1000) as distance FROM %[1]s;"

// insertQuery must be formatted locally to set the table name, then rendered with Prepare
var insertQuery = `INSERT INTO %[1]s (lon, lat, rlon_d, rlat_d, rlon_dd, rlat_dd, geom, number, street, unit, city, district, region, postcode) VALUES ( %[2]s, %[3]s, RADIANS(%[2]s), RADIANS(%[3]s), RADIANS(%[2]s), RADIANS(%[3]s),  POINT(%[2]s, %[3]s), "%[4]s", "%[5]s", "%[6]s", "%[7]s", "%[8]s", "%[9]s", "%[10]s");`

// GeoCommand stores all values provided on the command command line. These
// values are passed to the command functions listed below.
type GeoCommand struct {
	User     string
	Password string
	Host     string
	Port     int
	Schema   string
	Table    string
	Quiet    bool
	Lon      string
	Lat      string
	InFile   *os.File
	Postal   string
}

// Connect to the MySQL database
func (gc *GeoCommand) Connect() *sql.DB {
	connectStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?interpolateParams=true", gc.User, gc.Password, gc.Host, gc.Port, gc.Schema)

	// Create an sql.DB and check for errors
	db, err := sql.Open("mysql", connectStr)
	if err != nil {
		log.Fatal(err)
		panic(err.Error())
	}

	return db
}

func rad(deg float64) float64 {
	return deg * math.Pi / 180
}

// haversin(θ) function
func hav(theta float64) float64 {
	return .5 * (1 - math.Cos(theta))
}

func hav2(theta float64) float64 {
	return math.Pow(math.Sin(theta/2), 2)
}

// Calculate distance using law of cosines.
// d = acos( sin φ1 ⋅ sin φ2 + cos φ1 ⋅ cos φ2 ⋅ cos Δλ ) ⋅ R
//Values are in radians
func hDist(dlon, dlat, dtlon, dtlat float64) float64 {
	lon := rad(dlon)
	lat := rad(dlat)
	tlon := rad(dtlon)
	tlat := rad(dtlat)

	return Rk *
		math.Acos(
			math.Sin(lat)*math.Sin(tlat)+
				math.Cos(lat)*math.Cos(tlat)*
					math.Cos(lon-tlon))
}

// Calculate distance using Haversine formula. Values are in radians
func hDist2(dlon, dlat, dtlon, dtlat float64) float64 {
	lon := rad(dlon)
	lat := rad(dlat)
	tlon := rad(dtlon)
	tlat := rad(dtlat)

	h := hav(tlat-lat) + math.Cos(lat)*math.Cos(tlat)*hav(tlon-lon)

	return 2 * Rk * math.Asin(math.Sqrt(h))
}

// Distance calculates the distance between two points
func (gc *GeoCommand) Distance(context *kingpin.ParseContext) error {
	var dlon float64 = 11.8686483
	var dlat float64 = 47.2261598
	dtlon, _ := strconv.ParseFloat(gc.Lon, 64)
	dtlat, _ := strconv.ParseFloat(gc.Lat, 64)

	fmt.Printf("Method 1: %f\nMethod 2: %f\n", hDist(dlon, dlat, dtlon, dtlat), hDist2(dlon, dlat, dtlon, dtlat))
	return nil
}

// Load a file into the MySQL database
// 0:	LON,
// 1:	LAT,
// 2:	NUMBER,
// 3:	STREET,
// 4:	UNIT,
// 5:	CITY,
// 6:	DISTRICT,
// 7:	REGION,
// 8:	POSTCODE,
// 9:	ID,
// 10:	HASH
//
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
		query := fmt.Sprintf(insertQuery, gc.Table,
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

// Select uses the query statement above to fetch rows by the lon/lat provided
// on the command line.
func (gc *GeoCommand) doSelect(query string, context *kingpin.ParseContext) error {
	db := gc.Connect()
	defer db.Close()

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

// Select uses the query statement above to fetch rows by the lon/lat provided
// on the command line.
func (gc *GeoCommand) Select(context *kingpin.ParseContext) error {
	err := gc.doSelect(storedFuncQuery, context)
	if err != nil {
		return err
	}
	return gc.doSelect(rawQuery, context)
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
