package dialog

import (
	"fmt"
	"math"
	"strconv"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/list"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"
)

// ModelGroup represents a group of model items.
type ModelGroup struct {
	*list.Versioned
	Title      string
	Items      []*ModelItem
	configured bool
	t          *styles.Styles
}

// NewModelGroup creates a new ModelGroup.
func NewModelGroup(t *styles.Styles, title string, configured bool, items ...*ModelItem) ModelGroup {
	return ModelGroup{
		Versioned:  list.NewVersioned(),
		Title:      title,
		Items:      items,
		configured: configured,
		t:          t,
	}
}

// Finished implements list.Item. Model groups are immutable headers.
func (m *ModelGroup) Finished() bool {
	return true
}

// AppendItems appends [ModelItem]s to the group.
func (m *ModelGroup) AppendItems(items ...*ModelItem) {
	m.Items = append(m.Items, items...)
}

// Render implements [list.Item].
func (m *ModelGroup) Render(width int) string {
	var configured string
	if m.configured {
		configuredIcon := m.t.ToolCallSuccess.Render()
		configuredText := m.t.Dialog.Models.ConfiguredText.Render("Configured")
		configured = configuredIcon + " " + configuredText
	}

	title := " " + m.Title + " "
	// Keep the "Configured" badge only when the full title fits beside it
	// (plus a separator). Otherwise drop it and let the title use the whole
	// width, rather than truncating the title to reserve room for a badge
	// that common.Section would then drop anyway, leaving dead space.
	if configured != "" && lipgloss.Width(title)+lipgloss.Width(configured)+3 > width {
		configured = ""
	}
	if configured == "" {
		title = ansi.Truncate(title, max(0, width-1), "…")
	}

	return common.Section(m.t, title, width, configured)
}

// ModelItem represents a list item for a model type.
type ModelItem struct {
	*list.Versioned

	prov      catwalk.Provider
	model     catwalk.Model
	modelType ModelType

	cache         map[int]string
	t             *styles.Styles
	m             fuzzy.Match
	focused       bool
	current       bool
	showProvider  bool
	priceInWidth  int
	priceOutWidth int
}

// Finished implements list.Item. Model items are render-stable
// outside of explicit SetFocused / SetMatch.
func (m *ModelItem) Finished() bool {
	return true
}

// SelectedModel returns this model item as a [config.SelectedModel] instance.
func (m *ModelItem) SelectedModel() config.SelectedModel {
	return config.SelectedModel{
		Model:           m.model.ID,
		Provider:        string(m.prov.ID),
		ReasoningEffort: m.model.DefaultReasoningEffort,
		MaxTokens:       m.model.DefaultMaxTokens,
	}
}

// SelectedModelType returns the type of model represented by this item.
func (m *ModelItem) SelectedModelType() config.SelectedModelType {
	return m.modelType.Config()
}

var _ ListItem = &ModelItem{}

// NewModelItem creates a new ModelItem.
func NewModelItem(t *styles.Styles, prov catwalk.Provider, model catwalk.Model, typ ModelType, showProvider bool) *ModelItem {
	return &ModelItem{
		Versioned:    list.NewVersioned(),
		prov:         prov,
		model:        model,
		modelType:    typ,
		t:            t,
		cache:        make(map[int]string),
		showProvider: showProvider,
	}
}

// Filter implements ListItem.
func (m *ModelItem) Filter() string {
	return m.model.Name
}

// ID implements ListItem.
func (m *ModelItem) ID() string {
	return modelKey(string(m.prov.ID), m.model.ID)
}

// modelPriceLabel renders a model's input and output cost per million
// tokens as two table columns, e.g. `3.00  15.00`. Each number is
// right-justified to the given width so the columns line up across rows. It
// returns an empty string when the model has no pricing information.
func modelPriceLabel(model catwalk.Model, inWidth, outWidth int) string {
	if model.CostPer1MIn == 0 && model.CostPer1MOut == 0 {
		return ""
	}
	return fmt.Sprintf("%*s  %*s",
		inWidth, formatPrice(model.CostPer1MIn),
		outWidth, formatPrice(model.CostPer1MOut))
}

