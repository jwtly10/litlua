# LSP Support in LitLua

See here for the full LSP specification: 
https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/

LitLua provides built in LSP support but does not want to compete with existing Lua tooling, so instead it proxy to the 
[Lua Language Server](https://github.com/LuaLS/lua-language-server) for LSP support.
There are some tricks in the background inspired by https://github.com/jmbuhr/otter.nvim to make lua-language-server
think it's working with a real lua file, and we map diagnostics to the .md file.

## Pre-requisites
- [Lua-Language-Server](https://github.com/LuaLS/lua-language-server) - installation instructions are in the README


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

LSP Features Planned in next version (v0.0.3):
- textDocument/rename
- textDocument/documentSymbol
- textDocument/references
- textDocument/implementation
- textDocument/signatureHelp
- textDocument/declaration
- textDocument/typeDefinition

All other methods are not guaranteed to have full support (they are proxied to the lua-language-server, but no LitLua 
specific features are added). 

If you would like a specific LSP feature implemented please raise an Issue!

## Usage

Ideally you will never have to interact with the language server directly. All that is required is to install the LSP, and 
implement it within your editor of choice.

However in the [Configuration](#Configuration) section you can see all the opts the language server accepts.



### Neovim

(TODO) - This would be nice as a plugin, but for now you can add the following to your Neovim configuration (requires nvim-lspconfig):
```lua
vim.api.nvim_create_autocmd('FileType', {
    pattern = 'markdown',
    callback = function()
        vim.lsp.start({
            name = 'litlua',
            cmd = { 'litlua-ls'},
            root_dir = vim.fs.dirname(vim.fs.find({ '.git' }, { upward = true })[1]),
        })
    end,
})
```

### Vscode
(TODO) - I don't use VSCode, and only created an extension for debugging purposes. If you would like to contribute feel free to implement the VSCode extension.

## Configuration

The LSP accepts the current flags. If a Neovim plugin is created, it should support the following params:

(TODO)




