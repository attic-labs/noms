# Frequently Asked Questions

### How is this a database? There's not even a query language!

First, we do plan to have some sort query language one day. But such a thing is sort of like the last step of a long journey that we are just at the beginning of. Right now we are working on the underlying primitives that would allow a query system to be built and work well.

Second, the term *database* isn't well-defined. There are a lot of different systems that call themselves databases.

We like to think the word has a certain heft, and certain expectations that go along with it: A database should deal with lots of small, structured records well. A database should never lose or corrupt data. A database should support efficient queries on different attributes.

Noms can and should be a database, not merely a datastore. We're not quite there yet, but we're off to a good start, and we do think we have the [necessary tools](https://github.com/attic-labs/noms/blob/master/doc/intro.md#prolly-trees-probabilistic-b-trees).

### Decentralized like BitTorrent?

No, decentralized like Git.

Specifically, Noms isn't itself a peer-to-peer network. If you can get two instances to share data, somehow, then they can synchronize. Noms doesn't define how this should happen though. 

Currently, instances mainly share data via either HTTP/DNS or a filesystem. But it should be easy to add other mechanisms. For example, it seems like Noms could run well on top of BitTorrent, or IPFS. You should [look into it](https://github.com/attic-labs/noms/issues/2123).

### Isn't it wasteful to store every version?

Noms deduplicates chunks of data that are identical within one database. So if multiple versions of one dataset share a lot of data, or if the same data is present in multiple datasets, Noms only stores one copy.

That said, it is definitely possible to have write patterns that defeat this. Deduplication is done at the chunk level, and chunks are currently set to an average size of 4KB. So if you change about 1 byte in every 4096 in a single commit, and those changed bytes are well-distributed throughout the dataset, then we will end up making a complete copy of the dataset.

### Is there a way to not store the entire history?

Theoretically, definitely. In Git, for example, the concept of "shallow clones" exists, and we could do something similar in Noms. This has not been implemented yet.

### Is this another one of those NoSQL database thingies?

It depends on the specific features and/or functionality you use to define `NoSQL`. Does Noms support similar type of data? Yes. Does Noms support a flexible schema? Yes.  Does Noms "compromise consistency"? No. Is Noms “eventually consistent”? No, it is always consistent.

### Git can store files, how is this different?

Git is awesome at storing files, allowing different versions of files, tracking changes between commits, and having a strong sense of data integrity. If you commit 2 versions of a 2 GB text file with a large number of changes, then Git will give you information about line differences or sections of changes in those files.  If you imported those files into Noms as structured data, then the differences calculated by Noms would give you semantic information about the data that changed - and much more quickly than Git. Noms is great at working with data and this is only one example of how Noms is different than Git.

### Why is it called Noms?

1. It's insert-only. OMNOMNOM.
2. It's content addressed. Every value has its own hash, or [name](http://dictionary.reverso.net/french-english/nom).

### Are you sure Noms doesn't stand for something?

Pretty sure. But if you like, you can pretend it stands for Non-Mutable Store.

### You don't know what you're doing!

Not a question. But, fair enough.
