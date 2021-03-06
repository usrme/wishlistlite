package main

import "github.com/charmbracelet/bubbles/key"

type customKeyMap struct {
	Input   key.Binding
	Connect key.Binding
	Cancel  key.Binding
	Sort    key.Binding
}

var customKeys = customKeyMap{
	Input: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "input connection"),
	),
	Connect: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "connect"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel input"),
	),
	Sort: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "recently used"),
	),
}
