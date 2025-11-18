# Funterm Editor Support

## Features

All editors provide:
- Full syntax highlighting for Funterm
- Language block support (`python`, `py`, `lua`, `js`, `node`, `go`)
- Control flow highlighting
- Bitstring highlighting (`<<pattern>>`)
- Operator highlighting (including `++` for concatenation)
- Number highlighting (decimal, hex, binary, scientific notation)
- Code folding
- Comment highlighting
- String highlighting

## VS Code

### Installation
1. Install VSIX file: `syntax-support-vscode/funterm-syntax-1.0.0.vsix`
2. In VS Code: Extensions → ⋮ → Install from VSIX...
3. Select the file and restart VS Code

### Additional Features
- Bracket and quote auto-closing
- Import statements highlighting
- Qualified variables highlighting (`py.var`, `lua.var`)

## Sublime Text

### Installation
```bash
# macOS
cp -r syntax-support-sublime-text ~/Library/Application\ Support/Sublime\ Text/Packages/Funterm

# Linux
cp -r syntax-support-sublime-text ~/.config/sublime-text/Packages/Funterm

# Windows
cp -r syntax-support-sublime-text "%APPDATA%\Sublime Text\Packages\Funterm"
```

### Additional Features
- Automatic `.su` file detection

## Vim/Neovim

### Installation

#### Automatic installation
```bash
chmod +x syntax-support-vim/install_vim_plugin.sh
./syntax-support-vim/install_vim_plugin.sh
```

#### Manual installation

**Vim:**
```bash
mkdir -p ~/.vim/syntax ~/.vim/ftdetect
cp syntax-support-vim/syntax/funterm.vim ~/.vim/syntax/
cp syntax-support-vim/ftdetect/funterm.vim ~/.vim/ftdetect/
```

**Neovim:**
```bash
mkdir -p ~/.config/nvim/syntax ~/.config/nvim/ftdetect
cp syntax-support-vim/syntax/funterm.vim ~/.config/nvim/syntax/
cp syntax-support-vim/ftdetect/funterm.vim ~/.config/nvim/ftdetect/
```

### Additional Features
- Automatic `.su` file detection
- Short prefix support (`py.`, `lua.`)
- Color scheme compatibility

### Verification
1. Open a `.su` file
2. Run `:set filetype?` - should show `filetype=funterm`
3. If highlighting doesn't work: `:set filetype=funterm`
