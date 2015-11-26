package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"strings"

	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/tealeg/xlsx"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/types"
)

var date1904 bool = false

func main() {
	var ds *dataset.Dataset

	flag.Usage = func() {
		fmt.Println("Usage: crunchbase [options] file\n")
		flag.PrintDefaults()
	}

	dsFlags := dataset.NewFlags()
	flag.Parse()

	ds = dsFlags.CreateDataset()
	path := flag.Arg(0)
	if ds == nil || path == "" {
		flag.Usage()
		return
	}
	defer ds.Close()

	fmt.Printf("Opening Excel file (this takes a minute or so)...")
	t0 := time.Now()
	xlFile, err := xlsx.OpenFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(-1)
	}
	t1 := time.Now()
	fmt.Println("\rOpened Excel file in: ", t1.Sub(t0))

	date1904 = xlFile.Date1904

	// Read in Rounds and group  according to CompanyPermalink
	roundsByPermalink := map[string][]Round{}
	roundsSheet := xlFile.Sheet["Rounds"]
	numRounds := 0
	for i, row := range roundsSheet.Rows {
		if i != 0 {
			round := NewRoundFromRow(row)
			pl := round.CompanyPermalink()
			roundsByPermalink[pl] = append(roundsByPermalink[pl], round)
			numRounds++
		}
	}

	t2 := time.Now()

	// Read in Companies and map to permalink
	companyRefs := NewMapOfStringToRefOfCompany()
	companySheet := xlFile.Sheet["Companies"]
	for i, row := range companySheet.Rows {
		if i%1000 == 0 {
			fmt.Printf("\rProcessed %d of %d companies...", i, len(companySheet.Rows))
		}
		if i != 0 {
			company := NewCompanyFromRow(row)
			permalink := company.Permalink()

			rounds := roundsByPermalink[permalink]
			roundRefs := NewSetOfRefOfRound()
			for _, r := range rounds {
				ref := types.WriteValue(r, ds.Store())
				roundRefs = roundRefs.Insert(NewRefOfRound(ref))
			}
			company = company.SetRounds(roundRefs)
			ref := types.WriteValue(company, ds.Store())
			refOfCompany := NewRefOfCompany(ref)
			companyRefs = companyRefs.Set(company.Permalink(), refOfCompany)
		}
	}

	t3 := time.Now()
	fmt.Printf("\rRead %d companies in %s\n", len(companySheet.Rows), t3.Sub(t2))
	fmt.Println("Comitting...")

	// Write the list of companyRefs
	companiesRef := types.WriteValue(companyRefs, ds.Store())

	// Commit ref of the companiesRef list
	_, ok := ds.Commit(types.NewRef(companiesRef))
	d.Exp.True(ok, "Could not commit due to conflicting edit")

	fmt.Printf("Done. Imported %d companies with %d rounds\n", companyRefs.Len(), numRounds)
}

func NewCompanyFromRow(row *xlsx.Row) Company {
	cells := row.Cells

	company := CompanyDef{
		Permalink:       cells[0].Value,
		Name:            cells[1].Value,
		HomepageUrl:     cells[2].Value,
		CategoryList:    parseListOfCategory(cells[3].Value),
		Market:          cells[4].Value,
		FundingTotalUsd: parseFloatValue(cells[5], "Company.FundingTotalUsd"),
		Status:          cells[6].Value,
		CountryCode:     cells[7].Value,
		StateCode:       cells[8].Value,
		Region:          cells[9].Value,
		City:            cells[10].Value,
		FundingRounds:   uint16(parseIntValue(cells[11], "Company.FundingRounds")),
		FoundedAt:       parseTimeStamp(cells[12], "Company.FoundedAt"),
		// Skip FoundedMonth: 13
		// Skip FoundedYear:  14
		FirstFundingAt: parseTimeStamp(cells[15], "Company.FirstFundingAt"),
		LastFundingAt:  parseTimeStamp(cells[16], "Company.LastFundingAt"),
	}
	return company.New()
}

func NewRoundFromRow(row *xlsx.Row) Round {
	cells := row.Cells

	var raisedAmountUsd float64
	if len(cells) < 16 {
		fmt.Printf("warning: Found Round with only %d cells - expected 16!\n", len(cells))
		raisedAmountUsd = 0
	} else {
		raisedAmountUsd = parseFloatValue(cells[15], "Round.raisedAmountUsd")
	}

	round := RoundDef{
		CompanyPermalink:      cells[0].Value,
		FundingRoundPermalink: cells[8].Value,
		FundingRoundType:      cells[9].Value,
		FundingRoundCode:      cells[10].Value,
		FundedAt:              parseTimeStamp(cells[11], "Round.fundedAt"),
		// Skip FundedMonth:   12
		// Skip FundedQuarter: 13
		// Skip FundedYear:    14
		RaisedAmountUsd: raisedAmountUsd,
	}
	return round.New()
}

func parseListOfCategory(s string) SetOfStringDef {
	elems := strings.Split(s, "|")
	realElems := SetOfStringDef{}
	for _, elem := range elems {
		s1 := strings.TrimSpace(elem)
		if s1 != "" {
			realElems[s1] = true
		}
	}
	return realElems
}

func parseFloatValue(cell *xlsx.Cell, field string) float64 {
	v := strings.TrimSpace(cell.Value)
	parsedValue := float64(0)
	if v != "" {
		var err error
		parsedValue, err = cell.Float()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Parse failure on field: %s, err: %s\n", field, err)
		}
	}
	return float64(parsedValue)
}

func parseIntValue(cell *xlsx.Cell, field string) int {
	v := strings.TrimSpace(cell.Value)
	parsedValue := 0
	if v != "" {
		var err error
		parsedValue, err = cell.Int()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Parse failure on field: %s, err: %s\n", field, err)
		}
	}
	return int(parsedValue)
}

func parseTimeStamp(cell *xlsx.Cell, field string) int64 {
	if f, err := strconv.ParseFloat(cell.Value, 64); err == nil {
		return xlsx.TimeFromExcelTime(f, date1904).Unix()
	}
	const shortForm = "2006-01-02"
	if t, err := time.Parse(shortForm, cell.Value); err == nil {
		return t.Unix()
	}
	if cell.Value != "" {
		fmt.Fprintf(os.Stderr, "Could not parse field as date: %s, \"%s\"\n", field, cell.Value)
	}
	return 0
}
