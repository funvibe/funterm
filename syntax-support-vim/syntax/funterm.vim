" Vim syntax file for Funterm (.su files)
" Language: Funterm
" Maintainer: Funterm Team
" Latest Revision: 2024

if exists("b:current_syntax")
  finish
endif

" Keywords
syn keyword funtermKeyword if else while for in break continue return match import
syn keyword funtermBoolean true false nil
syn keyword funtermLanguage python lua js javascript node go py

" Comments
syn match funtermComment "//.*$"
syn region funtermBlockComment start="/\*" end="\*/" contains=funtermBlockComment

" Strings
syn region funtermString start='"' skip='\\"' end='"'
syn region funtermString start="'" skip="\\'" end="'"
syn region funtermMultiString start='"""' end='"""'
syn region funtermMultiString start="'''" end="'''"

" Numbers
syn match funtermNumber "\<\d\+\>"
syn match funtermNumber "\<\d\+\.\d*\>"
syn match funtermNumber "\<\d\+[eE][+-]\?\d\+\>"
syn match funtermNumber "\<\d\+\.\d*[eE][+-]\?\d\+\>"
syn match funtermHexNumber "\<0[xX][0-9a-fA-F]\+\>"
syn match funtermBinNumber "\<0[bB][01]\+\>"

" Operators
syn match funtermOperator "[+\-*/%=<>!&|^~]"
syn match funtermOperator "++\|--\|==\|!=\|<=\|>=\|&&\|||\|->\||>\|<<\|>>\|\*\*"

" Language blocks
syn region funtermPythonBlock start="python\s*{" end="}" contains=@funtermPython fold
syn region funtermLuaBlock start="lua\s*{" end="}" contains=@funtermLua fold
syn region funtermJSBlock start="\(js\|javascript\|node\)\s*{" end="}" contains=@funtermJS fold
syn region funtermGoBlock start="go\s*{" end="}" contains=@funtermGo fold

" Bitstrings
syn region funtermBitstring start="<<" end=">>" contains=funtermBitstringType,funtermNumber,funtermVariable
syn keyword funtermBitstringType binary integer float utf8 utf16 utf32 big little signed unsigned contained

" Variables and functions
syn match funtermFunction "\<[a-zA-Z_][a-zA-Z0-9_]*\s*("me=e-1
syn match funtermVariable "\<[a-zA-Z_][a-zA-Z0-9_]*\>"

" Language-specific function calls
syn match funtermLangCall "\<\(python\|py\|lua\|js\|javascript\|node\|go\)\.\w\+"

" Special characters
syn match funtermSpecial "[{}()\[\],;:]"

" Pipe operator
syn match funtermPipe "|>"

" Define highlighting
hi def link funtermKeyword       Keyword
hi def link funtermBoolean       Boolean
hi def link funtermLanguage      Type
hi def link funtermComment       Comment
hi def link funtermBlockComment  Comment
hi def link funtermString        String
hi def link funtermMultiString   String
hi def link funtermNumber        Number
hi def link funtermHexNumber     Number
hi def link funtermBinNumber     Number
hi def link funtermOperator      Operator
hi def link funtermPythonBlock   Special
hi def link funtermLuaBlock      Special
hi def link funtermJSBlock       Special
hi def link funtermGoBlock       Special
hi def link funtermBitstring     Special
hi def link funtermBitstringType StorageClass
hi def link funtermFunction      Function
hi def link funtermVariable      Identifier
hi def link funtermLangCall      Function
hi def link funtermSpecial       Special
hi def link funtermPipe          Operator

let b:current_syntax = "funterm"