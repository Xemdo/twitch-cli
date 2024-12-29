// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
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
	/*var err error
	if clientID == "" {
		clientIDPrompt := promptui.Prompt{
			Label: "Client ID",
			Validate: func(s string) error {
				if len(s) == 30 || len(s) == 31 {
					return nil
				}
				return errors.New("Invalid length for Client ID")
			},
		}

		clientID, err = clientIDPrompt.Run()
	}

	if clientSecret == "" {
		clientSecretPrompt := promptui.Prompt{
			Label: "Client Secret",
			Validate: func(s string) error {
				if len(s) == 30 || len(s) == 31 {
					return nil
				}
				return errors.New("Invalid length for Client Secret")
			},
		}

		clientSecret, err = clientSecretPrompt.Run()
	}

	if clientID == "" && clientSecret == "" {
		return fmt.Errorf("Must specify either the Client ID or Secret")
	}

	viper.Set("clientId", clientID)
	viper.Set("clientSecret", clientSecret)

	configPath, err := util.GetConfigPath()
	if err != nil {
		return err
	}

	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("Failed to write configuration: %v", err.Error())
	}

	fmt.Println("Updated configuration.")*/

	//p := tea.NewProgram(ConfigureCmdInitialModel())
	//if _, err := p.Run(); err != nil {
	//	log.Fatal(err)
	//}

	//var m tea.Model
	if clientID == "" {
		p := tea.NewProgram(ConfigureCmdInitialModel(ConfigureCmdStateClientID))

		m, err := p.Run()
		ccm := m.(configureCmdModel)

		if err != nil {
			return err
		}
		if ccm.err != nil {
			return ccm.err
		}

		fmt.Printf("%s\n", ccm.input.Value())
	}

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
