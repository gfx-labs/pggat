package strutil

import "testing"

type MatcherTestCase struct {
	Matcher  Matcher
	Haystack string
	Expected bool
}

var matcherTestCases = []MatcherTestCase{
	{
		Matcher:  "a*b",
		Haystack: "abb",
		Expected: true,
	},
	{
		Matcher:  "a*b",
		Haystack: "abbbb",
		Expected: true,
	},
	{
		Matcher:  "a*b",
		Haystack: "abbbc",
		Expected: false,
	},
	{
		Matcher:  "a*b",
		Haystack: "bab",
		Expected: false,
	},
	{
		Matcher:  "a*b",
		Haystack: "ab",
		Expected: true,
	},
	{
		Matcher:  "*abc",
		Haystack: "testabc",
		Expected: true,
	},
	{
		Matcher:  "*abc",
		Haystack: "gfdgfdgfdgfdgiret",
		Expected: false,
	},
	{
		Matcher:  "test*",
		Haystack: "foobar",
		Expected: false,
	},
	{
		Matcher:  "test*",
		Haystack: "testing",
		Expected: true,
	},
	{
		Matcher:  "*potatoe*",
		Haystack: "i like potatoes so much",
		Expected: true,
	},
	{
		Matcher:  "*potatoe*",
		Haystack: "potato",
		Expected: false,
	},
	{
		Matcher:  "abc",
		Haystack: "abc",
		Expected: true,
	},
	{
		Matcher:  "a**",
		Haystack: "a",
		Expected: true,
	},
	{
		Matcher:  "a*a*",
		Haystack: "aa",
		Expected: true,
	},
	{
		Matcher:  "*_ro",
		Haystack: "uniswap",
		Expected: false,
	},
	{
		Matcher:  "*_ro",
		Haystack: "uniswap_ro",
		Expected: true,
	},
	{
		Matcher:  "",
		Haystack: "",
		Expected: true,
	},
	{
		Matcher:  "",
		Haystack: "abc",
		Expected: false,
	},
}

func TestMatcher(t *testing.T) {
	for _, c := range matcherTestCases {
		if c.Matcher.Matches(c.Haystack) != c.Expected {
			t.Errorf("expected %s match %s to be %v", c.Matcher, c.Haystack, c.Expected)
		}
	}
}
