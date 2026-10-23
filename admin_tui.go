package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type adminModel struct {
	tab     int
	users   []User
	acls    []aclEntry
	audits  []AuditLog
	err     error
	loading bool
	width   int
	height  int

	mode   string // "", addUserName, addUserRole, deleteUser, grantUser, grantRepo, grantPerm, revokeUser, revokeRepo
	input  textinput.Model
	buf    string
	status string
}

type adminDataMsg struct {
	users  []User
	acls   []aclEntry
	audits []AuditLog
	err    error
}

func loadAdminData() tea.Msg {
	users, err := ListUsers()
	if err != nil {
		return adminDataMsg{err: err}
	}
	acls, err := listACL()
	if err != nil {
		return adminDataMsg{err: err}
	}
	audits, err := readAuditEntries()
	if err != nil {
		return adminDataMsg{err: err}
	}
	if len(audits) > 20 {
		audits = audits[len(audits)-20:]
	}
	return adminDataMsg{users: users, acls: acls, audits: audits}
}

func newAdminModel() adminModel {
	ti := textinput.New()
	ti.CharLimit = 128
	ti.Width = 40
	return adminModel{loading: true, input: ti}
}

func (m adminModel) Init() tea.Cmd {
	return loadAdminData
}

func (m adminModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case adminDataMsg:
		m.loading = false
		m.err = msg.err
		m.users = msg.users
		m.acls = msg.acls
		m.audits = msg.audits
	case tea.KeyMsg:
		if m.mode != "" {
			switch msg.String() {
			case "esc":
				m.mode = ""
				m.buf = ""
				m.input.SetValue("")
				m.input.Blur()
				return m, nil
			case "enter":
				val := strings.TrimSpace(m.input.Value())
				switch m.mode {
				case "addUserName":
					if val == "" {
						m.status = "name required"
						return m, nil
					}
					m.buf = val
					m.mode = "addUserRole"
					m.input.SetValue("")
					m.input.Placeholder = "role (admin/member/read-only)"
					m.input.Focus()
				case "addUserRole":
					if err := AddUser(m.buf, val); err != nil {
						m.status = err.Error()
					} else {
						m.status = "user added"
						logAudit("user-add", "", true, "")
					}
					m.mode = ""
					m.buf = ""
					m.input.SetValue("")
					m.input.Blur()
					m.loading = true
					return m, loadAdminData
				case "deleteUser":
					if err := RemoveUser(val); err != nil {
						m.status = err.Error()
					} else {
						m.status = "user removed"
						logAudit("user-remove", "", true, "")
					}
					m.mode = ""
					m.input.SetValue("")
					m.input.Blur()
					m.loading = true
					return m, loadAdminData
				case "grantUser":
					m.buf = val
					m.mode = "grantRepo"
					m.input.SetValue("")
					m.input.Placeholder = "repo key (host/owner/repo)"
					m.input.Focus()
				case "grantRepo":
					m.buf = m.buf + "\x00" + val
					m.mode = "grantPerm"
					m.input.SetValue("")
					m.input.Placeholder = "permission (read/write/admin)"
					m.input.Focus()
				case "grantPerm":
					parts := strings.SplitN(m.buf, "\x00", 2)
					user := parts[0]
					repo := ""
					if len(parts) > 1 {
						repo = parts[1]
					}
					if err := grantACL(user, repo, val); err != nil {
						m.status = err.Error()
					} else {
						m.status = "access granted"
						logAudit("acl-grant", repo, true, "")
					}
					m.mode = ""
					m.buf = ""
					m.input.SetValue("")
					m.input.Blur()
					m.loading = true
					return m, loadAdminData
				case "revokeUser":
					m.buf = val
					m.mode = "revokeRepo"
					m.input.SetValue("")
					m.input.Placeholder = "repo key (host/owner/repo)"
					m.input.Focus()
				case "revokeRepo":
					if err := revokeACL(m.buf, val); err != nil {
						m.status = err.Error()
					} else {
						m.status = "access revoked"
						logAudit("acl-revoke", val, true, "")
					}
					m.mode = ""
					m.buf = ""
					m.input.SetValue("")
					m.input.Blur()
					m.loading = true
					return m, loadAdminData
				}
				return m, nil
			default:
				m.input, _ = m.input.Update(msg)
				return m, nil
			}
		}

		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "right", "l":
			m.tab = (m.tab + 1) % 3
			m.status = ""
		case "shift+tab", "left", "h":
			m.tab = (m.tab + 2) % 3
			m.status = ""
		case "r":
			m.loading = true
			m.status = ""
			return m, loadAdminData
		}

		if m.tab == 0 {
			switch msg.String() {
			case "a":
				m.mode = "addUserName"
				m.input.Placeholder = "user name"
				m.input.SetValue("")
				m.input.Focus()
			case "d":
				m.mode = "deleteUser"
				m.input.Placeholder = "user name to delete"
				m.input.SetValue("")
				m.input.Focus()
			}
		}
		if m.tab == 1 {
			switch msg.String() {
			case "g":
				m.mode = "grantUser"
				m.input.Placeholder = "user name"
				m.input.SetValue("")
				m.input.Focus()
			case "x":
				m.mode = "revokeUser"
				m.input.Placeholder = "user name"
				m.input.SetValue("")
				m.input.Focus()
			}
		}
	}
	return m, nil
}

