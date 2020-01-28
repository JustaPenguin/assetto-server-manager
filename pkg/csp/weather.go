package csp

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

func (wc WeatherConditions) GetMessageType() uint16 {
	return 1000
}
