// Command gen rebuilds the crafting data files from the ao-bin-dumps exports.
//
// The game data is published by ao-bin-dumps in two shapes:
//
//	items.json            XML-shaped dump at the repository root, 17 MB, holds
//	                      craftingrequirements for every item
//	formatted/items.json  flat list with LocalizedNames, 24 MB
//
// Run it with go generate after a game patch, with both files on disk:
//
//	AODP_FULL=~/ao-bin-dumps/items.json \
//	AODP_FORMATTED=~/ao-bin-dumps/formatted/items.json \
//	go generate ./internal/crafting
//
// Two files are written next to the embedded data:
//
//	recipes.json  base -> tier -> enchant -> [{item, count}]
//	names.json    ingredient item id -> EN-US label, sizes collapsed to the
//	              tiered id (T4_METALBAR_LEVEL4 -> T4_METALBAR)
//
// items.json (the curated craftable list) is read, never written: it carries
// the hand-picked subset of gear the calculator offers.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

const (
	minTier      = 4
	maxTier      = 8
	maxEnchant   = 4
	recipesFile  = "recipes.json"
	namesFile    = "names.json"
	itemsFile    = "items.json"
	nameLanguage = "EN-US"
)

// levelSuffix is the enchant decoration on refined materials.
var levelSuffix = regexp.MustCompile(`_LEVEL\d+$`)

// ingredient is one recipe line. Field order matters: it keeps the generated
// JSON stable against the previous exporter.
type ingredient struct {
	Count  int    `json:"count"`
	ItemID string `json:"item"`
}

type item struct {
	UniqueName     string `json:"uniqueName"`
	BaseUniqueName string `json:"baseUniqueName"`
	Label          string `json:"label"`
}

