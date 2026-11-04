package ui

import (
	"context"

	"github.com/alayasan/albion-optimizer/web/internal/crafting"
)

// planIngredientColumns is how many ingredient cells each row reserves.
const planIngredientColumns = 4

// priceAgeLayout is how the market timestamp is shown in the table.
const priceAgeLayout = "2006-01-02 15:04"

// PlanCell is one ingredient of a crafting row.
type PlanCell struct {
	ItemID      string
	Label       string
	Count       int
	Price       float64
	HasPrice    bool
	Age         string
	Override    float64
	HasOverride bool
}

// PlanRow is one tier/enchant variant of the selected item.
type PlanRow struct {
	Tier    int
	Enchant int
	NewTier bool
	Cells   []PlanCell
}

// PlanTable is the recipe table for the selected item: one header per
// ingredient column plus one row per tier/enchant variant.
type PlanTable struct {
	Headers []string
	Rows    []PlanRow
}

// buildPlan turns the recipe table for the selected item into view rows, using
// manual overrides first and cached market prices second.
func buildPlan(ctx context.Context, prices *priceCache, state crafting.State, selected *crafting.Item) PlanTable {
	if selected == nil {
		return emptyPlan()
	}

	plans := crafting.Plans(selected.BaseUniqueName)
	if len(plans) == 0 {
		return emptyPlan()
	}

	seen := make(map[string]bool)
	ids := make([]string, 0, len(plans)*2)
	for _, plan := range plans {
		for _, ingredient := range plan.Ingredients {
			if !seen[ingredient.ItemID] {
				seen[ingredient.ItemID] = true
				ids = append(ids, ingredient.ItemID)
			}
		}
	}
	quotes := prices.Lookup(ctx, ids)

	rows := make([]PlanRow, 0, len(plans))
	previousTier := 0
	for _, plan := range plans {
		row := PlanRow{
			Tier:    plan.Tier,
			Enchant: plan.Enchant,
			NewTier: plan.Tier != previousTier,
			Cells:   make([]PlanCell, 0, planIngredientColumns),
		}
		previousTier = plan.Tier

		for _, ingredient := range plan.Ingredients {
			cell := PlanCell{
				ItemID: ingredient.ItemID,
				Label:  crafting.Name(ingredient.ItemID),
				Count:  ingredient.Count,
			}
			if override, ok := state.Overrides[ingredient.ItemID]; ok {
				cell.Price = override
				cell.HasPrice = true
				cell.Override = override
				cell.HasOverride = true
			} else if quote, ok := quotes[ingredient.ItemID]; ok && quote.SellPriceMin > 0 {
				cell.Price = float64(quote.SellPriceMin)
				cell.HasPrice = true
				cell.Age = quote.SellPriceMinDate.Format(priceAgeLayout)
			}
			row.Cells = append(row.Cells, cell)
		}

		for len(row.Cells) < planIngredientColumns {
			row.Cells = append(row.Cells, PlanCell{})
		}
		rows = append(rows, row)
	}

	return PlanTable{Headers: planHeaders(plans), Rows: rows}
}

// emptyPlan keeps the table shape when nothing is selected, so the header row
// still lines up with the empty-state row.
func emptyPlan() PlanTable {
	return PlanTable{Headers: make([]string, planIngredientColumns)}
}

// planHeaders names each ingredient column after the material family of the
// lowest tier that fills it, so the header stays valid for every row.
func planHeaders(plans []crafting.Plan) []string {
	headers := make([]string, planIngredientColumns)
	for column := range headers {
		for _, plan := range plans {
			if column >= len(plan.Ingredients) {
				continue
			}
			headers[column] = crafting.FamilyName(plan.Ingredients[column].ItemID)
			break
		}
	}
	return headers
}
