# multiplex spec

go-multiplex is an implementation of the multiplexing protocol used by
[multiplex](https://github.com/maxogden/multiplex). This document will attempt
to define a specification for the wire protocol and algorithm used in both
implementations. 

Multiplex is a very simple protocol that does not provide many features offered
by other stream multiplexers. Notably, multiplex does not provide backpressure
at the protocol level, or support half closed streams.

## Message format
Every communication in multiplex consists of a header, and a length prefixed data segment.
The header is an unsigned base128 varint, as defined in the [protocol buffers spec](https://developers.google.com/protocol-buffers/docs/encoding#varints). The lower three bits are the message flags, and the rest of the bits (shifted down by three bits) are the stream ID this message pertains to:

```
header = readUvarint()
flag = head & 0x07
id = flag >> 3
```

### Flag Values

| NewStream        | 0 |
| MessageReceiver  | 1 |
| MessageInitiator | 2 |
| CloseReceiver    | 3 |
| CloseInitiator   | 4 |
| ResetReceiver    | 5 |
| ResetInitiator   | 6 |

The data segment is length prefixed by another unsigned varint. This results in one message looking like:

| header  | length  | data           |
| uvarint | uvarint | 'length' bytes |


## Protocol

Multiplex operates over a reliable ordered pipe between two peers, such as a TCP
socket, or a unix pipe. One peer is designated the session 'initiator' (or the
dialer) and the other is the session 'receiver'. The session initiator does not
necessarily send the first packet, this distinction is just made to make the
allocation of stream ID's unambiguous.

### Opening a new stream

To open a new stream, first allocate a new unique stream ID; the session
initiator allocates odd IDs and the session receiver allocates even IDs. Then,
send a message with the flag set to `NewStream`, the ID set to the newly
allocated stream ID, and the data of the message set to the name of the stream.
Stream names are purely for interfaces and are not otherwise considered by the
protocol. An empty string may also be used for the stream name, and they may
also be repeated (using the same stream name for every stream is valid). Reusing
a stream ID after closing a stream may result in undefined behaviour.

The party that opens a stream is called the stream initiator. It's unclear why
this distinction was necessary but removing it isn't worth making a backwards
incompatible change to the protocol.

### Writing to a stream

To write data to a stream, one must send a message with the flag
`MessageReceiver` (1) or `MessageInitiator` (2) (depending on whether or not the
writer is the receiver or sender). The data field should contain the data you
wish to write to the stream, limited to a maximum size agreed upon out of band
(For reference, the go-multiplex implementation sets this to 1MB).

### Closing a stream

Multiplex supports half-closed streams. Closing a stream closes it for writing
and closes the remote end for reading but allows writing in the other direction.

To close a stream, send a message with a zero length body and a `CloseReceiver`
(3) or `CloseInitiator` (4) flag (depending on whether or not the closer is the
receiver or sender). Writing to a stream after it has been closed should result
in an error. Reading from a remote-closed stream should return all data send
before closing the stream and then EOF thereafter.

### Resetting a stream

To immediately close a stream for both reading and writing, use reset. This
should generally only be used on error; during normal operation, both sides
should close instead.

To reset a stream, send a message with a zero length body and a `ResetReceiver`
(5) or `ResetInitiator` (6) flag. Reset must not block and must immediately
close both ends of the stream for both reading and writing. All current and
future reads and writes must return errors (*not* EOF) and any data queued or in
flight should be dropped.

## Implementation notes

If a stream is being actively written to, the reader must take care to keep up
with inbound data. Due to the lack of back pressure at the protocol level, the
implementation must handle slow readers by doing one or both of:

1. Blocking the entire connection until the offending stream is read.
2. Resetting the offending stream.

For example, the go-multiplex implementation blocks for a short period of time
and then resets the stream if necessary.

