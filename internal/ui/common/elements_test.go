package common

import (
	"strings"
	"testing"

	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestFormatTokensAndCostPrefixesEstimatedUsage(t *testing.T) {
	t.Parallel()

	sty := styles.CharmtonePantera()

	rendered := formatTokensAndCost(&sty, 120, 1000, 0, true)
	actual := ansi.Strip(rendered)

	require.Contains(t, actual, "~12%")
	require.Contains(t, actual, "(120)")
	require.Contains(t, actual, "$0.00")
	require.True(t, strings.Contains(rendered, sty.ModelInfo.TokenPercentage.Render("~12%")))
}

func TestFormatTokensAndCostOmitsEstimatedPrefix(t *testing.T) {
	t.Parallel()

	sty := styles.CharmtonePantera()

	actual := ansi.Strip(formatTokensAndCost(&sty, 120, 1000, 0, false))

	require.Contains(t, actual, "12%")
	require.NotContains(t, actual, "~12%")
}

func TestFormatUsageTotalsShowsCacheHitRate(t *testing.T) {
	t.Parallel()

	sty := styles.CharmtonePantera()

	rendered := formatUsageTotals(&sty, &ModelContextInfo{
		TotalInputTokens:       2_000,
		TotalCachedInputTokens: 6_000,
		TotalOutputTokens:      1_500,
	})
	require.Equal(t, "2K in, 6K cache reads\n1.5K out", ansi.Strip(rendered))
}

func TestFormatUsageTotalsOmitsEmptySession(t *testing.T) {
	t.Parallel()

	sty := styles.CharmtonePantera()

	require.Empty(t, formatUsageTotals(&sty, &ModelContextInfo{}))
}
