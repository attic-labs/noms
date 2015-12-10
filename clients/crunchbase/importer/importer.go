package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"strings"

	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/tealeg/xlsx"
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/clients/util"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/dataset"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

const (
	// Change this version when the output of this importer changes given the same input
	codeVersion = uint32(1)

	permalink       = "permalink"
	name            = "name"
	homepageUrl     = "homepage_url"
	market          = "market"
	fundingTotalUsd = "fundingTotalUsd"
	status          = "status"
	countryCode     = "country_code"
	stateCode       = "state_code"
	region          = "region"
	city            = "city"
	fundingRounds   = "funding_rounds"
	foundedAt       = "founded_at"
	firstFundingAt  = "first_funding_at"
	lastFundingAt   = "last_funding_at"

	companyPermalink      = "company_permalink"
	companyName           = "company_name"
	companyCategoryList   = "company_category_list"
	companyMarket         = "company_market"
	companyCountryCode    = "company_country_code"
	companyState          = "company_state"
	companyRegion         = "company_region"
	companyCity           = "company_city"
	fundingRoundPermalink = "funding_round_permalink"
	fundingRoundType      = "funding_round_type"
	fundingRoundCode      = "funding_round_code"
	fundedAt              = "funded_at"
	raisedAmountUsd       = "raised_amount_usd"
)

var (
	date1904 = false
	dsFlags  = dataset.NewFlags()
)

