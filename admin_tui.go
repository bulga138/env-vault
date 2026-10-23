package main

import (
	"fmt"

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
	return adminModel{loading: true}
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
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "right", "l":
			m.tab = (m.tab + 1) % 3
		case "shift+tab", "left", "h":
			m.tab = (m.tab + 2) % 3
		case "r":
			m.loading = true
			return m, loadAdminData
		}
	}
	return m, nil
}

var (
	adminTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	adminTabStyle   = lipgloss.NewStyle().Padding(0, 2)
	adminActiveTab  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Padding(0, 2)
	adminErrStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
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
	case 1:
		body = "ACL entries:\n"
		if len(m.acls) == 0 {
			body += "  (none)\n"
		}
		for _, a := range m.acls {
			body += fmt.Sprintf("  %-16s %-24s %s\n", a.User, a.RepoKey, a.Permission)
		}
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

	return adminTitleStyle.Render("env-vault admin dashboard") + "\n" +
		tabLine +
		body +
		"\n[q] quit  [tab] next  [shift+tab] prev  [r] reload\n"
}

func runAdminTUI() error {
	p := tea.NewProgram(newAdminModel())
	_, err := p.Run()
	return err
}
