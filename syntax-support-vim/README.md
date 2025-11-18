# Funterm Syntax for Vim/Neovim

Syntax highlighting and support for Funterm (.su files) in Vim and Neovim.

## Installation

#### Automatic installation (recommended)
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

## Features
- Full syntax highlighting for Funterm
- Automatic `.su` file detection
- All keywords and operators support
- Short prefix support (`py.`, `lua.`, `js.`)
- Language blocks highlighting
- Bitstring highlighting
- Comment highlighting
- String highlighting (regular and multiline)
- Number highlighting (decimal, hex, binary, scientific notation)
- Code folding
- Compatibility with popular color schemes

## Usage

After installation, Vim/Neovim will automatically detect `.su` files.

### Verification
1. Open a `.su` file
2. Run `:set filetype?`
3. Should show `filetype=funterm`