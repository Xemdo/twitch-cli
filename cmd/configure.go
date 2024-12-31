// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/twitchdev/twitch-cli/internal/util"
	"github.com/twitchdev/twitch-cli/internal/util/tui"
)

var clientID string
var clientSecret string

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configures your Twitch CLI with your Client ID and Secret",
	RunE:  configureCmdRun,
}

func init() {
	rootCmd.AddCommand(configureCmd)

	configureCmd.Flags().StringVarP(&clientID, "client-id", "i", "", "Client ID to use.")
	configureCmd.Flags().StringVarP(&clientSecret, "client-secret", "s", "", "Client Secret to use.")
}

func configureCmdRun(cmd *cobra.Command, args []string) error {
	clientID = viper.GetString("clientId")
	clientSecret = viper.GetString("clientSecret")

	validationFunc := func(s string) error {
		if len(strings.TrimSpace(s)) == 30 || len(strings.TrimSpace(s)) == 31 {
			return nil
		}

		return fmt.Errorf("Invalid length")
	}

	inputPrompts := map[string]tui.TextInputPrompt{
		"clientid": {
			DisplayMessage: "Client ID",
			DefaultValue:   clientID,
			ValidateInput:  validationFunc,
			Order:          0,
		},
		"clientsecret": {
			DisplayMessage: "Client Secret",
			DefaultValue:   clientSecret,
			ValidateInput:  validationFunc,
			Order:          1,
		},
	}

	responses, err := tui.TextInput(tui.TextInputOptions{
		CharLimit: 31,
		Prompts:   inputPrompts,
	})

	if err != nil {
		return fmt.Errorf("%s\nNo configuration was updated. Please check inputs and try again.", err.Error())
	}

	// Check for operation being cancelled
	if err == nil && responses == nil {
		return nil
	}

	// Set the configuration using the new values
	clientID = responses["clientid"]
	clientSecret = responses["clientsecret"]

	viper.Set("clientId", clientID)
	viper.Set("clientSecret", clientSecret)

	configPath, err := util.GetConfigPath()
	if err != nil {
		return err
	}

	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("Failed to write configuration: %v", err.Error())
	}

	fmt.Println("Updated configuration.")

	return nil
}

type configureCmdState int

const (
	ConfigureCmdStateClientID = iota
	ConfigureCmdStateSecret   = iota
)

type configureCmdModel struct {
	input textinput.Model
	err   error
	state configureCmdState
}

func ConfigureCmdInitialModel(state configureCmdState) configureCmdModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 31

	return configureCmdModel{
		input: ti,
		err:   nil,
		state: state,
	}
}

func (m configureCmdModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m configureCmdModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			m.err = validateLength(m.input.Value(), m.state)
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case error:
		m.err = msg
		return m, nil
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m configureCmdModel) View() string {
	return fmt.Sprintf(
		"Client ID\n%s",
		m.input.View(),
	) + "\n"
}

func validateLength(s string, state configureCmdState) error {
	if len(strings.TrimSpace(s)) == 30 || len(strings.TrimSpace(s)) == 31 {
		return nil
	}

	errMsgExtra := ""
	if state == ConfigureCmdStateClientID {
		errMsgExtra = "for Client ID"
	} else if state == ConfigureCmdStateSecret {
		errMsgExtra = "for Client Secret"
	}

	return fmt.Errorf("Invalid length %s", errMsgExtra)
}
