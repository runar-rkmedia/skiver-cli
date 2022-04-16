package cmd

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/runar-rkmedia/go-common/logger"
	"github.com/runar-rkmedia/skiver/importexport"
	"github.com/runar-rkmedia/skiver/types"
	"github.com/runar-rkmedia/skiver/utils"
)

func BuildTranslationKeyFromApi(api Api, l logger.AppLogger, projectKeyLike, localeLike string) map[string]map[string]string {
	buf := bytes.Buffer{}
	err := api.Export(projectKeyLike, "raw", localeLike, &buf)
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to get exported project")
	}
	var ep types.ExtendedProject
	err = json.Unmarshal(buf.Bytes(), &ep)
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to unmarshal exported project")
	}
	m, err := FlattenExtendedProject(ep, []string{localeLike})
	if err != nil {
		l.Fatal().Err(err).Msg("Failed to flatten exported project")
	}
	if len(m) == 0 {
		l.Fatal().Msg("Found no matches")
	}
	return m
}

func buildTranslationKeyRegexFromMap(sorted []string) *regexp.Regexp {
	reg := `(.*")(`
	var regexKeys = make([]string, len(sorted))
	i := 0
	for _, k := range sorted {
		r := regexp.QuoteMeta(k)
		regexKeys[i] = r
		i++
	}
	reg += strings.Join(regexKeys, "|") + `)("(?: as any)?)(.*)`
	return regexp.MustCompile(reg)

}

// Creates a flattened map of translationKeys
// The source can either be a file (i18next), or it will fallback to getting from the api
func buildTranslationMapWithRegex(l logger.AppLogger, fromSourceFile *os.File, api Api, project, locale string) (map[string]struct{}, *regexp.Regexp) {
	translationKeys := map[string]struct{}{}
	// var r1 *regexp.Regexp
	if fromSourceFile != nil {
		b, err := ioutil.ReadAll(fromSourceFile)
		if err != nil {
			l.Fatal().Err(err).Msg("Failed to read from source")
		}
		var j map[string]interface{}
		if err := json.Unmarshal(b, &j); err != nil {
			l.Fatal().Err(err).Msg("Failed to unmarshal source")
		}
		flat := Flatten(j)
		// strip context
		for k := range flat {
			ts := strings.Split(k, ".")
			lastTs := ts[len(ts)-1]
			key, _ := importexport.SplitTranslationAndContext(lastTs, "_")
			joined := strings.Join(append(ts[:len(ts)-1], key), ".")
			translationKeys[joined] = struct{}{}
		}
	} else {
		mm := BuildTranslationKeyFromApi(api, l, project, locale)
		for k := range mm {
			translationKeys[k] = struct{}{}
		}
	}

	sorted := utils.SortedMapKeys(translationKeys)
	regex := buildTranslationKeyRegexFromMap(sorted)
	return translationKeys, regex
}

// Flatten takes a map and returns a new one where nested maps are replaced
// by dot-delimited keys.
func Flatten(m map[string]interface{}) map[string]interface{} {
	o := make(map[string]interface{})
	for k, v := range m {
		switch child := v.(type) {
		case map[string]interface{}:
			nm := Flatten(child)
			for nk, nv := range nm {
				o[k+"."+nk] = nv
			}
		default:
			o[k] = v
		}
	}
	return o
}

var newLineReplacer = strings.NewReplacer("\n", "", "\r", "")

func FlattenExtendedProject(ep types.ExtendedProject, locales []string) (map[string]map[string]string, error) {
	m := map[string]map[string]string{}
	if ep.CategoryTree.Translations != nil {
		c := ep.CategoryTree
		for _, t := range ep.CategoryTree.Translations {
			key := c.Key + "." + t.Key
			if c.Key == "" {
				key = t.Key
			}
			for _, tv := range t.Values {
				loc := matchesLocale(ep.Locales[tv.LocaleID], locales)
				if loc == "" {
					continue
				}
				mm := map[string]string{}
				if tv.Value != "" {
					mm[loc] = tv.Value
				}
				for k, c := range tv.Context {
					mm[loc+"_"+k] = c
				}

				if len(mm) == 0 {
					continue
				}
				m[key] = mm
			}
		}
	}
	for _, c := range ep.CategoryTree.Categories {
		for _, t := range c.Translations {
			key := c.Key + "." + t.Key
			if c.Key == "" {
				key = t.Key
			}
			for _, tv := range t.Values {
				loc := matchesLocale(ep.Locales[tv.LocaleID], locales)
				if loc == "" {
					continue
				}
				mm := map[string]string{}
				if tv.Value != "" {
					mm[loc] = tv.Value
				}
				for k, c := range tv.Context {
					mm[loc+"_"+k] = c
				}

				if len(mm) == 0 {
					continue
				}
				m[key] = mm

			}
		}
		mk, err := FlattenExtendedProject(types.ExtendedProject{CategoryTree: c, Locales: ep.Locales}, locales)
		if err != nil {
			return m, err
		}
		for k, v := range mk {
			m[k] = v
		}

	}

	return m, nil
}
func matchesLocale(l types.Locale, locales []string) string {
	for _, loc := range locales {
		if l.ID == loc {
			return loc
		}
		if l.IETF == loc {
			return loc
		}
		if l.Iso639_3 == loc {
			return loc
		}
		if l.Iso639_2 == loc {
			return loc
		}
		if l.Iso639_1 == loc {
			return loc
		}

	}

	return ""
}
