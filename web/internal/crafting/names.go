package crafting

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// namesJSON holds the EN-US label of every item that appears in a recipe,
// generated from the ao-bin-dumps formatted dump by data/gen.
//
//go:embed data/names.json
var namesJSON []byte

// names maps a tiered item id to its game label. The enchant decoration is
// collapsed before the lookup, because T4_METALBAR_LEVEL3 is still a
// "Steel Bar" in game.
var names map[string]string

// familyNames are the tier-neutral display names of the refined materials,
// which are the ingredients nearly every recipe uses.
var familyNames = map[string]string{
	"CLOTH":    "Cloth",
	"LEATHER":  "Leather",
	"METALBAR": "Metal Bar",
	"PLANKS":   "Planks",
}

// tierAdjectives are stripped from a label to get a tier-neutral family name,
// e.g. "Adept's Startouched Crystal" -> "Startouched Crystal".
var tierAdjectives = []string{
	"Beginner's ",
	"Novice's ",
	"Journeyman's ",
	"Adept's ",
	"Expert's ",
	"Master's ",
	"Grandmaster's ",
	"Elder's ",
}

func init() {
	if err := json.Unmarshal(namesJSON, &names); err != nil {
		panic(fmt.Sprintf("crafting: parse embedded names: %v", err))
	}
}

// Name returns the game label of a recipe item, falling back to the craftable
// item list and finally to the raw unique name, so an item added by a patch
// never renders blank.
func Name(uniqueName string) string {
	key := nameKey(uniqueName)
	if label, ok := names[key]; ok {
		return label
	}
	if item, ok := Lookup(uniqueName); ok {
		return item.Label
	}
	if item, ok := Lookup(key); ok {
		return item.Label
	}
	return uniqueName
}

// FamilyName returns the tier-neutral name of the ingredient column an item
// belongs to: T4_METALBAR and T6_METALBAR_LEVEL2 both resolve to "Metal Bar".
// The rows carry the tier, so the header only names the material.
func FamilyName(uniqueName string) string {
	key := tierPattern.ReplaceAllString(nameKey(uniqueName), "")
	if name, ok := familyNames[key]; ok {
		return name
	}
	if label := Name(uniqueName); label != uniqueName {
		if stripped := stripTierAdjective(label); stripped != "" {
			return stripped
		}
	}
	return humanize(key)
}

// nameKey removes the enchant decoration so every level shares one entry.
func nameKey(uniqueName string) string {
	return enchantPattern.ReplaceAllString(levelPattern.ReplaceAllString(uniqueName, ""), "")
}

func stripTierAdjective(label string) string {
	for _, adjective := range tierAdjectives {
		if strings.HasPrefix(label, adjective) {
			return strings.TrimPrefix(label, adjective)
		}
	}
	return label
}

// humanize turns an unresolved id into readable text: ARTEFACT_2H_BOW_AVALON
// becomes "Artefact 2H Bow Avalon".
func humanize(key string) string {
	words := strings.Split(key, "_")
	for i, word := range words {
		switch {
		case word == "":
			continue
		case strings.ContainsAny(word, "0123456789"):
			words[i] = strings.ToUpper(word)
		default:
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}
