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

type StringModifierFunc func(in string) string

func ChainStringModifierFuncs(funcs ...StringModifierFunc) StringModifierFunc {
	return func(in string) string {
		for _, f := range funcs {
			in = f(in)
		}
		return in
	}
}

type StringModifier interface {
	Modify(in string) string
}

func ToStringHandleFunc(modifier StringModifier) StringModifierFunc {
	return func(in string) string {
		return modifier.Modify(in)
	}
}

type ConstantReplacer struct {
	OldNew   []string
	replacer *strings.Replacer
	once     *sync.Once
}

func NewConstantReplacer(oldnew ...string) *ConstantReplacer {
	var once sync.Once
	return &ConstantReplacer{
		OldNew:   oldnew,
		replacer: nil,
		once:     &once,
	}
}

func NewConstantReplacerFromMap(m map[string]string) *ConstantReplacer {
	oldnew := make([]string, 2*len(m))
	i := 0
	for key, value := range m {
		oldnew[i] = key
		oldnew[i+1] = value
		i += 2
	}
	return NewConstantReplacer(oldnew...)
}

func (replacer *ConstantReplacer) Modify(in string) string {
	replacer.once.Do(func() {
		replacer.replacer = strings.NewReplacer(replacer.OldNew...)
	})
	return replacer.replacer.Replace(in)
}

func IgnoreInvalidUTF8(in string) string {
	return strings.ToValidUTF8(in, "")
}

type UTF8Normalizer struct {
	Form norm.Form
}

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

type RuneHandleFunc func(r rune) (handles bool, to string)

func KeepAllFunc(r rune) (bool, string) {
	return true, string(r)
}

func NewRuneHandleFuncFromMap(m map[rune]string) RuneHandleFunc {
	return func(r rune) (bool, string) {
		res, has := m[r]
		if has {
			return true, res
		}
		return false, ""
	}
}

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

func NewSpaceReplacerFunc(replaceBy string) RuneHandleFunc {
	return func(r rune) (bool, string) {
		if unicode.IsSpace(r) {
			return true, replaceBy
		}
		return false, ""
	}
}

func ValidSlugRuneReplaceFunc(r rune) (bool, string) {
	// if r is a valid character return this rune unchanged
	if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
		return true, string(r)
	}
	return false, ""
}

func ReplaceDashAndHyphens(r rune) (bool, string) {
	if unicode.In(r, unicode.Hyphen, unicode.Dash) {
		return true, "-"
	}
	return false, ""
}

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

// TODO broken
func NewTruncateFunc(maxLength int, wordSep string) StringModifierFunc {
	return func(in string) string {
		if maxLength < 0 {
			return in
		}
		if in == "" {
			return in
		}
		split := strings.Split(in, wordSep)
		//for _, bla := range split {
		//	fmt.Printf("\"%s\"\n", bla)
		//}
		//fmt.Println("joined:", strings.Join(split, wordSep))
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

func NewTrimFunc(cutset string) StringModifierFunc {
	return func(in string) string {
		return strings.Trim(in, cutset)
	}
}

// Workflow: preprocessing: valid, norm, language stuff
// process: ascii only, replace whitespace, dash and hyphens
// finalize, remove multiple slashes, truncate

func GetDefaultPreProcessors() []StringModifierFunc {
	return []StringModifierFunc{
		IgnoreInvalidUTF8,
		strings.ToLower,
		ToStringHandleFunc(NewUTF8Normalizer(norm.NFKC)),
	}
}

func GetDefaultProcessors() []StringModifierFunc {
	return []StringModifierFunc{
		RuneHandleFuncToStringModifierFunc(ChainRuneHandleFuncs(
			NewSpaceReplacerFunc("-"),
			ReplaceDashAndHyphens,
			ValidSlugRuneReplaceFunc,
		)),
	}
}

func GetDefaultFinalizers() []StringModifierFunc {
	return []StringModifierFunc{
		NewReplaceMultiOccurrencesFunc('-'),
		NewTrimFunc("-"),
	}
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

func NewSlugGenerator() *SlugGenerator {
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

var defaultGenerator *SlugGenerator

func init() {
	defaultGenerator = NewDefaultSlugGenerator()
}

func GenerateSlug(in string) string {
	return defaultGenerator.GenerateSlug(in)
}

// TODO is valid function

// TODO should not be modified
// customize: -, multiple maps, truncate length
