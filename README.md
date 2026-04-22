# l-connect3-cli

l-connect3-cli is a Windows-first command-line tool for controlling a Lian Li UNI HUB (SL Infinity VID/PID 0x0CF2:0xA102) directly over HID, without relying on the full L-Connect UI.

Each SL Infinity fan has **2 independent RGB channels**: one for the **fan RGB** (main ring/body) and one for the **side lights**. This allows independent effect control or synchronized effects across both lighting zones per fan.

![Fans](fans.gif)

# Why?
I have a stream deck and thought it would be nice to switch effects on the fly using macros. Next step will be to write an actual stream deck plugin. (If I get the time!)

## Features

- Direct HID probe
- Fan control per port and all ports
- RPM telemetry and combined HID status
- Static color control (all ports, mapped port, or raw channel)
- Effect control:
  - single mode (hid-effect <effect>)
  - linked palette mode (hid-effect linked)
  - split dual-effect mode (hid-effect split)
- Persisted port-to-channel mapping
- ASCII dashboard for per-port mode/state overview

## Requirements

- Windows
- L-Connect 3 installed
- Go (for build/run from source)

HID commands require hidapi.dll:

- Available on PATH, or
- Present at C:/Program Files/Lian-Li/L-Connect 3/hidapi.dll

## Build

```powershell
go build -o l-connect3-cli.exe .
```

## Quick Start

```powershell
.\l-connect3-cli.exe hid-probe
.\l-connect3-cli.exe hid-list
.\l-connect3-cli.exe hid-rpm
.\l-connect3-cli.exe hid-status
.\l-connect3-cli.exe hid-fan 1 50
.\l-connect3-cli.exe hid-effect rainbow --port 2 --speed 2 --brightness 100 --direction 1
.\l-connect3-cli.exe hid-effect linked mixing --port 2 --colors "#FF0000,#0000FF" --speed 2 --brightness 100 --direction 0
.\l-connect3-cli.exe hid-effect split door meteor --port 2 --primary-colors "#FF8800,#FFFFFF" --secondary-colors "#00D5FF,#FF4D7A" --speed 2 --brightness 100
.\l-connect3-cli.exe hid-set "#FF6600" 75
.\l-connect3-cli.exe hid-set-port 1 "#FF6600" 75
.\l-connect3-cli.exe hid-set-channel 0 "#FF6600" 75
.\l-connect3-cli.exe hid-map-show
.\l-connect3-cli.exe hid-map-set 2 2 3
.\l-connect3-cli.exe fan-all quiet
.\l-connect3-cli.exe examples
.\l-connect3-cli.exe examples linked
.\l-connect3-cli.exe examples split
.\l-connect3-cli.exe ascii-status
```

## Command Reference

### hid-probe

Tests direct HID open against VID/PID 0x0CF2:0xA102.

```powershell
.\l-connect3-cli.exe hid-probe
```

### hid-list

Enumerates all connected HID devices and prints their VID, PID, usage page, interface number, manufacturer, and product name. Useful for identifying the VID/PID of a different fan controller or hub.

```powershell
.\l-connect3-cli.exe hid-list
```

Example output:

```
VID     PID     Usage  Page   If#  Manufacturer                  Product
------------------------------------------------------------------------------------------
0x1B1C  0x1BF0  0x0006  0x0001  -1
0x0CF2  0xA102  0x00A1  0xFF72  1    ENE                           LianLi-SL-infinity-v1.4
0x1B1C  0x1B2D  0x0006  0x0001  0    Corsair                       Corsair Gaming K95 RGB PLATINUM Keyboard
0x1B1C  0x1B2D  0x0002  0x0001  0    Corsair                       Corsair Gaming K95 RGB PLATINUM Keyboard
0x0461  0x4E9D  0x0002  0x0001  0    DELL                          Alienware 610M
0x1B1C  0x1BF0  0x0002  0x0001  -1
0x0DB0  0x0B58  0x0001  0xFFC0  7    Generic                       USB Audio
0x0FD9  0x006C  0x0001  0x000C  0    Elgato                        Stream Deck XL
```

