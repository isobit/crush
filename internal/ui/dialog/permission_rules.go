package dialog

import (
	"context"
	"strconv"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/list"
	"github.com/charmbracelet/crush/internal/ui/util"
	uv "github.com/charmbracelet/ultraviolet"
)

// PermissionRulesID is the identifier for the permission rules dialog.
const PermissionRulesID = "permission_rules"

type permissionRulesMode uint8

const (
	permissionRulesModeNormal permissionRulesMode = iota
	permissionRulesModeDeleting
)

// PermissionRules is a dialog for managing persistent permission rules.
type PermissionRules struct {
	com          *common.Common
	help         help.Model
	list         *list.FilterableList
	input        textinput.Model
	rules        []db.PermissionRule
	sessionPerms []permission.PermissionRequest
	mode         permissionRulesMode

	keyMap struct {
		Select        key.Binding
		Next          key.Binding
		Previous      key.Binding
		UpDown        key.Binding
		Delete        key.Binding
		ConfirmDelete key.Binding
		CancelDelete  key.Binding
		Close         key.Binding
	}
}

var _ Dialog = (*PermissionRules)(nil)

// NewPermissionRules creates a new permission rules management dialog.
func NewPermissionRules(com *common.Common, sessionPerms []permission.PermissionRequest) (*PermissionRules, error) {
	p := new(PermissionRules)
	p.mode = permissionRulesModeNormal
	p.com = com
	p.sessionPerms = sessionPerms

	rules, err := com.Workspace.PermissionListRules(context.TODO())
	if err != nil {
		return nil, err
	}

	p.rules = rules

	h := help.New()
	h.Styles = com.Styles.DialogHelpStyles()
	p.help = h

	p.list = list.NewFilterableList(p.buildItems(permissionRulesModeNormal)...)
	p.list.Focus()
	p.list.SetSelected(0)

	p.input = textinput.New()
	p.input.SetVirtualCursor(false)
	p.input.Placeholder = "Type to filter"
	p.input.SetStyles(com.Styles.TextInput)
	p.input.Focus()

	p.keyMap.Select = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	)
	p.keyMap.Next = key.NewBinding(
		key.WithKeys("down", "ctrl+n"),
		key.WithHelp("↓", "next item"),
	)
	p.keyMap.Previous = key.NewBinding(
		key.WithKeys("up", "ctrl+p"),
		key.WithHelp("↑", "previous item"),
	)
	p.keyMap.UpDown = key.NewBinding(
		key.WithKeys("up", "down"),
		key.WithHelp("↑↓", "navigate"),
	)
	p.keyMap.Delete = key.NewBinding(
		key.WithKeys("ctrl+x"),
		key.WithHelp("ctrl+x", "delete"),
	)
	p.keyMap.ConfirmDelete = key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "delete"),
	)
	p.keyMap.CancelDelete = key.NewBinding(
		key.WithKeys("n", "esc"),
		key.WithHelp("n", "cancel"),
	)
	p.keyMap.Close = CloseKey

	return p, nil
}

// ID implements Dialog.
func (p *PermissionRules) ID() string {
	return PermissionRulesID
}

// HandleMsg implements Dialog.
func (p *PermissionRules) HandleMsg(msg tea.Msg) Action {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch p.mode {
		case permissionRulesModeDeleting:
			switch {
			case key.Matches(msg, p.keyMap.ConfirmDelete):
				action := p.confirmDeleteRule()
				p.list.SetItems(p.buildItems(permissionRulesModeNormal)...)
				if len(p.rules) > 0 || len(p.sessionPerms) > 0 {
					p.list.SelectFirst()
					p.list.ScrollToSelected()
				}
				return action
			case key.Matches(msg, p.keyMap.CancelDelete):
				p.mode = permissionRulesModeNormal
				p.list.SetItems(p.buildItems(permissionRulesModeNormal)...)
			}
		default:
			switch {
			case key.Matches(msg, p.keyMap.Close):
				return ActionClose{}
			case key.Matches(msg, p.keyMap.Delete):
				if len(p.rules) == 0 && len(p.sessionPerms) == 0 {
					return nil
				}
				p.mode = permissionRulesModeDeleting
				p.list.SetItems(p.buildItems(permissionRulesModeDeleting)...)
			case key.Matches(msg, p.keyMap.Previous):
				p.list.Focus()
				if p.list.IsSelectedFirst() {
					p.list.SelectLast()
				} else {
					p.list.SelectPrev()
				}
				p.list.ScrollToSelected()
			case key.Matches(msg, p.keyMap.Next):
				p.list.Focus()
				if p.list.IsSelectedLast() {
					p.list.SelectFirst()
				} else {
					p.list.SelectNext()
				}
				p.list.ScrollToSelected()
			default:
				var cmd tea.Cmd
				p.input, cmd = p.input.Update(msg)
				value := p.input.Value()
				p.list.SetFilter(value)
				p.list.ScrollToTop()
				p.list.SetSelected(0)
				return ActionCmd{cmd}
			}
		}
	}
	return nil
}

// Cursor returns the cursor position relative to the dialog.
func (p *PermissionRules) Cursor() *tea.Cursor {
	return InputCursor(p.com.Styles, p.input.Cursor())
}

