# LitLua v0.0.2

LitLua is a literate programming tool inspired by Emacs designed to support better formating of lua based configuration. It enables you to write and maintain well-documented Lua configurations by transforming literate Markdown docs into executable Lua code.

 
🚨 This is a very early work in progress, so there may be edge cases and bugs not found yet - until V1 its recommend to use a VCS to ensure your config is not overwritten :)  🚨


## Features

- 📝 **Markdown-Based**: Write your configurations in familiar Markdown format
- 🔍 **Code Block Extraction**: Automatically extracts and processes Lua code blocks
- 💾 **Redundancy Precautions**: Automatically creates backups of configuration files before overwriting
- 🛠 **Simple CLI Interface**: Easy to use optional command-line tooling
- 📦 **LSP Support!**: Powered by LuaLS, LitLua provides built-in LSP support for your Markdown->Lua configurations

##
Here is a small demo of Litlua handling nvim configuration. You can find the Nvim configuration repo using Litlua here:  https://github.com/jwtly10/config.nvim


https://github.com/user-attachments/assets/f928c89b-c42f-45da-8e7c-8c428b162882

## Why?

After spending some time with Emacs, I could see the value of having litterate configurations for more than just the emacs ecosystem - configurations being just a readable, documentation first markdown file, that will be 'transpiled' into the expected format!


> Check out [Examples](https://github.com/jwtly10/litlua/blob/main/examples/kickstart.nvim/kickstart_configuration.litlua.md) for a small showcase of what LitLua is about. 
> 
> I've re-written some of [kickstart.nvim](https://github.com/nvim-lua/kickstart.nvim) in markdown and used LitLua to generate the lua files, you can see the same for an example wezterm config - .wezterm.lua is the transpiled file, *.litlua.md is the markdown source.
> 
> If you have the LitLua LSP installed, you will also get LSP support for the markdown files, so you can hover over functions and go to definitions etc.


## Roadmap
- [X] Basic Markdown to Lua conversion to SFO
- [X] Built in LitLua LSP support 
- [X] Live file compilations (via LSP)
- [ ] Support compiling `litlua.md` files on Neovim startup
- [ ] Single file to multiple file output (master file, into multiple configuration .lua files)
- [ ] 'Tagging' of code blocks for easy reference and linking
- [ ] Hot swapping configuration management - switch between versions of configurations

Generally the idea is this should seamlessly integrate into existing lua based configuration setups, such as Neovim, and work alongside existing lua files.
With as much work done via the LSP as possible, to create a seamless experience and be editor agnostic.


## Installation

These installation instructions assume you have go installed - https://go.dev/doc/install

There are 2 installations available, the LSP and the CLI.

The LSP is editor agnostic and will try to take care of all compilation while the LSP is running. On save/exit, 
the lua file will be generated, given output pragma options are set (see below [FileFormat](#file-format)).

The CLI contains a subset of the LSP features, and is more suited directory or one off conversions, or for CI/Scripting support

It is recommend you read here before installing and getting started: [LSP.md](./LSP.md)

You can install both with:

```sh
go install github.com/jwtly10/litlua/cmd/...@latest
```

### LSP Installation

```sh
go install github.com/jwtly10/litlua/cmd/litlua-ls@latest
```

### CLI Installation

```sh
go install github.com/jwtly10/litlua/cmd/litlua@latest
```

### Build from source
```sh
# Clone the repo
git clone https://github.com/jwtly10/litlua.git
cd litlua

# Installing the LSP from source
go build -o litlua-ls cmd/litlua-ls/main.go
# Move the LSP binary to your PATH  
mv litlua-ls /usr/local/bin  

# Installed the CLI
go build -o litlua cmd/litlua/main.go
# Move the CLI binary to your PATH  
mv litlua /usr/local/bin  
```


More installation options will be available in the future.


### File Format

LitLua processes custom Markdown files (`.litlua.md`) containing Lua code blocks. Here's an example:

````markdown
<!-- @pragma output: init.lua -->

# Neovim Telescope Configuration

Our telescope setup with detailed explanations.

```lua
require('telescope').setup({
    defaults = {
        mappings = {
            i = {
                ['<C-u>'] = false,
                ['<C-d>'] = false,
            },
        },
    },
})
```

This configures the basic telescope behavior. Let's add some keymaps:

```lua
-- File browsing
vim.keymap.set('n', '<leader>ff', 
    require('telescope.builtin').find_files)

-- Live grep
vim.keymap.set('n', '<leader>fg', 
    require('telescope.builtin').live_grep)
```
````

LitLua will:
1. Parse the Markdown document
2. Extract pragma directives at the top of the file, such as `output:init.lua`
3. Extract Lua code blocks
4. Generate a clean Lua file with all the code to the output file `init.litlua.lua`
5. Create a backup of any existing output file

> Note: The output file will be `init.litlua.lua` for safety reason - please see [Configuration](#configuration) for details


## Usage

### LSP
See [LSP.md](./LSP.md) for more information on the LSP support and usage.

### CLI

#### Basic usage:

Converts `.litlua.md` markdown document to lua file (based on the `output` pragma) else defaults to original file name with `.litlua.lua` extension:

```bash
litlua <your_configuration_file_with_lua_src.litlua.md>
```

Enable debug logging during conversion:

```bash
litlua -debug <your_configuration_file_with_lua_src.litlua.md>
```

Converts a directory, looking for`.litlua.md` markdown documents to transform to lua files as above.

```bash
litlua  ./path/to/config/files
```

*Example Output:*

```sh
~/Projects/litlua ❯ litlua ./examples/                                                                                 22.11.0  1.23.3

🚀 Compilation is running:
  📄 Path     : ./examples/

Compilation Results:
Source                                                                 Output
--------------------------------------------------------------------------------------------------------------
lsp_example.litlua.md                                                  compiled.litlua.lua
wezterm/wezterm_configuration.litlua.md                                wezterm/.wezterm.lua
kickstart.nvim/kickstart_configuration.litlua.md                       kickstart.nvim/output.litlua.lua
--------------------------------------------------------------------------------------------------------------

✨ Compilation complete! Processed 3 files

```

#### Output

LitLua generates a single Lua file containing all the extracted code blocks, maintaining their original order as specified in the Markdown source.

#### Configuration


Litlua will always default to outputing files with a `.litlua.lua` extension. Even if you have specified `something.lua`. This is done as a safety precaution to prevent the case of accidently updating a file you did not back up. 

For most usecases, this is not a problem, as `.*.lua` will ensure the filetype still has all usual properties of a lua file. 

In order to FORCE the output path you can use:

``` markdown
<!-- @pragma output: init.lua -->
<!-- @pragma force: true -->
```

Which WILL output a file `init.lua`. Please use this at your own risk.


By default, LitLua will generate the output file in the same directory as the input file, with a `.litlua.lua` extension. You can customize the output path using pragmas **at the start** your document:

> NOTE: The file path will ALWAYS be relative to the input file

```markdown
<!-- @pragma output: init.lua -->

# My Neovim Configuration
...
```

## Development

### Setup locally

```bash
git clone https://github.com/jwtly10/litlua
go test ./... -v
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request or Issue.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
