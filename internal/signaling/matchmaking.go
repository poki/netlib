package signaling

import "strings"

// regionNeighbors lists the fallback regions to try, in order, after the player's region.
var regionNeighbors = map[string][]string{
	"eu-west":  {"eu-north", "eu-east"},
	"eu-north": {"eu-west", "eu-east"},
	"eu-east":  {"eu-west", "eu-north"},
	"ma":       {"eu-west"},

	"na-west":    {"na-central", "na-east"},
	"na-central": {"na-east", "na-west"},
	"na-east":    {"na-central", "na-west"},
	"us-ak":      {"na-west"},
	"us-hi":      {"na-west"},
	"us":         {"na-central", "na-east", "na-west"},
	"ca":         {"na-central", "na-east", "na-west"},

	"asia-east":      {"asia-southeast", "cn"},
	"asia-southeast": {"asia-east"},
	"asia-south":     {"asia-southeast"},
	"cn":             {"asia-east"},

	// In November 2026, eu-west and gulf tied on median latency from Pakistan.
	// Western Europe had the lower p90.
	"pk": {"eu-west", "gulf", "asia-southeast"},

	// Colombia, Ecuador, Peru and Venezuela had lower median and p90 latencies
	// to eastern North America than to southern South America.
	"sa-north": {"na-east", "sa-south"},
	"sa-south": {"sa-north"},
}

// matchmakingRegion returns the primary matchmaking region for a given country and subdivision.
func matchmakingRegion(country, subdivision string) string {
	country = strings.ToUpper(strings.TrimSpace(country))
	if len(country) != 2 || country == "XX" || country[0] < 'A' || country[0] > 'Z' || country[1] < 'A' || country[1] > 'Z' {
		return "eu-west" // This is our default fallback region.
	}

	subdivision = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(subdivision)), country+"-")

	switch country {
	case "US", "CA":
		switch subdivision {
		case "AK", "HI": // Alaska and Hawaii have their own regions.
			if country == "US" {
				return strings.ToLower(country + "-" + subdivision)
			}
		case "WA", "OR", "CA", "NV", "AZ", "UT", "ID", "MT", "WY", "CO", "NM", "BC", "AB", "YT":
			return "na-west" // Western North America.
		case "ND", "SD", "NE", "KS", "OK", "TX", "MN", "IA", "MO", "AR", "LA", "WI", "IL", "MI", "IN", "MS", "TN", "KY", "MB", "SK", "NT", "NU":
			return "na-central" // Central North America.
		case "ME", "NH", "VT", "MA", "CT", "RI", "NY", "NJ", "PA", "DE", "MD", "DC", "VA", "WV", "NC", "SC", "GA", "FL", "AL", "OH", "ON", "QC", "NB", "NS", "PE", "NL":
			return "na-east" // Eastern North America.
		}
		return strings.ToLower(country)
	case "AT", "BE", "CH", "DE", "ES", "FR", "GB", "IE", "IT", "LI", "LU", "MC", "NL", "PT", "AD", "SM", "VA", "GG", "IM", "JE":
		return "eu-west" // Western Europe.
	case "DK", "EE", "FI", "IS", "LT", "LV", "NO", "SE", "FO", "AX":
		return "eu-north" // Northern Europe.
	case "AL", "BA", "BG", "BY", "CZ", "GR", "HR", "HU", "MD", "ME", "MK", "PL", "RO", "RS", "SI", "SK", "UA", "XK":
		return "eu-east" // Eastern Europe.
	case "HK", "JP", "KR", "MO", "TW":
		return "asia-east" // Eastern Asia.
	case "BN", "ID", "KH", "LA", "MM", "MY", "PH", "SG", "TH", "TL", "VN":
		return "asia-southeast" // Southeast Asia.
	case "BD", "BT", "IN", "LK", "NP":
		return "asia-south" // South Asia.
	case "AU", "NZ":
		return "oceania" // Oceania.
	case "AR", "BR", "CL", "PY", "UY":
		return "sa-south" // Southern South America.
	case "BO", "CO", "EC", "PE", "VE", "GY", "SR":
		return "sa-north" // Northern South America.
	case "AE", "BH", "KW", "OM", "QA", "SA":
		return "gulf" // Gulf region.
	case "BW", "LS", "NA", "SZ", "ZA", "ZW":
		return "africa-south" // Southern Africa.
	}

	// Countries outside the shared pools use their country code as the matchmaking region.
	return strings.ToLower(country)
}