The Lian Li UNI HUB will appear as `ENE / LianLi-SL-infinity-v1.4` with VID `0x0CF2` and PID `0xA102`. If you have a different Lian Li hub or fan controller, look for the `ENE` manufacturer entry and note its VID/PID.

### hid-fan <port> <speed>

Sets one port to speed 0-100.

```powershell
.\l-connect3-cli.exe hid-fan 1 50
```

### fan-all <speed|preset>

Sets all four ports.

Accepted values:

- Speed: 0-100
- Presets: quiet (35), standard (55), performance (80)

### hid-rpm

Reads current RPM for ports 1-4.

### hid-status

Shows live RPM plus last fan target state.

### hid-set <hex-color> [brightness]

Sets static color on default primary channels (0,2,4,6)—the fan RGB channels for ports 1-4.

Channel layout: channels 0,1 = port 1 (fan + side), channels 2,3 = port 2, channels 4,5 = port 3, channels 6,7 = port 4. By default, this command targets the even channels (fan RGB only).

Color accepts either:

- hex: #RRGGBB or RRGGBB
- named: red, orange, yellow, green, blue, magenta, purple, cyan, aqua, teal, lime, violet, indigo, amber, pink, white, gray/grey, black
- shaded named: light red, dark red, very light blue, very dark orange (works with all named colors)

### hid-set-port <port> <hex-color> [brightness]

Sets static color on one mapped visible port (writes both the fan RGB and side light channels).

### hid-set-channel <channel> <hex-color> [brightness]

Sets static color on one raw HID channel (0-7).

### hid-map-show

Shows current persisted mapping.

### hid-map-set <port> <channelA> <channelB>

Updates mapping for one visible port. `channelA` is the fan RGB channel, `channelB` is the side light channel.

### hid-effect <effect>

Applies a single effect mode to one mapped port or all ports.

```powershell
.\l-connect3-cli.exe hid-effect breathing --port 2 --color "#FF0000" --speed 2 --brightness 70
.\l-connect3-cli.exe hid-effect 0x24 --port 3 --speed 3 --brightness 60 --direction 1
.\l-connect3-cli.exe hid-effect breathing --port 2 --color "light red" --speed 2 --brightness 70
```

Effect argument can be a named alias or numeric 0..255/0x00..0xFF.

Common names include:

- static, breathing, rainbow, rainbowmorph, runway, mixing, stack, colorcycle, meteor, groove, render, tunnel
- door, scan, mopup, heartbeat, heartbeatrunway, disco, electriccurrent, warning

### hid-effect linked <effect>

Applies one linked effect with a palette of 1-4 colors.

```powershell
.\l-connect3-cli.exe hid-effect linked mixing --port 2 --colors "#FF0000,#0000FF" --speed 2 --brightness 100 --direction 0
.\l-connect3-cli.exe hid-effect linked mixing --port 2 --colors "light red,dark blue" --speed 2 --brightness 100 --direction 0
```

### hid-effect split <primary-effect> <secondary-effect>

Applies two independent effects to one fan group: primary effect to the fan RGB channel, secondary effect to the side light channel.

```powershell
.\l-connect3-cli.exe hid-effect split door meteor --port 2 --primary-colors "#FF8800,#FFFFFF" --secondary-colors "#00D5FF,#FF4D7A" --speed 2 --brightness 100
.\l-connect3-cli.exe hid-effect split door meteor --port 2 --primary-colors "dark orange,white" --secondary-colors "light blue,magenta" --speed 2 --brightness 100
```

### ascii-status

Renders an ASCII dashboard of all four ports including:

- mapped channels
- effect mode (single/linked/split)
- palette summary
- fan target snapshot
- live RPM (when available)

```powershell
.\l-connect3-cli.exe ascii-status
```

### examples

Prints copy-paste command recipes directly in the terminal.

```powershell
.\l-connect3-cli.exe examples
.\l-connect3-cli.exe examples linked
.\l-connect3-cli.exe examples split
```

## Effect Cookbook

Ready-to-run examples for common lighting setups.

### Linked Mode (link icon behavior)

Rainbow wave on one group:

```powershell
.\l-connect3-cli.exe hid-effect linked rainbow --port 2 --colors "#FF0000,#0000FF,#00FF66,#FF8800" --speed 2 --brightness 100 --direction 1
```

