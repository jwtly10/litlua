-- Generated by LitLua (https://www.github.com/jwtly10/litlua) v0.0.1
-- Source: testdata/extract/basic.md
-- Generated: 2024-01-01T00:00:00Z

-- WARNING: This is an auto-generated file.
-- Do not modify this file directly as changes will be overwritten on next compilation.
-- Instead, modify the source markdown file and recompile.

vim.opt.number = true
vim.opt.relativenumber = true
vim.opt.expandtab = true
vim.opt.tabstop = 4
vim.opt.shiftwidth = 4

-- Leader key
vim.g.mapleader = " "

-- Quick save
vim.keymap.set('n', '<leader>w', ':w<CR>')

-- Quick quit
vim.keymap.set('n', '<leader>q', ':q<CR>')

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

