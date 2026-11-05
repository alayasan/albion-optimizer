package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/alayasan/albion-optimizer/web/internal/crafting"
	"github.com/alayasan/albion-optimizer/web/internal/market"
)

// planIngredientColumns is how many ingredient cells each row reserves.
const planIngredientColumns = 4

// planOutputColumns is how many columns follow the ingredients: price, demand,
// profit, gain, cost, books, fame and silver bought per day.
const planOutputColumns = 8

// sharedIngredientColumn is the first column whose material is not enchanted.
// Artefacts and rare materials are the same item for enchant 0 through 4, so
// those cells are merged across the tier's rows.
const sharedIngredientColumn = 2

// Price age is shown in whole hours since the API timestamp, capped at 48
// because anything older is not actionable for a crafting decision.
const (
	priceAgeMaxHours = 48
	priceAgeBuckets  = 6
)

// PlanCell is one priced item of a crafting row: an ingredient, or the crafted
// item itself.
type PlanCell struct {
	ItemID      string
	Label       string
	Count       int
	Price       float64
	HasPrice    bool
	AgeHours    string
	AgeClass    string
	Override    float64
	HasOverride bool
	// Rowspan is how many tier rows this cell covers; 0 renders a normal cell.
	Rowspan int
}

// Economics is one craft's money picture: what the materials and station cost,
// what the crafted item sells for, and the margin between them.
type Economics struct {
	Cost       float64
	Revenue    float64
	Profit     float64
	Gain       float64
	HasCost    bool
	HasRevenue bool
}

// HasProfit reports whether both sides of the margin are known.
func (e Economics) HasProfit() bool {
	return e.HasCost && e.HasRevenue
}

// HasGain reports whether the margin can be expressed as a percentage.
func (e Economics) HasGain() bool {
	return e.HasProfit() && e.Cost > 0
}

// PlanRow is one tier/enchant variant of the selected item.
type PlanRow struct {
	Tier    int
	Enchant int
	NewTier bool
	Cells   []PlanCell

	// Output is the crafted item itself and its market price.
	Output PlanCell

	Economics Economics

	// DemandUnits is how many units of the output traded in the window.
	DemandUnits int
	// BoughtPerDay is the silver traded in the output per day of that window.
	BoughtPerDay float64
	HasDemand    bool
	HasBought    bool
}

// PlanTable is the recipe table for the selected item: one header per
// ingredient column plus one row per tier/enchant variant.
type PlanTable struct {
	Headers []string
	Rows    []PlanRow
	// Cols is the total column count, used by the empty-state row.
	Cols int
}

// buildPlan turns the recipe table for the selected item into view rows, using
// manual overrides first and cached market data second.
func buildPlan(ctx context.Context, prices *priceCache, demands *demandCache, state crafting.State, selected *crafting.Item, now time.Time) PlanTable {
	if selected == nil {
		return emptyPlan()
	}

	plans := crafting.Plans(selected.BaseUniqueName)
	if len(plans) == 0 {
		return emptyPlan()
	}

	outputs := make([]string, 0, len(plans))
	for _, plan := range plans {
		outputs = append(outputs, crafting.ItemID(selected.BaseUniqueName, plan.Tier, plan.Enchant))
	}

	seen := make(map[string]bool, len(outputs)+len(plans)*planIngredientColumns)
	ids := make([]string, 0, len(seen))
	for _, id := range outputs {
		seen[id] = true
		ids = append(ids, id)
	}
	for _, plan := range plans {
		for _, ingredient := range plan.Ingredients {
			if !seen[ingredient.ItemID] {
				seen[ingredient.ItemID] = true
				ids = append(ids, ingredient.ItemID)
			}
		}
	}
	quotes := prices.Lookup(ctx, ids)
	traded := demands.Lookup(ctx, outputs, state.DemandDays)

	rows := make([]PlanRow, 0, len(plans))
	days := float64(demandDays(state))
	previousTier := 0
	for i, plan := range plans {
		row := PlanRow{
			Tier:    plan.Tier,
			Enchant: plan.Enchant,
			NewTier: plan.Tier != previousTier,
			Cells:   make([]PlanCell, 0, planIngredientColumns),
			Output: pricedCell(PlanCell{
				ItemID: outputs[i],
				Label:  selected.Label,
				Count:  1,
			}, state, quotes, now),
		}
		previousTier = plan.Tier

		for _, ingredient := range plan.Ingredients {
			row.Cells = append(row.Cells, pricedCell(PlanCell{
				ItemID: ingredient.ItemID,
				Label:  crafting.Name(ingredient.ItemID),
				Count:  ingredient.Count,
			}, state, quotes, now))
		}

		for len(row.Cells) < planIngredientColumns {
			row.Cells = append(row.Cells, PlanCell{})
		}

		row.Economics = economics(state, row)
		row.DemandUnits, row.HasDemand = traded[row.Output.ItemID]
		if row.HasDemand && row.Output.HasPrice {
			row.HasBought = true
			row.BoughtPerDay = float64(row.DemandUnits) / days * row.Output.Price
		}

		rows = append(rows, row)
	}

	return PlanTable{
		Headers: planHeaders(plans),
		Rows:    mergeSharedIngredients(rows),
		Cols:    1 + planIngredientColumns + planOutputColumns,
	}
}

