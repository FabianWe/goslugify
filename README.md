# goslugify

A slug is a "short label for something". In [the django glossary](https://docs.djangoproject.com/en/3.0/glossary/) it is defined as:

> A short label for something, containing only letters, numbers, underscores or hyphens. They’re generally used in URLs. For example, in a typical blog entry URL:
> https://www.djangoproject.com/weblog/2008/apr/12/spring/
> the last bit (spring) is the slug.

There are some libraries out there for Golang, for example [gosimple/slug](https://github.com/gosimple/slug).
These libraries were not as flexible as I would like, and so I created this (hopefully easy to use but flexible) library.

## Documentation
You can find the documentation on [godoc.org](https://godoc.org/github.com/FabianWe/goslugify) including some examples.

## Simple Usage
The documentation contains some simple example, but probably the easiest way to do something is to use the `GenerateSlug` function:

```go
import (
    "fmt"
    "github.com/FabianWe/goslugify"
)

func main() {
    fmt.Println(goslugify.GenerateSlug("Gophers! The most interesting species of rodents"))
}
```

This will generate `gophers-the-most-interesting-species-of-rodents`.
That's it.

## What Will it Do By Default?
By default, the following conversion rules are applied to convert a string to a slug:
1. Remove invalid UTF-8 codepoints
2. Convert to UTF-8 normal form NKFC, see [this blog post](https://blog.golang.org/normalization) and [this go package](https://godoc.org/golang.org/x/text/unicode/norm)
3. Convert to lower case
4. Replace whitespaces by `"-"`
5. Replace all dash symbols and hyphens by `"-"` (there is not just `"-"` in UTF-8)
6. Translate umlauts, for example `"ä"` to `"ae"`, `"ß"` to `"ss""`
7. Drop everything that is not in `a-zA-Z0-9-_`
8. Remove occurrences of two or more `"-"` by a single `"-"`
9. Remove all leading and trailing `"-"`

Important note: Don't assume that this is exactly what happens all the time over different versions.
Even in a new release of the same major release this behavior is likely to change if new functionality gets added.
So don't assume that `GenerateSlug` always returns the same string! Once for example emoji support is added the result
might look different.
If you want to generate a slug to identify an object (in a database for example) always store this slug with the object,
don't assume that a call to `GenerateSlug(name)` will return the exact same slug again (for the given object name).

If you don't write anything very specific for your own projects it's probably a good idea to share your function.
I would be more than happy to include useful (and general) extensions in this project.

## I Want to Customize the Generated Slugs
There are different ways to customize the behavior, one that is rather simple and should
be sufficient in 90% of all use cases.

### The Easy Way
Create an instance of [SlugConfig](https://godoc.org/github.com/FabianWe/goslugify#SlugConfig) and call
`Configure`.
This allows you to set the maximal length of the generated slugs, a different word separator,
control lower case behavior and provide additional replacement maps.
There are some examples in the documentation here, is another one:

```go
import (
    "fmt"
    "github.com/FabianWe/goslugify"
)

func main() {
    config := goslugify.NewSlugConfig()
    config.TruncateLength = 30
    config.WordSeparator = '_'
    config.ToLower = false
    config.AddReplaceMap(goslugify.GetLanguageMap("en"))
    generator := config.Configure()
    fmt.Println(generator.GenerateSlug("Gophers & Other Rodents: A survey"))
}
```

This will produce `"Gophers_and_Other_Rodents_A"`.

You can add more substitutions that should happen on the input string by calling [AddReplaceMap](https://godoc.org/github.com/FabianWe/goslugify#SlugConfig.AddReplaceMap).
The language `"de"` for German is available too.

Again: The default behavior might change even through different versions of the same major release.

### Extending With Custom Functions
Another way that is not too hard is to add your own functions that do some kind of string modification.
Either implement [StringModifierFunc](https://godoc.org/github.com/FabianWe/goslugify#StringModifierFunc)
or [StringModifier](https://godoc.org/github.com/FabianWe/goslugify#StringModifier), please read the documentation first.

Also make sure you understand the different phases of the generation as documented in [SlugGenerator](https://godoc.org/github.com/FabianWe/goslugify#SlugGenerator).
Then you can use an existing `SlugConfig` and call [GetPhases](https://godoc.org/github.com/FabianWe/goslugify#SlugConfig.GetPhases):
This will give you the modifiers used for all three phases. You can simply append (to the back), create a copy and
insert at the beginning, whatever you like.
Then simply create a `SlugGenerator` instance with it.

### Create a SlugGenerator by Hand
Probably the hardest way, you don't have the defaults that come with this library.
Make sure that you get the order of the functions right, because it is important that they're executed in the correct order (in most cases).
You probably want to look in the code anyway, so you might have a look at [GetDefaultPreProcessors](https://godoc.org/github.com/FabianWe/goslugify#GetDefaultPreProcessors),
[GetDefaultProcessors](https://godoc.org/github.com/FabianWe/goslugify#GetDefaultProcessors) and
[GetDefaultFinalizers](https://godoc.org/github.com/FabianWe/goslugify#GetDefaultFinalizers).
Again: Don't rely on the return value, it is probable that new modifiers will be added! So for example don't assume that the length will always be the same, or the element
on a specific index is a certain modifier.

## Contribute
As mentioned before new functionality might be added in the same major release and thus the generated slugs in default mode might change.
So if you for example plan to add emoji support or add a new language you can share it and I will happily add it to the project.
Just contact me, via [E-Mail](mailto:fabianwen@posteo.eu) or create a Pull Request.

### TODOs
* Add support for more languages
* Add support for emojis

## License
Apache License, Version 2.0. See [LICENSE](LICENSE) file.
