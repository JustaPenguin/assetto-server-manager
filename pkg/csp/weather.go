package csp

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

type Weather uint8

const (
	LightThunderstorm Weather = iota
	Thunderstorm
	HeavyThunderstorm
	LightDrizzle
	Drizzle
	HeavyDrizzle
	LightRain
	Rain
	HeavyRain
	LightSnow
	Snow
	HeavySnow
	LightSleet
	Sleet
	HeavySleet
	Clear
	FewClouds
	ScatteredClouds
	BrokenClouds
	OvercastClouds
	Fog
	Mist
	Smoke
	Haze
	Sand
	Dust
	Squalls
	Tornado
	Hurricane
	Cold
	Hot
	Windy
	Hail
)

var AvailableWeathers = []Weather{
	LightThunderstorm,
	Thunderstorm,
	HeavyThunderstorm,
	LightDrizzle,
	Drizzle,
	HeavyDrizzle,
	LightRain,
	Rain,
	HeavyRain,
	LightSnow,
	Snow,
	HeavySnow,
	LightSleet,
	Sleet,
	HeavySleet,
	Clear,
	FewClouds,
	ScatteredClouds,
	BrokenClouds,
	OvercastClouds,
	Fog,
	Mist,
	Smoke,
	Haze,
	Sand,
	Dust,
	Squalls,
	Tornado,
	Hurricane,
	Cold,
	Hot,
	Windy,
	Hail,
}

type WeatherConditions struct {
	Timestamp   uint64
	Current     Weather
	Next        Weather
	Transition  float32
	TimeToApply float32
}

func (wc WeatherConditions) ToRaceMessage() (string, error) {
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.BigEndian, wc)

	if err != nil {
		return "", err
	}

	encoding := base64.StdEncoding

	enc := encoding.EncodeToString(buf.Bytes())

	return fmt.Sprintf("\t\t\t\t$CSP0:%s", enc), nil
}
