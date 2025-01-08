# LSP Support in LitLua

See here for the full LSP specification: 
https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/

LitLua provides built in LSP support but does not want to compete with existing Lua tooling, so instead it proxies to the 
[Lua Language Server](https://github.com/LuaLS/lua-language-server) for LSP support.

See [Implementation](#Implementation) for an overview of how this works.

## Pre-requisites
- [Lua-Language-Server](https://github.com/LuaLS/lua-language-server) - installation instructions are in the README
> Note: If you have lua-ls installed via Mason in neovim, this should work out of the box, and does not require a secondary installation. If litlua-ls is unable to find it during startup, you can manually supply the location of the Mason installation through the flags

## Installation
You can install the LitLua language server here:
```sh
go install github.com/jwtly10/litlua/cmd/litlua-ls@latest
```

## Supported LSP Features

Current LSP methods support:
- textDocument/didOpen
- textDocument/didChange
- textDocument/hover
- textDocument/signatureHelp
- textDocument/completion
- textDocument/definition
- textDocument/didSave - (this will trigger a 'final' compilatoin to the pragma output file)

LSP methods planned for support in future:
- textDocument/rename
- textDocument/documentSymbol
- textDocument/references
- textDocument/implementation
- textDocument/signatureHelp
- textDocument/declaration
- textDocument/typeDefinition

If you would like a specific LSP feature implemented please raise an issue!

## Configuration

The LSP accepts a number of flags, all of which can be explained with `litlua-ls -h`:
```
Usage:
  litlua-ls [flags]

Flags:
  -debug
        Enable debug logging
  -luals string
        Custom path to lua-language-server
  -shadow-root string
        Custom path to shadow root directory (for LSP intermediate files)
  -version
        Print version information

Examples:
  # Start the server with default settings
  $ litlua-ls

  # Start with custom lua-language-server path
  $ litlua-ls -luals=/usr/local/bin/lua-language-server

  # Enable debug logging
  $ litlua-ls -debug
```

## Implementation

LitLua's Language Server Protocol (LSP) implementation takes a pragmatic approach by acting as a proxy between your editor and the official `lua-ls` language server. This design leverages existing, battle-tested Lua tooling while adding seamless support for Lua code embedded in Markdown.

### Shadow Workspace Architecture

The core of the implementation uses a "shadow workspace" approach:

```
   Editor                 litlua-ls                    lua-ls
     |                        |                           |
     |   [MD with Lua code]   |                           |
     | -------------------->  |                           |
     |                        |                           |
     |                        |  [Transform]              |
     |                        |  MD -> Lua                |
     |                        |  (preserve positions)     |
     |                        |                           |
     |                        |     [Shadow Files]        |
     |                        | -------------------->     |
     |                        |                           |
     |        [LSP responses] |    [LSP responses]        |
     |  <-------------------  |   <--------------------   |
     |  (mapped positions)    |                           |
```

1. When your editor sends LSP notifications (e.g., `textDocument/didChange`), LitLua-LSP intercepts these events
2. The Markdown document is transformed into pure Lua files, preserving exact line positions
3. These transformed files are stored in a shadow workspace (by default in the OS temp directory, configurable via flags)
4. LitLua-LSP forwards LSP requests to `lua-ls`, which operates on the shadow workspace
5. When `lua-ls` returns diagnostics or other LSP responses, LitLua-LSP maps these back to the original Markdown positions using the preserved line mappings

This architecture ensures that users get full Lua language features (completion, diagnostics, hover) while editing Markdown files, with accurate position mapping between the two formats.

### Final Compilation

Litlua provides a CLI for manually compiling the final lua file from markdown, but when using the LSP, on save we will trigger this final compilation if the file contains a valid output pragma. This reduces the friction of using Litlua, you edit your config, and the final lua file will be automatically generated on save.

## Usage

### Neovim
I have not implemented a neovim plugin yet, this is planned for a future version (or someone in the community :)), but for now you can add the following to your Neovim configuration (requires nvim-lspconfig):
```lua
vim.api.nvim_create_autocmd('BufRead', {
    pattern = '*.litlua.md',
    callback = function()
        vim.lsp.start({
            name = 'litlua',
            cmd = { 'litlua-ls'}, -- here you can add opts such as { 'litlua-ls', '-debug', '-luals=/usr/local/bin/lua-language-server'}
            root_dir = vim.fs.dirname(vim.fs.find({ '.git' }, { upward = true })[1]),
        })
    end,
})
```

### Vscode
I don't use VSCode, and only created an extension for debugging purposes, so it should have relatively good support out of the box. If you would like to contribute feel free to implement the VSCode extension.
