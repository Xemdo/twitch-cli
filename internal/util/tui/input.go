// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Pre-defined for user in the Values map in TextInputOptions. Will be updated when iterated through by Charm's Update().
type TextInputPrompt struct {
	// The message that's displayed indicating what the user is entering. e.g. "Client ID:"
	DisplayMessage string

	// Optional. The default value that will appear for the user to automatically press enter on.
	DefaultValue string

	// Validation function. If no validation needed, have it return nil.
	ValidateInput func(string) error

	// Whether or not to hide this value from being displayed to the user. Will use "<hidden>" instead.
	IsSensitive bool

	// If order matters for the map of prompts, this is used for ordering those prompts. Lowest has priority.
	Order int
}

type TextInputOptions struct {
	CharLimit int

	Prompts map[string]TextInputPrompt
}

type model struct {
	textInputModel   textinput.Model
	err              error
	cancelled        bool
	currentPromptKey string
	remainingPrompts map[string]TextInputPrompt
	answeredPrompts  map[string]string
}

// Determines what the next prompt will be from a given map. Does not modify the map.
func determineNextPrompt(remainingPrompts map[string]TextInputPrompt) string {
	lowestKey := ""
	lowestOrder := 2147483647

	for key, prompt := range remainingPrompts {
		if prompt.Order < lowestOrder {
			lowestKey = key
			lowestOrder = prompt.Order
		}
	}

	return lowestKey
}

func determinePlaceholderValue(prompt TextInputPrompt) string {
	if prompt.IsSensitive {
		return "<hidden>"
	}

	return prompt.DefaultValue
}

func determinePlaceholderStyle(prompt TextInputPrompt) lipgloss.Style {
	base := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")) // ANSI grey

	if prompt.IsSensitive {
		return base.Italic(true)
	}

	return base
}

func createInitialModel(options TextInputOptions) model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = options.CharLimit
	ti.Placeholder = determinePlaceholderValue(options.Prompts[determineNextPrompt(options.Prompts)])
	ti.PlaceholderStyle = determinePlaceholderStyle(options.Prompts[determineNextPrompt(options.Prompts)])

	return model{
		textInputModel:   ti,
		err:              nil,
		cancelled:        false,
		currentPromptKey: determineNextPrompt(options.Prompts),
		remainingPrompts: options.Prompts,
		answeredPrompts:  map[string]string{},
	}
}

// Returns (map[string]string, nil) if successful.
// Returns (nil, nil) if cancelled (ctrl+c, esc).
// Returns (nil, err) if error.
func TextInput(options TextInputOptions) (map[string]string, error) {
	p := tea.NewProgram(createInitialModel(options))

	m, err := p.Run()

	// Check for errors with the program itself
	if err != nil {
		return nil, err
	}

	// Check for errors returned by our Update loop
	if m.(model).err != nil {
		return nil, m.(model).err
	}

	// Ctrl+C/Esc
	if m.(model).cancelled {
		return nil, nil
	}

	return m.(model).answeredPrompts, nil
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit

		case tea.KeyEnter:
			currentPrompt := m.remainingPrompts[m.currentPromptKey]
			inputtedValue := m.textInputModel.Value()

			// Check if we should use the placeholder instead
			if len(inputtedValue) == 0 {
				inputtedValue = currentPrompt.DefaultValue
			}

			// Check string validation
			var validationErr error
			if currentPrompt.ValidateInput != nil {
				validationErr = currentPrompt.ValidateInput(inputtedValue)
			}

			if validationErr != nil {
				m.err = fmt.Errorf("Bad input for %s: %s", currentPrompt.DisplayMessage, validationErr.Error())
				return m, tea.Quit
			}

			// Update answered prompt list with the user's input
			m.answeredPrompts[m.currentPromptKey] = inputtedValue

			// Remove current prompt from remaining prompts list, then find the next one
			delete(m.remainingPrompts, m.currentPromptKey)
			m.currentPromptKey = determineNextPrompt(m.remainingPrompts)
			currentPrompt = m.remainingPrompts[m.currentPromptKey] // Update currentPrompt to reflect the changes

			// Reset input and set default for the next prompt
			m.textInputModel.Reset()
			m.textInputModel.Placeholder = determinePlaceholderValue(currentPrompt)
			m.textInputModel.PlaceholderStyle = determinePlaceholderStyle(currentPrompt)

			// Exit if that was the last prompt
			if len(m.remainingPrompts) == 0 {
				return m, tea.Quit
			}
		}

	// Handle all errors
	case error:
		m.err = msg
		return m, nil
	}

	m.textInputModel, cmd = m.textInputModel.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf("%s\n%s\n",
		m.remainingPrompts[m.currentPromptKey].DisplayMessage,
		m.textInputModel.View())
}
