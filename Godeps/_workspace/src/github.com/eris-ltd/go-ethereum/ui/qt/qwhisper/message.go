package qwhisper

import (
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/crypto"
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/ethutil"
	"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/whisper"
)

type Message struct {
	ref     *whisper.Message
	Flags   int32  `json:"flags"`
	Payload string `json:"payload"`
	From    string `json:"from"`
}

func ToQMessage(msg *whisper.Message) *Message {
	return &Message{
		ref:     msg,
		Flags:   int32(msg.Flags),
		Payload: "0x" + ethutil.Bytes2Hex(msg.Payload),
		From:    "0x" + ethutil.Bytes2Hex(crypto.FromECDSAPub(msg.Recover())),
	}
}
