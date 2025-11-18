#!/bin/bash

# Install Funterm Vim plugin from vim-extension directory
echo "Installing Funterm Vim plugin..."

# Create directories if they don't exist
mkdir -p ~/.vim/syntax
mkdir -p ~/.vim/ftdetect

# Copy files from vim-extension
cp syntax/funterm.vim ~/.vim/syntax/
cp ftdetect/funterm.vim ~/.vim/ftdetect/

# Update or create .vimrc if needed
if ! grep -q "filetype plugin" ~/.vimrc 2>/dev/null; then
    echo "" >> ~/.vimrc
    echo '" Enable filetype detection' >> ~/.vimrc
    echo 'filetype plugin indent on' >> ~/.vimrc
    echo 'syntax on' >> ~/.vimrc
fi

echo "Installation complete!"
echo ""
echo "Files installed:"
echo "  ~/.vim/syntax/funterm.vim"
echo "  ~/.vim/ftdetect/funterm.vim"
echo ""
echo "Testing with a sample file..."

# Create a test file
cat > /tmp/test_funterm.su << 'EOF'
// Test Funterm file
py.greeting = "Hello World"
lua.number = 42

if (lua.number > 40) {
    js.console.log("Number is big")
}

for i = 0, 10 {
    py.print(i)
}
EOF

echo "Created test file: /tmp/test_funterm.su"
echo ""
echo "To test the syntax highlighting:"
echo "1. Open the file: vim /tmp/test_funterm.su"
echo "2. If syntax is not highlighted, type: :set filetype=funterm"
echo "3. Keywords, strings, and language prefixes should be colored"
echo ""
echo "Note: Syntax highlighting works best in interactive Vim, not with -c flags"