package main

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

func newItemDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	tColor := lipgloss.Color("#a3be8c")
	dColor := lipgloss.Color("#7a8e69")
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(tColor).BorderLeftForeground(tColor)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(dColor).BorderLeftForeground(dColor)

	return d
}
