package importer

import "strings"

// countryISO2 maps API-Football country / national-team names to ISO-3166
// alpha-2 codes (the bundled /flags/<iso2>.svg names). UK home nations use
// the flag-icons sub-national names. Missing entries just render the
// code chip, so this list favours football nations over completeness.
var countryISO2 = map[string]string{
	// UEFA
	"albania": "al", "andorra": "ad", "armenia": "am", "austria": "at", "azerbaijan": "az",
	"belarus": "by", "belgium": "be", "bosnia and herzegovina": "ba", "bosnia & herzegovina": "ba",
	"bulgaria": "bg", "croatia": "hr", "cyprus": "cy", "czech republic": "cz", "czechia": "cz",
	"denmark": "dk", "england": "gb-eng", "estonia": "ee", "faroe islands": "fo", "finland": "fi",
	"france": "fr", "georgia": "ge", "germany": "de", "gibraltar": "gi", "greece": "gr",
	"hungary": "hu", "iceland": "is", "israel": "il", "italy": "it", "kazakhstan": "kz",
	"kosovo": "xk", "latvia": "lv", "liechtenstein": "li", "lithuania": "lt", "luxembourg": "lu",
	"malta": "mt", "moldova": "md", "monaco": "mc", "montenegro": "me", "netherlands": "nl",
	"north macedonia": "mk", "northern ireland": "gb-nir", "norway": "no", "poland": "pl",
	"portugal": "pt", "republic of ireland": "ie", "ireland": "ie", "romania": "ro", "russia": "ru",
	"san marino": "sm", "scotland": "gb-sct", "serbia": "rs", "slovakia": "sk", "slovenia": "si",
	"spain": "es", "sweden": "se", "switzerland": "ch", "turkey": "tr", "türkiye": "tr", "turkiye": "tr",
	"ukraine": "ua", "wales": "gb-wls",
	// CONMEBOL
	"argentina": "ar", "bolivia": "bo", "brazil": "br", "chile": "cl", "colombia": "co",
	"ecuador": "ec", "paraguay": "py", "peru": "pe", "uruguay": "uy", "venezuela": "ve",
	// CONCACAF
	"canada": "ca", "costa rica": "cr", "cuba": "cu", "curacao": "cw", "curaçao": "cw",
	"el salvador": "sv", "guatemala": "gt", "haiti": "ht", "honduras": "hn", "jamaica": "jm",
	"mexico": "mx", "nicaragua": "ni", "panama": "pa", "trinidad and tobago": "tt",
	"usa": "us", "united states": "us", "united states of america": "us",
	// AFC
	"australia": "au", "bahrain": "bh", "china": "cn", "china pr": "cn", "india": "in",
	"indonesia": "id", "iran": "ir", "ir iran": "ir", "iraq": "iq", "japan": "jp", "jordan": "jo",
	"korea republic": "kr", "south korea": "kr", "korea dpr": "kp", "north korea": "kp",
	"kuwait": "kw", "lebanon": "lb", "malaysia": "my", "oman": "om", "palestine": "ps",
	"qatar": "qa", "saudi arabia": "sa", "syria": "sy", "thailand": "th", "united arab emirates": "ae",
	"uzbekistan": "uz", "vietnam": "vn",
	// CAF
	"algeria": "dz", "angola": "ao", "benin": "bj", "burkina faso": "bf", "cameroon": "cm",
	"cape verde": "cv", "cape verde islands": "cv", "congo": "cg", "congo dr": "cd", "dr congo": "cd",
	"egypt": "eg", "equatorial guinea": "gq", "gabon": "ga", "gambia": "gm", "ghana": "gh",
	"guinea": "gn", "guinea-bissau": "gw", "ivory coast": "ci", "côte d'ivoire": "ci", "cote d'ivoire": "ci",
	"kenya": "ke", "libya": "ly", "madagascar": "mg", "mali": "ml", "mauritania": "mr", "morocco": "ma",
	"mozambique": "mz", "namibia": "na", "nigeria": "ng", "senegal": "sn", "south africa": "za",
	"sudan": "sd", "tanzania": "tz", "togo": "tg", "tunisia": "tn", "uganda": "ug", "zambia": "zm",
	"zimbabwe": "zw",
	// OFC
	"new zealand": "nz", "fiji": "fj", "papua new guinea": "pg", "solomon islands": "sb", "tahiti": "pf",
}

// CountryISO2 returns the ISO-3166 alpha-2 (or flag-icons sub-national) code
// for a country / national-team name, or "" when unknown.
func CountryISO2(name string) string {
	return countryISO2[strings.ToLower(strings.TrimSpace(name))]
}
