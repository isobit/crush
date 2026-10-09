package dialog

import (
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestFormatPrice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   float64
		want string
	}{
		{"whole", 3, "3.00"},
		{"rounds up", 14.6, "14.60"},
		{"large", 180, "180.00"},
		{"fraction", 0.1, "0.10"},
		{"rounds to cents", 0.126, "0.13"},
		{"float artifact", 0.39999999999999997, "0.40"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, formatPrice(tt.in))
		})
	}
}

func TestModelPriceLabel(t *testing.T) {
	t.Parallel()

	require.Equal(t, "3.00  15.00", modelPriceLabel(catwalk.Model{CostPer1MIn: 3, CostPer1MOut: 15}, 0, 0))
	require.Equal(t, " 3.00  15.00", modelPriceLabel(catwalk.Model{CostPer1MIn: 3, CostPer1MOut: 15}, 5, 5))
	require.Empty(t, modelPriceLabel(catwalk.Model{}, 4, 4))
}

func TestAlignPriceColumns(t *testing.T) {
	t.Parallel()

	cheap := &ModelItem{model: catwalk.Model{CostPer1MIn: 0.1, CostPer1MOut: 0.5}}
	pricey := &ModelItem{model: catwalk.Model{CostPer1MIn: 15, CostPer1MOut: 180}}
	unpriced := &ModelItem{model: catwalk.Model{}}
	groups := []ModelGroup{{Items: []*ModelItem{cheap, unpriced}}, {Items: []*ModelItem{pricey}}}

	alignPriceColumns(groups)

	require.Equal(t, " 0.10    0.50", modelPriceLabel(cheap.model, cheap.priceInWidth, cheap.priceOutWidth))
	require.Equal(t, "15.00  180.00", modelPriceLabel(pricey.model, pricey.priceInWidth, pricey.priceOutWidth))
	require.Empty(t, modelPriceLabel(unpriced.model, unpriced.priceInWidth, unpriced.priceOutWidth))
}

func TestModelItemRenderFitsWidthAndKeepsPrice(t *testing.T) {
	t.Parallel()

	st := styles.IsobitStyles()
	item := NewModelItem(&st, catwalk.Provider{Name: "A Very Long Provider Name"}, catwalk.Model{
		Name:         "Claude Sonnet 4.5",
		CostPer1MIn:  15,
		CostPer1MOut: 75,
	}, ModelTypeLarge, true)
	item.SetCurrent(true)
	alignPriceColumns([]ModelGroup{{Items: []*ModelItem{item}}})

	for _, width := range []int{30, 45, 60} {
		rendered := item.Render(width)
		require.Equal(t, width, lipgloss.Width(rendered))
		require.Contains(t, ansi.Strip(rendered), "15.00  75.00")
	}
}
