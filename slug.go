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

package goslugify

import (
	"golang.org/x/text/unicode/norm"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// StringReplaceMap describes a replacement that should take place.
// The keys from the map are substituted by the value of that key.
type StringReplaceMap map[string]string

// MergeStringReplaceMaps merges multiple maps into one.
// If an entry appears in more than one map the first occurrence of that key is used,
// that is maps coming later in the chain can only add new keys, not overwrite keys from previous maps.
func MergeStringReplaceMaps(maps ...StringReplaceMap) StringReplaceMap {
	res := make(StringReplaceMap)
	for _, m := range maps {
		for key, value := range m {
			if _, has := res[key]; !has {
				res[key] = value
			}
		}
	}
	return res
}

// StringModifierFunc is any function that takes a string and returns a modified one.
// These function should not have side effects, like changing variables in a closure and must be allowed
// to be called concurrently by multiple go routines.
//
// Such a function takes the whole string in question and modifies it (in contrast to for example a RuneHandleFunc).
// These function make up most of the functionality of this library.
//
// There is also an interface called StringModifier with a similar purpose.
// Such an interface instance can be converted to a modifier function by ToStringHandleFunc.
//
// Note that many of the functions in the go string package are of this form.
type StringModifierFunc func(in string) string

// ChainStringModifierFuncs takes a sequence of modifier functions and returns them as a single function.
// This function will apply all modifiers in the order in which they are given.
func ChainStringModifierFuncs(funcs ...StringModifierFunc) StringModifierFunc {
	return func(in string) string {
		for _, f := range funcs {
			in = f(in)
		}
		return in
	}
}

// StringModifier is an interface for types implementing a modification function.
// They can be converted to a StringModifierFunc with ToStringHandleFunc.
// As a StringModifierFunc implementations should not have side effects and must be safe to be called
// concurrently by multiple go routines.
type StringModifier interface {
	Modify(in string) string
}

// ToStringHandleFunc converts a StringModifier to a StringModifierFunc.
func ToStringHandleFunc(modifier StringModifier) StringModifierFunc {
	return func(in string) string {
		return modifier.Modify(in)
	}
}

// ConstantReplacer is an implementation of StringModifier that replaces all occurrences of a word
// by another word.
// For this the list OldNew is used, it describes pairs (key, value) and must therefor always
// contain an even number of strings.
// See strings.Replacer for more details.
//
// Note that you can append new key/value pairs to an existing replacer, but only before
// Modify is called for the first time.
type ConstantReplacer struct {
	OldNew   []string
	replacer *strings.Replacer
	once     *sync.Once
}

// NewConstantReplacer returns a new replacer given the key/value list.
func NewConstantReplacer(oldnew ...string) *ConstantReplacer {
	var once sync.Once
	return &ConstantReplacer{
		OldNew:   oldnew,
		replacer: nil,
		once:     &once,
	}
}

// NewConstantReplacerFromMap given the replacement strings as a map.
func NewConstantReplacerFromMap(m StringReplaceMap) *ConstantReplacer {
	oldnew := make([]string, 2*len(m))
	i := 0
	for key, value := range m {
		oldnew[i] = key
		oldnew[i+1] = value
		i += 2
	}
	return NewConstantReplacer(oldnew...)
}

// Modify replaces all occurrences in the string with the given key/values.
func (replacer *ConstantReplacer) Modify(in string) string {
	replacer.once.Do(func() {
		replacer.replacer = strings.NewReplacer(replacer.OldNew...)
	})
	return replacer.replacer.Replace(in)
}

// WordReplacer replaces occurrences of words within a string, it implements StringModifier.
// The difference between WordReplacer and ConstantReplacer is that WordReplacer does not replace
// all occurrences, but only complete words.
// A word is defined by splitting the string given the WordSeparator.
//
// So a replacement "@" --> "at" would behave on the string "something@-@" differently:
// ConstantReplacer would return "somethingat-at" and WordReplacer would return "something@-at"
// (given that the separator is "-").
type WordReplacer struct {
	WordMap       StringReplaceMap
	WordSeparator string
}

// NewWordReplacer returns a new replacer given the replacement map and the word separator.
func NewWordReplacer(wordMap StringReplaceMap, wordSeparator string) *WordReplacer {
	return &WordReplacer{
		WordMap:       wordMap,
		WordSeparator: wordSeparator,
	}
}

func (replacer *WordReplacer) Modify(in string) string {
	words := strings.Split(in, replacer.WordSeparator)
	first := true
	var buf strings.Builder
	for _, word := range words {
		replaceBy := word
		if entry, has := replacer.WordMap[word]; has {
			replaceBy = entry
		}
		if first {
			first = false
		} else {
			buf.WriteString(replacer.WordSeparator)
		}
		buf.WriteString(replaceBy)
	}
	return buf.String()
}

// IgnoreInvalidUTF8 is a StringModifierFunc that removes all invalid UTF-8 codepoints from the string.
// It is usually the first modifier called.
func IgnoreInvalidUTF8(in string) string {
	return strings.ToValidUTF8(in, "")
}

// UTF8Normalizer implements StringModifier, the string is normalized according to utf-8 normal forms.
// For details see the norm package https://godoc.org/golang.org/x/text/unicode/norm and this blog post
// https://blog.golang.org/normalization.
//
// The default form used in this package is NFKC.
// It is usually called right after IgnoreInvalidUTF8.
type UTF8Normalizer struct {
	Form norm.Form
}

// NewUTF8Normalizer returns a new normalizer given the form.
func NewUTF8Normalizer(form norm.Form) UTF8Normalizer {
	return UTF8Normalizer{form}
}

func (normalizer UTF8Normalizer) Modify(in string) string {
	switch normalizer.Form {
	case norm.NFC, norm.NFD, norm.NFKC, norm.NFKD:
		return normalizer.Form.String(in)
	default:
		return in
	}
}

// RuneHandleFunc is a function that operates on a single rune of the input string;
// in contrast to a StringModifierFunc.
// The idea is that in one iteration over the input string we can apply several handle function
// on each rune.
//
// The return value should be (true, SOME_STRING) if the handle function accepts this rune and has a
// rule for it. In this case the function says that the rune should be replaced by SOME_STRING.
// It should return (false, "") if the function is not interested in the string.
//
// Many such functions can be chained with ChainRuneHandleFuncs.
// The first function that returns true in such a sequence is then executed and this replacement
// takes place for a certain rune.
//
// Such a rune handle function can then be converted to string modification function with
// RuneHandleFuncToStringModifierFunc.
// All runes for which no function returns true will be ignored! This means if no function
// ever returns true for a rune the rune will be dropped from the string (is considered invalid).
type RuneHandleFunc func(r rune) (handles bool, to string)

// KeepAllFunc can be used to avoid to drop certain runes when a rune handle function is converted to a
// string modifier with RuneHandleFuncToStringModifierFunc.
// If this is the last function chained in a sequence of such functions all runes will be kept.
func KeepAllFunc(r rune) (bool, string) {
	return true, string(r)
}

// NewRuneHandleFuncFromMap performs a replace of a single rune given a pre-defined set of
// replacements.
// This function will return (true, m[r]) for all entries in m.
func NewRuneHandleFuncFromMap(m map[rune]string) RuneHandleFunc {
	return func(r rune) (bool, string) {
		res, has := m[r]
		if has {
			return true, res
		}
		return false, ""
	}
}

// ChainRuneHandleFuncs chains multiple rune functions into one, see documentation of
// RuneHandleFunc.
func ChainRuneHandleFuncs(funcs ...RuneHandleFunc) RuneHandleFunc {
	return func(r rune) (bool, string) {
		for _, handleFunc := range funcs {
			if handles, res := handleFunc(r); handles {
				// first function that handles r
				return true, res
			}
		}
		// no handle found
		return false, ""
	}
}

// RuneHandleFuncToStringModifierFunc converts a rune handle function to a string modifier function,
// usually this is a chained function.
// See RuneHandleFunc documentation for details.
func RuneHandleFuncToStringModifierFunc(runeHandler RuneHandleFunc) StringModifierFunc {
	return func(in string) string {
		var buf strings.Builder
		for _, r := range in {
			if handles, res := runeHandler(r); handles {
				buf.WriteString(res)
			}
		}
		return buf.String()
	}
}

// NewSpaceReplacerFunc returns a rune handle function that accepts all space runes
// (according to unicode.IsSpace) and replaces a space by a pre-defined string.
func NewSpaceReplacerFunc(replaceBy string) RuneHandleFunc {
	return func(r rune) (bool, string) {
		if unicode.IsSpace(r) {
			return true, replaceBy
		}
		return false, ""
	}
}

// ValidSlugRuneReplaceFunc accepts all runes that are by default allowed in slugs:
// A-Z, a-z, 0-9, - and _.
func ValidSlugRuneReplaceFunc(r rune) (bool, string) {
	// if r is a valid character return this rune unchanged
	if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
		return true, string(r)
	}
	return false, ""
}