// pricedCell fills in the manual override or cached quote for one item.
func pricedCell(cell PlanCell, state crafting.State, quotes map[string]market.Price, now time.Time) PlanCell {
	if override, ok := state.Overrides[cell.ItemID]; ok {
		cell.Price = override
		cell.HasPrice = true
		cell.Override = override
		cell.HasOverride = true
		return cell
	}
	if quote, ok := quotes[cell.ItemID]; ok && quote.SellPriceMin > 0 {
		cell.Price = float64(quote.SellPriceMin)
		cell.HasPrice = true
		cell.AgeHours, cell.AgeClass = priceAge(now, quote.SellPriceMinDate.Time)
	}
	return cell
}

// economics sums the material prices plus the station fee and sets them against
// the output price. Incomplete prices leave the margin blank rather than
// reporting a cost that is quietly too low.
func economics(state crafting.State, row PlanRow) Economics {
	econ := Economics{Cost: state.StationFee, HasCost: true}
	for _, cell := range row.Cells {
		if cell.ItemID == "" {
			continue
		}
		if !cell.HasPrice {
			econ.HasCost = false
			continue
		}
		econ.Cost += float64(cell.Count) * cell.Price
	}
	if row.Output.HasPrice {
		econ.HasRevenue = true
		econ.Revenue = row.Output.Price
	}
	econ.Profit = econ.Revenue - econ.Cost
	if econ.HasGain() {
		econ.Gain = econ.Profit / econ.Cost * 100
	}
	return econ
}

// demandDays keeps the window inside the range the state can hold.
func demandDays(state crafting.State) int {
	if state.DemandDays < 1 {
		return 1
	}
	return state.DemandDays
}

// mergeSharedIngredients collapses the unenchanted ingredient columns into one
// cell per tier, spanning that tier's enchant rows.
func mergeSharedIngredients(rows []PlanRow) []PlanRow {
	for start := 0; start < len(rows); {
		end := start
		for end < len(rows) && rows[end].Tier == rows[start].Tier {
			end++
		}
		span := end - start
		if span > 1 {
			// Columns are removed right to left so the remaining indexes stay
			// valid while rows shrink.
			for column := planIngredientColumns - 1; column >= sharedIngredientColumn; column-- {
				if !sharedInTier(rows[start:end], column) {
					continue
				}
				rows[start].Cells[column].Rowspan = span
				for i := start + 1; i < end; i++ {
					rows[i].Cells = append(rows[i].Cells[:column], rows[i].Cells[column+1:]...)
				}
			}
		}
		start = end
	}
	return rows
}

// sharedInTier reports whether every row of a tier uses the same material in
// the given column. Empty columns are never merged.
func sharedInTier(group []PlanRow, column int) bool {
	if column >= len(group[0].Cells) {
		return false
	}
	first := group[0].Cells[column]
	if first.ItemID == "" {
		return false
	}
	for _, row := range group[1:] {
		cell := row.Cells[column]
		if cell.ItemID != first.ItemID || cell.Count != first.Count {
			return false
		}
	}
	return true
}

// emptyPlan keeps the table shape when nothing is selected, so the header row
// still lines up with the empty-state row.
func emptyPlan() PlanTable {
	return PlanTable{
		Headers: make([]string, planIngredientColumns),
		Cols:    1 + planIngredientColumns + planOutputColumns,
	}
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

// priceAge renders how long ago a price was recorded: whole hours up to
// priceAgeMaxHours, then "48h+". The class steps from green to red so a stale
// quote is visible at a glance.
func priceAge(now, seen time.Time) (label, class string) {
	if seen.IsZero() {
		return "", ""
	}

	hours := int(now.Sub(seen).Hours())
	if hours < 0 {
		hours = 0
	}

	label = fmt.Sprintf("%dh", hours)
	if hours >= priceAgeMaxHours {
		hours = priceAgeMaxHours
		label = fmt.Sprintf("%dh+", priceAgeMaxHours)
	}

	bucket := hours * priceAgeBuckets / (priceAgeMaxHours + 1)
	if bucket >= priceAgeBuckets {
		bucket = priceAgeBuckets - 1
	}
	return label, fmt.Sprintf("age-%d", bucket)
}
