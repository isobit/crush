package model

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/home"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
)

// refreshSessionPermissions updates the cached session permissions from the
// permission service.
func (m *UI) refreshSessionPermissions() {
	if m.session == nil {
		return
	}
	m.sessionPermissions = m.com.Workspace.PermissionListSessionPermissions(m.session.ID)
}

// permissionsInfo renders the session permissions section showing permissions
// that have been granted for the current session.
func (m *UI) permissionsInfo(width, maxItems int, isSection bool) string {
	if len(m.sessionPermissions) == 0 {
		return ""
	}

	t := m.com.Styles

	title := t.Resource.Heading.Render("Session Permissions")
	if isSection {
		title = common.Section(t, title, width)
	}
	list := permissionsList(t, m.sessionPermissions, width, maxItems)

	return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s\n\n%s", title, list))
}

// permissionsList renders a list of session permissions with their tool name,
// action, and path, truncating to maxItems if needed.
func permissionsList(t *styles.Styles, perms []permission.PermissionRequest, width, maxItems int) string {
	if maxItems <= 0 {
		return ""
	}

	var rendered []string
	for _, p := range perms {
		if len(rendered) >= maxItems {
			break
		}

		icon := t.Resource.OnlineIcon.String()
		title := t.Resource.Name.Render(fmt.Sprintf("%s:%s", p.ToolName, p.Action))

		var description string
		if p.Path != "" {
			description = t.Resource.StatusText.Render(home.Short(p.Path))
		}

		rendered = append(rendered, common.Status(t, common.StatusOpts{
			Icon:        icon,
			Title:       title,
			Description: description,
		}, width))
	}

	if len(perms) > maxItems {
		remaining := len(perms) - maxItems
		rendered = append(rendered, t.Resource.AdditionalText.Render(fmt.Sprintf("…and %d more", remaining)))
	}

	return strings.Join(rendered, "\n")
}
