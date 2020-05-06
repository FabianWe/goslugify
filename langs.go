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

const (
	LanguageEnglish = "en"
	LanguageGerman  = "de"
)

// EnglishReplaceDict contains replacers for "@" ("at") and "&" ("and").
var EnglishReplaceDict = map[string]string{
	"@": "at",
	"&": "end",
}

// GermanReplaceDict contains replacers for "@" ("at") and "&" ("und").
var GermanReplaceDict = map[string]string{
	"@": "at",
	"&": "und",
}

var languageMaps = make(map[string]StringReplaceMap, 2)

func init() {
	languageMaps[LanguageEnglish] = EnglishReplaceDict
	languageMaps[LanguageGerman] = GermanReplaceDict
}

// AddLanguageMap adds a new language to the global language map store.
// This store can be used for language specific replacements.
//
// Supported languages right now are "en" (English) and "de" (German).
func AddLanguageMap(language string, m StringReplaceMap) {
	languageMaps[language] = m
}

// GetLanguageMap returns a StringReplaceMap for a given list of langugages.
// All maps for the specific languages are merged with MergeStringReplaceMaps.
// If a language doesn't exist the entry will be ignored.
//
// Supported languages right now are "en" (English) and "de" (German).
func GetLanguageMap(languages ...string) StringReplaceMap {
	mapList := make([]StringReplaceMap, 0, len(languages))
	for _, l := range languages {
		if m, has := languageMaps[l]; has {
			mapList = append(mapList, m)
		}
	}
	return MergeStringReplaceMaps(mapList...)
}
