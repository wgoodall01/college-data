package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

type PrincetonReview struct {
	ticker <-chan time.Time
}

func NewPrincetonReview(rateLimit time.Duration) *PrincetonReview {
	return &PrincetonReview{
		ticker: time.Tick(rateLimit),
	}
}

func (pr *PrincetonReview) FetchInfo(c *College) (err error) {
	url := fmt.Sprintf("https://www.princetonreview.com/college/x-%d", c.PrincetonReviewId)

	<-pr.ticker // rate limit
	res, fetchErr := http.Get(url)
	if fetchErr != nil {
		return fetchErr
	}

	doc, htmlParseErr := goquery.NewDocumentFromReader(res.Body)
	if htmlParseErr != nil {
		return htmlParseErr
	}

	// Recover from any parse errors
	defer func() {
		if panicked := recover(); panicked != nil {
			err = errors.Errorf("princeton parse error: %v", panicked)
		}
	}()

	// lock college for editing
	c.Lock()
	defer c.Unlock()

	DefaultInt(&c.NumApplicants, getPrincetonInt(doc, "Applicants"))
	DefaultFloat(&c.AcceptanceRate, getPrincetonFloat(doc, "Acceptance Rate"))
	DefaultFloat(&c.GPAAverage, getPrincetonFloat(doc, "Average HS GPA"))

	// ACT range
	ACTRangeLow, ACTRangeHigh := getPrincetonIntRange(doc, "ACT Composite")
	DefaultInt(&c.ACTRangeLow, ACTRangeLow)
	DefaultInt(&c.ACTRangeHigh, ACTRangeHigh)

	return nil
}

func getPrincetonItem(root *goquery.Document, label string) *string {
	parent := root.Find("div.col-sm-4").FilterFunction(func(i int, el *goquery.Selection) bool {
		siblings := el.Children()
		if siblings.Length() <= 1 {
			return false
		}
		labelEl := siblings.Slice(0, 1) // first child
		return strings.ToLower(labelEl.Text()) == strings.ToLower(label)
	})

	if parent.Length() == 0 {
		return nil
	}

	value := strings.TrimSpace(parent.Find("div:last-child").Text())
	return &value
}

func getPrincetonInt(root *goquery.Document, label string) *int {
	value := getPrincetonItem(root, label)
	if value == nil {
		return nil
	}

	valInt := MustParseInt(TrimFormattedNumber(*value))
	return &valInt
}

func getPrincetonIntRange(root *goquery.Document, label string) (low *int, hi *int) {
	valueStr := getPrincetonItem(root, label)
	if valueStr == nil {
		return nil, nil
	}

	split := strings.Split(*valueStr, " - ")
	if len(split) != 2 {
		panic(errors.Errorf("couldn't split range correctly for '%s'", label))
	}

	loInt := MustParseInt(split[0])
	hiInt := MustParseInt(split[1])

	return &loInt, &hiInt
}

func getPrincetonFloat(root *goquery.Document, label string) *float64 {
	value := getPrincetonItem(root, label)
	if value == nil {
		return nil
	}

	valFloat := MustParseFloat64(TrimFormattedNumber(*value))
	return &valFloat
}
