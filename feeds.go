package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

// Feed type for feed table
type Feed struct {
	ID               int
	UserID           int
	Slug             string
	Category         string
	Content          string
	Image1           string
	Image2           string
	Image3           string
	Image4           string
	Image5           string
	ReactionsHappy   int
	ReactionsLove    int
	ReactionsFunny   int
	ReactionsShocked int
	ReactionsSad     int
	ReactionsAngry   int
	Lat              float64
	Lng              float64
	Reviewed         int
	DateCreated      time.Time
}

// NewFeed constructs a new Feed
func NewFeed() *Feed {
	f := new(Feed)
	f.ID = 0
	f.UserID = 0
	f.Slug = ""
	f.Category = ""
	f.Content = ""
	f.Image1 = ""
	f.Image2 = ""
	f.Image3 = ""
	f.Image4 = ""
	f.Image4 = ""
	f.ReactionsHappy = 0
	f.ReactionsLove = 0
	f.ReactionsFunny = 0
	f.ReactionsShocked = 0
	f.ReactionsSad = 0
	f.ReactionsAngry = 0
	f.Lat = 0.0
	f.Lng = 0.0
	f.Reviewed = 0
	f.DateCreated = time.Now()
	return f
}

// Comment type for comment table
type Comment struct {
	ID          int
	ParentID    int
	UserID      int
	Content     string
	Reviewed    int
	DateCreated time.Time
}

// NewComment constructs a new comment
func NewComment() *Comment {
	c := new(Comment)
	c.ID = 0
	c.ParentID = 0
	c.UserID = 0
	c.Content = ""
	c.Reviewed = 0
	c.DateCreated = time.Now()
	return c
}

func createComments(feed *Feed, cluster []ClusterMember, rnd *rand.Rand, db *sql.DB) {
	// The feed is added.  Now create 1 - maxComments comments
	numComments := len(cluster)
	lastCommentID := GetRowCount("comments", db)

	fmt.Printf("\tCreating %d comments\n", numComments)

	lastCommentDate := feed.DateCreated
	for k := 0; k < numComments; k++ {
		comment := NewComment()
		comment.UserID = cluster[k].ID
		//comment.UserID, _, _ = GetNextID(feed.UserID, CommenterOffset, feed.Lng, feed.Lat, rnd, db)
		comment.ParentID = feed.ID
		lastCommentDate = lastCommentDate.Add(time.Duration(rnd.Intn(3600)) * time.Second)
		comment.DateCreated = lastCommentDate

		// Generate the content
		paras := rnd.Intn(MaxContentParagraphs-1) + 1
		paraSize := rnd.Intn(2)

		// Get the feed content
		comment.Content = Loripsum(paras, paraSize)

		// Insert Comment
		fmt.Printf("\t\tComment %d: User: %d at %s\n", lastCommentID+k, comment.UserID, comment.DateCreated.UTC().Format("2006-01-02 15:04:05"))
		InsertComment(comment, db)
	}
}

var totalSeconds = 60 * 60 * 10

// CreateFeeds generates a random number of feeds
func CreateFeeds(dayNo int, date time.Time, rnd *rand.Rand, db *sql.DB) {

	numFeeds := 150 + rnd.Intn(100)
	feedsPerHour := numFeeds / 10
	fmt.Printf("Creating %v feeds for %v\n", Green("%d", numFeeds), Green("%s", date))
	start := time.Now()
	// Since it's auto_incremented, the last id used  = row count
	lastFeedID := GetRowCount("feed", db)
	secondPerFeed := 3600 / feedsPerHour
	secondsOffset := 0
	lastHour := 0
	for j := 0; j < numFeeds; j++ {
		feed := NewFeed()

		// Generate a feed
		feed.ID = 1 + lastFeedID
		lastFeedID++

		// calculate the random creation time
		hour := j / feedsPerHour
		if hour > lastHour {
			lastHour = hour
			secondsOffset = 0
		}
		seconds := rnd.Intn(secondPerFeed)

		feed.DateCreated = date.Add(time.Duration(3600*hour+secondsOffset+seconds) * time.Second)
		secondsOffset += seconds
		// Generate random content for the feed
		paragraphs := rnd.Intn(MaxFeedParagraphs-1) + 1
		paragraphSize := rnd.Intn(2)
		feed.Content = Loripsum(paragraphs, paragraphSize)

		// Pick a random user for the feed, and get the user's location
		numComments := rnd.Intn(MaxComments-1) + 1

		userID, cluster := GetCluster(CommenterOffset, numComments, rnd, db)
		feed.UserID = userID
		lon, lat := GetLonLat(feed.UserID, false, db)
		feed.Lng = lon
		feed.Lat = lat

		// Insert the feed
		InsertFeed(feed, db)
		fmt.Printf("\tDay %v - Feed %d [%v/%v]: User: %d [%f, %f] at %s\n", Green("%d", dayNo), feed.ID, Cyan("%d", j), Yellow("%d", numFeeds), feed.UserID, feed.Lng, feed.Lat, feed.DateCreated.UTC().Format("2006-01-02 15:04:05"))
		// Now that the feed is created, add some comments
		createComments(feed, cluster, rnd, db)
	}
	queryTime := time.Since(start)
	fmt.Printf("Execution time: %vms", Round(float64(queryTime)/float64(time.Millisecond), 2))
}
