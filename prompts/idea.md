I want to write a error parser for a programming language, it could be either
- Go
- Rust
- C++
- Python
- Typescript
... or some other. What I want to get is a list of errors/warnings that I could put into neovim
entries:
Quickfix entry docs:
		    bufnr	buffer number; must be the number of a valid
				buffer
		    filename	name of a file; only used when "bufnr" is not
				present or it is invalid.
		    module	name of a module; if given it will be used in
				quickfix error window instead of the filename.
		    lnum	line number in the file
		    end_lnum	end of lines, if the item spans multiple lines
		    pattern	search pattern used to locate the error
		    col		column number
		    vcol	when non-zero: "col" is visual column
				when zero: "col" is byte index
		    end_col	end column, if the item spans multiple columns
		    nr		error number
		    text	description of the error
		    type	single-character error type, 'E', 'W', etc.
		    valid	recognized error message
		    user_data
				custom data associated with the item, can be
				any type.

I want to write a Go program that does this job. I don't want to write parsing logic, I want to define some grammars
The program should get output lines and which language they belong to. It outputs a json list of quickfix entries.
Give me a high level overview of this Go program. Which libraries to use? How to make parsing fast? How to write grammar without regexps? How to make it maintainable?
