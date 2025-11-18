module funterm

go 1.25.0

require (
	github.com/chzyer/readline v1.5.1
	github.com/funvibe/funbit v1.0.0
	github.com/stretchr/testify v1.8.4
	github.com/yuin/gopher-lua v1.1.1
	go-parser v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

// Локальный go-parser модуль
replace go-parser => ./go-parser

replace github.com/funvibe/funbit => ./funbit
