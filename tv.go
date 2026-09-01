package torrentparser

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// Some of these more complex regexes are adapted from https://github.com/TheBeastLT/parse-torrent-title
	// Season ranges (ie, S01-S03) - must have two capture groups to denote the start and end of the range
	seasonRange1 = regexp.MustCompile(`(?i)(?:complete\W|(?:seasons|series)?\W|\W|^)(?:s(\d{1,2})[, +/\\&-]+)+s(\d{1,2})\b`)
	seasonRange2 = regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?(?:seasons?|[Сс]езони?|temporadas?)[. ]?[-:]?[. ]?[([]?(?:(?:(\d{1,2})[., /\\&-]+)+(\d{1,2})\b)[)\]]?`)
	seasonRange3 = regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?\bseasons?\b[. -]?S?(\d{1,2})[. -]?(?:to|thru|and|\+|:)[. -]?(?:s?)(\d{1,2})\b`) // two capture groups

	// Season list matches a substring list of seasons (ie, 1,2,3,4,5)
	seasonList = regexp.MustCompile(`(?i)(?:(?:\bthe\W)?\bcomplete\W)?(?:seasons?|[Сс]езони?|temporadas?)[. ]?[-:]?[. ]?[([]?((?:\d{1,2}[., /\\&]+)+\d{1,2}\b)[)\]]?`)

	seasonGeneral  = regexp.MustCompile(`(?i)[^\w]S([0-9]{1,2})(?: ?E[0-9]{1,2})?`)
	seasonSaison   = regexp.MustCompile(`(?i)(?:\(?Saison|Season)[. _-]?([0-9]{1,2})`)
	seasonX        = regexp.MustCompile(`(?i)[^\d]+([0-9]{1,2})x[0-9]{1,2}[^\d]+`)
	episodeGeneral = regexp.MustCompile(`(?i)S[0-9]{1,2} ?E([0-9]{1,2})`)
	episodeSeason  = regexp.MustCompile(`(?i)\(Season \d+\) ([0-9]{1,3})\s`)
	episodeEpisode = regexp.MustCompile(`(?i)[ée]p(?:isode)?[. _-]?([0-9]{1,3})`)
	episodeAnime   = regexp.MustCompile(`(?i)- ([0-9]{1,3}) (?:\[|\()`)
	episodeX       = regexp.MustCompile(`(?i)[0-9]{1,2}x([0-9]{1,2})`)

	// Episode ranges (ie, S01E01-E05, E01-05) and multi-episode packs (ie, S01E01E02E03)
	episodeRange = regexp.MustCompile(`(?i)(?:s\d{1,2}[. _-]?)?e(\d{1,3})[. _-]*(?:-|to|thru|&|\+)[. _-]*e?(\d{1,3})`)
	episodeMulti = regexp.MustCompile(`(?i)(?:e\d{2,3}[. _-]?){2,}`)
	episodeNum   = regexp.MustCompile(`(?i)e(\d{1,3})`)
)

func (p *parser) GetSeasons() []int {
	// Try identify season ranges before individually defined seasons/single seasons
	seasonList := p.FindString("seasonList", seasonList, FindStringOptions{})
	if seasonList != "" {
		seasons := potentialSeasonListToInts(seasonList)
		if seasons != nil {
			return seasons
		}
	}

	for _, seasonRangeRX := range []*regexp.Regexp{seasonRange1, seasonRange2, seasonRange3} {
		seasons := p.FindNumbers("seasonRange", seasonRangeRX, FindNumbersOptions{})
		if seasons != nil && seasons[1] > seasons[0] {
			return intRange(seasons[0], seasons[1])
		}
	}

	season := p.FindNumbers("seasonGeneral", seasonGeneral, FindNumbersOptions{NilValue: nil})
	if season != nil {
		return season
	}
	season = p.FindNumbers("seasonSaison", seasonSaison, FindNumbersOptions{NilValue: nil})
	if season != nil {
		return season
	}
	return p.FindNumbers("seasonX", seasonX, FindNumbersOptions{NilValue: nil})
}

func (p *parser) GetEpisode() int {
	episode := p.FindNumber("episode", episodeGeneral, FindNumberOptions{})
	if episode != 0 {
		return episode
	}
	episode = p.FindNumber("episode", episodeEpisode, FindNumberOptions{})
	if episode != 0 {
		return episode
	}
	episode = p.FindNumber("episode", episodeSeason, FindNumberOptions{})
	if episode != 0 {
		return episode
	}
	episode = p.FindNumber("episode", episodeAnime, FindNumberOptions{})
	if episode != 0 {
		return episode
	}
	return p.FindNumber("episode", episodeX, FindNumberOptions{})
}

// getEpisodes returns every episode a name refers to: an inclusive range
// (S01E01-E05), a multi-episode pack (S01E01E02E03), or the single episode
// already parsed. It takes the single value to avoid re-running the stateful
// episode matcher.
func (p *parser) getEpisodes(single int) []int {
	if m := episodeRange.FindStringSubmatch(p.Name); m != nil {
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		if end > start {
			return intRange(start, end)
		}
	}
	if run := episodeMulti.FindString(p.Name); run != "" {
		var eps []int
		for _, e := range episodeNum.FindAllStringSubmatch(run, -1) {
			if n, err := strconv.Atoi(e[1]); err == nil {
				eps = append(eps, n)
			}
		}
		if len(eps) > 1 {
			return eps
		}
	}
	if single > 0 {
		return []int{single}
	}
	return nil
}

// potentialSeasonListToInts attempts to parse a season list separated by an unknown
// delimiter into a slice of ints.
// All expected delimiters are replaced with a pipe, then the string is split on the pipe
// to normalize inconsistent use of delimiters (ie, 1, 2 & 3).
func potentialSeasonListToInts(l string) []int {
	r := strings.NewReplacer(
		",", "|",
		".", "|",
		" ", "|",
		"/", "|",
		"\\", "|",
		"&", "|",
	)

	seasonParts := strings.Split(r.Replace(l), "|")
	seasons := make([]int, 0)
	for _, seasonPart := range seasonParts {
		if seasonPart == "" {
			continue
		}
		season, err := strconv.Atoi(seasonPart)
		if err != nil {
			return nil
		}
		seasons = append(seasons, season)
	}
	return seasons
}

// intRange returns a slice of integers from s to e inclusive.
func intRange(s, e int) []int {
	var r []int
	for i := s; i <= e; i++ {
		r = append(r, i)
	}
	return r
}
