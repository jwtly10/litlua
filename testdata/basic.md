<!-- @pragma output: init.lua -->
<!-- @pragma debug: true -->

# Basic Neovim Configuration

This is a basic Neovim configuration showing how we can use literate programming
to better document our setup.

## Core Settings

These are our basic Vim settings that we want to apply:

```lua
vim.opt.number = true
vim.opt.relativenumber = true
vim.opt.expandtab = true
vim.opt.tabstop = 4
vim.opt.shiftwidth = 4
```

## Key Mappings

Here we set up some basic key mappings:

```lua
-- Leader key
vim.g.mapleader = " "

-- Quick save
vim.keymap.set('n', '<leader>w', ':w<CR>')

-- Quick quit
vim.keymap.set('n', '<leader>q', ':q<CR>')
```

## Plugin Setup

Basic plugin setup using lazy.nvim:

```lua
local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not vim.loop.fs_stat(lazypath) then
    vim.fn.system({
        "git",
        "clone",
        "--filter=blob:none",
        "https://github.com/folke/lazy.nvim.git",
        "--branch=stable",
        lazypath,
    })
end
vim.opt.rtp:prepend(lazypath)
```