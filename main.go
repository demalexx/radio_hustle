package main

import (
	"github.com/spf13/cobra"
	"log"
	"radio_hustle/radio_hustle_calendar"
)

const (
	Version = "1.0.0"
)

func main() {
	rootCmd := initArgs()

	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

func initArgs() *cobra.Command {
	var (
		credentialsFile string
	)

	rootCmd := &cobra.Command{
		Use:     "radio_hustle_calendar",
		Args:    cobra.ExactArgs(1),
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			calendarService := radio_hustle_calendar.GetCalendarService(
				credentialsFile,
			)
			syncer := radio_hustle_calendar.CreateSyncer(
				calendarService,
				args[0],
			)
			syncResult := syncer.Sync()

			log.Printf(
				"Events added: %d, deleted: %d, updated: %d, up-to-date: %d",
				syncResult.Added,
				syncResult.Deleted,
				syncResult.Updated,
				syncResult.UpToDate,
			)
		},
	}
	rootCmd.PersistentFlags().StringVar(
		&credentialsFile,
		"credentials",
		"",
		"file with Google service account",
	)
	calendarCmd := &cobra.Command{
		Use: "calendar",
	}
	calendarListCmd := &cobra.Command{
		Use: "list",
		Run: func(cmd *cobra.Command, args []string) {
			calendarService := radio_hustle_calendar.GetCalendarService(
				credentialsFile,
			)
			radio_hustle_calendar.ListCalendars(calendarService)
		},
	}
	calendarCreateCmd := &cobra.Command{
		Use: "create",
		Run: func(cmd *cobra.Command, args []string) {
			calendarService := radio_hustle_calendar.GetCalendarService(
				credentialsFile,
			)
			radio_hustle_calendar.CreateCalendar(calendarService)
		},
	}
	calendarDelCmd := &cobra.Command{
		Use:  "del",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			calendarService := radio_hustle_calendar.GetCalendarService(
				credentialsFile,
			)
			radio_hustle_calendar.DelCalendar(calendarService, args[0])
		},
	}

	err := rootCmd.MarkPersistentFlagFilename("credentials")
	if err != nil {
		log.Fatal(err)
	}
	err = rootCmd.MarkPersistentFlagRequired("credentials")
	if err != nil {
		log.Fatal(err)
	}

	calendarCmd.AddCommand(calendarListCmd)
	calendarCmd.AddCommand(calendarCreateCmd)
	calendarCmd.AddCommand(calendarDelCmd)

	rootCmd.AddCommand(calendarCmd)

	return rootCmd
}
