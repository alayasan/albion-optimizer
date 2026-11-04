package crafting

import "testing"

func TestNameResolvesEnchantDecoration(t *testing.T) {
	cases := map[string]string{
		"T4_METALBAR":                        "Steel Bar",
		"T4_METALBAR_LEVEL4":                 "Steel Bar",
		"T8_PLANKS_LEVEL2":                   "Whitewood Planks",
		"T4_ARTEFACT_2H_ARCANESTAFF_CRYSTAL": "Adept's Startouched Crystal",
		// Gear is labelled from the craftable list, and the enchant suffix
		// does not change the name.
		"T8_MAIN_SWORD":   "Broadsword",
		"T8_MAIN_SWORD@4": "Broadsword",
		// Unknown ids fall back to the id rather than rendering blank.
		"T9_MADE_UP_ITEM": "T9_MADE_UP_ITEM",
	}
	for itemID, want := range cases {
		if got := Name(itemID); got != want {
			t.Errorf("Name(%q) = %q, want %q", itemID, got, want)
		}
	}
}

func TestNamesCoverEveryRecipeIngredient(t *testing.T) {
	for base, tiers := range recipes {
		for tier, levels := range tiers {
			for enchant, ingredients := range levels {
				for _, ingredient := range ingredients {
					if _, ok := names[nameKey(ingredient.ItemID)]; !ok {
						t.Errorf("%s T%d@%d ingredient %s has no label; regenerate data/names.json",
							base, tier, enchant, ingredient.ItemID)
					}
				}
			}
		}
	}
}

func TestFamilyName(t *testing.T) {
	cases := map[string]string{
		"T4_METALBAR":                        "Metal Bar",
		"T6_PLANKS_LEVEL2":                   "Planks",
		"T8_LEATHER_LEVEL4":                  "Leather",
		"T5_CLOTH_LEVEL1":                    "Cloth",
		"T4_ARTEFACT_2H_ARCANESTAFF_CRYSTAL": "Startouched Crystal",
		"QUESTITEM_TOKEN_ROYAL_T6":           "Royal Sigil",
		"T4_WEIRD_THING":                     "Weird Thing",
		"T4_2H_BOW":                          "2H Bow",
	}
	for itemID, want := range cases {
		if got := FamilyName(itemID); got != want {
			t.Errorf("FamilyName(%q) = %q, want %q", itemID, got, want)
		}
	}
}