// Draw implements [Dialog].
func (p *PermissionRules) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	t := p.com.Styles
	width := max(0, min(defaultDialogMaxWidth, area.Dx()-t.Dialog.View.GetHorizontalBorderSize()))
	height := max(0, min(defaultDialogHeight, area.Dy()-t.Dialog.View.GetVerticalBorderSize()))
	innerWidth := width - t.Dialog.View.GetHorizontalFrameSize()
	heightOffset := t.Dialog.Title.GetVerticalFrameSize() + titleContentHeight +
		t.Dialog.InputPrompt.GetVerticalFrameSize() + inputContentHeight +
		t.Dialog.HelpView.GetVerticalFrameSize() +
		t.Dialog.View.GetVerticalFrameSize()
	p.input.SetWidth(max(0, innerWidth-t.Dialog.InputPrompt.GetHorizontalFrameSize()-1))
	p.list.SetSize(innerWidth, height-heightOffset)
	p.help.SetWidth(innerWidth)

	var cur *tea.Cursor
	rc := NewRenderContext(t, width)
	rc.Title = "Permission Rules"

	switch p.mode {
	case permissionRulesModeDeleting:
		rc.TitleStyle = t.Dialog.Sessions.DeletingTitle
		rc.TitleGradientFromColor = t.Dialog.Sessions.DeletingTitleGradientFromColor
		rc.TitleGradientToColor = t.Dialog.Sessions.DeletingTitleGradientToColor
		rc.ViewStyle = t.Dialog.Sessions.DeletingView
		rc.AddPart(t.Dialog.Sessions.DeletingMessage.Render("Delete this permission rule?"))
	default:
		inputView := t.Dialog.InputPrompt.Render(p.input.View())
		cur = p.Cursor()
		rc.AddPart(inputView)
	}

	listView := t.Dialog.List.Height(p.list.Height()).Render(p.list.Render())
	rc.AddPart(listView)
	rc.Help = p.help.View(p)

	view := rc.Render()

	DrawCenterCursor(scr, area, view, cur)
	return cur
}

func (p *PermissionRules) confirmDeleteRule() Action {
	item := p.list.SelectedItem()
	p.mode = permissionRulesModeNormal
	if item == nil {
		return nil
	}

	switch v := item.(type) {
	case *PermissionRuleItem:
		p.removeRule(v.ID())
		return ActionCmd{p.deleteRuleCmd(v.PermissionRule.ID)}
	case *SessionPermissionItem:
		p.removeSessionPerm(v.PermissionRequest.ID)
		return ActionCmd{p.deleteSessionPermCmd(v.PermissionRequest.SessionID, v.PermissionRequest.ID)}
	default:
		return nil
	}
}

func (p *PermissionRules) removeRule(id string) {
	var newRules []db.PermissionRule
	for _, r := range p.rules {
		if strconv.FormatInt(r.ID, 10) == id {
			continue
		}
		newRules = append(newRules, r)
	}
	p.rules = newRules
}

// buildItems combines session permission items and persistent rule items into
// a single list.
func (p *PermissionRules) buildItems(mode permissionRulesMode) []list.FilterableItem {
	var items []list.FilterableItem
	items = append(items, sessionPermissionItems(p.com.Styles, mode, p.sessionPerms...)...)
	items = append(items, permissionRuleItems(p.com.Styles, mode, p.rules...)...)
	return items
}

func (p *PermissionRules) deleteRuleCmd(id int64) tea.Cmd {
	return func() tea.Msg {
		err := p.com.Workspace.PermissionDeleteRule(context.TODO(), id)
		if err != nil {
			return util.NewErrorMsg(err)
		}
		return nil
	}
}

func (p *PermissionRules) removeSessionPerm(permissionID string) {
	var filtered []permission.PermissionRequest
	for _, sp := range p.sessionPerms {
		if sp.ID == permissionID {
			continue
		}
		filtered = append(filtered, sp)
	}
	p.sessionPerms = filtered
}

func (p *PermissionRules) deleteSessionPermCmd(sessionID, permissionID string) tea.Cmd {
	return func() tea.Msg {
		p.com.Workspace.PermissionDeleteSessionPermission(sessionID, permissionID)
		return nil
	}
}

// ShortHelp implements [help.KeyMap].
func (p *PermissionRules) ShortHelp() []key.Binding {
	switch p.mode {
	case permissionRulesModeDeleting:
		return []key.Binding{
			p.keyMap.ConfirmDelete,
			p.keyMap.CancelDelete,
		}
	default:
		return []key.Binding{
			p.keyMap.UpDown,
			p.keyMap.Delete,
			p.keyMap.Close,
		}
	}
}

// FullHelp implements [help.KeyMap].
func (p *PermissionRules) FullHelp() [][]key.Binding {
	m := [][]key.Binding{}
	slice := []key.Binding{
		p.keyMap.UpDown,
		p.keyMap.Delete,
		p.keyMap.Close,
	}
	switch p.mode {
	case permissionRulesModeDeleting:
		slice = []key.Binding{
			p.keyMap.ConfirmDelete,
			p.keyMap.CancelDelete,
		}
	}
	for i := 0; i < len(slice); i += 4 {
		end := min(i+4, len(slice))
		m = append(m, slice[i:end])
	}
	return m
}
