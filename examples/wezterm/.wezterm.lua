-- Generated by LitLua (https://www.github.com/jwtly10/litlua) v0.0.2
-- Source: /Users/personal/Projects/litlua/examples/wezterm/wezterm_configuration.litlua.md
-- Generated: 2025-01-05T19:19:51Z

-- WARNING: This is an auto-generated file.
-- Do not modify this file directly as changes will be overwritten on next compilation.
-- Instead, modify the source markdown file and recompile.

local wezterm = require("wezterm")

local config = {}

if wezterm.config_builder then
    config = wezterm.config_builder()
end

config.font_size = 24
config.font = wezterm.font 'JetBrainsMono NF'
-- config.font = wezterm.font 'Iosevka NF'

config.colors = {
    background = "1C1C1C"
    -- background = "202020",
    -- background = "1C2021",
}

config.initial_cols = 100
config.initial_rows = 45

config.window_padding = {
    left = 0,
    right = 0,
    top = 1,
    bottom = 0,
}

local act = wezterm.action

config.keys = {
    { key = "LeftArrow",  mods = "OPT", action = act({ SendString = "\x1bb" }) },
    { key = "RightArrow", mods = "OPT", action = act({ SendString = "\x1bf" }) },
    { key = "3",          mods = "OPT", action = act.SendString("#") },
}

config.keys = {
    { key = "v", mods = "CTRL", action = act.PasteFrom("Clipboard") },
}

config.mouse_bindings = {
    {
        event = { Up = { streak = 1, button = "Left" } },
        mods = "META",
        action = act.OpenLinkAtMouseCursor,
    },
}

config.adjust_window_size_when_changing_font_size = false

config.window_close_confirmation = "NeverPrompt"
config.use_fancy_tab_bar = false
-- config.hide_tab_bar_if_only_one_tab = true

return config

