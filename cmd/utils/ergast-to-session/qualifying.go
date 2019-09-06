package main

type Qualifying struct {
	MRData struct {
		RaceTable struct {
			Races []struct {
				Circuit struct {
					Location struct {
						Country  string `json:"country"`
						Lat      string `json:"lat"`
						Locality string `json:"locality"`
						Long     string `json:"long"`
					} `json:"Location"`
					CircuitID   string `json:"circuitId"`
					CircuitName string `json:"circuitName"`
					URL         string `json:"url"`
				} `json:"Circuit"`
				QualifyingResults []struct {
					Constructor struct {
						ConstructorID string `json:"constructorId"`
						Name          string `json:"name"`
						Nationality   string `json:"nationality"`
						URL           string `json:"url"`
					} `json:"Constructor"`
					Driver struct {
						Code            string `json:"code"`
						DateOfBirth     string `json:"dateOfBirth"`
						DriverID        string `json:"driverId"`
						FamilyName      string `json:"familyName"`
						GivenName       string `json:"givenName"`
						Nationality     string `json:"nationality"`
						PermanentNumber string `json:"permanentNumber"`
						URL             string `json:"url"`
					} `json:"Driver"`
					Q1       string `json:"Q1"`
					Q2       string `json:"Q2"`
					Q3       string `json:"Q3"`
					Number   string `json:"number"`
					Position string `json:"position"`
				} `json:"QualifyingResults"`
				Date     string `json:"date"`
				RaceName string `json:"raceName"`
				Round    string `json:"round"`
				Season   string `json:"season"`
				Time     string `json:"time"`
				URL      string `json:"url"`
			} `json:"Races"`
			Round  string `json:"round"`
			Season string `json:"season"`
		} `json:"RaceTable"`
		Limit  string `json:"limit"`
		Offset string `json:"offset"`
		Series string `json:"series"`
		Total  string `json:"total"`
		URL    string `json:"url"`
		Xmlns  string `json:"xmlns"`
	} `json:"MRData"`
}
