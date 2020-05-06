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

package goslugify_test

import (
	"fmt"
	"github.com/FabianWe/goslugify"
)

func ExampleMergeStringReplaceMaps() {
	m1 := map[string]string{
		"a": "b",
	}
	// the 'a' in this mapping will be ignored (because it exists already in m1)
	// 'a' will be replaced by "b", even though m2 maps 'b' --> "c"
	m2 := map[string]string{
		"a": "fooo",
		"b": "c",
		"c": "d",
	}
	combined := goslugify.MergeStringReplaceMaps(m1, m2)
	modifier := goslugify.NewConstantReplacerFromMap(combined)
	fmt.Println(modifier.Modify("abcde"))
	// Output: bcdde
}

func ExampleConstantReplacer() {
	replacer := goslugify.NewConstantReplacer("foo", "bar")
	fmt.Println(replacer.Modify("hello foo bar"))
	// Output: hello bar bar
}

func ExampleNewConstantReplacerFromMap() {
	replacer := goslugify.NewConstantReplacerFromMap(map[string]string{
		"42": "21",
	})
	fmt.Println(replacer.Modify("42 is only half the truth"))
	// Output: 21 is only half the truth
}

func ExampleWordReplacer() {
	replacer := goslugify.NewWordReplacer(map[string]string{
		"@": "at",
	}, "-")
	fmt.Println(replacer.Modify("something@-@"))
	// Output: something@-at
}

func ExampleNewRuneHandleFuncFromMap() {
	f := goslugify.NewRuneHandleFuncFromMap(map[rune]string{
		'€': "euro",
		'$': "dollar",
	})
	// without KeepAllFunc every codepoint except € and $ would be dropped
	funcs := goslugify.ChainRuneHandleFuncs(f, goslugify.KeepAllFunc)
	modifier := goslugify.RuneHandleFuncToStringModifierFunc(funcs)
	fmt.Println(modifier("The USA use $ and Germany uses €"))
	// Output: The USA use dollar and Germany uses euro
}

func ExampleChainRuneHandleFuncs() {
	funcs := goslugify.ChainRuneHandleFuncs(goslugify.NewSpaceReplacerFunc("-"),
		goslugify.ValidSlugRuneReplaceFunc,
	)
	modifier := goslugify.RuneHandleFuncToStringModifierFunc(funcs)
	fmt.Println(modifier("!!hello world!!"))
	// Output: hello-world
}

func ExampleValidSlugRuneReplaceFunc() {
	modifier := goslugify.RuneHandleFuncToStringModifierFunc(goslugify.ValidSlugRuneReplaceFunc)
	fmt.Println(modifier("abc!09?€§ABZ-_"))
	// Output: abc09ABZ-_
}

func ExampleNewReplaceMultiOccurrencesFunc() {
	f := goslugify.NewReplaceMultiOccurrencesFunc('-')
	fmt.Println(f("foo--bar---hello"))
	// Output: foo-bar-hello
}

func ExampleNewTruncateFunc() {
	f := goslugify.NewTruncateFunc(5, "-")
	fmt.Println(f("foo-bar"))
	// Output: foo
}

func ExampleNewTruncateFunc_second() {
	f := goslugify.NewTruncateFunc(6, "+")
	fmt.Println(f("thisisaverylongword+foo"))
	// Output: thisis
}

func ExampleNewTruncateFunc_third() {
	// this is one of the "weird" cases mentioned
	f := goslugify.NewTruncateFunc(5, "-")
	fmt.Println(f("-a--foo"))
	// Output: -a-
}

func ExampleNewTrimFunc() {
	f := goslugify.NewTrimFunc("-_")
	fmt.Println(f("----hello---_"))
	// Output: hello
}

func ExampleSlugGenerator() {
	slugGenerator := goslugify.NewDefaultSlugGenerator()
	fmt.Println(slugGenerator.GenerateSlug("foo bar -- hello"))
	// Output: foo-bar-hello
}

func ExampleGenerateSlug() {
	fmt.Println(goslugify.GenerateSlug("Hello World!! Isn't this amazing?"))
	// Output: hello-world-isnt-this-amazing
}

func ExampleSlugConfig() {
	config := goslugify.NewSlugConfig()
	// set the allowed max length to 11
	config.TruncateLength = 11
	// don't convert to lower case
	config.ToLower = false
	generator := config.Configure()
	fmt.Println(generator.GenerateSlug("Hello World! Isn't this amazing?"))
	// Output: Hello-World
}

func ExampleSlugConfig_second() {
	config := goslugify.NewSlugConfig()
	config.AddReplaceMap(map[string]string{
		"world": "moon",
	})
	generator := config.Configure()
	fmt.Println(generator.GenerateSlug("Hello World!!!"))
	// Output: hello-moon
}

func ExampleGetLanguageMap() {
	config := goslugify.NewSlugConfig()
	config.AddReplaceMap(goslugify.GetLanguageMap("de"))
	generator := config.Configure()
	fmt.Println(generator.GenerateSlug("Aragorn & Arwen"))
	// Output: aragorn-und-arwen
}