var (
	adminTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	adminTabStyle    = lipgloss.NewStyle().Padding(0, 2)
	adminActiveTab   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Padding(0, 2)
	adminErrStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	adminStatusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

func (m adminModel) View() string {
	if m.loading {
		return "Loading admin dashboard...\n"
	}
	if m.err != nil {
		return adminErrStyle.Render(fmt.Sprintf("Error: %v\n", m.err))
	}

	tabs := []string{"Users", "ACL", "Audit"}
	var tabLine string
	for i, t := range tabs {
		if i == m.tab {
			tabLine += adminActiveTab.Render(t)
		} else {
			tabLine += adminTabStyle.Render(t)
		}
	}
	tabLine += "\n"

	var body string
	switch m.tab {
	case 0:
		body = "Users:\n"
		if len(m.users) == 0 {
			body += "  (none)\n"
		}
		for _, u := range m.users {
			body += fmt.Sprintf("  %-20s %-12s %s\n", u.Name, u.Role, u.CreatedAt.Format("2006-01-02 15:04"))
		}
		body += "\n  [a] add user  [d] delete user\n"
	case 1:
		body = "ACL entries:\n"
		if len(m.acls) == 0 {
			body += "  (none)\n"
		}
		for _, a := range m.acls {
			body += fmt.Sprintf("  %-16s %-24s %s\n", a.User, a.RepoKey, a.Permission)
		}
		body += "\n  [g] grant access  [x] revoke access\n"
	case 2:
		body = "Recent audit entries:\n"
		if len(m.audits) == 0 {
			body += "  (none)\n"
		}
		for _, a := range m.audits {
			status := "OK"
			if !a.Success {
				status = "FAIL"
			}
			body += fmt.Sprintf("  %s %-18s %-20s %s\n", a.Timestamp.Format("01-02 15:04"), status, a.Action, a.RepoName)
		}
	}

	out := adminTitleStyle.Render("env-vault admin dashboard") + "\n" +
		tabLine +
		body

	if m.mode != "" {
		out += "\n" + m.input.View() + "\n"
	}
	if m.status != "" {
		out += adminStatusStyle.Render(m.status) + "\n"
	}
	out += "\n[q] quit  [tab] next  [shift+tab] prev  [r] reload\n"
	return out
}

func runAdminTUI() error {
	p := tea.NewProgram(newAdminModel())
	_, err := p.Run()
	return err
}
