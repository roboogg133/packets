package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/roboogg133/packets/cmd/packets/database"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config {name or id}",
	Short: "Show package configuration file",
	Long:  "Show package configuration file",
	Args:  cobra.RangeArgs(1, 1),
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
			if err == sql.ErrNoRows {
				fmt.Printf("package %s not found\n", insertedName)
			} else {
				fmt.Println("Error getting package ID:", err)
			}
			os.Exit(1)
		}

		flags, err := database.GetAllFromFlag(id, "config", db)
		if err != nil {
			fmt.Println("Error getting flags:", err)
			os.Exit(1)
		}

		if len(flags) == 0 {
			fmt.Println("0 configuration flags set")
			os.Exit(0)
		}

		defer func() {
			if r := recover(); r == "can't solve user home directory" {
			} else {
				os.Exit(1)
			}
		}()
		usrhomeDir, err := os.UserHomeDir()
		if err != nil {
			panic("can't solve user home directory")
		}
		for _, flag := range flags {
			flag.Path = strings.ReplaceAll(flag.Path, UserHomeDirPlaceholder, usrhomeDir)
			flag.Path = strings.ReplaceAll(flag.Path, UsernamePlaceholder, os.Getenv("USER"))
			fmt.Printf("\033[1m[ %s ]\033[0m\n - \033[2m%s\033[0m\n\n", flag.Name, flag.Path)
		}

	},
}

var flagCmd = &cobra.Command{
	Use:   "flag {flag} {package name or package id}",
	Short: "Show package flags by a key",
	Long:  "Show package flags by a key",
	Args:  cobra.RangeArgs(2, 2),
	Run: func(cmd *cobra.Command, args []string) {
		insertedName := args[1]

		db, err := sql.Open("sqlite3", InternalDB)
		if err != nil {
			fmt.Println("Error opening database:", err)
			os.Exit(1)
		}
		defer db.Close()
		database.PrepareDataBase(db)

		id, err := database.GetPackageId(insertedName, db)
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Printf("package %s not found\n", insertedName)
			} else {
				fmt.Println("Error getting package ID:", err)
			}
			os.Exit(1)
		}

		flags, err := database.GetAllFromFlag(id, args[0], db)
		if err != nil {
			fmt.Println("Error getting flags:", err)
			os.Exit(1)
		}

		if len(flags) == 0 {
			fmt.Printf("0 %s flags set\n", args[0])
			os.Exit(0)
		}

		defer func() {
			if r := recover(); r == "can't solve user home directory" {
			} else {
				os.Exit(1)
			}
		}()
		usrhomeDir, err := os.UserHomeDir()
		if err != nil {
			panic("can't solve user home directory")
		}
		for _, flag := range flags {
			flag.Path = strings.ReplaceAll(flag.Path, UserHomeDirPlaceholder, usrhomeDir)
			flag.Path = strings.ReplaceAll(flag.Path, UsernamePlaceholder, os.Getenv("USER"))
			fmt.Printf("\033[1m[ %s ]\033[0m\n - \033[2m%s\033[0m\n\n", flag.Name, flag.Path)
		}

	},
}
