package main

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type College struct {
	sync.Mutex // Lock for when used by multiple threads.

	ID   string `json:"-"`
	Name string `json:"Name"`

	// IDs for the college in various providers
	BigFutureID       int `json:"_big_future_id"`
	PrincetonReviewId int `json:"_princeton_review_id"`

	// Public/private
	Ownership *string `json:"Ownership,omitempty"`

	// From BigFuture: deadlines, # undergrads, tuition
	NumUndergrads        *int      `json:"Num. Undergrads,omitempty"`
	InStateTuition       *float64  `json:"Tuition: In-State,omitempty"`
	OutOfStateTuition    *float64  `json:"Tuition: Out-of-State,omitempty"`
	StandardDeadline     *Deadline `json:"Standard Deadline,omitempty"`
	StandardNotification *Deadline `json:"Standard Notification,omitempty"`
	EarlyDeadline        *Deadline `json:"Early Deadline,omitempty"`
	EarlyNotification    *Deadline `json:"Early Notification,omitempty"`

	// From BigFuture: ACT/GPA breakdowns
	ACTComposite_30_36 *float64 `json:"ACT Composite 30-36,omitempty"`
	ACTComposite_24_29 *float64 `json:"ACT Composite 24-29,omitempty"`
	ACTComposite_18_23 *float64 `json:"ACT Composite 18-23,omitempty"`
	ACTComposite_12_17 *float64 `json:"ACT Composite 12-17,omitempty"`
	ACTMath_30_36      *float64 `json:"ACT Math 30-36,omitempty"`
	ACTMath_24_29      *float64 `json:"ACT Math 24-29,omitempty"`
	ACTMath_18_23      *float64 `json:"ACT Math 18-23,omitempty"`
	ACTMath_12_17      *float64 `json:"ACT Math 12-17,omitempty"`
	ACTEnglish_30_36   *float64 `json:"ACT English 30-36,omitempty"`
	ACTEnglish_24_29   *float64 `json:"ACT English 24-29,omitempty"`
	ACTEnglish_18_23   *float64 `json:"ACT English 18-23,omitempty"`
	ACTEnglish_12_17   *float64 `json:"ACT English 12-17,omitempty"`
	GPA_375_plus       *float64 `json:"GPA 3.75+,omitempty"`
	GPA_350_374        *float64 `json:"GPA 3.50-3.74,omitempty"`
	GPA_325_349        *float64 `json:"GPA 3.25-3.49,omitempty"`
	GPA_300_324        *float64 `json:"GPA 3.00-3.24,omitempty"`
	GPA_250_299        *float64 `json:"GPA 2.50-2.99,omitempty"`

	// From Princeton Review: ACT/GPA ranges, GPA avg, acceptange rate, num applicants.
	NumApplicants  *int     `json:"Num. Applicants,omitempty"`
	AcceptanceRate *float64 `json:"Acceptance Rate,omitempty"`
	GPAAverage     *float64 `json:"GPA Average,omitempty"`
	ACTRangeLow    *int     `json:"ACT Range Low,omitempty"`
	ACTRangeHigh   *int     `json:"ACT Range High,omitempty"`

	// Codes for test score submission
	SATCode *int `json:"SAT Code,omitempty"`
	ACTCode *int `json:"ACT Code,omitempty"`
}

// setters only if fields are empty
func DefaultInt(value **int, x *int) {
	if *value == nil {
		*value = x
	}
}

func DefaultFloat(value **float64, x *float64) {
	if *value == nil {
		*value = x
	}
}

func DefaultDeadline(value **Deadline, x *Deadline) {
	if *value == nil {
		*value = x
	}
}

func DefaultString(value **string, x *string) {
	if *value == nil {
		*value = x
	}
}

const DeadlineFormat = "2006-01-02"

type Deadline struct {
	Month time.Month
	Day   int
}

func ParseDeadline(format string, dateStr string) (*Deadline, error) {
	date, err := time.Parse(format, dateStr)
	if err != nil {
		return nil, err
	}
	return &Deadline{
		Month: date.Month(),
		Day:   date.Day(),
	}, nil
}

func (d *Deadline) NextDate() time.Time {
	date := time.Date(time.Now().Year(), time.Month(d.Month), d.Day, 0, 0, 0, 0, time.Local)

	// Scoot the date a year forward if it's in the past.
	if date.Before(time.Now()) {
		date = date.AddDate(1, 0, 0) // 1 year, 0 months, 0 days
	}

	return date
}

func (d *Deadline) MarshalJSON() ([]byte, error) {
	date := d.NextDate()
	dateStr := date.Format(DeadlineFormat)
	buf, err := json.Marshal(dateStr)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (d *Deadline) UnmarshalJSON(b []byte) error {
	var dateStr string
	if err := json.Unmarshal(b, &dateStr); err != nil {
		return err
	}

	date, parseErr := time.Parse(DeadlineFormat, dateStr)
	if parseErr != nil {
		return errors.Wrap(parseErr, "unmarshal deadline:")
	}
	d.Month = date.Month()
	d.Day = date.Day()

	return nil
}
