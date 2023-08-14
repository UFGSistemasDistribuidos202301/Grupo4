package main

import (
	"encoding/gob"
)

type message struct {
	RequestTimestamp int
	Timestamp        int
	TypeMessage      int
	Id               int
	Address          string
}

func (m *message) getType() string {
	s := ""
	if m.TypeMessage == REPLY {
		s = "REPLY"
	} else if m.TypeMessage == REQUEST {
		s = "REQUEST"
	} else {
		s = "PERMISSION"
	}
	return s
}

func (m *message) receiveAndDecodeMessage(dec *gob.Decoder) error {
	if err := dec.Decode(m); err != nil {
		return err
	}
	return nil
}

func (m message) encodeAndSendMessage(enc *gob.Encoder) error {
	if err := enc.Encode(m); err != nil {
		return err
	}
	return nil
}
