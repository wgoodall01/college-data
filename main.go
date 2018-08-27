package main

import (
	"fmt"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

type InfoFetcher func(c *College) error

func main() {
	AIRTABLE_API_KEY := os.Getenv("AIRTABLE_API_KEY")
	AIRTABLE_BASE := os.Getenv("AIRTABLE_BASE")
	AIRTABLE_TABLE := os.Getenv("AIRTABLE_TABLE")

	INFO_FETCHERS := []InfoFetcher{
		GetBigFutureInfo,
		GetPrincetonReviewInfo,
	}

	cb := &CollegeBase{
		ApiKey:    AIRTABLE_API_KEY,
		BaseID:    AIRTABLE_BASE,
		TableName: AIRTABLE_TABLE,
	}

	var start time.Time

	start = statusStart("Fetching colleges from Airtable...")
	colleges, fetchErr := cb.Colleges()
	if fetchErr != nil {
		statusFatal(fetchErr)
	}
	statusEnd(start)

	// Fetch info for all the colleges.
	statusLine("Fetching college info...")
	statusCollegeHeader()
	for _, college := range colleges {
		start = statusStartCollege(college)

		for _, fetcher := range INFO_FETCHERS {
			fetcherErr := fetcher(college)
			if fetcherErr != nil {
				statusFatal(fetcherErr)
			}
		}

		// Save changes back to Airtable
		patchErr := cb.Patch(college)
		if patchErr != nil {
			statusFatal(patchErr)
		}

		statusEnd(start)
	}

}

func statusLine(f string, args ...interface{}) (start time.Time) {
	fmt.Printf(f+"\n", args...)
	return time.Now()
}

func statusCollegeHeader() {
	fmt.Printf("%40s%12s%12s\n", "Name", "BigFuture", "Princeton")
}

func statusStartCollege(c *College) (start time.Time) {
	fmt.Printf("%40s%12d%12d  ", c.Name, c.BigFutureID, c.PrincetonReviewId)
	return time.Now()
}

func statusStart(f string, args ...interface{}) (start time.Time) {
	fmt.Printf(f, args...)
	return time.Now()
}

func statusEnd(start time.Time) {
	fmt.Printf(" [done in %.2fs]\n", time.Since(start).Seconds())
}

func statusFatal(v interface{}) {
	fmt.Println()
	fmt.Println(v)
	os.Exit(1)
}
