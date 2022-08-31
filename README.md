# go-filetype


go-filetype is a toolkit to deal with libmagic rule files (sources, not compiled)

It contains:

  * A parser, which turn magic rule files into an AST
  * An interpreter, which identifies a target by following
  the rules in the AST
  * A compiler, which generates go code to follow the
  rules in the AST




