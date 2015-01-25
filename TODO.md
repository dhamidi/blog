# Technical

## Required

- Add tests.

  Since this time it seems like the code is going to stick around for
  some time, and not just a few-hour throwaway prototype, writing tests
  is not a wasted effort.

## Nice-to-have

- An additional event store implementation

  Obvious candidates for backends are Sqlite, Redis or Postgres.
  Bolt[1] looks interesting too, especially from an operational point of
  view (i.e. no additional service necessary).

# Features

## Required

- Do not allow images in user comments

  From a legal perspective it is too risky to have images appear
  directly in the comments.

## Nice-to-have

- Generate/maintain a sitemap.xml

  This feature is mainly here to explore how easy it is to have maintain
  multiple different views.  I suspect it is pretty easy.

[1]: https://github.com/boltdb/bolt
