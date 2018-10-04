package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

type InfoFetcher func(c *College) error

func main() {
	AIRTABLE_API_KEY := os.Getenv("AIRTABLE_API_KEY")
	AIRTABLE_BASE := os.Getenv("AIRTABLE_BASE")
	AIRTABLE_TABLE := os.Getenv("AIRTABLE_TABLE")

	INFO_FETCHERS := []InfoFetcher{
		NewBigFuture(time.Second / 2).FetchInfo,
		NewPrincetonReview(time.Second / 2).FetchInfo,
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

	// waitGroup for them all to complete
	var wg sync.WaitGroup

	for _, c := range colleges {
		wg.Add(1)
		go func(college *College) {
			var cwg sync.WaitGroup
			for _, fetcher := range INFO_FETCHERS {
				fetcherErr := fetcher(college)
				if fetcherErr != nil {
					// give up and explode
					statusFatal(fetcherErr)
				}
			}
			cwg.Wait()

			// save changes back to airtable
			patcherr := cb.Patch(college)
			if patcherr != nil {
				statusFatal(patcherr)
			}

			// print status line
			statusCollege(college)
			wg.Done()
		}(c)
	}

	wg.Wait()

}

func statusLine(f string, args ...interface{}) (start time.Time) {
	fmt.Printf(f+"\n", args...)
	return time.Now()
}

func statusCollegeHeader() {
	fmt.Printf("%40s%12s%12s\n", "Name", "BigFuture", "Princeton")
}

func statusCollege(c *College) (start time.Time) {
	fmt.Printf("%40s%12d%12d\n", c.Name, c.BigFutureID, c.PrincetonReviewId)
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