Mixing with two colors:

```powershell
.\l-connect3-cli.exe hid-effect linked mixing --port 2 --colors "#00D5FF,#FF4D7A" --speed 2 --brightness 100 --direction 0
```

Runway with two colors:

```powershell
.\l-connect3-cli.exe hid-effect linked runway --port 2 --colors "#FF8800,#FFFFFF" --speed 2 --brightness 100 --direction 0
```

Apply same linked effect to all groups:

```powershell
.\l-connect3-cli.exe hid-effect linked rainbow --port 0 --colors "#FF0000,#0000FF,#00FF66,#FF8800" --speed 2 --brightness 100 --direction 1
```

### Split Mode (circle icon behavior)

Door + Meteor on one group:

```powershell
.\l-connect3-cli.exe hid-effect split door meteor --port 2 --primary-colors "#FF8800,#FFFFFF" --secondary-colors "#00D5FF,#FF4D7A" --speed 2 --brightness 100 --direction 0
```

Stack + Runway on one group:

```powershell
.\l-connect3-cli.exe hid-effect split stack runway --port 4 --primary-colors "#FF4D7A" --secondary-colors "#00D5FF,#FFFFFF" --speed 2 --brightness 100 --direction 1
```

Split mode across all groups:

```powershell
.\l-connect3-cli.exe hid-effect split door meteor --port 0 --primary-colors "#FF8800,#FFFFFF" --secondary-colors "#00D5FF,#FF4D7A" --speed 2 --brightness 100 --direction 0
```

### Quick Notes

- Use `--port 1..4` for one visible fan group, or `--port 0` for all groups.
- `linked` accepts `--colors` with 1-4 values.
- `split` accepts `--primary-colors` and `--secondary-colors`, each with 1-4 values.
- Use `ascii-status` after applying effects to confirm stored linked/split state per port.

## Persistence Files

The CLI persists local runtime state in the working directory:

- .l-connect3-cli-map.json
- .l-connect3-cli-state.json
- .l-connect3-cli-lighting.json
- .l-connect3-cli-fans.json

These are machine/controller-specific and should not be committed.

## Release Automation

This repository includes GitHub Actions workflows to automate version tags and release assets.

- `.github/workflows/bump-tag.yml`: manually bump semantic version tags.
- `.github/workflows/release.yml`: build and publish Windows executables when a tag is pushed.

### Recommended Flow

1. Push your latest changes to `dev`.
2. In GitHub, open **Actions -> Bump Tag -> Run workflow**.
3. Choose `release_type`:
  - `patch`: `v0.6.0 -> v0.6.1`
  - `minor`: `v0.6.0 -> v0.7.0`
  - `major`: `v0.6.0 -> v1.0.0`
4. Keep `target_ref` as `dev` (or choose another branch/commit).
5. The workflow creates and pushes the new tag.
6. Tag push triggers the release workflow automatically.
7. GitHub Release is created with:
  - `l-connect3-cli.exe`
  - `l-connect3-cli-<tag>-windows-amd64.exe`

### Notes

- Version format is Semantic Versioning (`vMAJOR.MINOR.PATCH`).
- Current baseline tag: `v0.6.0`.
- You can still create and push tags manually if preferred.

## Troubleshooting

If hid-probe fails:

- Verify L-Connect 3 is installed
- Verify hidapi.dll exists in C:/Program Files/Lian-Li/L-Connect 3
- Reopen terminal after installation
- Try an elevated shell if permissions are restricted

## Project Layout

- main.go: command handlers and persisted state helpers
- cmd.go: Cobra command wiring
- hid_windows.go: Windows HID bridge and packet writes
- hid_stub.go: non-Windows stubs
- ascii_status.go: ASCII dashboard rendering
- effects_advanced.go: linked/split effect workflows

## Current Limitations

- Some advanced effect aliases are convenience mappings and may vary by firmware; use numeric mode codes for precise testing.
- Controller selection is currently first-controller only.

## Future Improvements

- explicit controller/profile selection flags
- full direct-HID RGB mode/effect support
- interactive channel mapping helper command