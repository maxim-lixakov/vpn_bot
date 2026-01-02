package utils

func GetCountryName(code string, serverName string) string {
	// Если есть serverName, используем его
	if serverName != "" {
		return serverName
	}

	// Маппинг кодов стран на названия
	countryNames := map[string]string{
		"hk": "Hong Kong",
		"kz": "Kazakhstan",
		"us": "United States",
		"ru": "Russia",
		"de": "Germany",
		"fr": "France",
		"gb": "United Kingdom",
		"jp": "Japan",
		"sg": "Singapore",
		"nl": "Netherlands",
		"ch": "Switzerland",
		"se": "Sweden",
		"no": "Norway",
		"dk": "Denmark",
		"fi": "Finland",
		"pl": "Poland",
		"cz": "Czech Republic",
		"at": "Austria",
		"be": "Belgium",
		"ie": "Ireland",
		"es": "Spain",
		"it": "Italy",
		"pt": "Portugal",
		"gr": "Greece",
		"tr": "Turkey",
		"au": "Australia",
		"nz": "New Zealand",
		"ca": "Canada",
		"mx": "Mexico",
		"br": "Brazil",
		"ar": "Argentina",
		"cl": "Chile",
		"co": "Colombia",
		"pe": "Peru",
		"za": "South Africa",
		"eg": "Egypt",
		"ae": "United Arab Emirates",
		"sa": "Saudi Arabia",
		"il": "Israel",
		"in": "India",
		"kr": "South Korea",
		"tw": "Taiwan",
		"th": "Thailand",
		"my": "Malaysia",
		"id": "Indonesia",
		"ph": "Philippines",
		"vn": "Vietnam",
		"cn": "China",
	}

	if name, ok := countryNames[code]; ok {
		return name
	}

	return code
}
