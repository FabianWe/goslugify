# goslugify

A slug is a "short label for something". In [the django glossary](https://docs.djangoproject.com/en/3.0/glossary/) it is defined as

> A short label for something, containing only letters, numbers, underscores or hyphens. They’re generally used in URLs. For example, in a typical blog entry URL:
> https://www.djangoproject.com/weblog/2008/apr/12/spring/
> the last bit (spring) is the slug.

There are some libraries out there for Golang, for example [gosimple/slug](https://github.com/gosimple/slug).
But these libraries were not as flexible as I would like, and so I created this (hopefully easy to use but flexible) library.
