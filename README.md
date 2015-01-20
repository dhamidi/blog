# What

This will become my personal blog at some point.

**USE AT YOUR OWN RISK** or better: **DO NOT USE YET**.

# Why

To try out [Event Sourcing][1] and [CQRS][2], because it seems like an
interesting approach to developing applications.  The perceived[^1]
advantages over a "classical", relational database centric approach are:

- Better query performance: it is easy to maintain use-case optimized
  views through projections of the event stream.

- Time-travel: by replaying only a subset of events, previous
  application state can be reconstructed.

- Decoupled components: by using command and event objects for passing
  messages through the system, components are decoupled.

- Flexibility: additional projections can easily (easy as in "more of
  the same") be added (such as a third normal form relational database).
  Event stores are simple[^2] and can be easily replaced with something
  fitting the current requirements.

- Less accidental complexity: domain events can easily (de-)serialized,
  meaning that no [ORM][3] is *necessary* (that doesn't mean it's
  useless).  At the time of writing, the dependencies, apart from the Go
  standard library, are: an uuid generator, a markdown processor and a
  HTML sanitizer.  Of these only the uuid generation is essential.

Of course, everything has its drawbacks.  Finding out about is the goal
of this project.

[1]: http://martinfowler.com/eaaDev/EventSourcing.html
[2]: http://martinfowler.com/bliki/CQRS.html
[3]: http://blog.codinghorror.com/object-relational-mapping-is-the-vietnam-of-computer-science/

# Footnotes

[^1]: "perceived" because I have to experience yet to confirm these advantages.
[^2]: For an example implementation atop two database tables, see: https://cqrs.wordpress.com/documents/building-event-storage/