// ReplaceDashAndHyphens replaces any symbol that is considered a hyphen or a dash
// (according to unicode.Hyphen, unicode.Dash) and replaces it with the rune '-'.
func ReplaceDashAndHyphens(r rune) (bool, string) {
	if unicode.In(r, unicode.Hyphen, unicode.Dash) {
		return true, "-"
	}
	return false, ""
}

// NewReplaceMultiOccurrencesFunc returns a StringModifierFunc that will remove multiple occurrences
// of the same rune.
// For example if the separator is '-' you usually want exactly one '-' to separate word.
// So "foo--bar" should be transformed to "foo-bar".
func NewReplaceMultiOccurrencesFunc(in rune) StringModifierFunc {
	return func(s string) string {
		isLastInRune := false
		var buf strings.Builder
		for _, r := range s {
			if r == in {
				if !isLastInRune {
					buf.WriteRune(r)
					isLastInRune = true
				}
			} else {
				isLastInRune = false
				buf.WriteRune(r)
			}
		}
		return buf.String()
	}
}

// NewTruncateFunc returns a StringModifierFunc that will truncate the string to a given
// maximum length.
//
// It will to a "smart" split, i.e. it will only take whole words and not truncate in the
// middle of a string.
// So NewTruncateFunc(5, "-")("foo-bar") will return only "foo", because "foo-bar" would be
// too long.
// As a special case the first word as defined by wordSep might be truncated in the word if it
// already is too long.
//
// Note: If the string starts with wordSep so may the result, so you might want to trim time string,
// either before or after.
// Also wordSep should not have multiple occurrences, otherwise the result can be a bit "strange".
// See some of the tests if you want to exactly know what I mean.
// In general NewReplaceMultiOccurrencesFunc should be called first.
func NewTruncateFunc(maxLength int, wordSep string) StringModifierFunc {
	return func(in string) string {
		if maxLength < 0 {
			return in
		}
		if in == "" {
			return in
		}
		split := strings.Split(in, wordSep)
		firstRunes := []rune(split[0])
		// if first word is already too long truncate it
		if len(firstRunes) >= maxLength {
			return string(firstRunes[:maxLength])
		}
		// append words while length is still valid
		sepLen := utf8.RuneCountInString(wordSep)
		var buf strings.Builder
		buf.WriteString(split[0])
		currentLen := len(firstRunes)
		for _, word := range split[1:] {
			nextLen := currentLen + sepLen + utf8.RuneCountInString(word)
			if nextLen > maxLength {
				break
			}
			buf.WriteString(wordSep)
			buf.WriteString(word)
			currentLen = nextLen
		}
		return buf.String()
	}
}

