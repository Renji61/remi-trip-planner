package airlabs

import "testing"

func TestParseAirlinesResponseWrapped(t *testing.T) {
	body := []byte(`{"response":[{"name":"American Airlines","iata_code":"AA","icao_code":"AAL","country_code":"US"}]}`)
	rows, err := parseAirlinesResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].IATACode != "AA" || rows[0].Name != "American Airlines" {
		t.Fatalf("got %#v", rows)
	}
}

func TestParseAirlinesResponseBareArray(t *testing.T) {
	body := []byte(`[{"name":"Delta Air Lines","iata_code":"DL","icao_code":"DAL","country_code":"US"}]`)
	rows, err := parseAirlinesResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].IATACode != "DL" {
		t.Fatalf("got %#v", rows)
	}
}

func TestNormalizeAirlineSuggestQuery(t *testing.T) {
	if normalizeAirlineSuggestQuery("a") != "" {
		t.Fatal("single char")
	}
	if normalizeAirlineSuggestQuery("  al ") != "al" {
		t.Fatal()
	}
}

func TestIsTwoCharAirlineDesignator(t *testing.T) {
	if !isTwoCharAirlineDesignator("aa") || !isTwoCharAirlineDesignator("A1") {
		t.Fatal("expected true")
	}
	if isTwoCharAirlineDesignator("a") || isTwoCharAirlineDesignator("abc") {
		t.Fatal("expected false")
	}
}
