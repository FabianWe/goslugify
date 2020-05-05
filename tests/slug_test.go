// Copyright 2020 Fabian Wenzelmann <fabianwen@posteo.eu>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"github.com/FabianWe/goslugify"
	"strconv"
	"testing"
)

func TestConstantReplacer(t *testing.T) {
	replacer := goslugify.NewConstantReplacer("foo", "bar", "21", "42")
	// from map and also make to a function
	replacerMap := goslugify.NewConstantReplacerFromMap(map[string]string{
		"foo": "bar",
		"21":  "42",
	})
	replacerMapFunc := goslugify.ToStringHandleFunc(replacerMap)
	tests := []struct {
		in, expected string
	}{
		{"foo", "bar"},
		{"something", "something"},
		{"bla foo", "bla bar"},
		{"bla 21 blubb", "bla 42 blubb"},
		{"foo-bla-21-foo", "bar-bla-42-bar"},
	}
	for _, tc := range tests {
		got := replacer.Modify(tc.in)
		if got != tc.expected {
			t.Errorf("expected replacement of \"%s\" to be \"%s\", but got \"%s\"",
				tc.in, tc.expected, got)
		}
		gotMap := replacerMapFunc(tc.in)
		if gotMap != tc.expected {
			t.Errorf("expected replacement with map of \"%s\" to be \"%s\", but got \"%s\"",
				tc.in, tc.expected, got)
		}
	}
}

func TestIgnoreInvalidUTF8(t *testing.T) {
	invalidBytes := []byte{0xff, 0xfe, 0xfd}
	invalidString := string(invalidBytes)
	tests := []struct {
		in, expected string
	}{
		{"foo bar", "foo bar"},
		{"世界", "世界"},
		{"foo " + invalidString + " bar", "foo  bar"},
	}
	for _, tc := range tests {
		got := goslugify.IgnoreInvalidUTF8(tc.in)
		if got != tc.expected {
			t.Errorf("expected that the removement of invalid utf8 from \"%s\" is \"%s\", but got \"%s\"",
				tc.in, tc.expected, got)
		}
	}
}

func TestSpaceReplacerFunc(t *testing.T) {
	withDash := goslugify.RuneHandleFuncToStringModifierFunc(
		goslugify.ChainRuneHandleFuncs(goslugify.NewSpaceReplacerFunc("-"),
			goslugify.KeepAllFunc))
	withPlus := goslugify.RuneHandleFuncToStringModifierFunc(
		goslugify.ChainRuneHandleFuncs(goslugify.NewSpaceReplacerFunc("+++"),
			goslugify.KeepAllFunc))
	tests := []struct {
		in, expectedDash, expectedPlus string
	}{
		{"foo", "foo", "foo"},
		{"foo bar", "foo-bar", "foo+++bar"},
		{"foo\tbar\n42", "foo-bar-42", "foo+++bar+++42"},
		{"foo  bar", "foo--bar", "foo++++++bar"},
		{" foo ", "-foo-", "+++foo+++"},
	}
	for _, tc := range tests {
		gotDash := withDash(tc.in)
		gotPlus := withPlus(tc.in)

		if gotDash != tc.expectedDash {
			t.Errorf("expected replacement with - in \"%s\" to be \"%s\", but got \"%s\"",
				tc.in, tc.expectedDash, gotDash)
		}

		if gotPlus != tc.expectedPlus {
			t.Errorf("expected replacement with + in \"%s\" to be \"%s\", but got \"%s\"",
				tc.in, tc.expectedPlus, gotPlus)
		}
	}
}

func ValidSlugRuneReplaceFunc(t *testing.T) {
	tests := []struct {
		in             rune
		expectedBool   bool
		expectedString string
	}{
		{'A', true, "A"},
		{'H', true, "H"},
		{'c', true, "c"},
		{'z', true, "Z"},
		{'4', true, "4"},
		{'-', true, "-"},
		{'_', true, "_"},
		{'€', false, ""},
		{'@', false, ""},
		{'世', false, ""},
	}
	for _, tc := range tests {
		gotBool, gotString := goslugify.ValidSlugRuneReplaceFunc(tc.in)
		if gotBool != tc.expectedBool || gotString != tc.expectedString {
			t.Errorf("expected (valid slug) rune replacment of \"%s\" to be (%s, \"%s\"), but got (%s, \"%s\") instead",
				string(tc.in), strconv.FormatBool(tc.expectedBool), tc.expectedString, strconv.FormatBool(gotBool), gotString)
		}
	}
}

