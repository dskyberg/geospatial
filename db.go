package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
)

// SterlingQuery
var SterlingQuery = `
SELECT feed.id, feed.content, COUNT(cmnt.id) AS comments FROM feed feed JOIN comments cmnt ON feed.id = cmnt.parent_id WHERE DATE(feed.date_created) = CURRENT_DATE()-1 AND COALESCE(3959 * acos ( cos ( radians($account[lat]) ) * cos( radians( feed.lat ) ) * cos( radians( feed.lng ) - radians($account[lng]) ) + sin ( radians($account[lat]) ) * sin( radians( feed.lat ) ) ), 0) <= 25 GROUP BY feed.id ORDER BY comments DESC LIMIT 6
`

// UpdateGeomQuery
var UpdateGeomQuery = `
UPDATE address SET geom = GeomFromText('POINT(11.40045540 47.23693990)') WHERE id='48e42824-f529-11e6-b9aa-6c4008befb36';
`

// CreateTableQuery
var CreateTableQuery = `
GRANT ALL ON *.* TO 'root'@'%' IDENTIFIED BY 'password';

CREATE SCHEMA geo_data;
USE geo_data;
CREATE TABLE feed (
  id int(12) NOT NULL AUTO_INCREMENT,
  user_id int(12) NOT NULL,
  slug varchar(50) CHARACTER SET latin1 NOT NULL,
  category varchar(20) CHARACTER SET latin1 NOT NULL,
  content text CHARACTER SET latin1 NOT NULL,
  image1 varchar(40) CHARACTER SET latin1 NOT NULL,
  image2 varchar(40) CHARACTER SET latin1 NOT NULL,
  image3 varchar(40) CHARACTER SET latin1 NOT NULL,
  image4 varchar(40) CHARACTER SET latin1 NOT NULL,
  image5 varchar(40) CHARACTER SET latin1 NOT NULL,
  reactions_happy int(7) NOT NULL,
  reactions_love int(7) NOT NULL,
  reactions_funny int(7) NOT NULL,
  reactions_shocked int(7) NOT NULL,
  reactions_sad int(7) NOT NULL,
  reactions_angry int(7) NOT NULL,
  lat decimal(10,8) NOT NULL,
  lng decimal(11,8) NOT NULL,
  reviewed tinyint(1) NOT NULL,
  date_created timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY lat (lat),
  KEY lng (lng),
  KEY category (category),
  KEY user_id (user_id)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 AUTO_INCREMENT=1638 ;

CREATE TABLE comments (
  id int(12) NOT NULL AUTO_INCREMENT,
  parent_id int(12) NOT NULL,
  user_id int(12) NOT NULL,
  content text CHARACTER SET latin1 NOT NULL,
  reviewed tinyint(1) NOT NULL,
  date_created timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY parent_id (parent_id)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8 AUTO_INCREMENT=3408 ;

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

// InlineQuery
var InlineQuery = `
SELECT id, (3956 * ACOS(COS(RADIANS(%[3]s)) 
			* COS(RADIANS(lat)) 
			* COS(RADIANS(lon) - RADIANS(%[2]s)) 
			+ SIN(RADIANS(%[3]s)) * SIN(RADIANS(lat)) ) ) 
			AS distance FROM %[1]s HAVING distance < $[4]s;
`

// InlineRadiansQuery
var InlineRadiansQuery = `
SELECT id, (3956 * ACOS( COS(%[3]f) 
			* COS(rlat_d) 
			* COS(rlon_d - %[2]f) 
			+ SIN(%[3]f) * SIN(rlat_d) ) ) 
			AS distance FROM %[1]s HAVING distance < %[4]d;
