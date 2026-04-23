package airlabs

import (
	"strings"
	"testing"
)

func TestParseSuggestResponseDocShape(t *testing.T) {
	body := []byte(`{
		"airports": [{
			"country_code": "BZ",
			"iata_code": "SQS",
			"lng": -89.017821,
			"city": "San Ignacio",
			"timezone": "America/Belize",
			"name": "Matthew Spain Airport",
			"city_code": "SQS",
			"lat": 17.185045
		}],
		"airports_by_cities": [{
			"icao_code": "TTPP",
			"country_code": "TT",
			"iata_code": "POS",
			"lng": -61.337951,
			"city": "Port of Spain",
			"timezone": "America/Port_of_Spain",
			"name": "Piarco International Airport",
			"city_code": "POS",
			"lat": 10.59762
		}]
	}`)
	rows, err := parseSuggestResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].IATACode != "SQS" || rows[1].IATACode != "POS" {
		t.Fatalf("order or codes wrong: %#v", rows)
	}
}

func TestParseSuggestResponseWrapped(t *testing.T) {
	inner := `{"airports":[{"name":"Test","iata_code":"TST","icao_code":"","city":"X","country_code":"US","timezone":"UTC","lat":1,"lng":2}]}`
	body := []byte(`{"response":` + inner + `}`)
	rows, err := parseSuggestResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].IATACode != "TST" {
		t.Fatalf("got %#v", rows)
	}
}

func TestNormalizeSuggestQuery(t *testing.T) {
	if normalizeSuggestQuery("ab") != "" {
		t.Fatal("short query should be empty")
	}
	if normalizeSuggestQuery("  kochi  ") != "kochi" {
		t.Fatal()
	}
	long := strings.Repeat("x", 35)
	if len([]rune(normalizeSuggestQuery(long))) != 30 {
		t.Fatalf("got %q", normalizeSuggestQuery(long))
	}
}

func TestSuggestLangParam(t *testing.T) {
	if suggestLangParam("en-US") != "en" {
		t.Fatalf("got %q", suggestLangParam("en-US"))
	}
	if suggestLangParam("") != "" {
		t.Fatal()
	}
	if suggestLangParam("x") != "" {
		t.Fatal()
	}
}
