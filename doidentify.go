package main

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/yizhibenpao/go-filetype/wizardry/wizinterpreter"
	"github.com/yizhibenpao/go-filetype/wizardry/wizparser"
	"github.com/yizhibenpao/go-filetype/wizardry/wizutil"
	"os"
)

func doIdentify() error {
	magdir := *identifyArgs.magdir

	NoLogf := func(format string, args ...interface{}) {}

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

	target := *identifyArgs.target
	targetReader, err := os.Open(target)
	if err != nil {
		panic(err)
	}

	defer targetReader.Close()

	stat, _ := targetReader.Stat()

	ictx := &wizinterpreter.InterpretContext{
		Logf: NoLogf,
		Book: book,
	}

	if *appArgs.debugInterpreter {
		ictx.Logf = Logf
	}

	sr := wizutil.NewSliceReader(targetReader, 0, stat.Size())

	result, err := ictx.Identify(sr)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s: %s\n", target, wizutil.MergeStrings(result))

	return nil
}