`

//  StoredFuncQuery uses the stored function
var StoredFuncQuery = "SELECT id, distance_loc(lon, lat, %[2]s, %[3]s) AS distance FROM %[1]s;"

// SpatialFuncQuery use MySQL spatial function
var SpatialFuncQuery = "SELECT id, (st_distance_sphere(geom, POINT(%[2]s, %[3]s))/1000) as distance FROM %[1]s;"

// InsertQuery must be formatted locally to set the table name, then rendered with Prepare
var InsertQuery = `INSERT INTO %[1]s (lon, lat, rlon_d, rlat_d, rlon_dd, rlat_dd, geom, number, street, unit, city, district, region, postcode) VALUES ( %[2]s, %[3]s, RADIANS(%[2]s), RADIANS(%[3]s), RADIANS(%[2]s), RADIANS(%[3]s),  POINT(%[2]s, %[3]s), "%[4]s", "%[5]s", "%[6]s", "%[7]s", "%[8]s", "%[9]s", "%[10]s");`

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

// GetRowCount get the number of rows in a table
func GetRowCount(table string, db *sql.DB) int {
	// Since it's auto_incremented, the last id used  = row count
	var lastID int
	err := db.QueryRow(fmt.Sprintf("select count(id) from %s;", table)).Scan(&lastID)
	if err != nil {
		panic(err)
	}
	return lastID
}

// Point struct
type Point struct {
	Lon float64
	Lat float64
}

// ClusterMember hold the ID and distance of a row in proximity to another row
type ClusterMember struct {
	ID       int
	Distance float64
}

// Account type for accounts table
type Account struct {
	ID       int
	Lon      float64
	Lat      float64
	geom     Point
	Rlon     float64
	Rlat     float64
	Number   string
	Street   string
	Unit     string
	City     string
	District string
	Region   string
	Postcode string
}

// GetClusteredRows returns all the rows that are witnin 'radius' of the
// provided longitude and latitude (provided as radians)
func GetClusteredRows(rlon, rlat float64, radius int, db *sql.DB) []ClusterMember {
	var cluster []ClusterMember
	query := fmt.Sprintf(InlineRadiansQuery, "addr_inno", rlon, rlat, radius)
	rows, err := db.Query(query)
	if err != nil {
		fmt.Println(query)
		panic(err)
	}

	for rows.Next() {
		var rID int
		var rDistance float64
		if err := rows.Scan(&rID, &rDistance); err != nil {
			log.Fatal(err.Error())
		}
		cluster = append(cluster, ClusterMember{rID, rDistance})
	}
	return cluster
}

// GetLonLat gets the lon and lat values from an address row
func GetLonLat(ID int, radians bool, db *sql.DB) (float64, float64) {
	// Find a sucker to enter the feed
	var lon, lat float64
	var pt string
	if radians {
		pt = "rlon_d, rlat_d"
	} else {
		pt = "lon, lat"
	}
	err := db.QueryRow(fmt.Sprintf("SELECT %s FROM addr_inno WHERE id=%d;", pt, ID)).Scan(&lon, &lat)
	if err != nil {
		panic(err)
	}
	return lon, lat
}

// GetNextID returns the next record within a given distance.
func GetNextID(startingID int, offset int, startingLon, startingLat float64, rnd *rand.Rand, db *sql.DB) (int, float64, float64) {
	var lon, lat float64
	alreadyFlipped := false
	startingOffset := CoinFlip(rnd.Intn(offset), rnd)
	nextID := startingID + startingOffset
	for {
		// If we pass this many, skip the original startingId
		if nextID == startingID {
			continue
		}

		err := db.QueryRow(fmt.Sprintf("SELECT lon, lat FROM addr_inno WHERE id=%d;", nextID)).Scan(&lon, &lat)
		if err != nil {
			if err == sql.ErrNoRows {
				if !alreadyFlipped {
					nextID = startingID - startingOffset
					alreadyFlipped = true
					continue
				} else {
					alreadyFlipped = false
					startingOffset = CoinFlip(rnd.Intn(offset), rnd)
					nextID = startingID + startingOffset
				}
			} else {
				panic(err)
			}
		}
		if Distance(lon, lat, startingLon, startingLat) < 25.0 {
			return nextID, lon, lat
		}
		nextID++
	}
}

func in(value int, slice []int) bool {
	for _, b := range slice {
		if b == value {
			return true
		}
	}
	return false
}

func sample(popSize, sampleSize, mod int, rnd *rand.Rand) int {
	var target int
	for {
		// Get a random number 0 <= skew < mod
		skew := rnd.Intn(mod)
		target = skew*sampleSize + rnd.Intn(sampleSize)
		if target > popSize {
			continue
		}
		return target
	}
}

func samplePop(pop []ClusterMember, sampleSize int, rnd *rand.Rand) []ClusterMember {
	samples := make([]ClusterMember, sampleSize)
	popSize := len(pop)
	mod := popSize / sampleSize

	if mod == 0 {
		for i := 0; i < sampleSize; i++ {
			samples[i].ID = pop[i].ID
			samples[i].Distance = pop[i].Distance
		}
	} else {
		used := make([]int, sampleSize)
		idx := 0
		for idx < sampleSize {

			// Get a random number 0 <= skew < mod
			target := sample(popSize, sampleSize, mod, rnd)
			if in(pop[target].ID, used) {
				continue
			}
			used[idx] = pop[target].ID
			samples[idx].ID = pop[target].ID
			samples[idx].Distance = pop[target].Distance
			idx++
		}
	}

	return samples
}

// GetCluster uses channels to timeout fetching a set of clustered IDs
func GetCluster(offset, clusterSize int, rnd *rand.Rand, db *sql.DB) (int, []ClusterMember) {
	var anchor int
	var cluster []ClusterMember

	populationSize := GetRowCount("addr_inno", db)
	fmt.Printf("Looking for ")
	for {
		anchor = rnd.Intn(populationSize-1) + 1
		fmt.Printf("%d rows near %d...", clusterSize, anchor)
		rLon, rLat := GetLonLat(anchor, true, db)
		c := GetClusteredRows(rLon, rLat, 25, db)
		numRows := len(c)
		if numRows < clusterSize {
			fmt.Printf("too small (%d), retrying...", len(c))
			continue
		}
		fmt.Printf("found in %d rows\n", numRows)
		cluster = samplePop(c, clusterSize, rnd)
		break
	}

	return anchor, cluster
}

// InsertFeed inserts a Feed into the feed table
func InsertFeed(feed *Feed, db *sql.DB) {
	// Insert the feed
	query := fmt.Sprintf("INSERT INTO feed ( user_id, slug, category, content, image1, image2, image3, image4, image5, reactions_happy, reactions_love, reactions_funny, reactions_shocked, reactions_sad, reactions_angry, lat, lng, reviewed, date_created) VALUES (%d, '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', %d, %d, %d, %d, %d, %d, %f, %f, %d, '%s');",
		feed.UserID,
		feed.Slug,
		feed.Category,
		feed.Content,
		feed.Image1,
		feed.Image2,
		feed.Image3,
		feed.Image4,
		feed.Image5,
		feed.ReactionsHappy,
		feed.ReactionsLove,
		feed.ReactionsFunny,
		feed.ReactionsShocked,
		feed.ReactionsSad,
		feed.ReactionsAngry,
		feed.Lat, feed.Lng,
		feed.Reviewed,
		feed.DateCreated.UTC().Format("2006-01-02 15:04:05"))
	_, err := db.Exec(query)
	if err != nil {
		fmt.Println(query)
		panic(err)
	}
}

// InsertComment inserts a Feed into the feed table
func InsertComment(comment *Comment, db *sql.DB) {
	// Insert the feed
	query := fmt.Sprintf("INSERT INTO comments (parent_id, user_id, content, reviewed, date_created) VALUES (%d, %d, '%s', %d, '%s');",
		comment.ParentID,
		comment.UserID,
		comment.Content,
		comment.Reviewed,
		comment.DateCreated.UTC().Format("2006-01-02 15:04:05"))
	_, err := db.Exec(query)
	if err != nil {
		fmt.Println(query)
		panic(err)
	}
}
