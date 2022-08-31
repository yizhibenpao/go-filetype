package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/yizhibenpao/go-filetype/wizardry/wizcompiler"
	"github.com/yizhibenpao/go-filetype/wizardry/wizparser"
)

func doCompile() error {
	magdir := *compileArgs.magdir

	NoLogf := func(format string, args ...interface{}) {
		fmt.Print(fmt.Sprintf(format, args...) + "\n")
	}

	Logf := func(format string, args ...interface{}) {
		fmt.Println(fmt.Sprintf(format, args...))
	}

	pctx := &wizparser.ParseContext{
		Logf: NoLogf,
	}

	if *appArgs.debugParser {
		pctx.Logf = Logf
	}

	book := make(wizparser.Spellbook)
	err := pctx.ParseAll(magdir, book)
	if err != nil {
		return errors.WithStack(err)
	}

	err = wizcompiler.Compile(book, *compileArgs.output, *compileArgs.chatty, *compileArgs.emitComments, *compileArgs.pkg)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