// NewTrimFunc returns a new StringModifierFunc that removes all leading and trailing
// occurrences of cutset from a string.
// For example "--foo--" will be transformed to "foo".
func NewTrimFunc(cutset string) StringModifierFunc {
	return func(in string) string {
		return strings.Trim(in, cutset)
	}
}

func getDefaultPreProcessorsWithForm(form norm.Form, toLower bool) []StringModifierFunc {
	res := []StringModifierFunc{
		IgnoreInvalidUTF8,
	}
	switch form {
	case norm.NFC, norm.NFD, norm.NFKC, norm.NFKD:
		res = append(res, ToStringHandleFunc(NewUTF8Normalizer(form)))
	}
	if toLower {
		res = append(res, strings.ToLower)
	}

	return res
}

func GetDefaultPreProcessors() []StringModifierFunc {
	return getDefaultPreProcessorsWithForm(norm.NFKC, true)
}

func getDefaultProcessorsWithConfig(replaceBy string, firstActions ...StringModifierFunc) []StringModifierFunc {
	res := make([]StringModifierFunc, len(firstActions), len(firstActions)+1)
	copy(res, firstActions)

	defaultFunc := RuneHandleFuncToStringModifierFunc(ChainRuneHandleFuncs(
		NewSpaceReplacerFunc(replaceBy),
		ReplaceDashAndHyphens,
		ValidSlugRuneReplaceFunc,
	))

	res = append(res, defaultFunc)
	return res
}

func GetDefaultProcessors() []StringModifierFunc {
	return getDefaultProcessorsWithConfig("-")
}

