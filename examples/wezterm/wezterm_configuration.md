<!-- @pragma output:.wezterm.lua -->
<img height="128" alt="WezTerm Icon" src="https://raw.githubusercontent.com/wez/wezterm/main/assets/icon/wezterm-icon.svg" align="left">

‎

‎ ‎*Welcome to my literate for Wezterm!*

‎

‎

## Installation

Here you can give yourself custom instructions etc...

## Why wezterm?

Here you can give some brief information about what you like about wezterm etc...

## Configuration

Step one is to setup wezterm lib

```lua
local wezterm = require("wezterm")

local config = {}

if wezterm.config_builder then
    config = wezterm.config_builder()
end
```

<details>
<summary>UI</summary>

I use these fonts, and often switch between different color themes
depending on what I am doing

```lua
config.font_size = 24
config.font = wezterm.font 'JetBrainsMono NF'
-- config.font = wezterm.font 'Iosevka NF'

config.colors = {
    background = "1C1C1C"
    -- background = "202020",
    -- background = "1C2021",
}
```

I also prefer to start with the intial window larger

```lua
config.initial_cols = 100
config.initial_rows = 45

config.window_padding = {
    left = 0,
    right = 0,
    top = 1,
    bottom = 0,
}
```

</details>

<details>
<summary>Key Bindings</summary>

We can define a small utility var for making key bindings easier to read:

```lua
local act = wezterm.action
```

Now we can make option-left and option-right work as expected in the terminal

```lua
config.keys = {
    { key = "LeftArrow",  mods = "OPT", action = act({ SendString = "\x1bb" }) },
    { key = "RightArrow", mods = "OPT", action = act({ SendString = "\x1bf" }) },
    { key = "3",          mods = "OPT", action = act.SendString("#") },
}
```

And we can fix pasting from the correct terminal

```lua
config.keys = {
    { key = "v", mods = "CTRL", action = act.PasteFrom("Clipboard") },
}
```

Not a keybinding, but we are allowed to use the mouse right?

```lua
config.mouse_bindings = {
    {
        event = { Up = { streak = 1, button = "Left" } },
        mods = "META",
        action = act.OpenLinkAtMouseCursor,
    },
}
```

</details>

<details>
<summary>Others</summary>

Theses are just a few nice to have settings :)

```lua
config.adjust_window_size_when_changing_font_size = false

config.window_close_confirmation = "NeverPrompt"
config.use_fancy_tab_bar = false
-- config.hide_tab_bar_if_only_one_tab = true
```

</details>


### And finally...

... we can return our config when wezterm tries to read it!

```lua
return config
```

