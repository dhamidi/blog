# Technical

## Required

- Define proper types.

  Currently errors returned by different parts of the system are
  constant strings.  This makes it hard to distinguish between error
  causes and retrieve accurate information about the error.  Having
  detailed error information is required at least in the HTTP layer.

  Errors should be modelled after the [os][1] package.

- Define interface for event stores.

  This includes the required operations as well as expected errors and
  semantics.  For example, fetching the stream of all events should
  never return a "Not found" error, but just an empty list of events.

- Add tests.

  Since this time it seems like the code is going to stick around for
  some time, and not just a few-hour throwaway prototype, writing tests
  is not a wasted effort.

# Features

## Required

- Provide a frontend for rewording posts.

## Nice-to-have

- Generate/maintain a sitemap.xml

  This feature is mainly here to explore how easy it is to have maintain
  multiple different views.  I suspect it is pretty easy.

[1]: https://golang.org/pkg/os#PathError