func getDefaultFinalizersWithConfig(replaceBy rune, truncateLength int) []StringModifierFunc {
	res := []StringModifierFunc{
		NewReplaceMultiOccurrencesFunc(replaceBy),
		NewTrimFunc(string(replaceBy)),
	}
	if truncateLength >= 0 {
		res = append(res, NewTruncateFunc(truncateLength, string(replaceBy)))
	}
	return res
}

func GetDefaultFinalizers() []StringModifierFunc {
	return getDefaultFinalizersWithConfig('-', -1)
}

type SlugGenerator struct {
	PreProcessor StringModifierFunc
	Processor    StringModifierFunc
	Finalizer    StringModifierFunc
}

func (gen *SlugGenerator) GenerateSlug(in string) string {
	in = gen.PreProcessor(in)
	in = gen.Processor(in)
	in = gen.Finalizer(in)
	return in
}

func (gen *SlugGenerator) Modify(in string) string {
	return gen.GenerateSlug(in)
}

func NewEmptySlugGenerator() *SlugGenerator {
	return &SlugGenerator{
		PreProcessor: nil,
		Processor:    nil,
		Finalizer:    nil,
	}
}

func NewDefaultSlugGenerator() *SlugGenerator {
	return &SlugGenerator{
		PreProcessor: ChainStringModifierFuncs(GetDefaultPreProcessors()...),
		Processor:    ChainStringModifierFuncs(GetDefaultProcessors()...),
		Finalizer:    ChainStringModifierFuncs(GetDefaultFinalizers()...),
	}
}

func (gen *SlugGenerator) WithPreProcessor(modifier StringModifierFunc) *SlugGenerator {
	return &SlugGenerator{
		PreProcessor: ChainStringModifierFuncs(modifier, gen.PreProcessor),
		Processor:    gen.PreProcessor,
		Finalizer:    gen.Finalizer,
	}
}

func (gen *SlugGenerator) WithProcessor(modifier StringModifierFunc) *SlugGenerator {
	return &SlugGenerator{
		PreProcessor: gen.PreProcessor,
		Processor:    ChainStringModifierFuncs(modifier, gen.Processor),
		Finalizer:    gen.Finalizer,
	}
}

func (gen *SlugGenerator) WithFinalizer(modifier StringModifierFunc) *SlugGenerator {
	return &SlugGenerator{
		PreProcessor: gen.PreProcessor,
		Processor:    gen.Processor,
		Finalizer:    ChainStringModifierFuncs(modifier, gen.Finalizer),
	}
}

var defaultConfig = NewSlugConfig()
var defaultGenerator = defaultConfig.Configure()

func GenerateSlug(in string) string {
	return defaultGenerator.GenerateSlug(in)
}

type SlugConfig struct {
	TruncateLength int
	WordSeparator  rune
	Form           norm.Form
	ReplaceMaps    []StringReplaceMap
	ToLower        bool
}

func NewSlugConfig() *SlugConfig {
	return &SlugConfig{
		TruncateLength: -1,
		WordSeparator:  '-',
		Form:           norm.NFKC,
		ReplaceMaps:    nil,
		ToLower:        true,
	}
}

func (config *SlugConfig) Configure() *SlugGenerator {
	pre := getDefaultPreProcessorsWithForm(config.Form, config.ToLower)

	// first merge all maps into one
	replaceMap := MergeStringReplaceMaps(config.ReplaceMaps...)
	// if there is at least one entry we create a replacer and pass it in getDefaultProcessorsWithConfig
	// this replacer will substitute all occurrences, not just whole words
	var processors []StringModifierFunc
	if len(replaceMap) > 0 {
		constReplacer := NewConstantReplacerFromMap(replaceMap)
		processors = getDefaultProcessorsWithConfig(string(config.WordSeparator), ToStringHandleFunc(constReplacer))
	} else {
		processors = getDefaultProcessorsWithConfig(string(config.WordSeparator))
	}

	finalizers := getDefaultFinalizersWithConfig(config.WordSeparator, config.TruncateLength)

	return &SlugGenerator{
		PreProcessor: ChainStringModifierFuncs(pre...),
		Processor:    ChainStringModifierFuncs(processors...),
		Finalizer:    ChainStringModifierFuncs(finalizers...),
	}
}

// TODO is valid function?
// should be... making this a method of SlugGenerator would be a bit hard...

// TODO should not be modified
// customize: -, multiple maps, truncate length
// add lower case bool option
// add a config type with a to... method?
