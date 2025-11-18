# Funterm Syntax Highlighting

VS Code extension for Funterm (.su) file syntax highlighting.

## Installation
1. Install VSIX file: `syntax-support-vscode/funterm-syntax-1.0.0.vsix`
2. In VS Code: Extensions → ⋮ → Install from VSIX...
3. Select the file and restart VS Code

## Features
- Full syntax highlighting for Funterm constructs
- Language block support (`python`, `py`, `lua`, `js`, `node`, `go`)
- Control flow highlighting (`if`, `else`, `for`, `while`, `match`, `break`, `continue`)
- Bitstring highlighting (`<<pattern>>`)
- Operator highlighting (arithmetic, logical, bitwise, concatenation `++`)
- Qualified variables highlighting (`py.var`, `lua.var`)
- Comment highlighting (`//`, `#`, `--`, `/* */`)
- Bracket and quote auto-closing
- Code folding support
- Import statements (`import language "path"`)