func TestReplaceDashAndHyphens(t *testing.T) {
	tests := []struct {
		in       rune
		expected string
	}{
		{'-', "-"},
		{'—', "-"},
		{'⸚', "-"},
		{'－', "-"},
		{'a', ""},
		{'€', ""},
	}
	for _, tc := range tests {
		_, got := goslugify.ReplaceDashAndHyphens(tc.in)
		if got != tc.expected {
			t.Errorf("expected dash and hyphen replacement of \"%s\" to be \"%s\", but got \"%s\"",
				string(tc.in), tc.expected, got)
		}
	}
}

func TestNewReplaceMultiOccurrencesFunc(t *testing.T) {
	withDash := goslugify.NewReplaceMultiOccurrencesFunc('-')
	testsWithDash := []struct {
		in       string
		expected string
	}{
		{"foo-bar", "foo-bar"},
		{"foo bar", "foo bar"},
		{"-", "-"},
		{"---", "-"},
		{"--", "-"},
		{"-foo-", "-foo-"},
		{"foo--bar----------xyz", "foo-bar-xyz"},
		{"42-------21", "42-21"},
	}
	for _, tc := range testsWithDash {
		got := withDash(tc.in)
		if got != tc.expected {
			t.Errorf("expected that multiple '-' removed in \"%s\" leads to \"%s\",but got \"%s\"",
				tc.in, tc.expected, got)
		}
	}

	withPlus := goslugify.NewReplaceMultiOccurrencesFunc('+')
	testsWithPlus := []struct {
		in       string
		expected string
	}{
		{"foo+bar", "foo+bar"},
		{"+", "+"},
		{"foo+++bar", "foo+bar"},
		{"+foo+++++", "+foo+"},
	}
	for _, tc := range testsWithPlus {
		got := withPlus(tc.in)
		if got != tc.expected {
			t.Errorf("expected that multiple '+' removed in \"%s\" leads to \"%s\",but got \"%s\"",
				tc.in, tc.expected, got)
		}
	}
}

func TestTruncateSingleRune(t *testing.T) {
	withTwo := goslugify.NewTruncateFunc(2, "-")
	withFive := goslugify.NewTruncateFunc(5, "-")
	tests := []struct {
		in           string
		expectedTwo  string
		expectedFive string
	}{
		{"foo", "fo", "foo"},
		{"foobar", "fo", "fooba"},
		{"f-b", "f", "f-b"},
		{"f-bar", "f", "f-bar"},
		{"foo-bar", "fo", "foo"},
		{"--f", "-", "--f"},
		{"a-b-c-d", "a", "a-b-c"},
	}
	for _, tc := range tests {
		gotTwo := withTwo(tc.in)
		if gotTwo != tc.expectedTwo {
			t.Errorf("expected truncate(\"-\", 2)(\"%s\") to be \"%s\", but got \"%s\"",
				tc.in, tc.expectedTwo, gotTwo)
		}
		gotFive := withFive(tc.in)
		if gotFive != tc.expectedFive {
			t.Errorf("expected truncate(\"-\", 5)(\"%s\") to be \"%s\", but got \"%s\"",
				tc.in, tc.expectedFive, gotFive)
		}
	}
}

func TestTruncateMultiple(t *testing.T) {
	truncater := goslugify.NewTruncateFunc(5, "++")
	tests := []struct {
		in       string
		expected string
	}{
		{"++", "++"},
		{"foo++bar", "foo"},
		{"f++ba", "f++ba"},
		{"fo++ba", "fo"},
	}
	for _, tc := range tests {
		got := truncater(tc.in)
		if got != tc.expected {
			t.Errorf("expected truncate(\"++\", 5)(\"%s\") to be \"%s\", but got \"%s\"",
				tc.in, tc.expected, got)
		}
	}
}

func TestTruncateMore(t *testing.T) {
	truncater := goslugify.NewTruncateFunc(42, "€")
	in := "abcd€efgh€21€42€€84"
	expected := "abcd€efgh€21€42€€84"
	if got := truncater(in); got != expected {
		t.Errorf("expected truncate(\"€\", 42)(\"%s\") to be \"%s\", but got \"%s\"",
			in, expected, got)
	}
}
