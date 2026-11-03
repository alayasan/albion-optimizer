package crafting

import "testing"

func TestRecipeBaseAndEnchanted(t *testing.T) {
	cases := []struct {
		name    string
		base    string
		tier    int
		enchant int
		want    []Ingredient
	}{
		{
			name:    "base",
			base:    "MAIN_SWORD",
			tier:    4,
			enchant: 0,
			want:    []Ingredient{{ItemID: "T4_METALBAR", Count: 16}, {ItemID: "T4_LEATHER", Count: 8}},
		},
		{
			name:    "enchant 4 uses level materials",
			base:    "MAIN_SWORD",
			tier:    4,
			enchant: 4,
			want:    []Ingredient{{ItemID: "T4_METALBAR_LEVEL4", Count: 16}, {ItemID: "T4_LEATHER_LEVEL4", Count: 8}},
		},
		{
			name:    "single ingredient",
			base:    "2H_BOW",
			tier:    4,
			enchant: 0,
			want:    []Ingredient{{ItemID: "T4_PLANKS", Count: 32}},
		},
		{
			name:    "highest tier",
			base:    "MAIN_SWORD",
			tier:    8,
			enchant: 0,
			want:    []Ingredient{{ItemID: "T8_METALBAR", Count: 16}, {ItemID: "T8_LEATHER", Count: 8}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Recipe(tc.base, tc.tier, tc.enchant)
			if !ok {
				t.Fatalf("Recipe(%s, %d, %d) missing", tc.base, tc.tier, tc.enchant)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("ingredients = %+v, want %+v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("ingredient %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestRecipeUnknown(t *testing.T) {
	if _, ok := Recipe("NOT_AN_ITEM", 4, 0); ok {
		t.Error("expected no recipe for unknown base")
	}
	if _, ok := Recipe("MAIN_SWORD", 3, 0); ok {
		t.Error("expected no recipe below tier 4")
	}
	if _, ok := Recipe("MAIN_SWORD", 4, 5); ok {
		t.Error("expected no recipe above enchant 4")
	}
}

func TestPlansCoversEveryTierAndEnchant(t *testing.T) {
	plans := Plans("MAIN_SWORD")
	if len(plans) != len(PlanTiers)*len(PlanEnchants) {
		t.Fatalf("plans = %d, want %d", len(plans), len(PlanTiers)*len(PlanEnchants))
	}

	seen := make(map[[2]int]bool)
	for _, plan := range plans {
		seen[[2]int{plan.Tier, plan.Enchant}] = true
		if len(plan.Ingredients) == 0 || len(plan.Ingredients) > 4 {
			t.Errorf("tier %d enchant %d has %d ingredients", plan.Tier, plan.Enchant, len(plan.Ingredients))
		}
	}

	for _, tier := range PlanTiers {
		for _, enchant := range PlanEnchants {
			if !seen[[2]int{tier, enchant}] {
				t.Errorf("missing tier %d enchant %d", tier, enchant)
			}
		}
	}
}

func TestPlansAreOrdered(t *testing.T) {
	previous := Plan{}
	for _, plan := range Plans("MAIN_SWORD") {
		if plan.Tier < previous.Tier {
			t.Fatalf("tier went backwards: %d after %d", plan.Tier, previous.Tier)
		}
		if plan.Tier == previous.Tier && plan.Enchant <= previous.Enchant {
			t.Fatalf("enchant not ascending at tier %d: %d after %d", plan.Tier, plan.Enchant, previous.Enchant)
		}
		previous = plan
	}
}

func TestEveryCraftableItemHasPlans(t *testing.T) {
	// Non-gear entries in the item list have no craftingrequirements.
	skip := map[string]bool{
		"GVGSEASONREWARD_FAMEBUFF":   true,
		"LEARNINGPOINTS_NONTRADABLE": true,
		"QUESTITEM_TOKEN_ROYAL_T4":   true,
	}
	for _, item := range Items() {
		if skip[item.BaseUniqueName] {
			continue
		}
		if len(Plans(item.BaseUniqueName)) == 0 {
			t.Errorf("%s (%s) has no plans", item.UniqueName, item.BaseUniqueName)
		}
	}
}
