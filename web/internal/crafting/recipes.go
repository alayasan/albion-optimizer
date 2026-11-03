package crafting

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data/recipes.json
var recipesJSON []byte

// Ingredient is one material and how many are needed.
type Ingredient struct {
	ItemID string `json:"item"`
	Count  int    `json:"count"`
}

// recipeBook maps base unique name -> tier -> enchant level -> ingredients.
type recipeBook map[string]map[int]map[int][]Ingredient

var recipes recipeBook

func init() {
	if err := json.Unmarshal(recipesJSON, &recipes); err != nil {
		panic(fmt.Sprintf("crafting: parse embedded recipes: %v", err))
	}
}

// Recipe returns the ingredients for one tier and enchant level of a base item.
func Recipe(baseUniqueName string, tier, enchant int) ([]Ingredient, bool) {
	tiers, ok := recipes[baseUniqueName]
	if !ok {
		return nil, false
	}
	levels, ok := tiers[tier]
	if !ok {
		return nil, false
	}
	ingredients, ok := levels[enchant]
	if !ok {
		return nil, false
	}
	return ingredients, true
}

// Tiers and enchants shown in the crafting plan table.
var (
	PlanTiers    = []int{4, 5, 6, 7, 8}
	PlanEnchants = []int{0, 1, 2, 3, 4}
)

// Plan is one row of the crafting table: a tier/enchant variant and the
// materials its recipe needs.
type Plan struct {
	Tier        int
	Enchant     int
	Ingredients []Ingredient
}

// Plans returns every craftable tier/enchant variant of a base item, ordered by
// tier and then enchant. Variants without a recipe are skipped.
func Plans(baseUniqueName string) []Plan {
	plans := make([]Plan, 0, len(PlanTiers)*len(PlanEnchants))
	for _, tier := range PlanTiers {
		for _, enchant := range PlanEnchants {
			ingredients, ok := Recipe(baseUniqueName, tier, enchant)
			if !ok {
				continue
			}
			plans = append(plans, Plan{
				Tier:        tier,
				Enchant:     enchant,
				Ingredients: ingredients,
			})
		}
	}
	return plans
}
