package config

// Site represents a paragliding launch site.
type Site struct {
	Name      string
	Direction [2]string // [from, to] compass point names, e.g. ["SSW", "WSW"]
	Lat       float64
	Lon       float64
}

// Sites is the list of all configured launch sites.
var Sites = []Site{
	{Name: "Balberget Ramp", Direction: [2]string{"SSW", "WSW"}, Lat: 63.94344038093285, Lon: 19.046277812311036},
	{Name: "Balberget Stuga", Direction: [2]string{"ESE", "SSE"}, Lat: 63.94013281904288, Lon: 19.122045235179684},
	{Name: "Tavelsjö", Direction: [2]string{"ENE", "E"}, Lat: 64.01453496664449, Lon: 20.0159563},
	{Name: "Storuman", Direction: [2]string{"SE", "NW"}, Lat: 64.96104228812447, Lon: 17.69696781869336},
	{Name: "Dalsberget", Direction: [2]string{"ENE", "ESE"}, Lat: 62.91695106970932, Lon: 18.466744719924737},
	{Name: "Dundret", Direction: [2]string{"NW", "ENE"}, Lat: 67.11411862249734, Lon: 20.588067722234612},
	{Name: "Kittelfjäll", Direction: [2]string{"ESE", "SSE"}, Lat: 65.25436262582429, Lon: 15.487933185539914},
	{Name: "Klutmarksbacken", Direction: [2]string{"SSW", "WSW"}, Lat: 64.72117147014961, Lon: 20.782167371833356},
}

// MaxGusts is the maximum safe wind gust speed in km/h.
// Exposed so other packages can reference it if needed.
const MaxGusts = 25 // km/h
