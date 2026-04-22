package main

import "github.com/spf13/cobra"

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "l-connect3-cli",
		Short:         "Control a Lian Li SL Infinity hub over direct HID",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(
		newHIDListCmd(),
		newASCIIStatusCmd(),
		newExamplesCmd(),
		newHIDProbeCmd(),
		newHIDFanCmd(),
		newHIDEffectCmd(),
		newHIDSetCmd(),
		newHIDSetPortCmd(),
		newHIDSetChannelCmd(),
		newHIDMapShowCmd(),
		newHIDMapSetCmd(),
		newFanAllCmd(),
		newHIDRPMCmd(),
		newHIDStatusCmd(),
	)

	return rootCmd
}

func newASCIIStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ascii-status",
		Short: "Render an ASCII view of ports, effects, and fan state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runASCIIStatus()
		},
	}
}

func newHIDEffectCmd() *cobra.Command {
	port := 0
	color := ""
	speed := 0
	brightness := 100
	direction := 0

	cmd := &cobra.Command{
		Use:   "hid-effect <effect>",
		Short: "Apply an effect mode to all ports or one mapped port",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHIDEffect(args[0], color, port, speed, brightness, direction)
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "target port 1-4; 0 applies to all mapped ports")
	cmd.Flags().StringVar(&color, "color", "", "optional effect color in hex (RRGGBB or #RRGGBB)")
	cmd.Flags().IntVar(&speed, "speed", 0, "effect speed byte (0-255)")
	cmd.Flags().IntVar(&brightness, "brightness", 100, "effect brightness percent (0-100)")
	cmd.Flags().IntVar(&direction, "direction", 0, "effect direction byte (0-255)")

	cmd.AddCommand(
		newHIDEffectLinkedCmd(),
		newHIDEffectSplitCmd(),
	)

	return cmd
}

func newHIDEffectLinkedCmd() *cobra.Command {
	port := 0
	colors := ""
	speed := 0
	brightness := 100
	direction := 0

	cmd := &cobra.Command{
		Use:   "linked <effect>",
		Short: "Apply one linked effect with 1-4 colors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHIDEffectLinked(args[0], colors, port, speed, brightness, direction)
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "target port 1-4; 0 applies to all mapped ports")
	cmd.Flags().StringVar(&colors, "colors", "", "required comma-separated colors: #RRGGBB,#RRGGBB,... (1-4)")
	_ = cmd.MarkFlagRequired("colors")
	cmd.Flags().IntVar(&speed, "speed", 0, "effect speed byte (0-255)")
	cmd.Flags().IntVar(&brightness, "brightness", 100, "effect brightness percent (0-100)")
	cmd.Flags().IntVar(&direction, "direction", 0, "effect direction byte (0-255)")

	return cmd
}

func newHIDEffectSplitCmd() *cobra.Command {
	port := 0
	primaryColors := ""
	secondaryColors := ""
	speed := 0
	brightness := 100
	direction := 0

	cmd := &cobra.Command{
		Use:   "split <primary-effect> <secondary-effect>",
		Short: "Apply two effects per port (one per paired channel)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHIDEffectSplit(args[0], args[1], primaryColors, secondaryColors, port, speed, brightness, direction)
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "target port 1-4; 0 applies to all mapped ports")
	cmd.Flags().StringVar(&primaryColors, "primary-colors", "", "required comma-separated colors for primary effect (1-4)")
	cmd.Flags().StringVar(&secondaryColors, "secondary-colors", "", "required comma-separated colors for secondary effect (1-4)")
	_ = cmd.MarkFlagRequired("primary-colors")
	_ = cmd.MarkFlagRequired("secondary-colors")
	cmd.Flags().IntVar(&speed, "speed", 0, "effect speed byte (0-255)")
	cmd.Flags().IntVar(&brightness, "brightness", 100, "effect brightness percent (0-100)")
	cmd.Flags().IntVar(&direction, "direction", 0, "effect direction byte (0-255)")

	return cmd
}

func newHIDProbeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hid-probe",
		Short: "Probe HID connectivity to the SL Infinity controller",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHIDProbe()
		},
	}
}

func newHIDListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hid-list",
		Short: "List all connected HID devices with VID, PID, and product info",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHIDList()
		},
	}
}

func newHIDFanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hid-fan <port 1-4> <speed 0-100>",
		Short: "Set fan speed for one port",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHIDFan(args[0], args[1])
		},
	}
}

func newHIDSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hid-set <hex-color> [brightness 0-100]",
		Short: "Set static color on default all-port channels",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			brightnessArg := "100"
			if len(args) == 2 {
				brightnessArg = args[1]
			}
			return runHIDSet(args[0], brightnessArg)
		},
	}
}

func newHIDSetPortCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hid-set-port <port 1-4> <hex-color> [brightness 0-100]",
		Short: "Set static color on one mapped port",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			brightnessArg := "100"
			if len(args) == 3 {
				brightnessArg = args[2]
			}
			return runHIDSetPort(args[0], args[1], brightnessArg)
		},
	}
}

func newHIDSetChannelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hid-set-channel <channel 0-7> <hex-color> [brightness 0-100]",
		Short: "Set static color on one raw HID channel",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			brightnessArg := "100"
			if len(args) == 3 {
				brightnessArg = args[2]
			}
			return runHIDSetChannel(args[0], args[1], brightnessArg)
		},
	}
}

func newHIDMapShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hid-map-show",
		Short: "Show persisted port-to-channel mapping",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHIDMapShow()
		},
	}
}

func newHIDMapSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hid-map-set <port 1-4> <channelA 0-7> <channelB 0-7>",
		Short: "Set persisted channel mapping for one port",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHIDMapSet(args[0], args[1], args[2])
		},
	}
}

func newFanAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fan-all <speed 0-100|quiet|standard|performance>",
		Short: "Set all ports to one fan speed/preset",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFanAll(args[0])
		},
	}
}

func newHIDRPMCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hid-rpm",
		Short: "Read current RPM for ports 1-4",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHIDRPM()
		},
	}
}

func newHIDStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hid-status",
		Short: "Show current RPM and last fan target",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHIDStatus()
		},
	}
}
