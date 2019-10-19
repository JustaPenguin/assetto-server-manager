package main

type RaceOut struct {
	Extras           []RaceOutExtra   `json:"extras"`
	NumberOfSessions int              `json:"number_of_sessions"`
	Players          []RaceOutPlayer  `json:"players"`
	Sessions         []RaceOutSession `json:"sessions"`
	Track            string           `json:"track"`
}

type RaceOutSession struct {
	BestLaps   []RaceOutBestLap `json:"bestLaps"`
	Duration   int              `json:"duration"`
	Event      int              `json:"event"`
	Laps       []RaceOutLaps    `json:"laps"`
	LapsCount  int              `json:"lapsCount"`
	Lapstotal  []int            `json:"lapstotal"`
	Name       string           `json:"name"`
	RaceResult []int            `json:"raceResult"`
	Type       int              `json:"type"`
}

type RaceOutLaps struct {
	Car     int    `json:"car"`
	Cuts    int    `json:"cuts"`
	Lap     int    `json:"lap"`
	Sectors []int  `json:"sectors"`
	Time    int    `json:"time"`
	Tyre    string `json:"tyre"`
}

type RaceOutBestLap struct {
	Car  int `json:"car"`
	Lap  int `json:"lap"`
	Time int `json:"time"`
}

type RaceOutPlayer struct {
	Car  string `json:"car"`
	Name string `json:"name"`
	Skin string `json:"skin"`
}

type RaceOutExtra struct {
	Name string `json:"name"`
	Time int    `json:"time"`
}