func main() {
	full := flag.String("full", os.Getenv("AODP_FULL"),
		"path to the ao-bin-dumps items.json dump (contains craftingrequirements)")
	formatted := flag.String("formatted", os.Getenv("AODP_FORMATTED"),
		"path to the ao-bin-dumps formatted/items.json (contains localized names)")
	data := flag.String("data", "data", "directory holding items.json, recipes.json and names.json")
	flag.Parse()

	if err := run(*full, *formatted, *data); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func run(fullPath, formattedPath, dir string) error {
	if fullPath == "" || formattedPath == "" {
		return fmt.Errorf("both -full and -formatted are required (or set AODP_FULL / AODP_FORMATTED)")
	}

	items, err := readItems(filepath.Join(dir, itemsFile))
	if err != nil {
		return err
	}

	nodes, err := readNodes(fullPath)
	if err != nil {
		return err
	}

	recipes, used := buildRecipes(items, nodes)
	if err := writeJSON(filepath.Join(dir, recipesFile), recipes); err != nil {
		return err
	}

	labels, err := readLabels(formattedPath)
	if err != nil {
		return err
	}

	names, missing := buildNames(used, labels)
	if len(missing) > 0 {
		return fmt.Errorf("%d recipe ingredients have no %s label in %s: %v",
			len(missing), nameLanguage, formattedPath, missing)
	}
	if err := writeJSON(filepath.Join(dir, namesFile), names); err != nil {
		return err
	}

	fmt.Printf("recipes: %d bases, names: %d items\n", len(recipes), len(names))
	return nil
}

func readItems(path string) ([]item, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []item
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return items, nil
}

func readJSON(path string, dst any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, value any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// readNodes indexes every item node in the XML-shaped dump by unique name.
// The same unique name appears in several categories, and only one of those
// nodes carries the crafting requirements, so all candidates are kept.
func readNodes(path string) (map[string][]map[string]any, error) {
	var dump map[string]any
	if err := readJSON(path, &dump); err != nil {
		return nil, err
	}

	root, ok := dump["items"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: no items root", path)
	}

	nodes := make(map[string][]map[string]any)
	var walk func(value any)
	walk = func(value any) {
		switch node := value.(type) {
		case map[string]any:
			if name, ok := node["@uniquename"].(string); ok && name != "" {
				nodes[name] = append(nodes[name], node)
			}
			for _, child := range node {
				walk(child)
			}
		case []any:
			for _, child := range node {
				walk(child)
			}
		}
	}
	walk(root)
	return nodes, nil
}

// readLabels indexes the formatted dump by unique name.
func readLabels(path string) (map[string]string, error) {
	var dump []struct {
		UniqueName     string            `json:"UniqueName"`
		LocalizedNames map[string]string `json:"LocalizedNames"`
	}
	if err := readJSON(path, &dump); err != nil {
		return nil, err
	}

	labels := make(map[string]string, len(dump))
	for _, entry := range dump {
		if entry.UniqueName == "" {
			continue
		}
		if _, ok := labels[entry.UniqueName]; ok {
			continue
		}
		labels[entry.UniqueName] = entry.LocalizedNames[nameLanguage]
	}
	return labels, nil
}

// buildRecipes resolves the ingredient list of every tier/enchant variant of
// every craftable base item and returns the book plus every ingredient id it
// referenced.
func buildRecipes(items []item, nodes map[string][]map[string]any) (map[string]map[string]map[string][]ingredient, map[string]bool) {
	bases := make(map[string]bool)
	for _, item := range items {
		bases[item.BaseUniqueName] = true
	}
	sorted := make([]string, 0, len(bases))
	for base := range bases {
		sorted = append(sorted, base)
	}
	sort.Strings(sorted)

	book := make(map[string]map[string]map[string][]ingredient, len(sorted))
	used := make(map[string]bool)

	for _, base := range sorted {
		tiers := make(map[string]map[string][]ingredient)
		for tier := minTier; tier <= maxTier; tier++ {
			node := bestNode(nodes[fmt.Sprintf("T%d_%s", tier, base)])
			if node == nil {
				continue
			}
			levels := make(map[string][]ingredient)
			for enchant := 0; enchant <= maxEnchant; enchant++ {
				lines := recipe(node, enchant)
				if len(lines) == 0 {
					continue
				}
				levels[strconv.Itoa(enchant)] = lines
				for _, line := range lines {
					used[line.ItemID] = true
				}
			}
			if len(levels) > 0 {
				tiers[strconv.Itoa(tier)] = levels
			}
		}
		if len(tiers) > 0 {
			book[base] = tiers
		}
	}
	return book, used
}

// bestNode picks the candidate carrying the crafting requirements.
func bestNode(candidates []map[string]any) map[string]any {
	var best map[string]any
	bestScore := -1
	for _, node := range candidates {
		score := 0
		for key := range node {
			if key[0] != '@' {
				score++
			}
		}
		if _, ok := node["craftingrequirements"]; ok {
			score += 10
		}
		if score > bestScore {
			best, bestScore = node, score
		}
	}
	return best
}

// recipe returns the ingredients of one enchant level. Level 0 lives on the
// item node itself; enchanted levels live under enchantments.enchantment[].
func recipe(node map[string]any, enchant int) []ingredient {
	var requirements map[string]any
	if enchant == 0 {
		requirements = firstMap(asList(node["craftingrequirements"]))
	} else {
		enchantments, _ := node["enchantments"].(map[string]any)
		for _, entry := range asList(enchantments["enchantment"]) {
			value, _ := entry["@enchantmentlevel"].(string)
			if value != strconv.Itoa(enchant) {
				continue
			}
			requirements = firstMap(asList(entry["craftingrequirements"]))
			break
		}
	}
	if requirements == nil {
		return nil
	}

	var lines []ingredient
	for _, resource := range asList(requirements["craftresource"]) {
		name, _ := resource["@uniquename"].(string)
		count, _ := resource["@count"].(string)
		number, err := strconv.Atoi(count)
		if name == "" || err != nil {
			continue
		}
		lines = append(lines, ingredient{Count: number, ItemID: name})
	}
	return lines
}

// buildNames resolves one label per ingredient id, collapsing the enchant
// decoration: T4_METALBAR_LEVEL3 shares the name of T4_METALBAR.
func buildNames(used map[string]bool, labels map[string]string) (map[string]string, []string) {
	names := make(map[string]string, len(used))
	var missing []string
	for itemID := range used {
		key := levelSuffix.ReplaceAllString(itemID, "")
		if _, ok := names[key]; ok {
			continue
		}
		label, ok := labels[key]
		if !ok || label == "" {
			missing = append(missing, itemID)
			continue
		}
		names[key] = label
	}
	sort.Strings(missing)
	return names, missing
}

func asList(value any) []map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return []map[string]any{typed}
	case []any:
		list := make([]map[string]any, 0, len(typed))
		for _, entry := range typed {
			if node, ok := entry.(map[string]any); ok {
				list = append(list, node)
			}
		}
		return list
	}
	return nil
}

func firstMap(list []map[string]any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	return list[0]
}
