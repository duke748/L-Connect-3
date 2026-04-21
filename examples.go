package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newExamplesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "examples",
		Short: "Show copy-paste command examples",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			printCommonExamples()
		},
	}

	cmd.AddCommand(
		newExamplesLinkedCmd(),
		newExamplesSplitCmd(),
	)

	return cmd
}

func newExamplesLinkedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "linked",
		Short: "Show linked-effect examples",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			printLinkedExamples()
		},
	}
}

func newExamplesSplitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "split",
		Short: "Show split-effect examples",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			printSplitExamples()
		},
	}
}

func printCommonExamples() {
	fmt.Println("l-connect3-cli examples")
	fmt.Println()
	fmt.Println("Quick checks:")
	fmt.Println("  .\\l-connect3-cli.exe hid-probe")
	fmt.Println("  .\\l-connect3-cli.exe hid-rpm")
	fmt.Println("  .\\l-connect3-cli.exe hid-status")
	fmt.Println()
	fmt.Println("Single effect mode:")
	fmt.Println("  .\\l-connect3-cli.exe hid-effect rainbow --port 2 --speed 2 --brightness 100 --direction 1")
	fmt.Println("  .\\l-connect3-cli.exe hid-effect breathing --port 2 --color \"#FF0000\" --speed 2 --brightness 70")
	fmt.Println()
	fmt.Println("Linked and split recipes:")
	fmt.Println("  .\\l-connect3-cli.exe examples linked")
	fmt.Println("  .\\l-connect3-cli.exe examples split")
	fmt.Println()
	fmt.Println("Useful follow-up:")
	fmt.Println("  .\\l-connect3-cli.exe ascii-status")
}

func printLinkedExamples() {
	fmt.Println("Linked effect examples")
	fmt.Println()
	fmt.Println("Rainbow wave on one group:")
	fmt.Println("  .\\l-connect3-cli.exe hid-effect linked rainbow --port 2 --colors \"#FF0000,#0000FF,#00FF66,#FF8800\" --speed 2 --brightness 100 --direction 1")
	fmt.Println()
	fmt.Println("Mixing with two colors:")
	fmt.Println("  .\\l-connect3-cli.exe hid-effect linked mixing --port 2 --colors \"#00D5FF,#FF4D7A\" --speed 2 --brightness 100 --direction 0")
	fmt.Println()
	fmt.Println("Runway with two colors:")
	fmt.Println("  .\\l-connect3-cli.exe hid-effect linked runway --port 2 --colors \"#FF8800,#FFFFFF\" --speed 2 --brightness 100 --direction 0")
	fmt.Println()
	fmt.Println("Apply linked mode to all groups:")
	fmt.Println("  .\\l-connect3-cli.exe hid-effect linked rainbow --port 0 --colors \"#FF0000,#0000FF,#00FF66,#FF8800\" --speed 2 --brightness 100 --direction 1")
}

func printSplitExamples() {
	fmt.Println("Split effect examples")
	fmt.Println()
	fmt.Println("Door + Meteor on one group:")
	fmt.Println("  .\\l-connect3-cli.exe hid-effect split door meteor --port 2 --primary-colors \"#FF8800,#FFFFFF\" --secondary-colors \"#00D5FF,#FF4D7A\" --speed 2 --brightness 100 --direction 0")
	fmt.Println()
	fmt.Println("Stack + Runway on one group:")
	fmt.Println("  .\\l-connect3-cli.exe hid-effect split stack runway --port 4 --primary-colors \"#FF4D7A\" --secondary-colors \"#00D5FF,#FFFFFF\" --speed 2 --brightness 100 --direction 1")
	fmt.Println()
	fmt.Println("Split mode across all groups:")
	fmt.Println("  .\\l-connect3-cli.exe hid-effect split door meteor --port 0 --primary-colors \"#FF8800,#FFFFFF\" --secondary-colors \"#00D5FF,#FF4D7A\" --speed 2 --brightness 100 --direction 0")
}
