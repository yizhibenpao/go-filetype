package main

//func test() {
//
//	for idx, args := range os.Args {
//		if idx == 0{
//			continue
//		}
//		targetReader, err := os.Open(args)
//		if err != nil {
//			panic(err)
//		}
//		defer targetReader.Close()
//
//		stat, _ := targetReader.Stat()
//
//		sr := wizutil.NewSliceReader(targetReader, 0, stat.Size())
//
//		result := Identify(sr, 0)
//		if err != nil {
//			panic(err)
//		}
//
//		fmt.Printf("%s: %s\n", args, wizutil.MergeStrings(result))
//	}
//
//}
