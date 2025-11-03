package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/roboogg133/packets/cmd/packets/database"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config {name or id}",
	Short: "Show package configuration file",
	Long:  "Show package configuration file",
	Args:  cobra.RangeArgs(1, 1),
	PreRun: func(cmd *cobra.Command, args []string) {
		GrantPrivilegies()
	},
	Run: func(cmd *cobra.Command, args []string) {
		insertedName := args[0]

		db, err := sql.Open("sqlite3", InternalDB)
		if err != nil {
			fmt.Println("Error opening database:", err)
			os.Exit(1)
		}
		defer db.Close()
		database.PrepareDataBase(db)

		id, err := database.GetPackageId(insertedName, db)
		if err != nil {
			fmt.Println("Error getting package ID:", err)
			os.Exit(1)
		}

		flags, err := database.GetAllFromFlag(id, "config", db)
		if err != nil {
			fmt.Println("Error getting flags:", err)
			os.Exit(1)
		}

		var all []list.Item

		for _, flag := range flags {
			item := item{
				title: flag.Name,
				desc:  flag.Path,
			}
			all = append(all, item)
		}
		/*
			delegate1 := list.NewDefaultDelegate()
			delegate1.Styles.NormalTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#000000")).
				Margin(1)
			delegate1.Styles.NormalDesc = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#000000"))

			delegate1.Styles.SelectedTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#0056FF")).
				Bold(true).
				Blink(true)

			delegate1.Styles.SelectedDesc = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Faint(true)
		*/
		m := model{list: list.New(all, list.NewDefaultDelegate(), 0, 0)}
		m.list.Title = "Configuration files"

		p := tea.NewProgram(m, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}

	},
}
