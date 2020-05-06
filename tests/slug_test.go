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

func TestWordReplacer(t *testing.T) {
	replacer := goslugify.NewWordReplacer(map[string]string{
		"@":   "at",
		"foo": "bar",
		"&":   "and",
	}, "-")
	tests := []struct {
		in       string
		expected string
	}{
		{"", ""},
		{" @", " @"},
		{"@", "at"},
		{"-", "-"},
		{"foo", "bar"},
		{"hello-@-world", "hello-at-world"},
		{"there-&-back-again", "there-and-back-again"},
		{"foo-&-bar-@-go", "bar-and-bar-at-go"},
	}
	for _, tc := range tests {
		got := replacer.Modify(tc.in)
		if got != tc.expected {
			t.Errorf("expected replacement of \"%s\" to be \"%s\" but got \"%s\"",
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

func TestValidSlugRuneReplaceFunc(t *testing.T) {
	tests := []struct {
		in             rune
		expectedBool   bool
		expectedString string
	}{
		{'A', true, "A"},
		{'H', true, "H"},
		{'c', true, "c"},
		{'z', true, "z"},
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

func TestValidateNoBounds(t *testing.T) {
	config := goslugify.NewSlugConfig()
	validator := config.GetValidator()

	tests := []struct {
		in       string
		expected bool
	}{
		{"foo", true},
		{"foo__bar", true},
		{"-foo-bar", false},
		{"-foo-", false},
		{"foo-", false},
		{"foo--bar-42", false},
		{"foo-bar-21-42", true},
		{"", true},
		{"foo-bar@", false},
		{"foo-Bar", false},
		{"-", false},
		{"Foo", false},
		{"foo-Bar", false},
	}
	for _, tc := range tests {
		got := validator(tc.in)
		if got != tc.expected {
			t.Errorf("expected validate(\"%s\") to be %s, but got %s",
				tc.in, strconv.FormatBool(tc.expected), strconv.FormatBool(got))
		}
	}
}

func TestValidateWithBounds(t *testing.T) {
	config := goslugify.NewSlugConfig()
	config.TruncateLength = 5
	config.WordSeparator = '_'
	validator := config.GetValidator()
	tests := []struct {
		in       string
		expected bool
	}{
		{"foo_", false},
		{"_foo", false},
		{"", true},
		{"foo", true},
		{"fo_ba", true},
		{"_", false},
		{"a_b", true},
		{"gophers", false},
		{"fo-ba", true},
	}
	for _, tc := range tests {
		got := validator(tc.in)
		if got != tc.expected {
			t.Errorf("expected validate(\"%s\") to be %s, but got %s",
				tc.in, strconv.FormatBool(tc.expected), strconv.FormatBool(got))
		}
	}
}

func TestValidateWithoutLower(t *testing.T) {
	config := goslugify.NewSlugConfig()
	config.ToLower = false
	validator := config.GetValidator()

	tests := []struct {
		in       string
		expected bool
	}{
		{"foo-bar", true},
		{"-foo", false},
		{"Foo", true},
		{"foo-Bar", true},
	}
	for _, tc := range tests {
		got := validator(tc.in)
		if got != tc.expected {
			t.Errorf("expected validate(\"%s\") to be %s, but got %s",
				tc.in, strconv.FormatBool(tc.expected), strconv.FormatBool(got))
		}
	}
}

func TestTranslateUmlaut(t *testing.T) {
	modifier := goslugify.RuneHandleFuncToStringModifierFunc(goslugify.TranslateUmlaut)
	in := "abcd123öäüÖÄÜßẞ"
	expected := "oeaeueOeAeUessss"
	got := modifier(in)
	if got != expected {
		t.Errorf("expected TranslateUmlaut(\"%s\") to be \"%s\", but got \"%s\"",
			in, expected, got)
	}
}
