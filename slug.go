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

// TranslateUmlaut translates the umlaut symbols (ö, ä, ü) as well as ß to a string without the umlaut,
// for example 'ö' --> "oe", 'ß' --> "ss".
func TranslateUmlaut(r rune) (bool, string) {
	switch r {
	case 'Ö':
		return true, "Oe"
	case 'ö':
		return true, "oe"
	case 'Ä':
		return true, "Ae"
	case 'ä':
		return true, "ae"
	case 'Ü':
		return true, "Ue"
	case 'ü':
		return true, "ue"
	case 'ß', 'ẞ':
		return true, "ss"
	default:
		return false, ""
	}
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

func isValidSlugRuneLowerCase(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}

func isValidSlugRuneIgnoreCase(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}

// ValidSlugRuneReplaceFunc accepts all runes that are by default allowed in slugs:
// A-Z, a-z, 0-9, - and _.
func ValidSlugRuneReplaceFunc(r rune) (bool, string) {
	// if r is a valid character return this rune unchanged
	if isValidSlugRuneIgnoreCase(r) {
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
// maxLength is the number of runes in the string, not the number of bytes.
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
// To be more explicit: Each codepoint in cutset will be removed, see strings.Trim.
// For example "-+foo-+" will be transformed to "foo" for the cutset "-+".
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

// GetDefaultPreProcessors returns the default list of pre processors, see SlugGenerator for details.
// The result will contain: IgnoreInvalidUTF8, normalization to NKFC, transforming the string
// to lowercase codepoints.
//
// Note: There is no guarantee that these processor will always remain the same, it's probable that new ones
// might be added, even in the same major version (which shouldn't be a problem for most applications).
func GetDefaultPreProcessors() []StringModifierFunc {
	return getDefaultPreProcessorsWithForm(norm.NFKC, true)
}

func getDefaultProcessorsWithConfig(replaceBy string, firstActions ...StringModifierFunc) []StringModifierFunc {
	res := make([]StringModifierFunc, len(firstActions), len(firstActions)+1)
	copy(res, firstActions)

	defaultFunc := RuneHandleFuncToStringModifierFunc(ChainRuneHandleFuncs(
		NewSpaceReplacerFunc(replaceBy),
		ReplaceDashAndHyphens,
		TranslateUmlaut,
		ValidSlugRuneReplaceFunc,
	))

	res = append(res, defaultFunc)
	return res
}

// GetDefaultProcessors returns th default list of processors, see SlugGenerator for details.
// The result will contain: Replace spaces by "-", replace dashes and hyphens by "-",
// translate umlauts, finally keep only the default set of codepoints and drop all others
// (see ValidSlugRuneReplaceFunc).
//
// Note: There is no guarantee that these processor will always remain the same, it's probable that new ones
// might be added, even in the same major version (which shouldn't be a problem for most applications).
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

// GetDefaultFinalizers returns the default list of finalizers, see SlugGenerator for details.
// The result will contain: replace multiple occurrences of "-" by a single one, trim leading and tailing "-".
//
// Note: There is no guarantee that these processor will always remain the same, it's probable that new ones
// might be added, even in the same major version (which shouldn't be a problem for most applications).
func GetDefaultFinalizers() []StringModifierFunc {
	return getDefaultFinalizersWithConfig('-', -1)
}

// SlugGenerator is the type that actually creates all slugs.
//
// The conversion input --> slug is split up into three faces:
// Preprocessing, processing and postprocessing (PreProcessor, Processor and Finalizer).
//
// The idea behind this is to make it easier to add your own modifiers at "the right moment".
// The idea is that the string passes through all three phases, each doing something different:
//
// The pre processor prepares the string to be actually processed later.
// This by default includes: Remove invalid UTF-8 codepoints from the string, normalize the string
// to NKFC (see https://blog.golang.org/normalization) and making the string lower case.
//
// The string is then prepared to be actually be processed: Replacements can assume that the string is valid
// and everything in it is lowercase.
//
// The main phase thus is responsible to create a slug form, i.e. use only valid slug codepoints, replace
// strings etc.
// By default this processing phase will do the following: Replace all spaces (" ", newline etc.)
// by "-", replace all dash symbols (for example the UTF-8 ― by "-", they're different codepoints),
// translate umlauts like 'ä' --> "ae" or "ß" --> "ss", then drop everything that is not a valid slug codepoint.
//
// After that the string is finalized and converted to a "normal form".
// By default this includes that all occurrences of more than one "-" are replaced by a single
// "-" and the removal of all leading / trailing "-".
//
// There are different ways to modify the slug generator, see the project homepage at
// https://github.com/FabianWe/goslugify and the examples in Configure.
//
// Of course you can also completely write your own generator by just setting all three parts yourself,
// but then you should know what you're doing.
//
// An important note: It is not guaranteed that this order remains consistent over all versions of this package!
// Probably only more functionality will be added, but I don't make any promises that this doesn't change!
//
// As a rule: If you need slugs for example in a database to identify objects store the slug,
// don't rely on the slug generator to for example compute the same slug again and again for the same
// input.
type SlugGenerator struct {
	PreProcessor StringModifierFunc
	Processor    StringModifierFunc
	Finalizer    StringModifierFunc
}

// GenerateSlug generates a slug by performing all three phases.
func (gen *SlugGenerator) GenerateSlug(in string) string {
	in = gen.PreProcessor(in)
	in = gen.Processor(in)
	in = gen.Finalizer(in)
	return in
}

// Modify is not really required, but as a fact SlugGenerator also implements StringModifier.
func (gen *SlugGenerator) Modify(in string) string {
	return gen.GenerateSlug(in)
}

// NewEmptySlugGenerator without any processing steps, this should only be used if you want
// to implement your own workflow without any of the defaults.
func NewEmptySlugGenerator() *SlugGenerator {
	return &SlugGenerator{
		PreProcessor: nil,
		Processor:    nil,
		Finalizer:    nil,
	}
}

// NewDefaultSlugGenerator returns a slug generator that consists of the components
// as returned by GetDefaultPreProcessors, GetDefaultProcessors and GetDefaultFinalizers.
func NewDefaultSlugGenerator() *SlugGenerator {
	return &SlugGenerator{
		PreProcessor: ChainStringModifierFuncs(GetDefaultPreProcessors()...),
		Processor:    ChainStringModifierFuncs(GetDefaultProcessors()...),
		Finalizer:    ChainStringModifierFuncs(GetDefaultFinalizers()...),
	}
}

// WithPreProcessor adds a new pre processor to the generator.
// Note: If you plan to add a lot of processor it's probably better to append to GetDefaultPreProcessors
// and then chain all entries yourself.
func (gen *SlugGenerator) WithPreProcessor(modifier StringModifierFunc) *SlugGenerator {
	return &SlugGenerator{
		PreProcessor: ChainStringModifierFuncs(modifier, gen.PreProcessor),
		Processor:    gen.PreProcessor,
		Finalizer:    gen.Finalizer,
	}
}

// WithPreProcessor adds a new processor to the generator.
// Note: If you plan to add a lot of processor it's probably better to append to GetDefaultProcessors
// and then chain all entries yourself.
func (gen *SlugGenerator) WithProcessor(modifier StringModifierFunc) *SlugGenerator {
	return &SlugGenerator{
		PreProcessor: gen.PreProcessor,
		Processor:    ChainStringModifierFuncs(modifier, gen.Processor),
		Finalizer:    gen.Finalizer,
	}
}

// WithFinalizer adds a new finalizer to the generator.
// Note: If you plan to add a lot of finalizers it's probably better to append to GetDefaultFinalizers
// and then chain all entries yourself.
func (gen *SlugGenerator) WithFinalizer(modifier StringModifierFunc) *SlugGenerator {
	return &SlugGenerator{
		PreProcessor: gen.PreProcessor,
		Processor:    gen.Processor,
		Finalizer:    ChainStringModifierFuncs(modifier, gen.Finalizer),
	}
}

// SlugConfig gives an easy way to build a customized slug generator.
//
// It allows customization (instead of just using the global GenerateSlug function), but doesn't
// require a complete setup where you have to define a whole workflow for yourself.
//
// For most use cases this customization should be sufficient.
// Just create a NewSlugConfig (this gives you the same config as the global function uses),
// set any of the fields on the config and then use Configure to create a SlugGenerator with the given
// settings.
//
// The following fields can be adjusted:
//
// TruncateLength: If set to a value > 0 this is the maximal length that the slug is allowed to have,
// smart truncating is used to truncate the string. If you want more details about truncating have a look at
// NewTruncateFunc. Note that this is the number of runes in th string, not the number of bytes.
//
// WordSeparator defines which codepoint should be used to separate words in the string.
// For example: "foo bar" --> "foo-bar". Also multiple occurrences of this codepoint will be stripped,
// e.g. "foo--bar" --> "foo-bar". If the string has leading or trailing '-' separators they will be trimmed,
// e.g. "-foo-bar-" --> "foo-bar".
// This codepoint will also determine where a word begins / ends for truncating the string
// (if TruncateLength > 0, again see NewTruncateFunc).
//
// Form defines the UTF-8 normal form to use, the default should do in most cases, however
// see https://blog.golang.org/normalization.
//
// ReplaceMaps can be used to add your own custom replacers. They could for example contain
// language specific replacements. See MergeStringReplaceMaps how multiple maps are merged.
// This replacement takes place right after the pre processors, so they're the first step after
// the pre processing.
//
// ToLower is by default set to true and the whole string is transformed to all lowercase codepoints
// in th pre processing phase.
type SlugConfig struct {
	TruncateLength int
	WordSeparator  rune
	Form           norm.Form
	ReplaceMaps    []StringReplaceMap
	ToLower        bool
}

// NewSlugConfig returns the default config that is used by the global GenerateSlug function,
// just change the fields you want to customize and call Configure.
func NewSlugConfig() *SlugConfig {
	return &SlugConfig{
		TruncateLength: -1,
		WordSeparator:  '-',
		Form:           norm.NFKC,
		ReplaceMaps:    nil,
		ToLower:        true,
	}
}

// AddReplaceMap add a new replace map to the back of the ReplaceMaps list.
func (config *SlugConfig) AddReplaceMap(m StringReplaceMap) {
	config.ReplaceMaps = append(config.ReplaceMaps, m)
}

// GetPhases returns the modifiers described by this config.
// You can use this function if you want to add custom modifiers by your own.
func (config *SlugConfig) GetPhases() (pre, processors, final []StringModifierFunc) {
	pre = getDefaultPreProcessorsWithForm(config.Form, config.ToLower)

	// first merge all maps into one
	replaceMap := MergeStringReplaceMaps(config.ReplaceMaps...)
	// if there is at least one entry we create a replacer and pass it in getDefaultProcessorsWithConfig
	// this replacer will substitute all occurrences, not just whole words
	if len(replaceMap) > 0 {
		constReplacer := NewConstantReplacerFromMap(replaceMap)
		processors = getDefaultProcessorsWithConfig(string(config.WordSeparator), ToStringHandleFunc(constReplacer))
	} else {
		processors = getDefaultProcessorsWithConfig(string(config.WordSeparator))
	}

	final = getDefaultFinalizersWithConfig(config.WordSeparator, config.TruncateLength)
	return
}

// Configure creates a SlugGenerator from the given config.
func (config *SlugConfig) Configure() *SlugGenerator {
	pre, processors, finalizers := config.GetPhases()

	return &SlugGenerator{
		PreProcessor: ChainStringModifierFuncs(pre...),
		Processor:    ChainStringModifierFuncs(processors...),
		Finalizer:    ChainStringModifierFuncs(finalizers...),
	}
}

// GetValidator returns a function that validates if a string is a valid slug according to this specification.
// All slugs generated by Configure().GenerateSlug should by valid slugs.
//
// One note though: As already mentioned there is not guaranteed that the specification might not change.
// I think in general this type should not change much, but even in the same major release new fields might be
// added. It should not be anything breaking the code of other people or change the default behavior, I'm just saying
// that the output should be used more for testing, not some hard coded assertions.
//
// Also: This function only validates the options present in the config, if you added other modifiers yourself
// they, of course, will not be checked.
//
// Also note that the replacement maps are not checked, i.e. it is not checked if a word within s could have been
// replaced with the replacement maps. It's more or less a syntax test, not a semantic test.
func (config *SlugConfig) GetValidator() func(s string) bool {
	return func(s string) bool {
		// we test all requirements as given in config, let's start with some easy tests:
		// must contain only valid utf8
		if !utf8.ValidString(s) {
			return false
		}

		// the length must be in bounds
		if config.TruncateLength > 0 {
			if count := utf8.RuneCountInString(s); count > config.TruncateLength {
				return false
			}
		}

		// if the normal form is valid: it must be in that normal form
		switch config.Form {
		case norm.NFC, norm.NFD, norm.NFKC, norm.NFKD:
			if !config.Form.IsNormalString(s) {
				return false
			}
		}

		// string is not allowed to start or end with - (or whatever that rune is)
		sepAsString := string(config.WordSeparator)
		if strings.HasPrefix(s, sepAsString) || strings.HasSuffix(s, sepAsString) {
			return false
		}

		// now test: only valid runes are contained, taking into account if lower is set
		// no multiple occurrences of - (or whatever the separator is)
		// note: we already checked that s doesn't start / end with it
		isLastSep := false

		for _, r := range s {
			if config.ToLower {
				// make sure it is a lower case rune
				if !isValidSlugRuneLowerCase(r) {
					return false
				}
			} else {
				if !isValidSlugRuneIgnoreCase(r) {
					return false
				}
			}

			// if it is -, the character before it is not allowed to be -
			if r == config.WordSeparator {
				// if last was this too --> error
				if isLastSep {
					return false
				}
				isLastSep = true
			} else {
				isLastSep = false
			}
		}

		return true
	}
}

var defaultConfig = NewSlugConfig()
var defaultGenerator = defaultConfig.Configure()
var defaultValidator = defaultConfig.GetValidator()

// GenerateSlug generates a new slug containing only valid slug codepoints.
//
// This returns a slug that should be good enough for most cases, if you want to configure the
// returned slug, for example set a max length, you can customize a SlugGenerator.
// See documentation there.
func GenerateSlug(in string) string {
	return defaultGenerator.GenerateSlug(in)
}

// IsSlug tests if s is a valid slug.
//
// Note: The result of this function can be used for testing etc.
// But it should not be used like a hard-coded assertion in a live system.
// For details see SlugConfig.GetValidator.
func IsSlug(s string) bool {
	return defaultValidator(s)
}