// alignPriceColumns sets the price widths of every item in groups to the
// widest prices present, so the price columns line up when rendered.
func alignPriceColumns(groups []ModelGroup) {
	var inWidth, outWidth int
	for _, g := range groups {
		for _, item := range g.Items {
			if item.model.CostPer1MIn == 0 && item.model.CostPer1MOut == 0 {
				continue
			}
			inWidth = max(inWidth, len(formatPrice(item.model.CostPer1MIn)))
			outWidth = max(outWidth, len(formatPrice(item.model.CostPer1MOut)))
		}
	}
	for _, g := range groups {
		for _, item := range g.Items {
			item.SetPriceWidths(inWidth, outWidth)
		}
	}
}

// modelPriceHeader renders the column header that labels the price shown on
// each model row. It reuses the item renderer so the header lines up with
// the rows beneath it.
func modelPriceHeader(t *styles.Styles, width int) string {
	styles := ListItemStyles{
		ItemBlurred:     t.Dialog.NormalItem,
		ItemFocused:     t.Dialog.NormalItem,
		InfoTextBlurred: t.Dialog.ListItem.InfoBlurred,
		InfoTextFocused: t.Dialog.ListItem.InfoBlurred,
	}
	return renderItemWithSuffix(styles, "", "", " In/Out ($/M) ", false, width, nil, nil)
}

// formatPrice renders a price number with exactly two decimal places. Values
// are rounded first because catwalk stores some costs as float artifacts such
// as 0.39999999999999997.
func formatPrice(v float64) string {
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', 2, 64)
}

// Render implements ListItem.
func (m *ModelItem) Render(width int) string {
	var providerInfo string
	if m.current {
		providerInfo = "Active"
	}
	if m.showProvider {
		if providerInfo != "" {
			providerInfo += " · "
		}
		providerInfo += string(m.prov.Name)
	}
	// The price is pinned to the right edge rather than joined to the info
	// column, so a long provider name truncates before the price does.
	var suffix string
	if price := modelPriceLabel(m.model, m.priceInWidth, m.priceOutWidth); price != "" {
		suffix = " " + price + " "
	}
	styles := ListItemStyles{
		ItemBlurred:     m.t.Dialog.NormalItem,
		ItemFocused:     m.t.Dialog.SelectedItem,
		InfoTextBlurred: m.t.Dialog.ListItem.InfoBlurred,
		InfoTextFocused: m.t.Dialog.ListItem.InfoFocused,
	}
	return renderItemWithSuffix(styles, m.model.Name, providerInfo, suffix, m.focused, width, m.cache, &m.m)
}

// SetFocused implements ListItem.
func (m *ModelItem) SetFocused(focused bool) {
	if m.focused == focused {
		return
	}
	m.cache = nil
	m.focused = focused
	if m.Versioned != nil {
		m.Bump()
	}
}

// SetCurrent marks this item as the model currently in use.
func (m *ModelItem) SetCurrent(current bool) {
	if m.current == current {
		return
	}
	m.cache = nil
	m.current = current
	if m.Versioned != nil {
		m.Bump()
	}
}

// SetMatch implements ListItem.
func (m *ModelItem) SetMatch(fm fuzzy.Match) {
	if sameFuzzyMatch(m.m, fm) {
		return
	}
	m.cache = nil
	m.m = fm
	if m.Versioned != nil {
		m.Bump()
	}
}

// SetPriceWidths sets the column widths used to align this item's price with
// the other items in the picker.
func (m *ModelItem) SetPriceWidths(inWidth, outWidth int) {
	if m.priceInWidth == inWidth && m.priceOutWidth == outWidth {
		return
	}
	m.cache = nil
	m.priceInWidth = inWidth
	m.priceOutWidth = outWidth
	if m.Versioned != nil {
		m.Bump()
	}
}
