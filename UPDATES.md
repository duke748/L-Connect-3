# Updates

## April 22, 2026

### hid-list command
Added `hid-list` to enumerate all connected HID devices using `hid_enumerate` from hidapi.dll. Prints VID, PID, usage page, interface number, manufacturer, and product name for every HID device on the system. Useful for identifying the VID/PID of a different fan controller or hub.

```powershell
.\l-connect3-cli.exe hid-list
```

### Dual-channel awareness
Documented that each SL Infinity fan has **2 independent RGB channels**: one for the **fan RGB** (main ring/body) and one for the **side lights**. The channel layout is:

| Port | Fan RGB channel | Side light channel |
|------|-----------------|--------------------|
| 1    | 0               | 1                  |
| 2    | 2               | 3                  |
| 3    | 4               | 5                  |
| 4    | 6               | 7                  |

- `hid-set` targets fan RGB channels (0, 2, 4, 6) by default.
- `hid-set-port` writes to both channels for the selected port.
- `hid-map-set <port> <channelA> <channelB>` — `channelA` is fan RGB, `channelB` is side lights.
- `hid-effect split` applies the primary effect to the fan RGB channel and the secondary effect to the side light channel.
