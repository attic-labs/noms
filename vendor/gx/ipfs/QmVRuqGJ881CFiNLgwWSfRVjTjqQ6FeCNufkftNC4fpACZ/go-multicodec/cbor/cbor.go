package mc_cbor

import (
	"io"

	cbor "gx/ipfs/QmVQfuckfPnW5LGmgSWsJVJr6Xea1bWXD8sBixe8E9MQD6/cbor/go"

	mc "gx/ipfs/QmVRuqGJ881CFiNLgwWSfRVjTjqQ6FeCNufkftNC4fpACZ/go-multicodec"
)

var HeaderPath string
var Header []byte

func init() {
	HeaderPath = "/cbor"
	Header = mc.Header([]byte(HeaderPath))
}

type codec struct {
	mc bool
}

func Codec() mc.Codec {
	return &codec{
		mc: false,
	}
}

func Multicodec() mc.Multicodec {
	return &codec{
		mc: true,
	}
}

func (c *codec) Encoder(w io.Writer) mc.Encoder {
	return &encoder{
		w:   w,
		mc:  c.mc,
		enc: cbor.NewEncoder(w),
	}
}

func (c *codec) Decoder(r io.Reader) mc.Decoder {
	return &decoder{
		r:   r,
		mc:  c.mc,
		dec: cbor.NewDecoder(r),
	}
}

func (c *codec) Header() []byte {
	return Header
}

type encoder struct {
	w   io.Writer
	mc  bool
	enc *cbor.Encoder
}

type decoder struct {
	r   io.Reader
	mc  bool
	dec *cbor.Decoder
}

func (c *encoder) Encode(v interface{}) error {
	// if multicodec, write the header first
	if c.mc {
		if _, err := c.w.Write(Header); err != nil {
			return err
		}
	}
	return c.enc.Encode(v)
}

func (c *decoder) Decode(v interface{}) error {
	// if multicodec, consume the header first
	if c.mc {
		if err := mc.ConsumeHeader(c.r, Header); err != nil {
			return err
		}
	}
	return c.dec.Decode(v)
}
