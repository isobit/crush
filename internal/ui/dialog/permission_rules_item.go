package dialog

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/ui/list"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/sahilm/fuzzy"
)

// PermissionRuleItem wraps a [db.PermissionRule] to implement the [ListItem]
// interface.
type PermissionRuleItem struct {
	db.PermissionRule
	t       *styles.Styles
	mode    permissionRulesMode
	m       fuzzy.Match
	cache   map[int]string
	focused bool
}

var _ ListItem = &PermissionRuleItem{}

// Filter returns the filterable value of the permission rule.
func (p *PermissionRuleItem) Filter() string {
	return fmt.Sprintf("%s:%s %s %s", p.ToolName, p.Action, p.Path, paramsSummary(p.PermissionRule))
}

// ID returns the unique identifier of the permission rule.
func (p *PermissionRuleItem) ID() string {
	return fmt.Sprintf("%d", p.PermissionRule.ID)
}

// SetMatch sets the fuzzy match for the permission rule item.
func (p *PermissionRuleItem) SetMatch(m fuzzy.Match) {
	p.cache = nil
	p.m = m
}

// SetFocused sets the focus state of the permission rule item.
func (p *PermissionRuleItem) SetFocused(focused bool) {
	if p.focused != focused {
		p.cache = nil
	}
	p.focused = focused
}

// Render returns the string representation of the permission rule item.
func (p *PermissionRuleItem) Render(width int) string {
	title := fmt.Sprintf("%s:%s", p.ToolName, p.Action)
	if summary := paramsSummary(p.PermissionRule); summary != "" {
		title += " " + summary
	}
	info := p.Path
	sty := ListItemStyles{
		ItemBlurred:     p.t.Dialog.NormalItem,
		ItemFocused:     p.t.Dialog.SelectedItem,
		InfoTextBlurred: p.t.Subtle,
		InfoTextFocused: p.t.Base,
	}

	if p.mode == permissionRulesModeDeleting {
		sty.ItemBlurred = p.t.Dialog.Sessions.DeletingItemBlurred
		sty.ItemFocused = p.t.Dialog.Sessions.DeletingItemFocused
	}

	return renderItem(sty, title, info, p.focused, width, p.cache, &p.m)
}

// paramsSummary extracts a short human-readable summary from the stored
// JSON params. It picks the most relevant field for each tool type.
func paramsSummary(rule db.PermissionRule) string {
	if rule.Params == "" || rule.Params == "{}" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(rule.Params), &m); err != nil {
		return ""
	}

	for _, key := range []string{"command", "file_path", "url", "uri", "prompt", "path", "mcp_name"} {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// permissionRuleItems converts a slice of [db.PermissionRule] to a slice of
// [list.FilterableItem].
func permissionRuleItems(t *styles.Styles, mode permissionRulesMode, rules ...db.PermissionRule) []list.FilterableItem {
	items := make([]list.FilterableItem, len(rules))
	for i, r := range rules {
		items[i] = &PermissionRuleItem{PermissionRule: r, t: t, mode: mode}
	}
	return items
}
