# LitLua v0.0.1

LitLua is a literate programming tool inspired by Emacs designed to support better formating of lua based configuration. It enables you to write and maintain well-documented Lua configurations by transforming literate Markdown docs into executable Lua code.

🚨 This is a work in progress, and not truly ready to handle all your configuration needs yet!

## Features

- 📝 **Markdown-Based**: Write your configurations in familiar Markdown format
- 🔍 **Code Block Extraction**: Automatically extracts and processes Lua code blocks
- 💾 **Redundancy Precautions**: Automatically creates backups of configuration files before overwriting
- 🛠 **Simple CLI Interface**: Easy to use command-line tool


## Why?

After spending some time with Emacs, I could see the value of having litterate configurations for more than just the emacs eco system - configurations being just a readable 'pretty' markdown file, that will be 'transpiled' into the expected format!

For now its a simply extracter of lua source code but future versions will include syntax checking, and other features to make it a more complete tool.


> Check out [Examples](https://github.com/jwtly10/litlua/tree/64b8e4407167ddac72ccd8c92c97f5a331c24550/examples) for a small showcase of what LitLua is about. 
> 
> I've re-written some of [kickstart.nvim](https://github.com/nvim-lua/kickstart.nvim) in markdown and used LitLua to generate the lua files, you can see the same for an example wezterm config. *.lua is the transpiled file, *.md is the markdown source.

## Roadmap
- [X] Basic Markdown to Lua conversion to SFO
- [ ] Single file to multiple file output (master file, into multiple configuration .lua files)
- [ ] Hot swapping configuration management
- [ ] Built in lua LSP support (linter, formatter, etc)
- [ ] 'Tagging' of code blocks for easy reference
- [ ] Watch mode for live updating of configuration files

Generally the idea is this should seamlessly integrate into existing lua based configuration setups, such as Neovim, and work alongside existing lua files.

## Installation

### Install via Go
```bash
# Assuming you have go installed - https://go.dev/doc/install 
go install github.com/jwtly10/litlua@latest
```

### Build from source
```bash
git clone https://github.com/jwtly10/litlua.git
go build -o litlua cmd/litlua/main.go
# Move the binary to your PATH  
mv litlua /usr/local/bin  
```

More installation options will be available in the future.

## Usage


Basic usage:

Converts markdown document to lua file:

```bash
litlua -in configuration.md
```

Enable debug logging during conversion:

```bash
litlua -in init.luadoc -debug
```

### File Format

LitLua processes Markdown files (`.md`) containing Lua code blocks. Here's an example:

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

The tool will:
1. Parse the Markdown document
2. Extract pragma directives at the top of the file, such as `output`
2. Extract Lua code blocks
3. Generate a clean Lua file with all the code to the output file `init.lua`
4. Create a backup of any existing output file

### Output

LitLua generates a single Lua file containing all the extracted code blocks, maintaining their original order as specified in the Markdown source.

## Configuration

### Output Path Resolution

By default, LitLua will generate the output file in the same directory as the input file, with a `.lua` extension. You can customize the output path using pragmas **at the start** your document:

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
go run ./cmd/litlua/main.go -in testdata/basic.md
```

### Running Tests

```bash
go test ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.