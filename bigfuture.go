package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

type BigFuture struct {
	// ticker is the rate limiter for all requests.
	ticker <-chan time.Time
}

func NewBigFuture(rateLimit time.Duration) *BigFuture {
	// this leaks tickers, but doesn't matter because these are long-lived objects.
	return &BigFuture{
		ticker: time.Tick(rateLimit),
	}
}

func (bf *BigFuture) FetchInfo(c *College) (err error) {
	url := fmt.Sprintf("https://bigfuture.collegeboard.org/college-university-search/print-college-profile?id=%d", c.BigFutureID)

	<-bf.ticker // rate limit
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
			err = errors.Errorf("bigFuture parse error for %d: %v", c.BigFutureID, panicked)
		}
	}()

	// lock the college
	c.Lock()
	defer c.Unlock()

	if facts := bf.infoBlock(doc, "Quick Facts"); facts != nil {
		DefaultInt(&c.NumUndergrads, facts.propertyInt("Total undergraduates"))
		DefaultFloat(&c.InStateTuition, facts.propertyFloat("In-State Tuition"))
		DefaultFloat(&c.OutOfStateTuition, facts.propertyFloat("Out-of-State Tuition"))
	}

	if schoolType := bf.infoBlock(doc, "Type of School"); schoolType != nil {
		DefaultInt(&c.SATCode, schoolType.propertyInt("College Board Code"))

		// get ownership
		if len(schoolType.descriptors) >= 2 {
			ownership := strings.TrimSpace(schoolType.descriptors[1])
			if ownership == "Public" {
				DefaultString(&c.Ownership, &ownership)
			} else if ownership == "Private" {
				DefaultString(&c.Ownership, &ownership)
			}
		}
	}

	if admission := bf.infoBlock(doc, "Admission"); admission != nil {
		DefaultDeadline(&c.StandardDeadline, admission.propertyDeadline("Regular application due"))
		DefaultDeadline(&c.StandardNotification, admission.propertyDeadline("College will notify student of admission"))
	}

	if early := bf.infoBlock(doc, "Early Decision and Action"); early != nil {
		DefaultDeadline(&c.EarlyDeadline, early.propertyDeadline("Early action application due"))
		DefaultDeadline(&c.EarlyNotification, early.propertyDeadline("College will notify student of early action admission by"))
	}

	if actComposite := bf.infoBlock(doc, "ACT Composite"); actComposite != nil {
		DefaultFloat(&c.ACTComposite_30_36, actComposite.propertyFloat("30 - 36"))
		DefaultFloat(&c.ACTComposite_24_29, actComposite.propertyFloat("24 - 29"))
		DefaultFloat(&c.ACTComposite_18_23, actComposite.propertyFloat("18 - 23"))
		DefaultFloat(&c.ACTComposite_12_17, actComposite.propertyFloat("12 - 17"))
	}

	if actMath := bf.infoBlock(doc, "ACT Math"); actMath != nil {
		DefaultFloat(&c.ACTMath_30_36, actMath.propertyFloat("30 - 36"))
		DefaultFloat(&c.ACTMath_24_29, actMath.propertyFloat("24 - 29"))
		DefaultFloat(&c.ACTMath_18_23, actMath.propertyFloat("18 - 23"))
		DefaultFloat(&c.ACTMath_12_17, actMath.propertyFloat("12 - 17"))
	}

	if actEnglish := bf.infoBlock(doc, "ACT English"); actEnglish != nil {
		DefaultFloat(&c.ACTEnglish_30_36, actEnglish.propertyFloat("30 - 36"))
		DefaultFloat(&c.ACTEnglish_24_29, actEnglish.propertyFloat("24 - 29"))
		DefaultFloat(&c.ACTEnglish_18_23, actEnglish.propertyFloat("18 - 23"))
		DefaultFloat(&c.ACTEnglish_12_17, actEnglish.propertyFloat("12 - 17"))
	}

	if gpa := bf.infoBlock(doc, "GPAs of incoming freshmen"); gpa != nil {
		DefaultFloat(&c.GPA_375_plus, gpa.propertyFloat("3.75+"))
		DefaultFloat(&c.GPA_350_374, gpa.propertyFloat("3.5 - 3.74"))
		DefaultFloat(&c.GPA_325_349, gpa.propertyFloat("3.25 - 3.49"))
		DefaultFloat(&c.GPA_300_324, gpa.propertyFloat("3.00 - 3.24"))
		DefaultFloat(&c.GPA_250_299, gpa.propertyFloat("2.50 - 2.99"))
	}

	return nil
}

type bigFutureInfoBlock struct {
	blockName   string // for debugging, mainly
	descriptors []string
}

func (bf *BigFuture) infoBlock(root *goquery.Document, headerText string) *bigFutureInfoBlock {
	subhead := root.Find("td>h2").FilterFunction(func(i int, el *goquery.Selection) bool {
		return strings.ToLower(el.Text()) == strings.ToLower(headerText)
	})

	if subhead.Length() != 1 {
		return nil
	}

	descriptorEls := subhead.SiblingsFiltered("p")
	descriptors := make([]string, descriptorEls.Length())
	descriptorEls.Each(func(i int, el *goquery.Selection) {
		descriptors[i] = el.Text()
	})

	return &bigFutureInfoBlock{
		blockName:   subhead.Text(),
		descriptors: descriptors,
	}
}

func (ib *bigFutureInfoBlock) property(prop string) *string {
	for _, desc := range ib.descriptors {
		split := strings.Split(desc, ":")
		if len(split) > 2 {
			panic(errors.Errorf("line with multiple ':': '%s'", desc))
		} else if len(split) < 2 {
			continue // ignore the line
		}

		key := strings.ToLower(strings.TrimSpace(split[0]))
		value := strings.TrimSpace(split[1])

		if key == strings.ToLower(prop) {
			return &value
		}
	}

	return nil // Key does not exist.
}

func (ib *bigFutureInfoBlock) propertyDeadline(prop string) *Deadline {
	dStrP := ib.property(prop)
	if dStrP == nil {
		return nil
	}
	dStr := *dStrP

	if strings.HasPrefix(dStr, "No ") || dStr == "--" {
		// e.g. "No regular app...". Not "Nov"
		return nil
	}

	deadline, err := ParseDeadline("Jan _2", dStr)
	if err != nil {
		panic(errors.Wrap(err, "couldn't parse deadline"))
	}
	return deadline

}

func (ib *bigFutureInfoBlock) propertyFloat(prop string) *float64 {
	value := ib.property(prop)
	if value == nil {
		return nil
	}

	valFloat := MustParseFloat64(TrimFormattedNumber(*value))
	return &valFloat
}

func (ib *bigFutureInfoBlock) propertyInt(prop string) *int {
	value := ib.property(prop)
	if value == nil {
		return nil
	}

	valInt := MustParseInt(TrimFormattedNumber(*value))
	return &valInt
}