func main() {
	var ds *dataset.Dataset

	flag.Usage = func() {
		fmt.Println("Usage: crunchbase [options] url\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	httpClient := util.CachingHttpClient()
	ds = dsFlags.CreateDataset()
	fmt.Println(ds.Store().Root())
	url := flag.Arg(0)
	if httpClient == nil || ds == nil || url == "" {
		flag.Usage()
		return
	}
	defer ds.Close()

	fmt.Println("Fetching excel file - this can take a minute or so...")
	resp, err := httpClient.Get(url)
	d.Exp.NoError(err)
	defer resp.Body.Close()

	tempFile, err := ioutil.TempFile(os.TempDir(), "")
	defer tempFile.Close()
	d.Chk.NoError(err)

	h := sha1.New()
	_, err = io.Copy(io.MultiWriter(h, tempFile), resp.Body)
	d.Chk.NoError(err)

	inputs := InputsDef{codeVersion, ref.FromHash(h).String()}
	if !needsReimport(*ds, inputs.New(ds.Store())) {
		fmt.Println("Excel file hasn't changed since last run, nothing to do.")
		return
	}

	companiesRef := importCompanies(*ds, tempFile.Name())
	imp := ImportDef{
		inputs,
		DateDef{time.Now().Format(time.RFC3339)},
		companiesRef,
	}.New(ds.Store())
	impRef := NewRefOfImport(types.WriteValue(imp, ds.Store()))

	// Commit ref of the companiesRef list
	_, err = ds.Commit(impRef)
	d.Exp.NoError(err)
}

func importCompanies(ds dataset.Dataset, fileName string) ref.Ref {
	fmt.Println("Opening excel file - this can take a minute or so...")

	xlFile, err := xlsx.OpenFile(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(-1)
	}

	date1904 = xlFile.Date1904

	// Read in Rounds and group  according to CompanyPermalink
	roundsByPermalink := map[string][]Round{}
	roundsSheet := xlFile.Sheet["Rounds"]
	roundsColIndexes := readIndexesFromHeaderRow(roundsSheet)
	numRounds := 0
	for i, row := range roundsSheet.Rows {
		if i != 0 {
			round := NewRoundFromRow(ds.Store(), roundsColIndexes, row, i)
			pl := round.CompanyPermalink()
			roundsByPermalink[pl] = append(roundsByPermalink[pl], round)
			numRounds++
		}
	}

	// Read in Companies and map to permalink
	companyRefsDef := MapOfStringToRefOfCompanyDef{}
	companySheet := xlFile.Sheet["Companies"]
	companyColIndexes := readIndexesFromHeaderRow(companySheet)
	for i, row := range companySheet.Rows {
		fmt.Printf("\rImporting %d of %d companies... (%.2f%%)", i, len(companySheet.Rows), float64(i)/float64(len(companySheet.Rows))*float64(100))
		if i != 0 {
			company := NewCompanyFromRow(ds.Store(), companyColIndexes, row, i)
			permalink := company.Permalink()

			rounds := roundsByPermalink[permalink]
			roundRefs := SetOfRefOfRoundDef{}
			for _, r := range rounds {
				ref := types.WriteValue(r, ds.Store())
				roundRefs[ref] = true
			}
			company = company.SetRounds(roundRefs.New(ds.Store()))
			ref := types.WriteValue(company, ds.Store())
			companyRefsDef[company.Permalink()] = ref
		}
	}
	fmt.Println("")

	companyRefs := companyRefsDef.New(ds.Store())

	// Uncomment this line of code once Len() is implemented on compoundLists
	//	fmt.Printf("\rImported %d companies with %d rounds\n", companyRefs.Len(), numRounds)

	// Write the list of companyRefs
	return types.WriteValue(companyRefs, ds.Store())
}

func needsReimport(ds dataset.Dataset, input Inputs) bool {
	if head, ok := ds.MaybeHead(); ok {
		if existing, ok := head.Value().(RefOfImport); ok {
			if existing.TargetValue(ds.Store()).Input().Ref() == input.Ref() {
				return false
			}
		}
	}
	return true
}

func NewCompanyFromRow(cs chunks.ChunkStore, idxs columnIndexes, row *xlsx.Row, rowNum int) Company {
	cells := row.Cells

	company := CompanyDef{
		Permalink:       idxs.getString(permalink, cells),
		Name:            idxs.getString(name, cells),
		HomepageUrl:     idxs.getString(homepageUrl, cells),
		CategoryList:    idxs.getListOfCategory(companyCategoryList, cells),
		Market:          idxs.getString(market, cells),
		FundingTotalUsd: idxs.getFloat(fundingTotalUsd, cells, "Company.FundingTotalUsd", rowNum),
		Status:          idxs.getString(status, cells),
		CountryCode:     idxs.getString(countryCode, cells),
		StateCode:       idxs.getString(stateCode, cells),
		Region:          idxs.getString(region, cells),
		City:            idxs.getString(city, cells),
		FundingRounds:   uint16(idxs.getInt(fundingRounds, cells, "Company.FundingRounds", rowNum)),
		FoundedAt:       idxs.getTimeStamp(foundedAt, cells, "Company.FoundedAt", rowNum),
		FirstFundingAt:  idxs.getTimeStamp(firstFundingAt, cells, "Company.FirstFundingAt", rowNum),
		LastFundingAt:   idxs.getTimeStamp(lastFundingAt, cells, "Company.LastFundingAt", rowNum),
	}
	return company.New(cs)
}

func NewRoundFromRow(cs chunks.ChunkStore, idxs columnIndexes, row *xlsx.Row, rowNum int) Round {
	cells := row.Cells

	round := RoundDef{
		CompanyPermalink:      idxs.getString(companyPermalink, cells),
		FundingRoundPermalink: idxs.getString(fundingRoundPermalink, cells),
		FundingRoundType:      idxs.getString(fundingRoundType, cells),
		FundingRoundCode:      idxs.getString(fundingRoundCode, cells),
		FundedAt:              idxs.getTimeStamp(fundedAt, cells, "Round.fundedAt", rowNum),
		RaisedAmountUsd:       idxs.getFloat(raisedAmountUsd, cells, "Round.raisedAmountUsd", rowNum),
	}
	return round.New(cs)
}

type columnIndexes map[string]int

func readIndexesFromHeaderRow(sheet *xlsx.Sheet) columnIndexes {
	m := map[string]int{}
	for i, cell := range sheet.Rows[0].Cells {
		m[cell.Value] = i
	}
	return m
}

func (cn columnIndexes) getString(key string, cells []*xlsx.Cell) string {
	if cellIndex, ok := cn[key]; ok {
		return cells[cellIndex].Value
	}
	return ""
}

func (cn columnIndexes) getListOfCategory(key string, cells []*xlsx.Cell) SetOfStringDef {
	realElems := SetOfStringDef{}
	if cellIndex, ok := cn[key]; ok {
		s := cells[cellIndex].Value
		elems := strings.Split(s, "|")
		for _, elem := range elems {
			s1 := strings.TrimSpace(elem)
			if s1 != "" {
				realElems[s1] = true
			}
		}
	}
	return realElems
}

func (cn columnIndexes) getFloat(key string, cells []*xlsx.Cell, field string, rowNum int) float64 {
	parsedValue := float64(0)
	if cellIndex, ok := cn[key]; ok && cellIndex < len(cells) {
		var err error
		parsedValue, err = cells[cellIndex].Float()
		if err != nil {
			v := cells[cellIndex].Value
			if v != "" && v != "-" {
				fmt.Fprintf(os.Stderr, "Unable to parse Float, row: %d, field: %s, err: %s\n", rowNum, field, err)
			}
			parsedValue = float64(0)
		}
	}
	return float64(parsedValue)
}

func (cn columnIndexes) getInt(key string, cells []*xlsx.Cell, field string, rowNum int) int {
	parsedValue := 0
	if cellIndex, ok := cn[key]; ok && cellIndex < len(cells) {
		var err error
		parsedValue, err = cells[cellIndex].Int()
		if err != nil {
			v := cells[cellIndex].Value
			if v != "" && v != "-" {
				fmt.Fprintf(os.Stderr, "Unable to parse Int, row: %d, field: %s, err: %s\n", rowNum, field, err)
			}
			parsedValue = 0
		}
	}
	return int(parsedValue)
}

func (cn columnIndexes) getTimeStamp(key string, cells []*xlsx.Cell, field string, rowNum int) int64 {
	if cellIndex, ok := cn[key]; ok {
		v := cells[cellIndex].Value
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return xlsx.TimeFromExcelTime(f, date1904).Unix()
		}
		const shortForm = "2006-01-02"
		var err error
		var t time.Time
		if t, err = time.Parse(shortForm, v); err == nil {
			return t.Unix()
		}
		if v != "" {
			fmt.Fprintf(os.Stderr, "Unable to parse Date, row: %d, field: %s, value: %s, err: %s\n", rowNum, field, v, err)
		}
	}
	return 0
}
