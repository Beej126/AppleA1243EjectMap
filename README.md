# Apple A1243 Keyboard Eject Key Remapper ⏏️

A lightweight, zero-dependency Windows utility written in Go that hooks raw HID input packets from the Apple A1243 USB Extended Keyboard to remap the hardware **Eject** key to any key combination, system action, executable, or Virtual Key (VK) code.

Because the Apple A1243 Eject key operates on an explicit HID Consumer Control / Vendor report page rather than standard keyboard scancodes, standard remapping tools like SharpKeys cannot detect or remap it on its own. This tool bridges that gap by running silently in the background, listening for the raw HID packet, and natively triggering the target shortcut or action in Windows.

---

## Features

* **Task Tray Icon** with High-DPI support (Windhawk taskbar tweak compatible =) 
* **Friendly Shortcut Parsing:** supports multi-modifier key combinations (e.g., `Win+Shift+S`, `Ctrl+Shift+Esc`, `Alt+F4`) and System Actions `lock`, `sleep`, `hibernate`, `monitors-off`, `reset-gfx`, `taskmgr`, `clipboard`, `gamebar`, `mute`, `volup`, `voldown`
* **Startup Installer:** Easily installs a shortcut to the Windows Startup folder configured with your target arguments.

### TBD

* Take over handling **<kbd>Fn</kbd>** + F7-F12 as media events... and F1-F4 for that matter... starts to feel like there should be a config file now (instead of command line args), where we make Fn+Fx mappings to the special Windows functions already implemented: play/pause, skip, reverse, lock-screen, task-view, etc.

---

## Usage

### Command Line Syntax

`AppleA1243EjectMap.exe <SHORTCUT|ACTION|VK_CODE> [FLAGS]`

### Supported Input Types

| Input | Syntax | Description | Example |
| --- | --- | --- | --- |
| **Key Combos** | `[MOD+]KEY` | Friendly modifier + key sequences | `Win+Shift+S`, `Ctrl+Shift+Esc`, `Alt+F4` |
| **System Actions** | Named Action | Direct Win32 OS action commands | `monitors-off`, `reset-gfx`, `lock`, `sleep` |
| **Executables** | `run:<command>` | Launch a program, app, or script | `run:wt.exe`, `run:notepad.exe` |
| **Single Keys** | `KEY` / `0xVK` | Single named key or Hex Virtual Key | `F24`, `PrintScreen`, `0x87` |

### Built-in System Actions

* `monitors-off`: Turn off all connected displays instantly.
* `reset-gfx`: Restart Windows graphics driver stack (`Win+Ctrl+Shift+B`).
* `lock`: Lock Windows Workstation natively.
* `sleep`: Put PC into Sleep mode.
* `hibernate`: Put PC into Hibernation.
* `taskmgr`: Open Task Manager directly.
* `clipboard`: Open Clipboard History (`Win+V`).
* `gamebar`: Open Xbox Game Bar (`Win+G`).
* `mute` / `volup` / `voldown`: Native system volume adjustments.

### Flags

* `-debug`: Launches an interactive debug console window displaying real-time raw HID packet logs and injection events.
* `-install`: Generates a shortcut in your Windows Startup directory pointing to the binary with your specified action argument, ensuring it runs automatically on boot.

---

### Examples

**Turn off monitors instantly:**
`.\AppleA1243EjectMap.exe monitors-off`

**Reset Graphics Driver:**
`.\AppleA1243EjectMap.exe reset-gfx`

**Trigger Windows Snipping Tool:**
`.\AppleA1243EjectMap.exe Win+Shift+S`

**Lock Computer directly:**
`.\AppleA1243EjectMap.exe lock`

**Launch Windows Terminal:**
`.\AppleA1243EjectMap.exe run:wt.exe`

**Install any shortcut to Windows Startup:**
`.\AppleA1243EjectMap.exe monitors-off -install`

**Run in interactive debug mode:**
`.\AppleA1243EjectMap.exe reset-gfx -debug`

---

## References

Remapping non-standard or cross-platform keyboards on Windows often requires operating across multiple layers of the OS input stack. Below are key tools and references used when configuring Apple hardware on Windows, alongside their strengths and limitations.

| Tool / Method | Layer | Pros | Cons |
| --- | --- | --- | --- |
| [Hidfalum driver](https://web.archive.org/web/20260414025507/https://maxtdp.com/software/hidfalum/) 👍💯 | low level filter driver | solves the main painpoints (except for eject key) - swapping Win & Alt to the Windows key locations, flipping Fn to be Insert, Clear to be NumLock, F13-F15 = Prt Screen, etc. | none - it's pretty perfect 
| [SharpKeys](https://github.com/randyrants/sharpkeys) 👍 | Hardware Registry (`Scancode Map`) | Writes directly to `HKLM\SYSTEM\CurrentControlSet\Control\Keyboard Layout`; zero background resource usage; completely native OS remapping. | Cannot map hardware keys that do not produce standard Windows scancodes (such as the A1243 Eject key). |
| [PowerToys Keyboard Manager](https://learn.microsoft.com/en-us/windows/powertoys/keyboard-manager) 👉👈 | User-Land Windows Hook | Native Microsoft GUI; supports application-specific remapping and key combinations on the fly. | Requires the heavy PowerToys background process to remain active; cannot intercept raw HID Vendor/Consumer pages. |
| [3RVX](https://3rvx.com/) 👍👍 | User-Land OSD & Hotkey Hook | Provides clean macOS-style translucent HUDs for volume and brightness; native support for custom hotkey actions and drive eject notifications. | Focused primarily on volume/OSD feedback rather than low-level scancode remapping for utility keys. |
| AutoHotkey (AHK) 👎 | User-Land Hooking | Extremely flexible scripting engine; capable of handling complex hotkey logic, conditional shortcuts, and window management. | High barrier to entry for simple key swaps; requires running script interpreters in the background; struggles with raw HID consumer reports without complex third-party libraries. |

🎉 Have some fun, get a cheap thermal label maker to create custom keycaps, e.g. [Nelko P21](https://www.amazon.com/Nelko-P21-Bluetooth-Templates-Organizing/dp/B0CHJR142G?th=1)

<img height="300" alt="image" src="https://github.com/user-attachments/assets/69f2ec44-c3c4-4b54-bf04-fca83acc5513" />

---

## Build Instructions

one time compiler install: `winget install --scope machine --id GoLang.Go`

`build.cmd`

