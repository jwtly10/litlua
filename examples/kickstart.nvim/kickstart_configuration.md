<!-- @pragma output:output.lua -->

![image-removebg-preview](https://github.com/user-attachments/assets/e1ce4825-968f-4b22-a764-dd63c5e1cca5)


# What is Kickstart?

  Kickstart.nvim is *not* a distribution.

  Kickstart.nvim is a starting point for your own configuration.

    The goal is that you can read every line of code, top-to-bottom, understand
    what your configuration is doing, and modify it to suit your needs.

    Once you've done that, you can start exploring, configuring and tinkering to
    make Neovim your own! That might mean leaving Kickstart just the way it is for a while
    or immediately breaking it into modular pieces. It's up to you!

    If you don't know anything about Lua, I recommend taking some time to read through
    a guide. One possible example which will only take 10-15 minutes:
      - https://learnxinyminutes.com/docs/lua/

    After understanding a bit more about Lua, you can use `:help lua-guide` as a
    reference for how Neovim integrates Lua.
    - :help lua-guide
    - (or HTML version): https://neovim.io/doc/user/lua-guide.html

# Kickstart Guide:

  TODO: The very first thing you should do is to run the command `:Tutor` in Neovim.

    If you don't know what this means, type the following:
      - <escape key>
      - :
      - Tutor
      - <enter key>

    (If you already know the Neovim basics, you can skip this step.)

  Once you've completed that, you can continue working through **AND READING** the rest
  of the kickstart init.lua.

  Next, run AND READ `:help`.
    This will open up a help window with some basic information
    about reading, navigating and searching the builtin help documentation.

    This should be the first place you go to look when you're stuck or confused
    with something. It's one of my favorite Neovim features.

    MOST IMPORTANTLY, we provide a keymap "<space>sh" to [s]earch the [h]elp documentation,
    which is very useful when you're not exactly sure of what you're looking for.

  I have left several `:help X` comments throughout the init.lua
    These are hints about where to find more information about the relevant settings,
    plugins or Neovim features used in Kickstart.

   NOTE: Look for lines like this

    Throughout the file. These are for you, the reader, to help you understand what is happening.
    Feel free to delete them once you know what you're doing, but they should serve as a guide
    for when you are first encountering a few different constructs in your Neovim config.

If you experience any errors while trying to install kickstart, run `:checkhealth` for more info.

I hope you enjoy your Neovim journey,

-- TJ

P.S. You can delete this when you're done too. It's your config now! :)

<details>
<summary>Base Settings</summary>

> NOTE: Must happen before plugins are loaded (otherwise wrong leader will be used)

## Set &lt;space&gt; as the leader key
See `:help mapleader`

```lua
vim.g.mapleader = ' '
vim.g.maplocalleader = ' '
```
## Set to true if you have a Nerd Font installed and selected in the terminal
```lua
vim.g.have_nerd_font = false
```
</details>

<details>
<summary>Setting options</summary>

See `:help vim.opt`

For more options, you can see `:help option-list`

> NOTE: You can change these options as you wish!

## Make line numbers default

```lua
vim.opt.number = true
```

## You can also add relative line numbers, to help with jumping.
 Experiment for yourself to see if you like it!
```lua
-- vim.opt.relativenumber = true
```

## Enable mouse mode, can be useful for resizing splits for example!
```lua
vim.opt.mouse = 'a'
```

## Don't show the mode, since it's already in the status line
```lua
vim.opt.showmode = false
```

## Sync clipboard between OS and Neovim.
Schedule the setting after `UiEnter` because it can increase startup-time.
Remove this option if you want your OS clipboard to remain independent.
See `:help 'clipboard'`
```lua
vim.schedule(function()
  vim.opt.clipboard = 'unnamedplus'
end)
```

## Enable break indent
```lua
vim.opt.breakindent = true
```

## Save undo history
```lua
vim.opt.undofile = true
```

### Case-insensitive searching UNLESS \C or one or more capital letters in the search term
```lua
vim.opt.ignorecase = true
vim.opt.smartcase = true
```

### Keep signcolumn on by default
```lua
vim.opt.signcolumn = 'yes'
```

### Decrease update time
```lua
vim.opt.updatetime = 250
```

### Decrease mapped sequence wait time
```lua
vim.opt.timeoutlen = 300
```

### Configure how new splits should be opened
```lua
vim.opt.splitright = true
vim.opt.splitbelow = true
```

### Sets how neovim will display certain whitespace characters in the editor.
See `:help 'list'` and `:help 'listchars'`
```lua
vim.opt.list = true
vim.opt.listchars = { tab = '» ', trail = '·', nbsp = '␣' }
```

### Preview substitutions live, as you type!
```lua
vim.opt.inccommand = 'split'
```

### Show which line your cursor is on
```lua
vim.opt.cursorline = true
```

### Minimal number of screen lines to keep above and below the cursor.
```lua
vim.opt.scrolloff = 10
```
</details>

<details>
<summary>Basic Keymaps</summary>

See `:help vim.keymap.set()`

### Clear highlights on search when pressing <Esc> in normal mode
See `:help hlsearch`
```lua
vim.keymap.set('n', '<Esc>', '<cmd>nohlsearch<CR>')
```

### Diagnostic keymaps
```lua
vim.keymap.set('n', '<leader>q', vim.diagnostic.setloclist, { desc = 'Open diagnostic [Q]uickfix list' })
```

### Exit terminal mode in the builtin terminal with a shortcut
For people to discover more easily. Otherwise, you normally need to press <C-\><C-n>, which
is not what someone will guess without a bit more experience.

> NOTE: This won't work in all terminal emulators/tmux/etc. Try your own mapping
or just use <C-\><C-n> to exit terminal mode
 
```lua
vim.keymap.set('t', '<Esc><Esc>', '<C-\\><C-n>', { desc = 'Exit terminal mode' })
```

### TIP: Disable arrow keys in normal mode
```lua
-- vim.keymap.set('n', '<left>', '<cmd>echo "Use h to move!!"<CR>')
-- vim.keymap.set('n', '<right>', '<cmd>echo "Use l to move!!"<CR>')
-- vim.keymap.set('n', '<up>', '<cmd>echo "Use k to move!!"<CR>')
-- vim.keymap.set('n', '<down>', '<cmd>echo "Use j to move!!"<CR>')
```

### Keybinds to make split navigation easier
Use CTRL+<hjkl> to switch between windows

See `:help wincmd` for a list of all window commands
```lua
vim.keymap.set('n', '<C-h>', '<C-w><C-h>', { desc = 'Move focus to the left window' })
vim.keymap.set('n', '<C-l>', '<C-w><C-l>', { desc = 'Move focus to the right window' })
vim.keymap.set('n', '<C-j>', '<C-w><C-j>', { desc = 'Move focus to the lower window' })
vim.keymap.set('n', '<C-k>', '<C-w><C-k>', { desc = 'Move focus to the upper window' })
```

</details>